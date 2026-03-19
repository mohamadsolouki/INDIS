// WebAuthn integration for INDIS Citizen PWA
// PRD FR-001.4: Private keys SHALL be generated on citizen's device only.
// WebAuthn Authenticator binds the key to the device TPM / secure enclave.

const RP_ID   = window.location.hostname;
const RP_NAME = 'INDIS — هویت دیجیتال ملی';
const TIMEOUT = 60_000; // 60s

/** Create a new WebAuthn credential (enrollment / first-time setup). */
export async function registerWebAuthn(userId: string, displayName: string): Promise<PublicKeyCredential> {
  const challenge = crypto.getRandomValues(new Uint8Array(32));
  const userIdBytes = new TextEncoder().encode(userId);

  const options: PublicKeyCredentialCreationOptions = {
    rp: { id: RP_ID, name: RP_NAME },
    user: { id: userIdBytes, name: userId, displayName },
    challenge,
    pubKeyCredParams: [
      { alg: -8,  type: 'public-key' }, // Ed25519
      { alg: -7,  type: 'public-key' }, // ES256 (P-256)
      { alg: -257, type: 'public-key' }, // RS256 fallback
    ],
    authenticatorSelection: {
      authenticatorAttachment: 'platform', // device-bound
      userVerification: 'required',
      residentKey: 'preferred',
    },
    attestation: 'none',
    timeout: TIMEOUT,
  };

  const credential = await navigator.credentials.create({ publicKey: options });
  if (!credential) throw new Error('WebAuthn credential creation failed');
  return credential as PublicKeyCredential;
}

/** Authenticate with an existing WebAuthn credential. Returns assertion for JWT exchange. */
export async function authenticateWebAuthn(credentialId?: string): Promise<PublicKeyCredential> {
  const challenge = crypto.getRandomValues(new Uint8Array(32));

  const options: PublicKeyCredentialRequestOptions = {
    rpId: RP_ID,
    challenge,
    userVerification: 'required',
    timeout: TIMEOUT,
    allowCredentials: credentialId
      ? [{ id: base64urlToBuffer(credentialId), type: 'public-key' }]
      : [],
  };

  const assertion = await navigator.credentials.get({ publicKey: options });
  if (!assertion) throw new Error('WebAuthn authentication failed');
  return assertion as PublicKeyCredential;
}

/** Check if WebAuthn is available on this device. */
export async function isWebAuthnAvailable(): Promise<boolean> {
  return (
    typeof window.PublicKeyCredential !== 'undefined' &&
    (await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable())
  );
}

/** Retrieve stored WebAuthn credential ID from localStorage. */
export function getStoredCredentialId(): string | null {
  return localStorage.getItem('indis_webauthn_cred');
}

/** Persist WebAuthn credential ID. */
export function storeCredentialId(credId: string): void {
  localStorage.setItem('indis_webauthn_cred', credId);
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
  const binary = atob(base64);
  const buffer = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) buffer[i] = binary.charCodeAt(i);
  return buffer.buffer;
}

export function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}
