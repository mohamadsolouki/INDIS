// INDIS — Merkle Proof Verification Template
// Verifies a Merkle inclusion (or non-membership) proof using Poseidon hashing.
//
// For inclusion proofs: leaf = H(credential_data), root = known credential registry root.
// For non-membership proofs: leaf = 0 (empty node), root = exclusion list root.
//
// Uses Poseidon(2) for internal node hashing (left, right order based on path index bit).

pragma circom 2.0.0;

include "poseidon.circom";

// depth: number of levels in the Merkle tree (e.g. 20 for a tree of 2^20 leaves).
template MerkleProof(depth) {
    signal input leaf;
    signal input path[depth];      // sibling hash at each level
    signal input pathIdx[depth];   // 0 = current node is left child, 1 = right child

    signal output root;

    component hasher[depth];
    signal levelHash[depth + 1];
    levelHash[0] <== leaf;

    for (var i = 0; i < depth; i++) {
        pathIdx[i] * (1 - pathIdx[i]) === 0;  // enforce pathIdx is boolean

        hasher[i] = Poseidon(2);

        // If pathIdx[i] == 0: current node is LEFT child → hash(current, sibling)
        // If pathIdx[i] == 1: current node is RIGHT child → hash(sibling, current)
        signal left[depth];
        signal right[depth];
        left[i] <== (1 - pathIdx[i]) * levelHash[i] + pathIdx[i] * path[i];
        right[i] <== pathIdx[i] * levelHash[i] + (1 - pathIdx[i]) * path[i];

        hasher[i].inputs[0] <== left[i];
        hasher[i].inputs[1] <== right[i];
        levelHash[i + 1] <== hasher[i].out;
    }

    root <== levelHash[depth];
}
