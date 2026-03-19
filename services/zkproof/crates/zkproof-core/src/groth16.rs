use std::sync::OnceLock;

use ark_bn254::{Bn254, Fr};
use ark_ff::{BigInteger, PrimeField};
use ark_groth16::{prepare_verifying_key, Groth16, Proof as ArkProof, ProvingKey};
use ark_r1cs_std::{alloc::AllocVar, eq::EqGadget, fields::fp::FpVar};
use ark_relations::r1cs::{ConstraintSynthesizer, ConstraintSystemRef, SynthesisError};
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use base64::{engine::general_purpose, Engine};
use rand::SeedableRng;
use rand_chacha::ChaCha20Rng;
use serde::{Deserialize, Serialize};
use sha3::{Digest, Sha3_256};

use crate::{Proof, ProofGenerator, ProofSystem, ProofVerifier, VerificationResult, ZkError};

const GROTH16_DEV_PREFIX: &[u8] = b"indis:groth16:dev:v1";

#[derive(Clone)]
struct EqualityCircuit {
    witness: Option<Fr>,
    public_input: Option<Fr>,
}

impl ConstraintSynthesizer<Fr> for EqualityCircuit {
    fn generate_constraints(self, cs: ConstraintSystemRef<Fr>) -> Result<(), SynthesisError> {
        let witness = FpVar::<Fr>::new_witness(cs.clone(), || {
            self.witness.ok_or(SynthesisError::AssignmentMissing)
        })?;
        let public_input = FpVar::<Fr>::new_input(cs, || {
            self.public_input.ok_or(SynthesisError::AssignmentMissing)
        })?;

        witness.enforce_equal(&public_input)?;
        Ok(())
    }
}

struct Groth16Parameters {
    proving_key: ProvingKey<Bn254>,
    prepared_vk: ark_groth16::PreparedVerifyingKey<Bn254>,
}

fn initialize_params() -> Result<Groth16Parameters, ZkError> {
    let mut rng = ChaCha20Rng::from_seed([19u8; 32]);
    let setup_circuit = EqualityCircuit {
        witness: Some(Fr::from(1u64)),
        public_input: Some(Fr::from(1u64)),
    };

    let proving_key = Groth16::<Bn254>::generate_random_parameters_with_reduction(
        setup_circuit,
        &mut rng,
    )
    .map_err(|e| ZkError::GenerationFailed(format!("groth16 setup failed: {e}")))?;

    let prepared_vk = prepare_verifying_key(&proving_key.vk);

    Ok(Groth16Parameters {
        proving_key,
        prepared_vk,
    })
}

fn params() -> Result<&'static Groth16Parameters, ZkError> {
    static PARAMS: OnceLock<Result<Groth16Parameters, ZkError>> = OnceLock::new();
    PARAMS
        .get_or_init(initialize_params)
        .as_ref()
        .map_err(|e| ZkError::GenerationFailed(format!("groth16 params unavailable: {e}")))
}

