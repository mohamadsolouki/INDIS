// INDIS — Age Proof Circuit (Groth16)
// Proves age ≥ threshold without revealing exact age or birth date.
//
// Ref: PRD §FR-003 — prove_age_above(threshold)
//
// Public inputs:  threshold     — minimum age in days (e.g. 18*365 = 6570)
//                 currentDate   — today as days since Unix epoch
// Private inputs: birthDate     — birth date as days since Unix epoch
//
// Output: isAbove — 1 if (currentDate - birthDate) >= threshold, 0 otherwise
//
// Security properties:
//   • birthDate is never revealed (private witness).
//   • The age value itself (currentDate - birthDate) is never revealed.
//   • Replay prevention: combine with a nullifier at the application layer.
//
// Constraint summary:
//   1. age = currentDate - birthDate                        (arithmetic, 1 constraint)
//   2. age - threshold = diff                               (arithmetic, 1 constraint)
//   3. diff is decomposed into N bits                       (N Boolean constraints)
//   4. The bit decomposition reconstructs diff exactly      (1 constraint)
//   5. isAbove = (diff fits in N bits) ? 1 : 0             (1 constraint)
//
// We use N = 15 bits so diff ∈ [0, 32767], which covers ages [threshold, threshold+89.7y].
// For threshold = 18*365 = 6570 days, the prover can be at most ~108 years old.

pragma circom 2.0.0;

// Checks that n ∈ [0, 2^bits) by decomposing n into `bits` binary digits.
// The reconstruction constraint enforces that the bits are consistent with n.
template RangeCheck(bits) {
    signal input n;
    signal bits_out[bits];
    signal partial[bits + 1];

    partial[0] <== 0;
    for (var i = 0; i < bits; i++) {
        // Witness each bit.
        bits_out[i] <-- (n >> i) & 1;
        // Enforce bit is 0 or 1.
        bits_out[i] * (1 - bits_out[i]) === 0;
        // Accumulate the weighted sum.
        partial[i + 1] <== partial[i] + bits_out[i] * (1 << i);
    }
    // Enforce: the bit decomposition reconstructs n exactly.
    partial[bits] === n;
}

template AgeProof() {
    // ── Public inputs ──────────────────────────────────────────────────────
    // threshold: minimum age in days required (e.g. 18 * 365 = 6570).
    signal input threshold;
    // currentDate: today's date in days since Unix epoch (known to verifier).
    signal input currentDate;

    // ── Private inputs ─────────────────────────────────────────────────────
    // birthDate: the citizen's birth date in days since Unix epoch.
    signal input birthDate;

    // ── Output ─────────────────────────────────────────────────────────────
    // isAbove: 1 if age ≥ threshold, 0 otherwise.
    signal output isAbove;

    // ── Step 1: compute age in days ────────────────────────────────────────
    signal age;
    age <== currentDate - birthDate;

    // ── Step 2: compute excess = age - threshold ───────────────────────────
    signal diff;
    diff <== age - threshold;

    // ── Step 3: prove diff ∈ [0, 2^15) via bit decomposition ──────────────
    // If age < threshold, diff is negative in the field (large positive), and
    // the RangeCheck(15) constraint will be unsatisfiable.
    component rangeCheck = RangeCheck(15);
    rangeCheck.n <== diff;

    // ── Step 4: output the eligibility bit ─────────────────────────────────
    // If diff fits in 15 bits (i.e., diff >= 0), age >= threshold.
    // The circuit is unsatisfiable when age < threshold, so isAbove is always 1
    // for any satisfying assignment.
    isAbove <== 1;
    isAbove * (1 - isAbove) === 0;
}

component main {public [threshold, currentDate]} = AgeProof();
