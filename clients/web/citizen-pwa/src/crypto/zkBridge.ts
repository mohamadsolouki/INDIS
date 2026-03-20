/**
 * WASM ZK Bridge — offline zero-knowledge proof generation for the Citizen PWA.
 *
 * This module abstracts over two proof generation strategies:
 *   1. Online: delegates to the INDIS gateway (/v1/verifier/respond).
 *   2. Offline: generates proofs locally via a Rust/WASM bundle
 *      (services/zkproof compiled to WASM with wasm-pack).
 *
 * The WASM bundle is loaded lazily and cached in the module scope so it is
 * only fetched once per page lifetime. A 72-hour revocation list (PRD FR-006)
 * must be cached before offline proofs are considered valid.
 *
 * Production integration note:
 *   Build the WASM target with:
 *     cd services/zkproof && wasm-pack build --target web --out-dir ../../clients/web/citizen-pwa/public/wasm
 *   Then set VITE_ZK_WASM_PATH=/wasm/indis_zkproof_bg.wasm in your .env.
 */

export type ProofSystem = 'groth16' | 'plonk' | 'bulletproofs';

export interface ZKProofRequest {
  /** W3C DID of the holder. */
  did: string;
  /** Credential ID to generate a proof for. */
  credentialId: string;
  /** The claim type to disclose (e.g. "age_gte_18", "is_citizen"). */
  claimType: string;
  /** Proof system to use — defaults to groth16. */
  proofSystem?: ProofSystem;
  /** Unix timestamp (seconds) for the proof's validity window. */
  validUntil?: number;
}

export interface ZKProof {
  proofSystem: ProofSystem;
  proof: string;        // base64-encoded proof bytes
  publicSignals: string[]; // public inputs (boolean claims only — PRD FR-013)
  claimType: string;
  did: string;
  generatedAt: number; // Unix ms
  isOffline: boolean;
}

// Module-scoped WASM instance cache.
let wasmModule: WasmZKModule | null = null;
let wasmLoadPromise: Promise<WasmZKModule | null> | null = null;

interface WasmZKModule {
  generate_groth16_proof(did: string, credential_id: string, claim_type: string): string;
  generate_plonk_proof(did: string, credential_id: string, claim_type: string): string;
}

/**
 * Attempts to load the WASM ZK module from the bundled path.
 * Returns null if WASM is not available (network offline, bundle missing).
 */
async function loadWasmModule(): Promise<WasmZKModule | null> {
  if (wasmModule) return wasmModule;
  if (wasmLoadPromise) return wasmLoadPromise;

  wasmLoadPromise = (async () => {
    const wasmPath = import.meta.env?.VITE_ZK_WASM_PATH as string | undefined
      ?? '/wasm/indis_zkproof.js';

    try {
      // Dynamic import of the wasm-pack generated JS glue.
      // The actual WASM binary is fetched by the glue module.
      const mod = await import(/* @vite-ignore */ wasmPath) as Record<string, unknown>;
      if (typeof mod.default === 'function') {
        // wasm-pack init function — call it to instantiate WASM.
        await (mod.default as () => Promise<void>)();
      }
      wasmModule = mod as unknown as WasmZKModule;
      return wasmModule;
    } catch (err) {
      // WASM bundle not yet built or unavailable offline.
      // Fall back to dev mock (see generateOfflineProofMock).
      console.warn('[zkBridge] WASM module unavailable, using mock proof:', err);
      return null;
    }
  })();

  return wasmLoadPromise;
}

/**
 * Generates a mock ZK proof for development / pre-WASM environments.
 * The mock proof is recognisable by its "MOCK:" prefix and must never be
 * accepted by a production verifier.
 */
function generateOfflineProofMock(req: ZKProofRequest): ZKProof {
  const payload = `${req.did}|${req.credentialId}|${req.claimType}|${Date.now()}`;
  const mockProofBytes = btoa(payload);
  return {
    proofSystem: req.proofSystem ?? 'groth16',
    proof: `MOCK:${mockProofBytes}`,
    // Public signals: only the boolean result of the claim (PRD FR-013).
    publicSignals: ['1'], // "1" = claim is satisfied
    claimType: req.claimType,
    did: req.did,
    generatedAt: Date.now(),
    isOffline: true,
  };
}

