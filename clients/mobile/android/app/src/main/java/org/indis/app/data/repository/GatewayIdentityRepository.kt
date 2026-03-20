package org.indis.app.data.repository

import android.content.Context
import org.indis.app.data.network.GatewayApiClient
import org.indis.app.domain.did.DIDManager
import org.json.JSONObject

/**
 * Concrete [IdentityRepository] that registers a citizen's DID with the
 * INDIS gateway and persists the resulting JWT + DID locally.
 *
 * Flow:
 *  1. [DIDManager.generateDid] — derives a DID from the device AndroidKeyStore key.
 *  2. POST /v1/identity/register — anchors the DID on the backend.
 *  3. On success: store DID + JWT in SharedPreferences.
 *
 * PRD FR-001: device-bound Ed25519/P-256 key generates a deterministic DID suffix.
 */
class GatewayIdentityRepository(
    private val context: Context,
    private val api: GatewayApiClient,
    private val didManager: DIDManager = DIDManager(),
) : IdentityRepository {

    private fun prefs() = context.getSharedPreferences("indis_prefs", Context.MODE_PRIVATE)

    override suspend fun enrollNationalId(nationalId: String): Result<String> {
        return runCatching {
            // 1. Generate (or retrieve cached) DID from AndroidKeyStore.
            val existingDid = prefs().getString("did", null)
            val did = existingDid ?: didManager.generateDid()

            // 2. Register with the gateway.
            val body = JSONObject().apply {
                put("national_id", nationalId)
                put("did",         did)
            }.toString()

            val raw = api.post("/v1/identity/register", body, token = "")
            val resp = JSONObject(raw)

            val jwt = resp.optString("token", "")
            val returnedDid = resp.optString("did", did)

            // 3. Persist.
            prefs().edit()
                .putString("did", returnedDid)
                .putString("jwt_token", jwt)
                .apply()

            returnedDid
        }
    }

    /** Returns the locally cached DID, or empty string if not yet registered. */
    fun localDid(): String = prefs().getString("did", "") ?: ""

    /** Returns the locally cached JWT, or empty string if not yet registered. */
    fun localToken(): String = prefs().getString("jwt_token", "") ?: ""
}
