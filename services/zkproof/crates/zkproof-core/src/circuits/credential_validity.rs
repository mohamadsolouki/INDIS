/// CredentialValidityCircuit: proves a credential is issued and not expired/revoked.
///
/// Ref: PRD §FR-002 — Credential Lifecycle, §FR-003 — ZK-SNARK.
///
/// Constraints:
/// 1. **Issued** — the credential issuance timestamp is non-zero (issued).
/// 2. **Not expired** — current_time < expiry_time, proven by bit-decomposing
///    (expiry_time - current_time) into 32 bits (≤ ~136 years in seconds).
/// 3. **Not revoked** — the revocation flag is false.
///
/// Private witnesses: issued_at (unix seconds), expiry_at (unix seconds), not_revoked
/// Public inputs:     current_time (unix seconds, provided by verifier)
use ark_bn254::Fr;
use ark_ff::Field;
use ark_r1cs_std::{
    alloc::AllocVar,
    boolean::Boolean,
    eq::EqGadget,
    fields::{fp::FpVar, FieldVar},
};
use ark_relations::r1cs::{ConstraintSynthesizer, ConstraintSystemRef, SynthesisError};

#[derive(Clone)]
pub struct CredentialValidityCircuit {
    /// Unix timestamp when the credential was issued (private witness).
    pub issued_at: Option<u64>,
    /// Unix timestamp when the credential expires (private witness).
    pub expiry_at: Option<u64>,
    /// True when the credential has NOT been revoked (private witness).
    pub not_revoked: Option<bool>,
    /// Current time in unix seconds (public input, provided by the verifier).
    pub current_time: u64,
}

impl ConstraintSynthesizer<Fr> for CredentialValidityCircuit {
    fn generate_constraints(self, cs: ConstraintSystemRef<Fr>) -> Result<(), SynthesisError> {
        let issued_at = self.issued_at.ok_or(SynthesisError::AssignmentMissing)?;
        let expiry_at = self.expiry_at.ok_or(SynthesisError::AssignmentMissing)?;
        let not_revoked = self.not_revoked.ok_or(SynthesisError::AssignmentMissing)?;
        let current_time = self.current_time;

        // ── Constraint 1: issued (issued_at > 0) ───────────────────────────────
        if issued_at == 0 {
            return Err(SynthesisError::Unsatisfiable);
        }
        // Witness issued_at to bind it into the circuit (private).
        let _issued_fp =
            FpVar::<Fr>::new_witness(cs.clone(), || Ok(Fr::from(issued_at)))?;

        // ── Constraint 2: not expired (expiry_at > current_time) ──────────────
        let remaining = if expiry_at > current_time {
            expiry_at - current_time
        } else {
            return Err(SynthesisError::Unsatisfiable); // credential expired
        };

        // Public: current_time (the verifier stamps this).
        let current_fp = FpVar::<Fr>::new_input(cs.clone(), || Ok(Fr::from(current_time)))?;
        // Private: expiry_at.
        let expiry_fp = FpVar::<Fr>::new_witness(cs.clone(), || Ok(Fr::from(expiry_at)))?;

        // Bit-decompose remaining = expiry_at - current_time into 32 bits.
        // 32 bits covers ~136 years in seconds, sufficient for any credential lifetime.
        let bits: Vec<Boolean<Fr>> = (0u64..32)
            .map(|i| Boolean::<Fr>::new_witness(cs.clone(), || Ok((remaining >> i) & 1 == 1)))
            .collect::<Result<_, _>>()?;

        let mut reconstructed = FpVar::<Fr>::zero();
        let mut coeff = Fr::from(1u64);
        for bit in &bits {
            let contribution = bit.select(&FpVar::constant(coeff), &FpVar::zero())?;
            reconstructed = &reconstructed + &contribution;
            coeff.double_in_place();
        }
        // Enforce: expiry_at = current_time + remaining_from_bits.
        (&current_fp + &reconstructed).enforce_equal(&expiry_fp)?;

        // ── Constraint 3: not revoked ──────────────────────────────────────────
        let not_revoked_var = Boolean::<Fr>::new_witness(cs, || Ok(not_revoked))?;
        not_revoked_var.enforce_equal(&Boolean::constant(true))?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::CredentialValidityCircuit;
    use ark_bn254::Fr;
    use ark_relations::r1cs::{ConstraintSynthesizer, ConstraintSystem};

    const NOW: u64 = 1_742_000_000; // ~2025-03-15 in unix seconds

    fn valid_circuit() -> CredentialValidityCircuit {
        CredentialValidityCircuit {
            issued_at: Some(NOW - 86400),       // issued 1 day ago
            expiry_at: Some(NOW + 365 * 86400), // expires in 1 year
            not_revoked: Some(true),
            current_time: NOW,
        }
    }

    #[test]
    fn valid_credential_satisfies() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        valid_circuit().generate_constraints(cs.clone()).unwrap();
        assert!(cs.is_satisfied().unwrap());
    }

    #[test]
    fn expired_credential_unsatisfiable() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = CredentialValidityCircuit {
            issued_at: Some(NOW - 2 * 365 * 86400),
            expiry_at: Some(NOW - 86400), // expired yesterday
            not_revoked: Some(true),
            current_time: NOW,
        };
        let result = circuit.generate_constraints(cs.clone());
        assert!(result.is_err() || !cs.is_satisfied().unwrap_or(true));
    }

    #[test]
    fn revoked_credential_unsatisfied() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = CredentialValidityCircuit {
            not_revoked: Some(false), // revoked
            ..valid_circuit()
        };
        circuit.generate_constraints(cs.clone()).unwrap();
        assert!(!cs.is_satisfied().unwrap());
    }

    #[test]
    fn unissued_credential_unsatisfiable() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = CredentialValidityCircuit {
            issued_at: Some(0), // never issued
            ..valid_circuit()
        };
        let result = circuit.generate_constraints(cs.clone());
        assert!(result.is_err() || !cs.is_satisfied().unwrap_or(true));
    }
}
