package org.indis.app.domain.biometric

import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity

/**
 * BiometricAuthHelper — wraps AndroidX BiometricPrompt for fingerprint / face unlock.
 *
 * Use this to gate access to the credential wallet and privacy center
 * with the device's built-in biometric authenticator.
 *
 * Supports: BIOMETRIC_STRONG (fingerprint, iris, 3D face) and
 * DEVICE_CREDENTIAL (PIN/pattern/password) as fallback.
 *
 * Usage:
 * ```kotlin
 * val helper = BiometricAuthHelper(activity)
 * if (helper.canAuthenticate()) {
 *     helper.authenticate(
 *         title    = "باز کردن کیف پول",
 *         subtitle = "برای دسترسی به اعتبارنامه‌ها تأیید کنید",
 *         onSuccess = { /* proceed */ },
 *         onError   = { msg -> showError(msg) },
 *     )
 * } else {
 *     // No biometric enrolled — proceed directly or show enrollment prompt
 * }
 * ```
 */
class BiometricAuthHelper(private val activity: FragmentActivity) {

    private val manager = BiometricManager.from(activity)

    /**
     * Returns true when at least one strong biometric authenticator is enrolled
     * and the hardware is available.
     */
    fun canAuthenticate(): Boolean {
        val result = manager.canAuthenticate(
            BiometricManager.Authenticators.BIOMETRIC_STRONG or
            BiometricManager.Authenticators.DEVICE_CREDENTIAL
        )
        return result == BiometricManager.BIOMETRIC_SUCCESS
    }

    /**
     * Shows the system biometric prompt.
     *
     * @param title       Primary title shown in the dialog.
     * @param subtitle    Secondary text (e.g., action description).
     * @param description Optional longer description.
     * @param onSuccess   Called on the main thread when authentication succeeds.
     * @param onError     Called on the main thread on failure or cancellation.
     */
    fun authenticate(
        title: String,
        subtitle: String,
        description: String = "",
        onSuccess: () -> Unit,
        onError: (String) -> Unit,
    ) {
        val executor = ContextCompat.getMainExecutor(activity)

        val callback = object : BiometricPrompt.AuthenticationCallback() {
            override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                onSuccess()
            }

            override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                if (errorCode != BiometricPrompt.ERROR_USER_CANCELED &&
                    errorCode != BiometricPrompt.ERROR_NEGATIVE_BUTTON
                ) {
                    onError(errString.toString())
                }
            }

            override fun onAuthenticationFailed() {
                // Single failure — the system shows its own feedback; don't dismiss yet.
            }
        }

        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle(title)
            .setSubtitle(subtitle)
            .apply { if (description.isNotBlank()) setDescription(description) }
            .setAllowedAuthenticators(
                BiometricManager.Authenticators.BIOMETRIC_STRONG or
                BiometricManager.Authenticators.DEVICE_CREDENTIAL
            )
            .build()

        BiometricPrompt(activity, executor, callback).authenticate(promptInfo)
    }
}
