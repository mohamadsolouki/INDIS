// INDIS — Citizenship Proof Circuit (Groth16)
// Proves Iranian citizenship without revealing any identifier.
// See PRD §FR-003 — prove_citizenship()

pragma circom 2.0.0;

// TODO: Implement citizenship proof circuit
// - Verify NIA signature on citizenship credential
// - Prove credential is not revoked (Merkle proof against revocation tree)
// - Prove credential is not expired
// - Output: boolean (valid citizen or not)
// - Reveal: NOTHING beyond the boolean result

template CitizenshipProof() {
    // Public inputs
    signal input revocationTreeRoot;
    signal input currentTimestamp;

    // Private inputs
    signal input credentialData;
    signal input issuerSignature;
    signal input revocationMerkleProof;

    // Output
    signal output isValidCitizen;

    // Placeholder
    isValidCitizen <-- 1;
    isValidCitizen * (1 - isValidCitizen) === 0;
}

component main {public [revocationTreeRoot, currentTimestamp]} = CitizenshipProof();
