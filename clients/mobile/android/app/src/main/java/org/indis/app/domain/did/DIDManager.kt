package org.indis.app.domain.did

import java.security.KeyPairGenerator
import java.security.MessageDigest
import java.security.spec.ECGenParameterSpec

class DIDManager {
    fun generateDid(): String {
        val keyPairGenerator = KeyPairGenerator.getInstance("EC", "AndroidKeyStore")
        val alias = "indis_device_signing_key"
        val spec = android.security.keystore.KeyGenParameterSpec.Builder(
            alias,
            android.security.keystore.KeyProperties.PURPOSE_SIGN or android.security.keystore.KeyProperties.PURPOSE_VERIFY
        )
            .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
            .setDigests(
                android.security.keystore.KeyProperties.DIGEST_SHA256,
                android.security.keystore.KeyProperties.DIGEST_SHA512
            )
            .setUserAuthenticationRequired(false)
            .build()

        keyPairGenerator.initialize(spec)
        val keyPair = keyPairGenerator.generateKeyPair()
        val pubEncoded = keyPair.public.encoded
        val digest = MessageDigest.getInstance("SHA-256").digest(pubEncoded)
        val didSuffix = digest.copyOfRange(0, 20).joinToString("") { "%02x".format(it) }
        return "did:indis:$didSuffix"
    }
}
