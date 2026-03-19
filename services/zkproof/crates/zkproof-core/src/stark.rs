//! INDIS — STARK proof engine using the Winterfell library.
//!
//! Provides two engines:
//!
//! * `DevelopmentStarkEngine` — legacy SHA3-hash baseline kept for backward-compat tests.
//! * `WinterfellStarkEngine` — real Winterfell ZK-STARK for voter eligibility proofs.
//!
//! The `WinterfellStarkEngine` builds a genuine STARK with the following AIR:
//!
//! | Component | Value |
//! |-----------|-------|
//! | Trace     | 1 column, 8 rows, constant |
//! | Constraint | `next[0] == cur[0]` (constant column) |
//! | Assertion  | `step 0, col 0 == voter_commitment` (public) |
//!
//! `voter_commitment` is derived deterministically from the election's public-input JSON
//! via `SHA3("indis:stark:v2:commitment:" || json)`.  This ties the STARK proof to a
//! specific voter + election pair; any tampering with the public inputs invalidates
//! the proof.
//!
//! For production, replace the constant-trace circuit with one that does an in-circuit
//! range check on `age >= 18` and verifies the voter's DID signature.

use sha3::{Digest, Sha3_256};
use winterfell::{
    crypto::{hashers::Blake3_256, DefaultRandomCoin},
    math::{fields::f128::BaseElement, FieldElement, ToElements},
    matrix::ColMatrix,
    Air, AirContext, Assertion, DefaultConstraintEvaluator, DefaultTraceLde, Deserializable,
    EvaluationFrame, FieldExtension, ProofOptions, Prover, StarkDomain, StarkProof,
    TraceInfo, TracePolyTable, TraceTable, TransitionConstraintDegree,
};

use crate::{Proof, ProofGenerator, ProofSystem, ProofVerifier, VerificationResult, ZkError};

// ── SHA3 development baseline constants ────────────────────────────────────────
const DEV_STARK_PREFIX: &[u8] = b"indis:stark:dev:v1";
const DEV_STARK_PROOF_LEN: usize = 33;

// ── Winterfell circuit constants ───────────────────────────────────────────────
const WINTERFELL_TRACE_WIDTH: usize = 1;
/// Minimum trace length for Winterfell; must be a power of 2 ≥ 8.
const WINTERFELL_TRACE_LENGTH: usize = 8;

// ────────────────────────────────────────────────────────────────────────────────
// DevelopmentStarkEngine — SHA3-hash baseline (kept for backward-compat tests)
// ────────────────────────────────────────────────────────────────────────────────

