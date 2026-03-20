//! INDIS — STARK proof engine using the Winterfell library.
//!
//! Provides two engines:
//!
//! * `DevelopmentStarkEngine` — legacy SHA3-hash baseline kept for backward-compat tests.
//! * `WinterfellStarkEngine` — real Winterfell ZK-STARK for voter eligibility proofs.
//!
//! ## Circuit design (v3 — 3-column eligibility AIR)
//!
//! The AIR encodes three pillars of voter eligibility as separate constant columns,
//! each derived with a domain-separated SHA3 commitment from the public inputs:
//!
//! | Column | Meaning                         | Public assertion          |
//! |--------|---------------------------------|---------------------------|
//! | 0      | voter_commitment (DID + elec)   | step 0 and step length-1  |
//! | 1      | age_commitment (age + elec)     | step 0 and step length-1  |
//! | 2      | nullifier_commitment (null+elec)| step 0 and step length-1  |
//!
//! All three columns are constant throughout the trace:
//! `next[i] - cur[i] = 0` (degree-1 transition constraint).
//!
//! Six public assertions (start + end of each column) bind the proof to the
//! specific (voter, age-eligibility, election, nullifier) tuple. Any tampering
//! with any pillar of the public inputs changes one or more committed values,
//! immediately invalidating the proof.
//!
//! ## Production upgrade path
//!
//! Replace the constant-column age commitment with a binary range decomposition
//! that enforces `age_value >= 18` in-circuit (8 bit-columns + carry constraint).
//! The service layer already enforces this policy, but in-circuit enforcement
//! removes the trust assumption on the service.

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
/// Three eligibility pillars: voter_commitment, age_commitment, nullifier_commitment.
const WINTERFELL_TRACE_WIDTH: usize = 3;
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
/// Each field is the *start* value of one trace column.  The *end* value is
/// deterministically computed as `start × 2^(TRACE_LENGTH−1)`, which is also
/// publicly asserted so that the verifier can re-derive both boundary values
/// independently.
///
/// Three eligibility pillars are derived with domain-separated SHA3 hashes:
///
/// - `voter_commitment` — `SHA3("indis:stark:voter:"     || voter_did_commitment_b64 || ":" || election_id)[0..8]`
/// - `age_commitment`   — `SHA3("indis:stark:age:"       || age_commitment_b64       || ":" || election_id)[0..8]`
/// - `nullifier`        — `SHA3("indis:stark:nullifier:" || nullifier_b64            || ":" || election_id)[0..8]`
#[derive(Debug, Clone)]
pub struct VoterPublicInputs {
    /// Start value of column 0 (voter DID × election binding).
    pub voter_commitment: BaseElement,
    /// Start value of column 1 (age-eligibility × election binding).
    pub age_commitment: BaseElement,
    /// Start value of column 2 (single-use nullifier × election binding).
    pub nullifier: BaseElement,
}

impl ToElements<BaseElement> for VoterPublicInputs {
    fn to_elements(&self) -> Vec<BaseElement> {
        vec![self.voter_commitment, self.age_commitment, self.nullifier]
    }
}

/// Compute the end value of a doubling column.
///
/// Each column doubles every row: `row[i] = start × 2^i`.
/// The last row therefore holds `start × 2^(TRACE_LENGTH−1)`.
fn doubling_end(start: BaseElement) -> BaseElement {
    let factor = BaseElement::new(1u128 << (WINTERFELL_TRACE_LENGTH - 1));
    start * factor
}

