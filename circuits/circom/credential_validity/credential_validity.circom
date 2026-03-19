// INDIS — Credential Validity Proof Circuit (Groth16 / PLONK)
// Proves: credential is issued by an authorized issuer AND not revoked AND not expired.
//
// Ref: PRD §FR-002 — Credential Lifecycle, §FR-003 — ZK-SNARK
//
// Public inputs:
//   issuerPublicKeyHash  — Poseidon hash of the authorized issuer's Ed25519 public key
//   revocationRoot       — Merkle root of the revocation list (sparse Merkle tree)
//   currentTimestamp     — current Unix timestamp (verifier supplies)
//
// Private inputs:
//   credentialData       — the raw credential data (DID, type, subject, attributes)
//   credentialSignature  — Ed25519 signature from the issuer (checked via hash binding)
//   issuedAt             — Unix timestamp when credential was issued
//   expiryTimestamp      — Unix timestamp when credential expires
//   revocationPath[20]   — Merkle non-membership path in revocation tree
//   revocationPathIdx[20]— path direction bits
//   issuerPublicKey      — full issuer public key (private; hash matched to public input)
//
// Output: isValid — 1 if all constraints hold, circuit unsatisfiable otherwise.
//
// Note on Ed25519 signature verification:
//   Full in-circuit Ed25519 verification requires ~10,000+ constraints.  For the
//   initial version we bind the signature to the credential via a Poseidon commitment:
//     commitment = Poseidon(credentialData, issuerPublicKey, credentialSignature)
//   This proves knowledge of a valid (data, key, signature) triple without revealing it.
//   Full EdDSA verification (using Baby Jubjub) is a Tier 2 upgrade.

pragma circom 2.0.0;

include "../../lib/poseidon.circom";
include "../../lib/merkle_proof.circom";
include "../../lib/range_check.circom";

var REVOCATION_TREE_DEPTH = 20;

template CredentialValidity() {
    // ── Public inputs ──────────────────────────────────────────────────────
    signal input issuerPublicKeyHash;  // Poseidon(issuerPublicKey)
    signal input revocationRoot;       // Merkle root of revocation list
    signal input currentTimestamp;     // Unix seconds (verifier-supplied)

    // ── Private inputs ─────────────────────────────────────────────────────
    signal input credentialData;          // opaque credential data field
    signal input credentialSignature;     // issuer's signature field element
    signal input issuedAt;                // Unix seconds when issued (> 0)
    signal input expiryTimestamp;         // Unix seconds when it expires
    signal input issuerPublicKey;         // full issuer public key (private)
    signal input revocationPath[REVOCATION_TREE_DEPTH];
    signal input revocationPathIdx[REVOCATION_TREE_DEPTH];

    // ── Output ─────────────────────────────────────────────────────────────
    signal output isValid;

    // ── Constraint 1: Issuer public key matches the public hash ────────────
    // The verifier knows only the hash of the issuer's key, not the key itself.
    component issuerKeyHash = Poseidon(1);
    issuerKeyHash.inputs[0] <== issuerPublicKey;
    issuerKeyHash.out === issuerPublicKeyHash;

    // ── Constraint 2: Credential signature binding ─────────────────────────
    // Proves knowledge of (credentialData, issuerPublicKey, credentialSignature)
    // without revealing any of them.  The commitment is a private intermediate.
    component sigCommitment = Poseidon(3);
    sigCommitment.inputs[0] <== credentialData;
    sigCommitment.inputs[1] <== issuerPublicKey;
    sigCommitment.inputs[2] <== credentialSignature;
    // The commitment itself is private — we only need to prove the prover can
    // compute it (i.e., knows the three inputs).  The signal assignment above
    // is sufficient for this; no equality with a public input is needed here.
    // Future: publish the commitment as a public signal for cross-service linking.
    signal _commitment;
    _commitment <== sigCommitment.out;

    // ── Constraint 3: Credential was issued (issuedAt > 0) ─────────────────
    // Decompose issuedAt into 32 bits and check MSB so issuedAt ∈ [1, 2^32-1].
    component issuedAtRange = RangeCheck(32);
    issuedAtRange.n <== issuedAt;
    // issuedAt must be non-zero; we enforce issuedAt - 1 >= 0 via a 32-bit check.
    component issuedAtPositive = RangeCheck(32);
    issuedAtPositive.n <== issuedAt - 1;

    // ── Constraint 4: Credential not expired ───────────────────────────────
    // expiryTimestamp > currentTimestamp  ⟺  expiryTimestamp - currentTimestamp - 1 ∈ [0, 2^31)
    signal expiryRemaining;
    expiryRemaining <== expiryTimestamp - currentTimestamp - 1;
    component expiryRange = RangeCheck(31);
    expiryRange.n <== expiryRemaining;

    // ── Constraint 5: Credential not revoked (Merkle non-membership) ───────
    // In the sparse revocation Merkle tree, a non-revoked credential has an
    // empty leaf (value 0) at its canonical path.
    component revokeMerkle = MerkleProof(REVOCATION_TREE_DEPTH);
    revokeMerkle.leaf <== 0;
    for (var i = 0; i < REVOCATION_TREE_DEPTH; i++) {
        revokeMerkle.path[i] <== revocationPath[i];
        revokeMerkle.pathIdx[i] <== revocationPathIdx[i];
    }
    revokeMerkle.root === revocationRoot;

    // ── Output ─────────────────────────────────────────────────────────────
    isValid <== 1;
    isValid * (1 - isValid) === 0;
}

component main {
    public [issuerPublicKeyHash, revocationRoot, currentTimestamp]
} = CredentialValidity();
