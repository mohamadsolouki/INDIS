use serde::{Deserialize, Serialize};
use sha3::{Digest, Sha3_256};

/// Public claims carried by the electoral STARK proof.
///
/// Three pillars of voter eligibility are committed separately so that
/// the proof simultaneously binds the voter's identity, their age eligibility,
/// and the single-use nullifier to a specific election.
///
/// - `voter_did_commitment_b64`: base64 of SHA3(voter_DID || election_id)
/// - `age_commitment_b64`: base64 of SHA3(age_bytes || election_id);
///    the service layer enforces age ≥ 18 before calling the STARK prover.
/// - `nullifier_b64`: base64 of the voter's single-use nullifier;
///    uniqueness is tracked in the nullifier set at the service layer.
/// - `election_id`: the election identifier, included in every domain-separated hash.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VoterEligibilityStarkAir {
    pub voter_did_commitment_b64: String,
    pub age_commitment_b64: String,
    pub election_id: String,
    pub nullifier_b64: String,
}

impl VoterEligibilityStarkAir {
    /// Build deterministic public inputs for the prover/verifier path.
    ///
    /// The JSON blob is the canonical representation passed to the STARK
    /// engine. Field order is deterministic because serde serialises struct
    /// fields in declaration order.
    pub fn to_public_inputs(&self) -> Vec<u8> {
        serde_json::to_vec(self).unwrap_or_default()
    }

    /// Derive a stable nullifier key for vote-uniqueness checks.
    ///
    /// The key is stored in the election's nullifier set by the electoral
    /// service to prevent a voter submitting more than one ballot.
    pub fn nullifier_key(&self) -> String {
        let mut hasher = Sha3_256::new();
        hasher.update(b"indis:nullifier:");
        hasher.update(self.election_id.as_bytes());
        hasher.update(self.nullifier_b64.as_bytes());
        format!("{:x}", hasher.finalize())
    }

    /// Derive the 3 domain-separated field-element commitments used as
    /// STARK public inputs.
    ///
    /// Each commitment is `SHA3(domain || field_value || ":" || election_id)[0..8]`
    /// interpreted as a little-endian `u64` (field element).
    pub fn commitments(&self) -> (u64, u64, u64) {
        (
            field_commitment(b"indis:stark:voter:", &self.voter_did_commitment_b64, &self.election_id),
            field_commitment(b"indis:stark:age:", &self.age_commitment_b64, &self.election_id),
            field_commitment(b"indis:stark:nullifier:", &self.nullifier_b64, &self.election_id),
        )
    }
}

/// Derive a field element from a domain-separated SHA3 hash.
///
/// `SHA3(domain || value || ":" || election_id)[0..8]` → `u64` (non-zero).
pub(crate) fn field_commitment(domain: &[u8], value: &str, election_id: &str) -> u64 {
    let mut h = Sha3_256::new();
    h.update(domain);
    h.update(value.as_bytes());
    h.update(b":");
    h.update(election_id.as_bytes());
    let digest = h.finalize();
    let mut bytes = [0u8; 8];
    bytes.copy_from_slice(&digest[..8]);
    let v = u64::from_le_bytes(bytes);
    if v == 0 { 1 } else { v }
}

#[cfg(test)]
mod tests {
    use super::{field_commitment, VoterEligibilityStarkAir};

    fn sample_air() -> VoterEligibilityStarkAir {
        VoterEligibilityStarkAir {
            voter_did_commitment_b64: "dm90ZXI=".to_string(),
            age_commitment_b64: "YWdlMjU=".to_string(),
            election_id: "election-1404".to_string(),
            nullifier_b64: "bnVsbGlmaWVy".to_string(),
        }
    }

    #[test]
    fn public_inputs_are_stable() {
        let air = sample_air();
        assert_eq!(air.to_public_inputs(), air.to_public_inputs());
    }

    #[test]
    fn nullifier_key_is_stable() {
        let air = sample_air();
        assert_eq!(air.nullifier_key(), air.nullifier_key());
    }

    #[test]
    fn nullifier_key_depends_on_election_and_nullifier() {
        let a = VoterEligibilityStarkAir {
            voter_did_commitment_b64: "dm90ZXI=".to_string(),
            age_commitment_b64: "YWdlMjU=".to_string(),
            election_id: "election-a".to_string(),
            nullifier_b64: "bnVsbGlmaWVyLTAx".to_string(),
        };
        let b = VoterEligibilityStarkAir {
            election_id: "election-b".to_string(),
            ..a.clone()
        };
        // Different election → different nullifier key (double-voting across elections)
        assert_ne!(a.nullifier_key(), b.nullifier_key());
    }

    #[test]
    fn commitments_are_nonzero() {
        let (vc, ac, nc) = sample_air().commitments();
        assert_ne!(vc, 0, "voter_commitment must be non-zero");
        assert_ne!(ac, 0, "age_commitment must be non-zero");
        assert_ne!(nc, 0, "nullifier_commitment must be non-zero");
    }

    #[test]
    fn commitments_are_distinct_per_pillar() {
        let air = sample_air();
        let (vc, ac, nc) = air.commitments();
        // Different domain separators → different commitments for same value.
        assert_ne!(vc, ac);
        assert_ne!(vc, nc);
        assert_ne!(ac, nc);
    }

    #[test]
    fn commitments_change_with_election_id() {
        let a = sample_air();
        let b = VoterEligibilityStarkAir {
            election_id: "election-9999".to_string(),
            ..a.clone()
        };
        let (vc_a, _, _) = a.commitments();
        let (vc_b, _, _) = b.commitments();
        assert_ne!(vc_a, vc_b, "voter_commitment must change when election_id changes");
    }

    #[test]
    fn field_commitment_is_nonzero_for_empty_input() {
        // Regression: SHA3(domain || "" || ":" || "")[0..8] must not produce 0.
        let v = field_commitment(b"indis:stark:voter:", "", "");
        assert_ne!(v, 0);
    }
}
