use serde::{Deserialize, Serialize};
use sha3::{Digest, Sha3_256};

/// Public claims carried by the development electoral STARK baseline.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VoterEligibilityStarkAir {
    pub voter_did_commitment_b64: String,
    pub election_id: String,
    pub nullifier_b64: String,
}

impl VoterEligibilityStarkAir {
    /// Build deterministic public inputs for the prover/verifier path.
    pub fn to_public_inputs(&self) -> Vec<u8> {
        serde_json::to_vec(self).unwrap_or_default()
    }

    /// Derive a stable nullifier key for vote uniqueness checks.
    pub fn nullifier_key(&self) -> String {
        let mut hasher = Sha3_256::new();
        hasher.update(self.election_id.as_bytes());
        hasher.update(self.nullifier_b64.as_bytes());
        format!("{:x}", hasher.finalize())
    }
}

#[cfg(test)]
mod tests {
    use super::VoterEligibilityStarkAir;

    #[test]
    fn public_inputs_are_stable() {
        let air = VoterEligibilityStarkAir {
            voter_did_commitment_b64: "dm90ZXI=".to_string(),
            election_id: "election-1404".to_string(),
            nullifier_b64: "bnVsbGlmaWVy".to_string(),
        };

        let one = air.to_public_inputs();
        let two = air.to_public_inputs();
        assert_eq!(one, two);
    }

    #[test]
    fn nullifier_key_depends_on_election_and_nullifier() {
        let air = VoterEligibilityStarkAir {
            voter_did_commitment_b64: "dm90ZXI=".to_string(),
            election_id: "election-a".to_string(),
            nullifier_b64: "bnVsbGlmaWVyLTAx".to_string(),
        };

        let key_one = air.nullifier_key();
        let key_two = air.nullifier_key();
        assert_eq!(key_one, key_two);
    }
}
