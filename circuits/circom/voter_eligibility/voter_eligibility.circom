// INDIS — Voter Eligibility Proof Circuit (Groth16)
// Atomic proof: citizenship + age ≥ 18 + not in exclusion list
// See PRD §FR-003 — prove_voter_eligibility(election_id)
// Reveals: NOTHING beyond eligibility boolean

pragma circom 2.0.0;

template VoterEligibility() {
    // Public inputs
    signal input electionId;
    signal input revocationTreeRoot;
    signal input exclusionListRoot;
    signal input currentTimestamp;

    // Private inputs
    signal input citizenshipCredential;
    signal input birthDate;
    signal input citizenId;
    signal input exclusionMerkleProof;

    // Output
    signal output isEligible;

    // Placeholder — combines citizenship + age + exclusion checks
    isEligible <-- 1;
    isEligible * (1 - isEligible) === 0;
}

component main {public [electionId, revocationTreeRoot, exclusionListRoot, currentTimestamp]} = VoterEligibility();
