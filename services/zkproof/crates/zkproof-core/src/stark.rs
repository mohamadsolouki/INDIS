use sha3::{Digest, Sha3_256};

use crate::{Proof, ProofGenerator, ProofSystem, ProofVerifier, VerificationResult, ZkError};

const DEV_STARK_PREFIX: &[u8] = b"indis:stark:dev:v1";
const DEV_STARK_PROOF_LEN: usize = 33;

/// Development baseline STARK engine.
///
/// This implementation is deterministic and hash-based to keep Tier 2
/// integration work moving. It is not cryptographically sound and will be
/// replaced by a real Winterfell prover/verifier in production.
#[derive(Debug, Default, Clone)]
pub struct DevelopmentStarkEngine;

impl DevelopmentStarkEngine {
    fn digest(circuit_id: &str, verification_key: &[u8], public_inputs: &[Vec<u8>]) -> [u8; 32] {
        let mut hasher = Sha3_256::new();
        hasher.update(DEV_STARK_PREFIX);
        hasher.update(circuit_id.as_bytes());
        hasher.update(verification_key);

        for input in public_inputs {
            hasher.update((input.len() as u64).to_le_bytes());
            hasher.update(input);
        }

        let digest = hasher.finalize();
        let mut out = [0u8; 32];
        out.copy_from_slice(&digest[..32]);
        out
    }

    fn encode_proof(digest: [u8; 32]) -> Vec<u8> {
        let mut data = Vec::with_capacity(DEV_STARK_PROOF_LEN);
        data.push(1); // version byte
        data.extend_from_slice(&digest);
        data
    }
}

impl ProofGenerator for DevelopmentStarkEngine {
    fn generate(
        &self,
        circuit_id: &str,
        _private_inputs: &[Vec<u8>],
        public_inputs: &[Vec<u8>],
    ) -> Result<Proof, ZkError> {
        if circuit_id.trim().is_empty() {
            return Err(ZkError::InvalidCircuit(
                "circuit_id cannot be empty".to_string(),
            ));
        }

        if public_inputs.is_empty() {
            return Err(ZkError::GenerationFailed(
                "public_inputs are required for STARK proofs".to_string(),
            ));
        }

        let digest = Self::digest(circuit_id, &[], public_inputs);
        let data = Self::encode_proof(digest);

        Ok(Proof {
            system: ProofSystem::Stark,
            data,
            public_inputs: public_inputs.to_vec(),
        })
    }
}

impl ProofVerifier for DevelopmentStarkEngine {
    fn verify(
        &self,
        proof: &Proof,
        verification_key: &[u8],
        public_inputs: &[Vec<u8>],
    ) -> Result<VerificationResult, ZkError> {
        if !matches!(proof.system, ProofSystem::Stark) {
            return Err(ZkError::UnsupportedSystem(
                "proof is not a STARK proof".to_string(),
            ));
        }

        if proof.data.len() != DEV_STARK_PROOF_LEN {
            return Ok(VerificationResult {
                valid: false,
                system: ProofSystem::Stark,
            });
        }

        if proof.data[0] != 1 {
            return Ok(VerificationResult {
                valid: false,
                system: ProofSystem::Stark,
            });
        }

        let expected = Self::digest("voter_eligibility", verification_key, public_inputs);
        let provided = &proof.data[1..33];

        Ok(VerificationResult {
            valid: provided == expected,
            system: ProofSystem::Stark,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::DevelopmentStarkEngine;
    use crate::{ProofGenerator, ProofSystem, ProofVerifier};

    #[test]
    fn generate_rejects_empty_public_inputs() {
        let engine = DevelopmentStarkEngine;
        let result = engine.generate("voter_eligibility", &[], &[]);
        assert!(result.is_err());
    }

    #[test]
    fn stark_proof_round_trip_validates() {
        let engine = DevelopmentStarkEngine;
        let public_input = b"{\"nullifier\":\"abc\"}".to_vec();
        let proof = engine
            .generate("voter_eligibility", &[], &[public_input.clone()])
            .expect("proof should be generated");

        assert!(matches!(proof.system, ProofSystem::Stark));

        let result = engine
            .verify(
                &proof,
                &[],
                &[public_input],
            )
            .expect("verification should succeed");

        assert!(result.valid);
    }

    #[test]
    fn stark_proof_rejects_mismatched_public_input() {
        let engine = DevelopmentStarkEngine;
        let public_input = b"{\"nullifier\":\"abc\"}".to_vec();
        let proof = engine
            .generate("voter_eligibility", &[], &[public_input])
            .expect("proof should be generated");

        let result = engine
            .verify(
                &proof,
                &[],
                &[b"{\"nullifier\":\"different\"}".to_vec()],
            )
            .expect("verification should run");

        assert!(!result.valid);
    }
}
