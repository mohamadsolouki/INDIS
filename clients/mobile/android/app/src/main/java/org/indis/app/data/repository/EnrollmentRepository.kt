package org.indis.app.data.repository

import org.indis.app.data.network.GatewayApiClient
import org.json.JSONObject

/**
 * Submits enrollment packages to the INDIS gateway.
 *
 * All network calls use the synchronous OkHttp API; callers must dispatch
 * to a background thread (e.g. [kotlinx.coroutines.Dispatchers.IO]).
 *
 * PRD UC-001 (standard enrollment), POST /v1/enrollment/submit.
 */
class EnrollmentRepository(private val api: GatewayApiClient) {

    /**
     * Submits document + face images for enrollment.
     *
     * @param docImageB64  Base64-encoded JPEG of the identity document rear face.
     * @param faceImageB64 Base64-encoded JPEG of the citizen's face.
     * @param token        Citizen's JWT (may be empty for first-time enrollment).
     * @param onSuccess    Called on the calling thread when the server accepts.
     * @param onError      Called on the calling thread with the failure message.
     */
    fun submit(
        docImageB64: String,
        faceImageB64: String,
        token: String,
        onSuccess: () -> Unit,
        onError: (String) -> Unit,
    ) {
        val body = JSONObject().apply {
            put("document_image", docImageB64)
            put("face_image",     faceImageB64)
        }.toString()

        runCatching { api.post("/v1/enrollment/submit", body, token) }
            .fold(
                onSuccess = { onSuccess() },
                onFailure = { onError(it.message ?: "Enrollment failed") },
            )
    }
}
