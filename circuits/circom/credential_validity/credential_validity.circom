// INDIS — Credential Validity Proof Circuit (Groth16 / PLONK)
// Proves: issued by authorized issuer AND not revoked AND not expired
// See PRD §FR-003 — prove_credential_valid(credential_type)

pragma circom 2.0.0;

template CredentialValidity() {
    // Public inputs
    signal input issuerPublicKey;
    signal input revocationTreeRoot;
    signal input currentTimestamp;

    // Private inputs
    signal input credentialData;
    signal input issuerSignature;
    signal input expiryTimestamp;
    signal input revocationMerkleProof;

    // Output
    signal output isValid;

    // Placeholder
    isValid <-- 1;
    isValid * (1 - isValid) === 0;
}

component main {public [issuerPublicKey, revocationTreeRoot, currentTimestamp]} = CredentialValidity();
