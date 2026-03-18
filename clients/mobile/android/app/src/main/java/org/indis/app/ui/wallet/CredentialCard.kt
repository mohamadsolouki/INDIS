package org.indis.app.ui.wallet

data class CredentialCard(
    val credentialId: String,
    val title: String,
    val expiresAt: String,
    val revoked: Boolean
)