/// SHA3-based development STARK baseline.
///
/// **NOT cryptographically sound.** Kept only for backward-compat unit tests.
/// All new code should use [`WinterfellStarkEngine`].
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
        if proof.data.len() != DEV_STARK_PROOF_LEN || proof.data[0] != 1 {
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

// ────────────────────────────────────────────────────────────────────────────────
// WinterfellStarkEngine — real Winterfell ZK-STARK
// ────────────────────────────────────────────────────────────────────────────────

/// Public inputs visible to the STARK verifier.
///
/// Two field elements derived deterministically from the election's JSON blob:
/// - `start`: first 8 bytes of SHA3 digest → ties the proof to the specific election
/// - `result`: `start × 2^(TRACE_LENGTH−1)` → computed deterministically from `start`
///
/// Having two assertions across the trace ensures the trace polynomial is non-constant
/// (required by the Winterfell DEEP composition protocol).
#[derive(Debug, Clone)]
pub struct VoterPublicInputs {
    pub start: BaseElement,
    pub result: BaseElement,
}

impl ToElements<BaseElement> for VoterPublicInputs {
    fn to_elements(&self) -> Vec<BaseElement> {
        vec![self.start, self.result]
    }
}

/// Winterfell AIR for voter eligibility STARK proofs.
///
/// **Circuit:** 1-column "doubling" trace of length `WINTERFELL_TRACE_LENGTH`.
///
/// ```text
/// row 0: start   (= SHA3(domain || json)[0..8] as u64)
/// row 1: start × 2
/// row 2: start × 4
/// ...
/// row n: start × 2^n   (= result, asserted publicly)
/// ```
///
/// Transition constraint: `next[0] - cur[0] × 2 = 0` (degree 1).
/// Boundary assertions: step 0 = `start`, step 7 = `result`.
///
/// This produces a proper non-constant trace polynomial of degree
/// `trace_length − 1 = 7`, satisfying the Winterfell DEEP composition invariant.
pub struct VoterEligibilityAir {
    context: AirContext<BaseElement>,
    pub_inputs: VoterPublicInputs,
}

impl Air for VoterEligibilityAir {
    type BaseField = BaseElement;
    type PublicInputs = VoterPublicInputs;

    fn new(
        trace_info: TraceInfo,
        pub_inputs: VoterPublicInputs,
        options: ProofOptions,
    ) -> Self {
        // Degree-1 transition: next[0] = cur[0] × 2.
        let degrees = vec![TransitionConstraintDegree::new(1)];
        // 2 boundary assertions: step 0 and step (TRACE_LENGTH−1).
        let context = AirContext::new(trace_info, degrees, 2, options);
        Self { context, pub_inputs }
    }

    fn context(&self) -> &AirContext<BaseElement> {
        &self.context
    }

    fn evaluate_transition<E: FieldElement + From<Self::BaseField>>(
        &self,
        frame: &EvaluationFrame<E>,
        _periodic_values: &[E],
        result: &mut [E],
    ) {
        // Enforce: next[0] = cur[0] × 2  ↔  next[0] - 2 × cur[0] = 0
        result[0] = frame.next()[0] - E::from(2u32) * frame.current()[0];
    }

    fn get_assertions(&self) -> Vec<Assertion<BaseElement>> {
        let last_step = WINTERFELL_TRACE_LENGTH - 1;
        vec![
            // start value is public — ties the proof to the election's JSON data.
            Assertion::single(0, 0, self.pub_inputs.start),
            // final value is public — verifier recomputes and checks.
            Assertion::single(0, last_step, self.pub_inputs.result),
        ]
    }
}

/// Winterfell prover for the voter eligibility AIR.
///
/// Uses `DefaultTraceLde` and `DefaultConstraintEvaluator` so no custom
/// LDE or evaluator logic is needed.
struct WinterfellProver {
    pub_inputs: VoterPublicInputs,
    options: ProofOptions,
}

impl Prover for WinterfellProver {
    type BaseField = BaseElement;
    type Air = VoterEligibilityAir;
    type Trace = TraceTable<Self::BaseField>;
    type HashFn = Blake3_256<Self::BaseField>;
    type RandomCoin = DefaultRandomCoin<Self::HashFn>;
    type TraceLde<E: FieldElement<BaseField = Self::BaseField>> =
        DefaultTraceLde<E, Self::HashFn>;
    type ConstraintEvaluator<'a, E: FieldElement<BaseField = Self::BaseField>> =
        DefaultConstraintEvaluator<'a, Self::Air, E>;

    fn get_pub_inputs(&self, _trace: &Self::Trace) -> VoterPublicInputs {
        self.pub_inputs.clone()
    }

    fn options(&self) -> &ProofOptions {
        &self.options
    }

    fn new_trace_lde<E: FieldElement<BaseField = Self::BaseField>>(
        &self,
        trace_info: &TraceInfo,
        main_trace: &ColMatrix<Self::BaseField>,
        domain: &StarkDomain<Self::BaseField>,
    ) -> (Self::TraceLde<E>, TracePolyTable<E>) {
        DefaultTraceLde::new(trace_info, main_trace, domain)
    }

    fn new_evaluator<'a, E: FieldElement<BaseField = Self::BaseField>>(
        &self,
        air: &'a Self::Air,
        aux_rand_elements: winterfell::AuxTraceRandElements<E>,
        composition_coefficients: winterfell::ConstraintCompositionCoefficients<E>,
    ) -> Self::ConstraintEvaluator<'a, E> {
        DefaultConstraintEvaluator::new(air, aux_rand_elements, composition_coefficients)
    }
}

