//! INDIS ZK Circuit Bindings
//!
//! Rust bindings for Circom (Groth16/PLONK) and Cairo (STARK) circuits.
//! This crate bridges the ZK core library with the compiled circuit artifacts.
//!
//! # Circuit Types
//!
//! - `age_proof` — Proves age ≥ threshold without revealing exact age
//! - `citizenship_proof` — Proves Iranian citizenship without any identifier
//! - `voter_eligibility` — Atomic proof: citizenship + age ≥ 18 + not excluded
//! - `credential_validity` — Proves credential is issued, not revoked, not expired

pub mod electoral_stark;

pub use electoral_stark::VoterEligibilityStarkAir;

/// Public input descriptor for a Bulletproofs citizenship range proof.
///
/// Used by the Justice service for anonymous testimony flows where a citizen
/// proves membership (age, citizenship flag) in a valid range without
/// disclosing raw identity attributes.
///
/// Ref: Bulletproofs paper <https://eprint.iacr.org/2017/1066>
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq, Eq)]
pub struct CitizenshipRangePublicInputs {
    /// Human-readable context label (e.g. `"justice:testimony:citizenship"`).
    pub context: String,
    /// Range bit width — the committed value must be in `[0, 2^n_bits)`.
    /// Defaults to 32 when deserialising from JSON without this field.
    #[serde(default = "default_n_bits")]
    pub n_bits: u32,
    /// Base64-encoded Pedersen commitment returned by the proof generation
    /// step. Verifiers use this to check the proof without learning the value.
    pub commitment_b64: String,
}

fn default_n_bits() -> u32 {
    32
}

impl CitizenshipRangePublicInputs {
    /// Serialise the public inputs to a canonical JSON byte vector.
    ///
    /// Stability: field order follows the struct declaration order as produced
    /// by `serde_json`. Callers must not rely on the precise byte layout for
    /// cryptographic binding — use the proof commitment for that.
    pub fn to_public_inputs(&self) -> Vec<u8> {
        serde_json::to_vec(self).expect("CitizenshipRangePublicInputs serialisation is infallible")
    }
}

/// Placeholder for circuit loading and binding functionality.
pub fn placeholder() {
    // TODO: Load compiled circuit artifacts
    // TODO: Bind to Circom WASM outputs
    // TODO: Bind to Cairo STARK prover
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn placeholder_test() {
        // Placeholder test — circuits are not yet compiled
        assert!(true);
    }

    #[test]
    fn citizenship_range_public_inputs_round_trip() {
        let original = CitizenshipRangePublicInputs {
            context: "justice:testimony:citizenship".to_string(),
            n_bits: 32,
            commitment_b64: "dGVzdA==".to_string(),
        };

        let bytes = original.to_public_inputs();
        let restored: CitizenshipRangePublicInputs =
            serde_json::from_slice(&bytes).expect("deserialisation failed");

        assert_eq!(original, restored, "round-trip must be lossless");
    }

    #[test]
    fn citizenship_range_public_inputs_default_n_bits() {
        // Ensure missing n_bits field deserialises to 32.
        let json = r#"{"context":"test","commitment_b64":"dGVzdA=="}"#;
        let inputs: CitizenshipRangePublicInputs =
            serde_json::from_str(json).expect("deserialisation failed");
        assert_eq!(inputs.n_bits, 32);
    }

    #[test]
    fn citizenship_range_public_inputs_serialisation_stability() {
        // Serialise twice — output must be identical (no random field ordering).
        let inputs = CitizenshipRangePublicInputs {
            context: "justice:testimony:citizenship".to_string(),
            n_bits: 64,
            commitment_b64: "YWJjZA==".to_string(),
        };
        let first = inputs.to_public_inputs();
        let second = inputs.to_public_inputs();
        assert_eq!(first, second, "serialisation must be deterministic");
    }
}
