package org.indis.app.ui.wallet

class PrivacyCenter {
    fun summarizeDisclosure(acceptedClaims: List<String>): String {
        return "Disclosed claims: ${acceptedClaims.joinToString(", ")}"
    }
}
