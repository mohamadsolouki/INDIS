//! Bulletproofs range proof engine for INDIS anonymous testimony (Justice service).
//!
//! Ref: Bulletproofs paper <https://eprint.iacr.org/2017/1066>
//!
//! Proves that a committed value is in the range `[0, 2^n_bits)` without
//! revealing the value. Used for citizenship / age range proofs in
//! anonymous testimony flows.

use bulletproofs::{BulletproofGens, PedersenGens, RangeProof};
use curve25519_dalek::ristretto::CompressedRistretto;
use merlin::Transcript;
use serde::{Deserialize, Serialize};

use crate::{Proof, ProofGenerator, ProofSystem, ProofVerifier, VerificationResult, ZkError};

// ── Serialisable wire format ─────────────────────────────────────────────────

/// Wire format for a serialised range proof + Pedersen commitment.
#[derive(Serialize, Deserialize)]
struct BulletproofsProofData {
    /// Serialised `bulletproofs::RangeProof` bytes.
    proof_bytes: Vec<u8>,
    /// Compressed Ristretto commitment bytes (32 bytes).
    commitment_bytes: Vec<u8>,
}

/// Public input descriptor for a Bulletproofs range statement.
#[derive(Serialize, Deserialize)]
struct RangePublicInputs {
    /// Number of bits for the range — value must be in `[0, 2^n_bits)`.
    #[serde(default = "default_n_bits")]
    n_bits: u32,
    /// Human-readable context label (e.g. `"justice:testimony:citizenship"`).
    #[serde(default)]
    context: String,
}

fn default_n_bits() -> u32 {
    32
}

// ── Engine ───────────────────────────────────────────────────────────────────

/// Bulletproofs range proof engine.
///
/// Ref: Bulletproofs paper <https://eprint.iacr.org/2017/1066>
pub struct BulletproofsEngine;

impl ProofGenerator for BulletproofsEngine {
    /// Generate a Bulletproofs range proof.
    ///
    /// # Inputs
    /// - `private_inputs[0]` — little-endian `u64` bytes of the secret value.
    ///   If `private_inputs` is empty a development dummy value of `42u64` is used.
    /// - `public_inputs[0]` — JSON blob matching `{"n_bits": 32, "context": "..."}`.
    ///   Defaults to `n_bits = 32` when absent.
    fn generate(
        &self,
        _circuit_id: &str,
        private_inputs: &[Vec<u8>],
        public_inputs: &[Vec<u8>],
    ) -> Result<Proof, ZkError> {
        // ── Decode secret value ──────────────────────────────────────────────
        let secret: u64 = if private_inputs.is_empty() {
            // Dev path: no private input supplied.
            42u64
        } else {
            let raw = &private_inputs[0];
            if raw.len() < 8 {
                // Pad to 8 bytes (little-endian).
                let mut buf = [0u8; 8];
                let len = raw.len().min(8);
                buf[..len].copy_from_slice(&raw[..len]);
                u64::from_le_bytes(buf)
            } else {
                let mut buf = [0u8; 8];
                buf.copy_from_slice(&raw[..8]);
                u64::from_le_bytes(buf)
            }
        };

        // ── Decode public parameters ─────────────────────────────────────────
        let range_params: RangePublicInputs = if public_inputs.is_empty() {
            RangePublicInputs {
                n_bits: 32,
                context: String::new(),
            }
        } else {
            serde_json::from_slice(&public_inputs[0]).map_err(|e| {
                ZkError::GenerationFailed(format!("invalid public inputs JSON: {}", e))
            })?
        };

        let n_bits = range_params.n_bits as usize;
        // bulletproofs only supports power-of-two bit widths in {8,16,32,64}.
        if !matches!(n_bits, 8 | 16 | 32 | 64) {
            return Err(ZkError::GenerationFailed(format!(
                "n_bits must be 8, 16, 32, or 64; got {}",
                n_bits
            )));
        }

        // ── Range check ──────────────────────────────────────────────────────
        if n_bits < 64 {
            let max = 1u64 << n_bits;
            if secret >= max {
                return Err(ZkError::GenerationFailed(format!(
                    "secret value {} exceeds 2^{} range",
                    secret, n_bits
                )));
            }
        }

        // ── Prove ────────────────────────────────────────────────────────────
        let pc_gens = PedersenGens::default();
        let bp_gens = BulletproofGens::new(n_bits, 1);

        let blinding = curve25519_dalek::scalar::Scalar::random(
            &mut rand::thread_rng(),
        );

        let mut transcript = Transcript::new(b"INDIS-range-proof");

        let (proof, commitment) = RangeProof::prove_single(
            &bp_gens,
            &pc_gens,
            &mut transcript,
            secret,
            &blinding,
            n_bits,
        )
        .map_err(|e| ZkError::GenerationFailed(format!("bulletproofs prove_single failed: {:?}", e)))?;

        // ── Serialise ────────────────────────────────────────────────────────
        let proof_bytes = proof.to_bytes();
        let commitment_bytes = commitment.as_bytes().to_vec();

        let wire = BulletproofsProofData {
            proof_bytes,
            commitment_bytes: commitment_bytes.clone(),
        };
        let serialised = serde_json::to_vec(&wire)
            .map_err(|e| ZkError::GenerationFailed(format!("serialisation failed: {}", e)))?;

        Ok(Proof {
            system: ProofSystem::Bulletproofs,
            data: serialised,
            public_inputs: vec![commitment_bytes],
        })
    }
}

