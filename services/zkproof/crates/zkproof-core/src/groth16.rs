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

use crate::circuits::{
    canonical_circuit_id, AgeRangeCircuit, CredentialValidityCircuit, VoterEligibilityCircuit,
    VOTER_AGE_THRESHOLD,
};
use crate::{Proof, ProofGenerator, ProofSystem, ProofVerifier, VerificationResult, ZkError};

const GROTH16_DEV_PREFIX: &[u8] = b"indis:groth16:dev:v1";

// ── Fallback equality circuit (for unknown circuit IDs) ──────────────────────

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

// ── Per-circuit proving key cache ────────────────────────────────────────────

struct Groth16Parameters {
    proving_key: ProvingKey<Bn254>,
    prepared_vk: ark_groth16::PreparedVerifyingKey<Bn254>,
}

macro_rules! circuit_params {
    ($fn_name:ident, $seed:expr, $circuit:expr) => {
        fn $fn_name() -> Result<&'static Groth16Parameters, ZkError> {
            static PARAMS: OnceLock<Result<Groth16Parameters, ZkError>> = OnceLock::new();
            PARAMS
                .get_or_init(|| {
                    let mut rng = ChaCha20Rng::from_seed($seed);
                    let pk = Groth16::<Bn254>::generate_random_parameters_with_reduction(
                        $circuit,
                        &mut rng,
                    )
                    .map_err(|e| ZkError::GenerationFailed(format!("groth16 setup: {e}")))?;
                    let pvk = prepare_verifying_key(&pk.vk);
                    Ok(Groth16Parameters { proving_key: pk, prepared_vk: pvk })
                })
                .as_ref()
                .map_err(|e| ZkError::GenerationFailed(format!("params unavailable: {e}")))
        }
    };
}

circuit_params!(
    params_age_proof,
    [11u8; 32],
    AgeRangeCircuit { age: Some(18), threshold: 18 }
);

circuit_params!(
    params_voter_eligibility,
    [22u8; 32],
    VoterEligibilityCircuit {
        age: Some(18),
        credential_hash_low: Some(1),
        not_excluded: Some(true),
    }
);

circuit_params!(
    params_credential_validity,
    [33u8; 32],
    CredentialValidityCircuit {
        issued_at: Some(1),
        expiry_at: Some(2),
        not_revoked: Some(true),
        current_time: 1,
    }
);

circuit_params!(
    params_equality,
    [19u8; 32],
    EqualityCircuit { witness: Some(Fr::from(1u64)), public_input: Some(Fr::from(1u64)) }
);

fn field_from_input(input: &[u8]) -> Fr {
    let mut hasher = Sha3_256::new();
    hasher.update(GROTH16_DEV_PREFIX);
    hasher.update((input.len() as u64).to_le_bytes());
    hasher.update(input);
    let digest = hasher.finalize();
    Fr::from_le_bytes_mod_order(&digest)
}

// ── Input parsing helpers ────────────────────────────────────────────────────

/// Parse private inputs JSON for age_proof circuit.
/// Expected JSON: `{"age": 35}` or `{"age": 35, "threshold": 18}`
fn parse_age_proof_input(
    private_inputs: &[Vec<u8>],
) -> Result<AgeRangeCircuit, ZkError> {
    let raw = private_inputs
        .first()
        .ok_or_else(|| ZkError::GenerationFailed("age_proof requires private input".into()))?;
    let v: serde_json::Value = serde_json::from_slice(raw)
        .map_err(|e| ZkError::GenerationFailed(format!("age_proof input JSON: {e}")))?;
    let age = v["age"]
        .as_u64()
        .ok_or_else(|| ZkError::GenerationFailed("age_proof: 'age' field required".into()))?;
    let threshold = v["threshold"].as_u64().unwrap_or(18);
    Ok(AgeRangeCircuit { age: Some(age), threshold })
}

