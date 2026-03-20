package org.indis.app.ui.wallet

/**
 * UI model for a single W3C Verifiable Credential in the wallet list.
 *
 * Intentionally separate from the data-layer entity so the UI does not
 * depend on Room schema details.
 */
data class CredentialCard(
    val id: String,
    val type: String,
    val title: String,
    val issuedAt: String,
    val expiresAt: String,
    val isRevoked: Boolean,
) {
    /** Kept for source-compatibility with the old [revoked] field name. */
    val revoked: Boolean get() = isRevoked
}
