// INDIS — Range Check Template
// Proves n ∈ [0, 2^bits) via bit decomposition.
// Reused by AgeProof, VoterEligibility, CredentialValidity circuits.

pragma circom 2.0.0;

template RangeCheck(bits) {
    signal input n;

    signal bit[bits];
    signal partial[bits + 1];

    partial[0] <== 0;
    for (var i = 0; i < bits; i++) {
        bit[i] <-- (n >> i) & 1;
        bit[i] * (1 - bit[i]) === 0;
        partial[i + 1] <== partial[i] + bit[i] * (1 << i);
    }
    partial[bits] === n;
}