/**
 * Generates a ZK proof offline using the WASM bundle.
 * Falls back to a mock proof when the bundle is not loaded.
 */
async function generateOfflineProof(req: ZKProofRequest): Promise<ZKProof> {
  const wasm = await loadWasmModule();
  if (!wasm) {
    return generateOfflineProofMock(req);
  }

  try {
    let proofJson: string;
    switch (req.proofSystem ?? 'groth16') {
      case 'plonk':
        proofJson = wasm.generate_plonk_proof(req.did, req.credentialId, req.claimType);
        break;
      case 'groth16':
      default:
        proofJson = wasm.generate_groth16_proof(req.did, req.credentialId, req.claimType);
        break;
    }

    const parsed = JSON.parse(proofJson) as { proof: string; publicSignals: string[] };
    return {
      proofSystem: req.proofSystem ?? 'groth16',
      proof: parsed.proof,
      publicSignals: parsed.publicSignals,
      claimType: req.claimType,
      did: req.did,
      generatedAt: Date.now(),
      isOffline: true,
    };
  } catch (err) {
    console.error('[zkBridge] WASM proof generation failed, falling back to mock:', err);
    return generateOfflineProofMock(req);
  }
}

/**
 * Checks whether the 72-hour revocation cache (PRD FR-006) is fresh.
 * Offline proofs must be refused if the cache is stale to prevent
 * presenting proofs for revoked credentials.
 */
export function isRevocationCacheFresh(): boolean {
  try {
    const raw = localStorage.getItem('indis_revocation_cache_ts');
    if (!raw) return false;
    const cachedAt = parseInt(raw, 10);
    const ageMs = Date.now() - cachedAt;
    const maxAgeMs = 72 * 60 * 60 * 1000; // 72 hours — PRD FR-006
    return ageMs < maxAgeMs;
  } catch {
    return false;
  }
}

/**
 * generateZKProof is the primary entry point for the Verify page.
 *
 * Strategy:
 *   - If online → POST to gateway (server-side Groth16/STARK).
 *   - If offline AND revocation cache is fresh → generate via WASM locally.
 *   - If offline AND cache is stale → throw RevocationCacheStaleError.
 */
export class RevocationCacheStaleError extends Error {
  constructor() {
    super(
      'Revocation cache is older than 72 hours (PRD FR-006). ' +
      'Connect to the internet to refresh before generating offline proofs.'
    );
    this.name = 'RevocationCacheStaleError';
  }
}

export async function generateZKProof(
  req: ZKProofRequest,
  options: { forceOffline?: boolean } = {}
): Promise<ZKProof> {
  const isOnline = navigator.onLine && !options.forceOffline;

  if (isOnline) {
    // Online path: delegate to gateway — proof generated server-side.
    // The gateway call is handled by the Verify page via http.post(); this
    // function returns a synthetic ZKProof wrapper for UI consistency.
    return {
      proofSystem: req.proofSystem ?? 'groth16',
      proof: '', // gateway returns the proof in the verify response
      publicSignals: [],
      claimType: req.claimType,
      did: req.did,
      generatedAt: Date.now(),
      isOffline: false,
    };
  }

  // Offline path.
  if (!isRevocationCacheFresh()) {
    throw new RevocationCacheStaleError();
  }

  return generateOfflineProof(req);
}

/**
 * Encodes a ZKProof as a compact JSON string suitable for QR code display.
 * Only includes public signals (boolean claims) — never raw identity attributes.
 * This implements the PRD FR-013 "boolean-only verifier result" requirement.
 */
export function encodeProofForQR(proof: ZKProof): string {
  return JSON.stringify({
    v: 1,
    ps: proof.proofSystem,
    did: proof.did,
    claim: proof.claimType,
    sig: proof.publicSignals,
    ts: proof.generatedAt,
    offline: proof.isOffline,
    // Proof bytes omitted from QR — verifier fetches full proof by DID+ts.
  });
}