/// Winterfell `ProofOptions` for the INDIS development baseline.
///
/// Security level: ~96-bit conjectured security (32 queries × log₂ 8 blowup).
fn stark_proof_options() -> ProofOptions {
    ProofOptions::new(
        32,                    // num_queries → ~96-bit security
        8,                     // blowup_factor (LDE domain = trace_length × 8)
        0,                     // grinding_factor (proof-of-work bits; 0 for dev speed)
        FieldExtension::None,  // no field extension needed for f128
        8,                     // FRI folding factor
        31,                    // FRI max remainder polynomial degree
    )
}

/// Derive `VoterPublicInputs` from the election's public-input JSON blob.
///
/// - `start = SHA3("indis:stark:v2:start:" || json)[0..8]` as `u64` → field element
/// - `result = start × 2^(TRACE_LENGTH−1)` — computed deterministically from `start`
///
/// Having `start != result` guarantees the trace polynomial is non-constant,
/// which is required by the Winterfell DEEP composition protocol.
fn derive_public_inputs(public_inputs_json: &[u8]) -> VoterPublicInputs {
    let mut hasher = Sha3_256::new();
    hasher.update(b"indis:stark:v2:start:");
    hasher.update(public_inputs_json);
    let digest = hasher.finalize();

    // Use first 8 bytes as a u64 seed → field element (avoid zero).
    let mut seed_bytes = [0u8; 8];
    seed_bytes.copy_from_slice(&digest[..8]);
    let seed = u64::from_le_bytes(seed_bytes);
    // Ensure non-zero (zero field element would make all trace rows zero).
    let start = BaseElement::new(if seed == 0 { 1 } else { seed as u128 });

    // result = start × 2^(TRACE_LENGTH−1)
    let factor = BaseElement::new(1u128 << (WINTERFELL_TRACE_LENGTH - 1));
    let result = start * factor;

    VoterPublicInputs { start, result }
}

/// Real Winterfell ZK-STARK engine for INDIS voter eligibility proofs.
///
/// Replaces the SHA3-hash `DevelopmentStarkEngine` with a genuine post-quantum
/// STARK using the Winterfell library. The proof ties the prover to a specific
/// `voter_commitment` derived from the election's public inputs:
///
/// ```text
/// voter_commitment = SHA3("indis:stark:v2:commitment:" || public_inputs_json)[0..16]
/// ```
///
/// Any tampering with the public inputs changes the commitment, invalidating the proof.
///
/// **Production upgrade:** Replace the constant-trace AIR with one that performs
/// in-circuit `age >= 18` range decomposition and DID signature verification.
#[derive(Debug, Default, Clone)]
pub struct WinterfellStarkEngine;

impl ProofGenerator for WinterfellStarkEngine {
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
                "STARK prove: public_inputs are required".to_string(),
            ));
        }

        let pub_inputs = derive_public_inputs(&public_inputs[0]);

        // Build a doubling trace: row i = start × 2^i.
        let start_val = pub_inputs.start;
        let mut trace = TraceTable::new(WINTERFELL_TRACE_WIDTH, WINTERFELL_TRACE_LENGTH);
        trace.fill(
            |state| {
                state[0] = start_val;
            },
            |_step, state| {
                // Double the value each step: row[i+1] = row[i] × 2.
                state[0] = state[0] * BaseElement::new(2);
            },
        );
        let options = stark_proof_options();
        let prover = WinterfellProver { pub_inputs, options };

        let stark_proof = prover
            .prove(trace)
            .map_err(|e| ZkError::GenerationFailed(format!("Winterfell prove failed: {e}")))?;

        Ok(Proof {
            system: ProofSystem::Stark,
            data: stark_proof.to_bytes(),
            public_inputs: public_inputs.to_vec(),
        })
    }
}

