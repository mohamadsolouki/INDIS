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
     * @param documentBase64 Base64-encoded JPEG of the identity document.
     * @param faceBase64     Base64-encoded JPEG of the citizen's face.
     * @param pathway        Enrollment pathway: "standard", "enhanced", or "social".
     * @return               Server-assigned enrollment tracking ID.
     * @throws Exception     On network or server error.
     */
    fun submitEnrollment(
        documentBase64: String,
        faceBase64: String,
        pathway: String,
    ): String {
        val body = JSONObject().apply {
            put("document_image", documentBase64)
            put("face_image",     faceBase64)
            put("pathway",        pathway)
        }.toString()

        val response = api.post("/v1/enrollment/submit", body, token = "")
        return JSONObject(response).getString("enrollment_id")
    }
}
