package org.indis.app.data.repository

import android.content.Context
import org.indis.app.data.local.CredentialEntity
import org.indis.app.data.local.EncryptedWalletDatabase
import org.indis.app.data.network.GatewayApiClient
import org.indis.app.ui.wallet.CredentialCard
import org.json.JSONArray
import org.json.JSONObject

/**
 * Concrete [CredentialRepository] that fetches W3C Verifiable Credentials
 * from the INDIS gateway and caches them in the local Room database.
 *
 * Network strategy:
 *   1. Try GET /v1/identity/{did}/credentials from the gateway.
 *   2. On success: upsert every credential into Room and return the list.
 *   3. On network failure: return the cached Room rows (offline fallback).
 *
 * PRD FR-006: credentials must be available offline for up to 72 hours.
 */
class GatewayCredentialRepository(
    private val context: Context,
    private val api: GatewayApiClient,
) : CredentialRepository {

    private val dao by lazy {
        EncryptedWalletDatabase.getInstance(context).credentialDao()
    }

    private fun prefs() = context.getSharedPreferences("indis_prefs", Context.MODE_PRIVATE)

    private fun did(): String = prefs().getString("did", "") ?: ""
    private fun token(): String = prefs().getString("jwt_token", "") ?: ""

    override suspend fun listCredentials(): List<CredentialCard> {
        val did = did()
        if (did.isNotEmpty() && token().isNotEmpty()) {
            runCatching { fetchAndCache(did, token()) }
            // Swallow network errors — fall through to Room cache.
        }
        return dao.getAllCredentials().map { it.toCard() }
    }

    /**
     * Sync remote credentials into Room.
     * Expects the gateway to return:
     * { "credentials": [ { "id", "type", "issuer", "issuedAt", "expiresAt", "vc", "status" } ] }
     */
    private fun fetchAndCache(did: String, token: String) {
        val raw = api.get("/v1/identity/$did/credentials", token)
        val root = JSONObject(raw)
        val arr: JSONArray = root.optJSONArray("credentials") ?: return

        for (i in 0 until arr.length()) {
            val obj = arr.getJSONObject(i)
            dao.let { /* called in IO dispatcher by callers */ }
            val entity = CredentialEntity(
                id             = obj.getString("id"),
                credentialType = obj.optString("type", "UnknownCredential"),
                issuer         = obj.optString("issuer", ""),
                issuedAt       = obj.optString("issuedAt", ""),
                expiresAt      = obj.optString("expiresAt", ""),
                vcJson         = obj.optString("vc", obj.toString()),
                status         = obj.optString("status", "active"),
            )
            // upsert() is a suspend fun — actual DB writes happen on IO via
            // the suspend fun upsert() in the DAO; callers must use Dispatchers.IO.
            kotlinx.coroutines.runBlocking { dao.upsert(entity) }
        }
    }

    private fun CredentialEntity.toCard() = CredentialCard(
        credentialId = id,
        title        = credentialType,
        expiresAt    = expiresAt,
        revoked      = status == "revoked",
    )
}
