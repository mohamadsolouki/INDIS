// INDIS — Age Proof Circuit (Groth16)
// Proves age ≥ threshold without revealing exact age.
// See PRD §FR-003 — prove_age_above(threshold)
//
// Public inputs:  threshold, currentDate
// Private inputs: birthDate, issuerSignature
// Output:         1 if age ≥ threshold, 0 otherwise

pragma circom 2.0.0;

// TODO: Implement age proof circuit
// - Verify issuer signature on birth date credential
// - Calculate age from birthDate and currentDate
// - Prove age ≥ threshold without revealing birthDate
// - Include nullifier to prevent replay

template AgeProof() {
    // Public inputs
    signal input threshold;
    signal input currentDate;

    // Private inputs
    signal input birthDate;

    // Output
    signal output isAbove;

    // Placeholder constraint
    // TODO: Replace with actual age comparison logic
    isAbove <-- (currentDate - birthDate) >= threshold ? 1 : 0;
    isAbove * (1 - isAbove) === 0; // Boolean constraint
}

component main {public [threshold, currentDate]} = AgeProof();
