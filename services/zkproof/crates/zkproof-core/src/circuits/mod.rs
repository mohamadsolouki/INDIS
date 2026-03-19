/// INDIS ZK Circuit Library
///
/// Real R1CS circuits using arkworks for Groth16 proof generation and verification.
/// All circuits target BN254 (Bn254 pairing-friendly curve) for compatibility with
/// the Ethereum ecosystem and existing trusted setup tooling.
pub mod age_range;
pub mod credential_validity;
pub mod voter_eligibility;

pub use age_range::AgeRangeCircuit;
pub use credential_validity::CredentialValidityCircuit;
pub use voter_eligibility::{VoterEligibilityCircuit, VOTER_AGE_THRESHOLD};

/// Maps a `circuit_id` string (from HTTP API) to a canonical circuit name.
/// Returns `None` for unknown circuit IDs.
pub fn canonical_circuit_id(id: &str) -> Option<&'static str> {
    match id.to_lowercase().as_str() {
        "age_proof" | "age_range" | "age-proof" | "age-range" => Some("age_proof"),
        "voter_eligibility" | "voter-eligibility" => Some("voter_eligibility"),
        "credential_validity" | "credential_valid" | "credential-validity" => {
            Some("credential_validity")
        }
        "citizenship_proof" | "citizenship" => Some("citizenship_proof"),
        _ => None,
    }
}