/// Winterfell AIR for voter eligibility STARK proofs (v3 — 3-column eligibility).
///
/// ## Trace layout
///
/// | Col | Semantics                       | Row 0               | Row i                   |
/// |-----|---------------------------------|---------------------|-------------------------|
/// | 0   | voter_commitment doubling chain | `voter_commitment`  | `voter_commitment × 2^i`|
/// | 1   | age_commitment doubling chain   | `age_commitment`    | `age_commitment × 2^i`  |
/// | 2   | nullifier doubling chain        | `nullifier`         | `nullifier × 2^i`       |
///
/// ## Transition constraints (degree 1, per column)
/// `next[i] − 2 × cur[i] = 0`
///
/// ## Boundary assertions (6 total, all public)
/// For each column: row 0 = `start`, row (TRACE_LENGTH−1) = `start × 2^(TRACE_LENGTH−1)`.
///
/// Having distinct start and end values ensures that each trace column's polynomial
/// has degree ≥ 1, satisfying Winterfell's DEEP composition invariant.
/// Any tampering with any pillar of the public inputs changes that column's
/// start and end values, immediately invalidating the proof.
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
        // Three degree-1 doubling constraints, one per column.
        let degrees = vec![
            TransitionConstraintDegree::new(1),
            TransitionConstraintDegree::new(1),
            TransitionConstraintDegree::new(1),
        ];
        // 6 boundary assertions: start + end for each of the 3 columns.
        let context = AirContext::new(trace_info, degrees, 6, options);
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
        // Each column doubles: next[i] = cur[i] × 2.
        let two = E::from(2u32);
        result[0] = frame.next()[0] - two * frame.current()[0];
        result[1] = frame.next()[1] - two * frame.current()[1];
        result[2] = frame.next()[2] - two * frame.current()[2];
    }

    fn get_assertions(&self) -> Vec<Assertion<BaseElement>> {
        let last = WINTERFELL_TRACE_LENGTH - 1;
        vec![
            // Col 0 — voter_commitment
            Assertion::single(0, 0,    self.pub_inputs.voter_commitment),
            Assertion::single(0, last, doubling_end(self.pub_inputs.voter_commitment)),
            // Col 1 — age_commitment
            Assertion::single(1, 0,    self.pub_inputs.age_commitment),
            Assertion::single(1, last, doubling_end(self.pub_inputs.age_commitment)),
            // Col 2 — nullifier
            Assertion::single(2, 0,    self.pub_inputs.nullifier),
            Assertion::single(2, last, doubling_end(self.pub_inputs.nullifier)),
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

/// Derive a non-zero field element from a domain-separated SHA3 hash.
///
/// `SHA3(domain || value_bytes || ":" || election_id)[0..8]` → `u64` → `BaseElement`
fn field_element_from_hash(domain: &[u8], value: &[u8], election_id: &[u8]) -> BaseElement {
    let mut h = Sha3_256::new();
    h.update(domain);
    h.update(value);
    h.update(b":");
    h.update(election_id);
    let digest = h.finalize();
    let mut bytes = [0u8; 8];
    bytes.copy_from_slice(&digest[..8]);
    let v = u64::from_le_bytes(bytes);
    BaseElement::new(if v == 0 { 1u128 } else { v as u128 })
}

/// Derive `VoterPublicInputs` from the election's public-input JSON blob.
///
/// Parses the JSON for `voter_did_commitment_b64`, `age_commitment_b64`,
/// `nullifier_b64`, and `election_id`, then derives three domain-separated
/// field elements — one per eligibility pillar.
///
/// Falls back to hashing the raw JSON blob for any missing field so that
/// malformed input still produces a deterministic (but likely invalid) proof
/// rather than panicking.
fn derive_public_inputs(public_inputs_json: &[u8]) -> VoterPublicInputs {
    // Best-effort JSON parse — fall back to raw-blob hashing if any field is absent.
    let (voter_val, age_val, nullifier_val, election_id) =
        if let Ok(v) = serde_json::from_slice::<serde_json::Value>(public_inputs_json) {
            let voter = v["voter_did_commitment_b64"]
                .as_str()
                .unwrap_or("")
                .as_bytes()
                .to_vec();
            let age = v["age_commitment_b64"]
                .as_str()
                .unwrap_or("")
                .as_bytes()
                .to_vec();
            let nullifier = v["nullifier_b64"]
                .as_str()
                .unwrap_or("")
                .as_bytes()
                .to_vec();
            let election = v["election_id"]
                .as_str()
                .unwrap_or("")
                .as_bytes()
                .to_vec();
            (voter, age, nullifier, election)
        } else {
            // Malformed JSON: hash the raw blob for all three pillars.
            let raw = public_inputs_json.to_vec();
            (raw.clone(), raw.clone(), raw.clone(), b"unknown".to_vec())
        };

    VoterPublicInputs {
        voter_commitment: field_element_from_hash(
            b"indis:stark:voter:",
            &voter_val,
            &election_id,
        ),
        age_commitment: field_element_from_hash(
            b"indis:stark:age:",
            &age_val,
            &election_id,
        ),
        nullifier: field_element_from_hash(
            b"indis:stark:nullifier:",
            &nullifier_val,
            &election_id,
        ),
    }
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

        // Build a 3-column doubling trace.
        // Each column starts at its eligibility commitment and doubles every row,
        // giving a non-constant trace polynomial (required by Winterfell DEEP composition).
        let vc = pub_inputs.voter_commitment;
        let ac = pub_inputs.age_commitment;
        let nc = pub_inputs.nullifier;
        let two = BaseElement::new(2);
        let mut trace = TraceTable::new(WINTERFELL_TRACE_WIDTH, WINTERFELL_TRACE_LENGTH);
        trace.fill(
            |state| {
                state[0] = vc;
                state[1] = ac;
                state[2] = nc;
            },
            |_step, state| {
                // Double each column independently each step.
                state[0] = state[0] * two;
                state[1] = state[1] * two;
                state[2] = state[2] * two;
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

    fn sample_inputs() -> Vec<u8> {
        serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "dm90ZXI=",
            "age_commitment_b64": "YWdlMjU=",
            "election_id": "election-1404",
            "nullifier_b64": "bnVsbGlmaWVy"
        }))
        .unwrap()
    }

    #[test]
    fn winterfell_rejects_empty_public_inputs() {
        let engine = WinterfellStarkEngine;
        let result = engine.generate("voter_eligibility", &[], &[]);
        assert!(result.is_err());
    }

    #[test]
    fn winterfell_round_trip_validates() {
        let engine = WinterfellStarkEngine;
        let public_input = sample_inputs();

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
    fn winterfell_rejects_tampered_voter_commitment() {
        let engine = WinterfellStarkEngine;
        let original = sample_inputs();
        let tampered = serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "TAMPERED==",
            "age_commitment_b64": "YWdlMjU=",
            "election_id": "election-1404",
            "nullifier_b64": "bnVsbGlmaWVy"
        }))
        .unwrap();

        let proof = engine
            .generate("voter_eligibility", &[], &[original])
            .expect("proof should be generated");
        let result = engine
            .verify(&proof, &[], &[tampered])
            .expect("verification should run");
        assert!(!result.valid, "tampered voter_commitment must invalidate proof");
    }

    #[test]
    fn winterfell_rejects_tampered_age_commitment() {
        let engine = WinterfellStarkEngine;
        let original = sample_inputs();
        let tampered = serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "dm90ZXI=",
            "age_commitment_b64": "TAMPERED==",
            "election_id": "election-1404",
            "nullifier_b64": "bnVsbGlmaWVy"
        }))
        .unwrap();

        let proof = engine
            .generate("voter_eligibility", &[], &[original])
            .expect("proof should be generated");
        let result = engine
            .verify(&proof, &[], &[tampered])
            .expect("verification should run");
        assert!(!result.valid, "tampered age_commitment must invalidate proof");
    }

    #[test]
    fn winterfell_rejects_tampered_nullifier() {
        let engine = WinterfellStarkEngine;
        let original = sample_inputs();
        let tampered = serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "dm90ZXI=",
            "age_commitment_b64": "YWdlMjU=",
            "election_id": "election-1404",
            "nullifier_b64": "TAMPERED=="
        }))
        .unwrap();

        let proof = engine
            .generate("voter_eligibility", &[], &[original])
            .expect("proof should be generated");
        let result = engine
            .verify(&proof, &[], &[tampered])
            .expect("verification should run");
        assert!(!result.valid, "tampered nullifier must invalidate proof");
    }

    #[test]
    fn winterfell_rejects_tampered_election_id() {
        let engine = WinterfellStarkEngine;
        let original = sample_inputs();
        let tampered = serde_json::to_vec(&serde_json::json!({
            "voter_did_commitment_b64": "dm90ZXI=",
            "age_commitment_b64": "YWdlMjU=",
            "election_id": "election-EVIL",
            "nullifier_b64": "bnVsbGlmaWVy"
        }))
        .unwrap();

        let proof = engine
            .generate("voter_eligibility", &[], &[original])
            .expect("proof should be generated");
        let result = engine
            .verify(&proof, &[], &[tampered])
            .expect("verification should run");
        assert!(!result.valid, "tampered election_id must invalidate proof");
    }

    #[test]
    fn winterfell_proof_system_is_stark() {
        let engine = WinterfellStarkEngine;
        let proof = engine
            .generate("voter_eligibility", &[], &[sample_inputs()])
            .expect("proof should be generated");
        assert!(matches!(proof.system, ProofSystem::Stark));
    }

    #[test]
    fn winterfell_malformed_json_produces_deterministic_proof() {
        // Malformed input should not panic; it should produce a valid (but
        // all-fallback) proof that round-trips with the same input.
        let engine = WinterfellStarkEngine;
        let bad = b"not-json".to_vec();
        let proof = engine
            .generate("voter_eligibility", &[], &[bad.clone()])
            .expect("should not panic on malformed JSON");
        let result = engine
            .verify(&proof, &[], &[bad])
            .expect("verification should run");
        assert!(result.valid, "malformed-input proof must round-trip");
    }
}