fn field_from_input(input: &[u8]) -> Fr {
    let mut hasher = Sha3_256::new();
    hasher.update(GROTH16_DEV_PREFIX);
    hasher.update((input.len() as u64).to_le_bytes());
    hasher.update(input);
    let digest = hasher.finalize();
    Fr::from_le_bytes_mod_order(&digest)
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Groth16ProofEnvelope {
    version: u8,
    circuit_id: String,
    proof_b64: String,
    public_input_b64: String,
}

/// Development Groth16 engine.
///
/// Uses a real Groth16 prover/verifier from arkworks with deterministic
/// development setup parameters. The circuit is intentionally minimal and proves
/// equality between a private witness and a public input derived from the input.
#[derive(Debug, Default, Clone)]
pub struct DevelopmentGroth16Engine;

impl DevelopmentGroth16Engine {
    fn encode_proof(
        circuit_id: &str,
        proof: &ArkProof<Bn254>,
        public_input: Fr,
    ) -> Result<Vec<u8>, ZkError> {
        let mut proof_bytes = Vec::new();
        proof
            .serialize_compressed(&mut proof_bytes)
            .map_err(|e| ZkError::GenerationFailed(format!("serialize proof failed: {e}")))?;

        let public_bytes = public_input.into_bigint().to_bytes_le();

        let envelope = Groth16ProofEnvelope {
            version: 1,
            circuit_id: circuit_id.to_string(),
            proof_b64: general_purpose::STANDARD.encode(proof_bytes),
            public_input_b64: general_purpose::STANDARD.encode(public_bytes),
        };

        serde_json::to_vec(&envelope)
            .map_err(|e| ZkError::GenerationFailed(format!("encode envelope failed: {e}")))
    }

    fn decode_proof(data: &[u8]) -> Result<(Groth16ProofEnvelope, ArkProof<Bn254>, Fr), ZkError> {
        let envelope: Groth16ProofEnvelope = serde_json::from_slice(data)
            .map_err(|e| ZkError::VerificationFailed(format!("invalid envelope: {e}")))?;

        if envelope.version != 1 {
            return Err(ZkError::VerificationFailed(
                "unsupported groth16 proof version".to_string(),
            ));
        }

        let proof_bytes = general_purpose::STANDARD.decode(&envelope.proof_b64).map_err(|e| {
            ZkError::VerificationFailed(format!("invalid envelope proof encoding: {e}"))
        })?;
        let public_input_bytes = general_purpose::STANDARD
            .decode(&envelope.public_input_b64)
            .map_err(|e| {
            ZkError::VerificationFailed(format!("invalid envelope public input encoding: {e}"))
        })?;

        let proof = ArkProof::<Bn254>::deserialize_compressed(proof_bytes.as_slice())
            .map_err(|e| ZkError::VerificationFailed(format!("invalid groth16 proof bytes: {e}")))?;
        let public_input = Fr::from_le_bytes_mod_order(&public_input_bytes);

        Ok((envelope, proof, public_input))
    }
}

impl ProofGenerator for DevelopmentGroth16Engine {
    fn generate(
        &self,
        circuit_id: &str,
        private_inputs: &[Vec<u8>],
        _public_inputs: &[Vec<u8>],
    ) -> Result<Proof, ZkError> {
        if circuit_id.trim().is_empty() {
            return Err(ZkError::InvalidCircuit(
                "circuit_id cannot be empty".to_string(),
            ));
        }

        let private_input = private_inputs
            .first()
            .ok_or_else(|| ZkError::GenerationFailed("private input is required".to_string()))?;

        let witness_value = field_from_input(private_input);
        let circuit = EqualityCircuit {
            witness: Some(witness_value),
            public_input: Some(witness_value),
        };

        let mut rng = ChaCha20Rng::from_seed([7u8; 32]);
        let proving_key = &params()?.proving_key;
        let proof = Groth16::<Bn254>::create_random_proof_with_reduction(circuit, proving_key, &mut rng)
            .map_err(|e| ZkError::GenerationFailed(format!("groth16 proof generation failed: {e}")))?;

        let encoded = Self::encode_proof(circuit_id, &proof, witness_value)?;

        Ok(Proof {
            system: ProofSystem::Groth16,
            data: encoded,
            public_inputs: vec![],
        })
    }
}

impl ProofVerifier for DevelopmentGroth16Engine {
    fn verify(
        &self,
        proof: &Proof,
        _verification_key: &[u8],
        public_inputs: &[Vec<u8>],
    ) -> Result<VerificationResult, ZkError> {
        if !matches!(proof.system, ProofSystem::Groth16) {
            return Err(ZkError::UnsupportedSystem(
                "proof is not a Groth16 proof".to_string(),
            ));
        }

        let (_envelope, ark_proof, embedded_public_input) = Self::decode_proof(&proof.data)?;

        if let Some(first_public_input) = public_inputs.first() {
            let external_public = field_from_input(first_public_input);
            if external_public != embedded_public_input {
                return Ok(VerificationResult {
                    valid: false,
                    system: ProofSystem::Groth16,
                });
            }
        }

        let result = Groth16::<Bn254>::verify_proof(
            &params()?.prepared_vk,
            &ark_proof,
            &[embedded_public_input],
        )
        .map_err(|e| ZkError::VerificationFailed(format!("groth16 verification failed: {e}")))?;

        Ok(VerificationResult {
            valid: result,
            system: ProofSystem::Groth16,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::DevelopmentGroth16Engine;
    use crate::{ProofGenerator, ProofSystem, ProofVerifier};

    #[test]
    fn groth16_round_trip_validates() {
        let engine = DevelopmentGroth16Engine;
        let input = b"holder:did:indis:test".to_vec();

        let proof = engine
            .generate("credential_validity", &[input.clone()], &[])
            .expect("proof should be generated");

        assert!(matches!(proof.system, ProofSystem::Groth16));

        let result = engine
            .verify(&proof, &[], &[input])
            .expect("verification should run");

        assert!(result.valid);
    }

    #[test]
    fn groth16_rejects_mismatched_public_input() {
        let engine = DevelopmentGroth16Engine;
        let proof = engine
            .generate("credential_validity", &[b"input-a".to_vec()], &[])
            .expect("proof should be generated");

        let result = engine
            .verify(&proof, &[], &[b"input-b".to_vec()])
            .expect("verification should run");

        assert!(!result.valid);
    }

    #[test]
    fn groth16_rejects_tampered_proof() {
        let engine = DevelopmentGroth16Engine;
        let mut proof = engine
            .generate("credential_validity", &[b"input-a".to_vec()], &[])
            .expect("proof should be generated");

        // Break JSON envelope so decoding fails.
        proof.data[0] = b'X';

        let result = engine.verify(&proof, &[], &[]);
        assert!(result.is_err());
    }
}