/// Parse private inputs JSON for voter_eligibility circuit.
/// Expected JSON: `{"age": 25, "credential_hash_low": 12345678, "not_excluded": true}`
fn parse_voter_eligibility_input(
    private_inputs: &[Vec<u8>],
) -> Result<VoterEligibilityCircuit, ZkError> {
    let raw = private_inputs.first().ok_or_else(|| {
        ZkError::GenerationFailed("voter_eligibility requires private input".into())
    })?;
    let v: serde_json::Value = serde_json::from_slice(raw).map_err(|e| {
        ZkError::GenerationFailed(format!("voter_eligibility input JSON: {e}"))
    })?;
    let age = v["age"]
        .as_u64()
        .ok_or_else(|| ZkError::GenerationFailed("voter_eligibility: 'age' required".into()))?;
    let credential_hash_low = v["credential_hash_low"].as_u64().unwrap_or(0);
    let not_excluded = v["not_excluded"].as_bool().unwrap_or(true);
    Ok(VoterEligibilityCircuit {
        age: Some(age),
        credential_hash_low: Some(credential_hash_low),
        not_excluded: Some(not_excluded),
    })
}

/// Parse private inputs JSON for credential_validity circuit.
/// Expected JSON: `{"issued_at": 1700000000, "expiry_at": 1800000000, "not_revoked": true}`
/// `current_time` comes from the public inputs.
fn parse_credential_validity_input(
    private_inputs: &[Vec<u8>],
    public_inputs: &[Vec<u8>],
) -> Result<CredentialValidityCircuit, ZkError> {
    let raw = private_inputs.first().ok_or_else(|| {
        ZkError::GenerationFailed("credential_validity requires private input".into())
    })?;
    let v: serde_json::Value = serde_json::from_slice(raw).map_err(|e| {
        ZkError::GenerationFailed(format!("credential_validity input JSON: {e}"))
    })?;
    let issued_at = v["issued_at"].as_u64().ok_or_else(|| {
        ZkError::GenerationFailed("credential_validity: 'issued_at' required".into())
    })?;
    let expiry_at = v["expiry_at"].as_u64().ok_or_else(|| {
        ZkError::GenerationFailed("credential_validity: 'expiry_at' required".into())
    })?;
    let not_revoked = v["not_revoked"].as_bool().unwrap_or(true);
    // current_time from public inputs JSON or default to 0.
    let current_time = if let Some(pub_raw) = public_inputs.first() {
        serde_json::from_slice::<serde_json::Value>(pub_raw)
            .ok()
            .and_then(|pv| pv["current_time"].as_u64())
            .unwrap_or(0)
    } else {
        0
    };
    Ok(CredentialValidityCircuit {
        issued_at: Some(issued_at),
        expiry_at: Some(expiry_at),
        not_revoked: Some(not_revoked),
        current_time,
    })
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Groth16ProofEnvelope {
    version: u8,
    circuit_id: String,
    proof_b64: String,
    /// All public inputs as base64-encoded little-endian field element bytes.
    public_inputs_b64: Vec<String>,
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
        public_inputs: &[Fr],
    ) -> Result<Vec<u8>, ZkError> {
        let mut proof_bytes = Vec::new();
        proof
            .serialize_compressed(&mut proof_bytes)
            .map_err(|e| ZkError::GenerationFailed(format!("serialize proof failed: {e}")))?;

        let encoded_inputs: Vec<String> = public_inputs
            .iter()
            .map(|pi| general_purpose::STANDARD.encode(pi.into_bigint().to_bytes_le()))
            .collect();

        let envelope = Groth16ProofEnvelope {
            version: 1,
            circuit_id: circuit_id.to_string(),
            proof_b64: general_purpose::STANDARD.encode(proof_bytes),
            public_inputs_b64: encoded_inputs,
        };

        serde_json::to_vec(&envelope)
            .map_err(|e| ZkError::GenerationFailed(format!("encode envelope failed: {e}")))
    }

    fn decode_proof(
        data: &[u8],
    ) -> Result<(Groth16ProofEnvelope, ArkProof<Bn254>, Vec<Fr>), ZkError> {
        let envelope: Groth16ProofEnvelope = serde_json::from_slice(data)
            .map_err(|e| ZkError::VerificationFailed(format!("invalid envelope: {e}")))?;

        if envelope.version != 1 {
            return Err(ZkError::VerificationFailed(
                "unsupported groth16 proof version".to_string(),
            ));
        }

        let proof_bytes =
            general_purpose::STANDARD.decode(&envelope.proof_b64).map_err(|e| {
                ZkError::VerificationFailed(format!("invalid envelope proof encoding: {e}"))
            })?;

        let proof = ArkProof::<Bn254>::deserialize_compressed(proof_bytes.as_slice())
            .map_err(|e| {
                ZkError::VerificationFailed(format!("invalid groth16 proof bytes: {e}"))
            })?;

        let public_inputs: Vec<Fr> = envelope
            .public_inputs_b64
            .iter()
            .map(|b64| {
                let bytes = general_purpose::STANDARD.decode(b64).map_err(|e| {
                    ZkError::VerificationFailed(format!("invalid public input encoding: {e}"))
                })?;
                Ok(Fr::from_le_bytes_mod_order(&bytes))
            })
            .collect::<Result<_, ZkError>>()?;

        Ok((envelope, proof, public_inputs))
    }
}

