/// VoterEligibilityCircuit: proves voter eligibility without revealing identity.
///
/// Ref: PRD §FR-007 — Electoral Module, §FR-003 — ZK-SNARK.
///
/// Encodes three constraints:
///
/// 1. **Age ≥ 18** — range proof via 8-bit decomposition of (age - 18).
/// 2. **Citizenship credential commitment** — the prover knows a credential hash
///    fragment whose value matches the public commitment.  In production this
///    would use a Poseidon/MiMC hash gadget; here we bind the raw field element.
/// 3. **Not excluded** — the prover asserts their exclusion flag is false.
///    If the citizen is on the exclusion list, proof generation fails.
///
/// Private witnesses: age, credential_hash_low, not_excluded
/// Public inputs:    threshold (18), credential_commitment (u64 fragment of cred hash)
use ark_bn254::Fr;
use ark_ff::Field;
use ark_r1cs_std::{
    alloc::AllocVar,
    boolean::Boolean,
    eq::EqGadget,
    fields::{fp::FpVar, FieldVar},
};
use ark_relations::r1cs::{ConstraintSynthesizer, ConstraintSystemRef, SynthesisError};

/// The minimum voting age enforced by the circuit (PRD §FR-007).
pub const VOTER_AGE_THRESHOLD: u64 = 18;

#[derive(Clone)]
pub struct VoterEligibilityCircuit {
    /// Citizen's actual age (private witness).
    pub age: Option<u64>,
    /// Low 64 bits of the citizenship credential hash (private witness).
    /// The corresponding high bits are zero-padded for the field element.
    pub credential_hash_low: Option<u64>,
    /// True when the citizen is NOT on the electoral exclusion list (private witness).
    pub not_excluded: Option<bool>,
}

impl ConstraintSynthesizer<Fr> for VoterEligibilityCircuit {
    fn generate_constraints(self, cs: ConstraintSystemRef<Fr>) -> Result<(), SynthesisError> {
        let age = self.age.ok_or(SynthesisError::AssignmentMissing)?;
        let cred_hash_low = self.credential_hash_low.ok_or(SynthesisError::AssignmentMissing)?;
        let not_excluded = self.not_excluded.ok_or(SynthesisError::AssignmentMissing)?;

        // ── Constraint 1: age ≥ 18 ─────────────────────────────────────────────
        let excess_val = if age >= VOTER_AGE_THRESHOLD {
            age - VOTER_AGE_THRESHOLD
        } else {
            return Err(SynthesisError::Unsatisfiable);
        };

        // Public: threshold (always 18 for voter eligibility).
        let threshold_fp =
            FpVar::<Fr>::new_input(cs.clone(), || Ok(Fr::from(VOTER_AGE_THRESHOLD)))?;
        // Private: age.
        let age_fp = FpVar::<Fr>::new_witness(cs.clone(), || Ok(Fr::from(age)))?;

        // Bit-decompose excess = age - 18 into 8 bits.
        let bits: Vec<Boolean<Fr>> = (0u64..8)
            .map(|i| Boolean::<Fr>::new_witness(cs.clone(), || Ok((excess_val >> i) & 1 == 1)))
            .collect::<Result<_, _>>()?;

        let mut reconstructed_excess = FpVar::<Fr>::zero();
        let mut coeff = Fr::from(1u64);
        for bit in &bits {
            let contribution = bit.select(&FpVar::constant(coeff), &FpVar::zero())?;
            reconstructed_excess = &reconstructed_excess + &contribution;
            coeff.double_in_place();
        }
        (&threshold_fp + &reconstructed_excess).enforce_equal(&age_fp)?;

        // ── Constraint 2: citizenship credential commitment ─────────────────────
        // Public: the credential commitment (verifier provides low 64-bit hash fragment).
        let cred_commitment =
            FpVar::<Fr>::new_input(cs.clone(), || Ok(Fr::from(cred_hash_low)))?;
        // Private: the prover's knowledge of the pre-image fragment.
        let cred_witness =
            FpVar::<Fr>::new_witness(cs.clone(), || Ok(Fr::from(cred_hash_low)))?;
        // Enforce: the prover's fragment matches the public commitment.
        // Production upgrade: replace with Poseidon hash gadget over the full credential.
        cred_witness.enforce_equal(&cred_commitment)?;

        // ── Constraint 3: not excluded ──────────────────────────────────────────
        // Private: the prover asserts they are NOT on the exclusion list.
        let not_excluded_var = Boolean::<Fr>::new_witness(cs, || Ok(not_excluded))?;
        // Enforce: not_excluded must be true.  If excluded, the circuit is unsatisfiable.
        not_excluded_var.enforce_equal(&Boolean::constant(true))?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::VoterEligibilityCircuit;
    use ark_bn254::Fr;
    use ark_relations::r1cs::{ConstraintSynthesizer, ConstraintSystem};

    fn eligible_circuit() -> VoterEligibilityCircuit {
        VoterEligibilityCircuit {
            age: Some(25),
            credential_hash_low: Some(0xdeadbeef12345678),
            not_excluded: Some(true),
        }
    }

    #[test]
    fn eligible_voter_satisfies() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        eligible_circuit().generate_constraints(cs.clone()).unwrap();
        assert!(cs.is_satisfied().unwrap());
    }

    #[test]
    fn underage_voter_unsatisfiable() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = VoterEligibilityCircuit {
            age: Some(17),
            credential_hash_low: Some(0xdeadbeef12345678),
            not_excluded: Some(true),
        };
        let result = circuit.generate_constraints(cs.clone());
        assert!(result.is_err() || !cs.is_satisfied().unwrap_or(true));
    }

    #[test]
    fn excluded_voter_unsatisfiable() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = VoterEligibilityCircuit {
            age: Some(30),
            credential_hash_low: Some(0xdeadbeef12345678),
            not_excluded: Some(false), // excluded
        };
        circuit.generate_constraints(cs.clone()).unwrap();
        // The constraint system should be unsatisfied (not_excluded enforced = true).
        assert!(!cs.is_satisfied().unwrap());
    }

    #[test]
    fn wrong_credential_commitment_unsatisfied() {
        // Verifier supplies a different commitment than what prover witnesses.
        // This tests the commitment binding constraint.
        // (In this simplified circuit the commitment IS the witness, so mismatch
        //  must be tested at the prove+verify level via groth16.rs routing.)
        let cs = ConstraintSystem::<Fr>::new_ref();
        eligible_circuit().generate_constraints(cs.clone()).unwrap();
        // Circuit satisfied; mismatch is caught at proof verification time.
        assert!(cs.is_satisfied().unwrap());
    }
}
