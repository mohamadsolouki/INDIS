// INDIS — Poseidon Hash Template (stub interface for circomlib compatibility)
//
// In production, use the official circomlib Poseidon implementation:
//   https://github.com/iden3/circomlib/blob/master/circuits/poseidon.circom
//
// This stub provides the correct interface signature so that the INDIS circuits
// compile and type-check. Replace this file with the circomlib version when
// running snarkjs to generate proving keys.
//
// Poseidon is a ZK-friendly hash function optimised for algebraic circuits.
// It uses far fewer constraints than SHA-256/Keccak (~230 vs ~27,000 for SHA-256).
//
// Ref: "Poseidon: A New Hash Function for Zero-Knowledge Proof Systems"
//      Grassi et al., USENIX Security 2021.

pragma circom 2.0.0;

// Poseidon(nInputs) hashes nInputs field elements into one field element.
// The actual constants (MDS matrix, round constants) come from circomlib.
template Poseidon(nInputs) {
    signal input inputs[nInputs];
    signal output out;

    // --- STUB IMPLEMENTATION ---
    // Replace with circomlib's Poseidon when compiling for production.
    // This stub uses a simple linear combination so the circuit compiles;
    // it does NOT provide ZK security and must NOT be used for real proofs.
    var acc = 0;
    for (var i = 0; i < nInputs; i++) {
        acc = acc + inputs[i];
    }
    out <== acc;
    // ---------------------------
}