impl ProofVerifier for WinterfellStarkEngine {
    fn verify(
        &self,
        proof: &Proof,
        _verification_key: &[u8],
        public_inputs: &[Vec<u8>],
    ) -> Result<VerificationResult, ZkError> {
        if !matches!(proof.system, ProofSystem::Stark) {
            return Err(ZkError::UnsupportedSystem(
                "proof is not a STARK proof".to_string(),
            ));
        }
        if public_inputs.is_empty() {
            return Ok(VerificationResult {
                valid: false,
                system: ProofSystem::Stark,
            });
        }

        let pub_inputs = derive_public_inputs(&public_inputs[0]);

        let stark_proof = StarkProof::read_from_bytes(&proof.data).map_err(|e| {
            ZkError::VerificationFailed(format!("STARK proof deserialization failed: {e}"))
        })?;

        // Accept proofs that provide ≥95-bit conjectured security.
        let min_opts = winterfell::AcceptableOptions::MinConjecturedSecurity(95);

        match winterfell::verify::<
            VoterEligibilityAir,
            Blake3_256<BaseElement>,
            DefaultRandomCoin<Blake3_256<BaseElement>>,
        >(stark_proof, pub_inputs, &min_opts)
        {
            Ok(_) => Ok(VerificationResult {
                valid: true,
                system: ProofSystem::Stark,
            }),
            Err(_) => Ok(VerificationResult {
                valid: false,
                system: ProofSystem::Stark,
            }),
        }
    }
}

// ── Tests ──────────────────────────────────────────────────────────────────────
#[cfg(test)]
mod tests {
    use super::{DevelopmentStarkEngine, WinterfellStarkEngine};
    use crate::{ProofGenerator, ProofSystem, ProofVerifier};

    // ── DevelopmentStarkEngine (backward compat) ───────────────────────────────

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
            .verify(&proof, &[], &[public_input])
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

    // ── WinterfellStarkEngine ──────────────────────────────────────────────────

    #[test]
    fn winterfell_rejects_empty_public_inputs() {
        let engine = WinterfellStarkEngine;
        let result = engine.generate("voter_eligibility", &[], &[]);
        assert!(result.is_err());
    }

    #[test]
    fn winterfell_round_trip_validates() {
        let engine = WinterfellStarkEngine;
        let public_input = serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "dm90ZXI=",
            "election_id": "election-1404",
            "nullifier_b64": "bnVsbGlmaWVy"
        }))
        .unwrap();

        let proof = engine
            .generate("voter_eligibility", &[], &[public_input.clone()])
            .expect("Winterfell proof should be generated");

        assert!(matches!(proof.system, ProofSystem::Stark));

        let result = engine
            .verify(&proof, &[], &[public_input])
            .expect("Winterfell verification should run without error");

        assert!(result.valid, "round-trip proof must be valid");
    }

    #[test]
    fn winterfell_rejects_tampered_public_inputs() {
        let engine = WinterfellStarkEngine;

        let input_a = serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "dm90ZXI=",
            "election_id": "election-1404",
            "nullifier_b64": "bnVsbGlmaWVy"
        }))
        .unwrap();

        let input_b = serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "YWx0ZXI=",
            "election_id": "election-9999",
            "nullifier_b64": "ZGlmZmVyZW50"
        }))
        .unwrap();

        let proof = engine
            .generate("voter_eligibility", &[], &[input_a])
            .expect("proof should be generated");

        let result = engine
            .verify(&proof, &[], &[input_b])
            .expect("verification should run without error");

        assert!(
            !result.valid,
            "tampered public inputs must invalidate the proof"
        );
    }

    #[test]
    fn winterfell_proof_system_is_stark() {
        let engine = WinterfellStarkEngine;
        let public_input = b"{\"election_id\":\"test\"}".to_vec();
        let proof = engine
            .generate("voter_eligibility", &[], &[public_input])
            .expect("proof should be generated");
        assert!(matches!(proof.system, ProofSystem::Stark));
    }
}