impl ProofGenerator for DevelopmentGroth16Engine {
    fn generate(
        &self,
        circuit_id: &str,
        private_inputs: &[Vec<u8>],
        public_inputs: &[Vec<u8>],
    ) -> Result<Proof, ZkError> {
        if circuit_id.trim().is_empty() {
            return Err(ZkError::InvalidCircuit("circuit_id cannot be empty".into()));
        }

        let mut rng = ChaCha20Rng::from_seed([7u8; 32]);

        // Dispatch to the correct real circuit based on circuit_id.
        let (ark_proof, pub_inputs_fp): (ArkProof<Bn254>, Vec<Fr>) =
            match canonical_circuit_id(circuit_id) {
                Some("age_proof") => {
                    let circuit = parse_age_proof_input(private_inputs)?;
                    let threshold = circuit.threshold;
                    let pk = &params_age_proof()?.proving_key;
                    let proof = Groth16::<Bn254>::create_random_proof_with_reduction(
                        circuit, pk, &mut rng,
                    )
                    .map_err(|e| ZkError::GenerationFailed(format!("age_proof: {e}")))?;
                    (proof, vec![Fr::from(threshold)])
                }
                Some("voter_eligibility") => {
                    let circuit = parse_voter_eligibility_input(private_inputs)?;
                    let cred_hash_low = circuit.credential_hash_low.unwrap_or(0);
                    let pk = &params_voter_eligibility()?.proving_key;
                    let proof = Groth16::<Bn254>::create_random_proof_with_reduction(
                        circuit, pk, &mut rng,
                    )
                    .map_err(|e| ZkError::GenerationFailed(format!("voter_eligibility: {e}")))?;
                    // Public inputs match circuit new_input order: [threshold, cred_commitment].
                    (proof, vec![Fr::from(VOTER_AGE_THRESHOLD), Fr::from(cred_hash_low)])
                }
                Some("credential_validity") => {
                    let circuit =
                        parse_credential_validity_input(private_inputs, public_inputs)?;
                    let current_time = circuit.current_time;
                    let pk = &params_credential_validity()?.proving_key;
                    let proof = Groth16::<Bn254>::create_random_proof_with_reduction(
                        circuit, pk, &mut rng,
                    )
                    .map_err(|e| ZkError::GenerationFailed(format!("credential_validity: {e}")))?;
                    (proof, vec![Fr::from(current_time)])
                }
                _ => {
                    let raw = private_inputs.first().ok_or_else(|| {
                        ZkError::GenerationFailed("private input is required".into())
                    })?;
                    let witness_value = field_from_input(raw);
                    let circuit = EqualityCircuit {
                        witness: Some(witness_value),
                        public_input: Some(witness_value),
                    };
                    let pk = &params_equality()?.proving_key;
                    let proof = Groth16::<Bn254>::create_random_proof_with_reduction(
                        circuit, pk, &mut rng,
                    )
                    .map_err(|e| ZkError::GenerationFailed(format!("equality: {e}")))?;
                    (proof, vec![witness_value])
                }
            };

        let encoded = Self::encode_proof(circuit_id, &ark_proof, &pub_inputs_fp)?;
        Ok(Proof { system: ProofSystem::Groth16, data: encoded, public_inputs: vec![] })
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
            return Err(ZkError::UnsupportedSystem("proof is not a Groth16 proof".into()));
        }

