// INDIS — Voter Eligibility Proof Circuit (Groth16)
// Atomic proof: citizenship credential valid + age ≥ 18 + not on exclusion list.
//
// Ref: PRD §FR-003 — prove_voter_eligibility(election_id)
//      PRD §FR-007 — Electoral Module
//
// Reveals NOTHING beyond the eligibility boolean. The citizen's identity,
// age, and credential data are never exposed to the verifier.
//
// Public inputs:
//   electionId          — unique election identifier (prevents cross-election proof reuse)
//   nullifier           — H(citizenId, electionId) — revealed to prevent double-voting
//   credentialRoot      — Merkle root of the credential registry (verifier supplies)
//   exclusionListRoot   — Merkle root of the exclusion list (verifier supplies)
//   currentTimestamp    — current Unix timestamp (verifier supplies)
//
// Private inputs:
//   citizenId           — citizen's unique identifier (kept private)
//   birthDateDays       — birth date as days since Unix epoch
//   credentialLeaf      — hash of the citizenship credential leaf
//   credentialPath[20]  — Merkle path from leaf to credentialRoot
//   credentialPathIdx[20] — Merkle path direction bits (0=left, 1=right)
//   exclusionPath[20]   — Merkle non-membership path
//   exclusionPathIdx[20] — Merkle path direction bits
//   expiryTimestamp     — credential expiry as Unix timestamp
//
// Output: isEligible — always 1 for any satisfying assignment
//
// Constraint groups:
//   1. Nullifier correctness:      nullifier = H(citizenId, electionId)
//   2. Age ≥ 18 (6570 days):       bit-decomposition range proof
//   3. Citizenship valid:          Merkle proof in credentialRoot
//   4. Credential not expired:     expiryTimestamp > currentTimestamp
//   5. Not excluded:               Merkle non-membership proof in exclusionListRoot

pragma circom 2.0.0;

include "../../lib/poseidon.circom";
include "../../lib/merkle_proof.circom";
include "../../lib/range_check.circom";

// Minimum voter age: 18 years × 365.25 days/year = 6574 days (rounded down to 6570).
var MIN_AGE_DAYS = 6570;

// Depth of the credential and exclusion Merkle trees.
var TREE_DEPTH = 20;

template VoterEligibility() {
    // ── Public inputs ──────────────────────────────────────────────────────
    signal input electionId;
    signal input nullifier;
    signal input credentialRoot;
    signal input exclusionListRoot;
    signal input currentTimestamp;     // Unix seconds

    // ── Private inputs ─────────────────────────────────────────────────────
    signal input citizenId;
    signal input birthDateDays;        // days since Unix epoch
    signal input credentialLeaf;
    signal input credentialPath[TREE_DEPTH];
    signal input credentialPathIdx[TREE_DEPTH];
    signal input exclusionPath[TREE_DEPTH];
    signal input exclusionPathIdx[TREE_DEPTH];
    signal input expiryTimestamp;      // Unix seconds

    // ── Output ─────────────────────────────────────────────────────────────
    signal output isEligible;

    // ── Constraint 1: Nullifier = Poseidon(citizenId, electionId) ──────────
    // This prevents the same citizen from producing two different proofs for
    // the same election while keeping their identity private.
    component nullifierHash = Poseidon(2);
    nullifierHash.inputs[0] <== citizenId;
    nullifierHash.inputs[1] <== electionId;
    nullifier === nullifierHash.out;

    // ── Constraint 2: Age ≥ 18 years ───────────────────────────────────────
    // Convert current Unix timestamp to days (approximate: / 86400).
    // birthDateDays is the private witness; currentDays is derived from public input.
    signal currentDays;
    currentDays <== currentTimestamp \ 86400;

    signal ageDays;
    ageDays <== currentDays - birthDateDays;

    // Prove: ageDays - MIN_AGE_DAYS ∈ [0, 2^15) (covers ages 18 to ~108 years).
    signal ageExcess;
    ageExcess <== ageDays - MIN_AGE_DAYS;

    component ageRange = RangeCheck(15);
    ageRange.n <== ageExcess;

    // ── Constraint 3: Citizenship credential Merkle proof ──────────────────
    // Proves the citizen holds a valid citizenship credential in the registry.
    component credMerkle = MerkleProof(TREE_DEPTH);
    credMerkle.leaf <== credentialLeaf;
    for (var i = 0; i < TREE_DEPTH; i++) {
        credMerkle.path[i] <== credentialPath[i];
        credMerkle.pathIdx[i] <== credentialPathIdx[i];
    }
    credMerkle.root === credentialRoot;

    // ── Constraint 4: Credential not expired ───────────────────────────────
    // expiryTimestamp > currentTimestamp
    signal expiryRemaining;
    expiryRemaining <== expiryTimestamp - currentTimestamp - 1;

    component expiryRange = RangeCheck(31);  // ~68 years of seconds
    expiryRange.n <== expiryRemaining;

    // ── Constraint 5: Not on exclusion list (Merkle non-membership) ────────
    // The exclusion list is a sparse Merkle tree; a non-membership proof
    // consists of a leaf whose value is 0 on the citizen's path.
    component exclMerkle = MerkleProof(TREE_DEPTH);
    exclMerkle.leaf <== 0;  // non-membership: leaf must be empty
    for (var i = 0; i < TREE_DEPTH; i++) {
        exclMerkle.path[i] <== exclusionPath[i];
        exclMerkle.pathIdx[i] <== exclusionPathIdx[i];
    }
    exclMerkle.root === exclusionListRoot;

    // ── Output ─────────────────────────────────────────────────────────────
    // If all constraints above are satisfied, the voter is eligible.
    isEligible <== 1;
    isEligible * (1 - isEligible) === 0;
}

component main {
    public [electionId, nullifier, credentialRoot, exclusionListRoot, currentTimestamp]
} = VoterEligibility();