impl ProofVerifier for BulletproofsEngine {
    /// Verify a Bulletproofs range proof.
    ///
    /// `proof.data` must be a `BulletproofsProofData` JSON blob.
    /// `proof.public_inputs[0]` (or the first entry of the caller-supplied
    /// `public_inputs` slice) is the expected commitment bytes.
    fn verify(
        &self,
        proof: &Proof,
        _verification_key: &[u8],
        public_inputs: &[Vec<u8>],
    ) -> Result<VerificationResult, ZkError> {
        // ── Deserialise proof data ───────────────────────────────────────────
        let wire: BulletproofsProofData = serde_json::from_slice(&proof.data).map_err(|e| {
            ZkError::VerificationFailed(format!("failed to deserialise proof data: {}", e))
        })?;

        // ── Recover commitment from proof data or caller-supplied public inputs ──
        let commitment_bytes: &[u8] = if !public_inputs.is_empty() {
            &public_inputs[0]
        } else if !proof.public_inputs.is_empty() {
            &proof.public_inputs[0]
        } else {
            &wire.commitment_bytes
        };

        if commitment_bytes.len() != 32 {
            return Ok(VerificationResult {
                valid: false,
                system: ProofSystem::Bulletproofs,
            });
        }

        let mut arr = [0u8; 32];
        arr.copy_from_slice(commitment_bytes);
        let compressed = CompressedRistretto(arr);

        // ── Deserialise the RangeProof ───────────────────────────────────────
        let range_proof = match RangeProof::from_bytes(&wire.proof_bytes) {
            Ok(rp) => rp,
            Err(_) => {
                return Ok(VerificationResult {
                    valid: false,
                    system: ProofSystem::Bulletproofs,
                });
            }
        };

        // ── Default to 32-bit range (matches generation default) ─────────────
        let n_bits: usize = 32;
        let pc_gens = PedersenGens::default();
        let bp_gens = BulletproofGens::new(n_bits, 1);

        let mut transcript = Transcript::new(b"INDIS-range-proof");

        let valid = range_proof
            .verify_single(&bp_gens, &pc_gens, &mut transcript, &compressed, n_bits)
            .is_ok();

        Ok(VerificationResult {
            valid,
            system: ProofSystem::Bulletproofs,
        })
    }
}

// ── Tests ────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{ProofGenerator as _, ProofVerifier as _};

    fn make_public_inputs(n_bits: u32) -> Vec<Vec<u8>> {
        let json = format!(r#"{{"n_bits":{n_bits},"context":"test"}}"#);
        vec![json.into_bytes()]
    }

    fn secret_bytes(v: u64) -> Vec<Vec<u8>> {
        vec![v.to_le_bytes().to_vec()]
    }

    #[test]
    fn bulletproofs_round_trip_valid() {
        let engine = BulletproofsEngine;
        let private = secret_bytes(1000);
        let public = make_public_inputs(32);

        let proof = engine.generate("age_range", &private, &public).unwrap();

        let result = engine.verify(&proof, &[], &[]).unwrap();
        assert!(result.valid, "round-trip verification must succeed");
    }

    #[test]
    fn bulletproofs_rejects_wrong_commitment() {
        let engine = BulletproofsEngine;
        let private = secret_bytes(500);
        let public = make_public_inputs(32);

        let mut proof = engine.generate("age_range", &private, &public).unwrap();

        // Tamper with the commitment stored in public_inputs
        if let Some(commitment) = proof.public_inputs.get_mut(0) {
            commitment[0] ^= 0xFF;
        }

        let result = engine
            .verify(&proof, &[], &proof.public_inputs.clone())
            .unwrap();
        assert!(!result.valid, "tampered commitment must fail verification");
    }

    #[test]
    fn bulletproofs_rejects_oversized_value() {
        let engine = BulletproofsEngine;
        // 2^33 is out-of-range for n_bits=32
        let oversized: u64 = 1u64 << 33;
        let private = secret_bytes(oversized);
        let public = make_public_inputs(32);

        let result = engine.generate("age_range", &private, &public);
        assert!(
            result.is_err(),
            "oversized value must be rejected at generation time"
        );
    }
}