        let (envelope, ark_proof, embedded_public_inputs) = Self::decode_proof(&proof.data)?;

        // If the caller supplies external public inputs (equality-circuit style),
        // check that the first embedded input matches the hash of the first external input.
        if let Some(first_external) = public_inputs.first() {
            let external_fp = field_from_input(first_external);
            if embedded_public_inputs.first() != Some(&external_fp) {
                return Ok(VerificationResult { valid: false, system: ProofSystem::Groth16 });
            }
        }

        // Select the correct prepared verifying key based on the circuit_id in the envelope.
        let pvk = match canonical_circuit_id(&envelope.circuit_id) {
            Some("age_proof") => &params_age_proof()?.prepared_vk,
            Some("voter_eligibility") => &params_voter_eligibility()?.prepared_vk,
            Some("credential_validity") => &params_credential_validity()?.prepared_vk,
            _ => &params_equality()?.prepared_vk,
        };

        let result =
            Groth16::<Bn254>::verify_proof(pvk, &ark_proof, &embedded_public_inputs).map_err(
                |e| ZkError::VerificationFailed(format!("groth16 verification failed: {e}")),
            )?;

        Ok(VerificationResult { valid: result, system: ProofSystem::Groth16 })
    }
}

#[cfg(test)]
mod tests {
    use super::DevelopmentGroth16Engine;
    use crate::{ProofGenerator, ProofSystem, ProofVerifier};

    /// For generic equality circuit tests we use an unknown circuit_id
    /// so the fallback EqualityCircuit path is used.
    const GENERIC_ID: &str = "generic_test_circuit";

    #[test]
    fn groth16_round_trip_validates() {
        let engine = DevelopmentGroth16Engine;
        let input = b"holder:did:indis:test".to_vec();

        let proof = engine
            .generate(GENERIC_ID, &[input.clone()], &[])
            .expect("proof should be generated");

        assert!(matches!(proof.system, ProofSystem::Groth16));

        let result = engine.verify(&proof, &[], &[input]).expect("verification should run");
        assert!(result.valid);
    }

    #[test]
    fn groth16_rejects_mismatched_public_input() {
        let engine = DevelopmentGroth16Engine;
        let proof = engine
            .generate(GENERIC_ID, &[b"input-a".to_vec()], &[])
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
            .generate(GENERIC_ID, &[b"input-a".to_vec()], &[])
            .expect("proof should be generated");

        // Break JSON envelope so decoding fails.
        proof.data[0] = b'X';
        assert!(engine.verify(&proof, &[], &[]).is_err());
    }

    #[test]
    fn groth16_age_proof_round_trip() {
        let engine = DevelopmentGroth16Engine;
        let input = br#"{"age": 35, "threshold": 18}"#.to_vec();

        let proof = engine
            .generate("age_proof", &[input], &[])
            .expect("age_proof should be generated");

        let result = engine.verify(&proof, &[], &[]).expect("age_proof verification should run");
        assert!(result.valid, "age=35 >= threshold=18 should be valid");
    }

    #[test]
    fn groth16_voter_eligibility_round_trip() {
        let engine = DevelopmentGroth16Engine;
        let input =
            br#"{"age": 25, "credential_hash_low": 12345678, "not_excluded": true}"#.to_vec();

        let proof = engine
            .generate("voter_eligibility", &[input], &[])
            .expect("voter_eligibility should be generated");

        let result =
            engine.verify(&proof, &[], &[]).expect("voter_eligibility verification should run");
        assert!(result.valid, "eligible voter should produce valid proof");
    }

    #[test]
    fn groth16_credential_validity_round_trip() {
        let engine = DevelopmentGroth16Engine;
        let now: u64 = 1_742_000_000;
        let private = format!(
            r#"{{"issued_at": {}, "expiry_at": {}, "not_revoked": true}}"#,
            now - 86400,
            now + 365 * 86400
        );
        let public = format!(r#"{{"current_time": {}}}"#, now);

        let proof = engine
            .generate("credential_validity", &[private.into_bytes()], &[public.into_bytes()])
            .expect("credential_validity proof should be generated");

        let result = engine
            .verify(&proof, &[], &[])
            .expect("credential_validity verification should run");
        assert!(result.valid, "valid credential should produce valid proof");
    }
}
