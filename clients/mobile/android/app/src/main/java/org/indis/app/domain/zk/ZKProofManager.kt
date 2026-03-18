package org.indis.app.domain.zk

class ZKProofManager {
    fun verifyProof(proofBytes: ByteArray): Boolean {
        // JNI bridge to Rust zk module will be added in a later iteration.
        return proofBytes.isNotEmpty()
    }
}
