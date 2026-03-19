/// AgeRangeCircuit: proves age >= threshold without revealing the exact age.
///
/// Ref: PRD §FR-003 — ZK-SNARK credential verification.
///
/// Private witness: `age` (the citizen's actual age)
/// Public input:   `threshold` (the minimum required age, e.g. 18)
///
/// Constraints (3 total):
/// 1. Boolean constraints on 8 bits: b_i * (1 - b_i) = 0  (enforced by Boolean gadget)
/// 2. Linear combination:  sum(b_i * 2^i) = age - threshold
/// 3. age = threshold + excess  (enforced by enforce_equal)
///
/// This proves  age - threshold ∈ [0, 255],  i.e.  threshold ≤ age < threshold + 256.
/// For all practical ages (18–130) and threshold = 18 this covers the full human range.
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
pub struct AgeRangeCircuit {
    /// The citizen's actual age (private witness).
    pub age: Option<u64>,
    /// The minimum age to prove against (public input, e.g. 18).
    pub threshold: u64,
}

impl ConstraintSynthesizer<Fr> for AgeRangeCircuit {
    fn generate_constraints(self, cs: ConstraintSystemRef<Fr>) -> Result<(), SynthesisError> {
        let age = self.age.ok_or(SynthesisError::AssignmentMissing)?;
        let threshold = self.threshold;

        // The prover cannot satisfy the circuit if age < threshold.
        let excess_val = if age >= threshold {
            age - threshold
        } else {
            return Err(SynthesisError::Unsatisfiable);
        };

        // Public input: threshold (the verifier knows this).
        let threshold_fp = FpVar::<Fr>::new_input(cs.clone(), || Ok(Fr::from(threshold)))?;

        // Private witness: age.
        let age_fp = FpVar::<Fr>::new_witness(cs.clone(), || Ok(Fr::from(age)))?;

        // Bit-decompose excess = age - threshold into 8 bits.
        // Each Boolean::new_witness automatically enforces the bit is 0 or 1.
        let bits: Vec<Boolean<Fr>> = (0u64..8)
            .map(|i| Boolean::<Fr>::new_witness(cs.clone(), || Ok((excess_val >> i) & 1 == 1)))
            .collect::<Result<_, _>>()?;

        // Reconstruct the excess value from bits: excess = sum(b_i * 2^i).
        let mut reconstructed = FpVar::<Fr>::zero();
        let mut coeff = Fr::from(1u64);
        for bit in &bits {
            // If bit is 1 add coeff, otherwise add 0.
            let contribution = bit.select(&FpVar::constant(coeff), &FpVar::zero())?;
            reconstructed = &reconstructed + &contribution;
            coeff.double_in_place();
        }

        // Enforce: age = threshold + excess_from_bits.
        // This proves age - threshold is represented exactly by 8 bits
        // (hence in [0, 255]) and that age = threshold + that excess.
        (&threshold_fp + &reconstructed).enforce_equal(&age_fp)?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::AgeRangeCircuit;
    use ark_bn254::Fr;
    use ark_relations::r1cs::{ConstraintSynthesizer, ConstraintSystem};

    #[test]
    fn age_above_threshold_satisfies() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = AgeRangeCircuit { age: Some(35), threshold: 18 };
        circuit.generate_constraints(cs.clone()).unwrap();
        assert!(cs.is_satisfied().unwrap());
    }

    #[test]
    fn age_at_threshold_satisfies() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = AgeRangeCircuit { age: Some(18), threshold: 18 };
        circuit.generate_constraints(cs.clone()).unwrap();
        assert!(cs.is_satisfied().unwrap());
    }

    #[test]
    fn age_below_threshold_unsatisfiable() {
        let cs = ConstraintSystem::<Fr>::new_ref();
        let circuit = AgeRangeCircuit { age: Some(17), threshold: 18 };
        let result = circuit.generate_constraints(cs.clone());
        // Should return SynthesisError::Unsatisfiable or leave cs unsatisfied.
        assert!(result.is_err() || !cs.is_satisfied().unwrap_or(true));
    }
}
