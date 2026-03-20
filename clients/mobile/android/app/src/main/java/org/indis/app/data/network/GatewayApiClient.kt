package org.indis.app.data.network

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.TimeUnit

/**
 * Thin wrapper around OkHttp for INDIS gateway calls.
 *
 * All calls are synchronous (run on IO dispatcher via coroutines in the repository layer).
 * Auth header is injected here so every call carries the citizen's JWT automatically.
 */
class GatewayApiClient(private val baseUrl: String) {

    private val http = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    private val json = "application/json; charset=utf-8".toMediaType()

    /** Convenience URL builder. */
    fun endpoint(path: String): String = "\${baseUrl.trimEnd('/')}/${path.trimStart('/')}"

    /** HTTP GET returning raw response body or throws on non-2xx. */
    fun get(path: String, token: String): String {
        val req = Request.Builder()
            .url(endpoint(path))
            .header("Authorization", "Bearer \$token")
            .build()
        http.newCall(req).execute().use { resp ->
            check(resp.isSuccessful) { "GET \$path failed: \${resp.code}" }
            return resp.body?.string() ?: ""
        }
    }

    /** HTTP POST with JSON body returning raw response body or throws on non-2xx. */
    fun post(path: String, jsonBody: String, token: String): String {
        val body = jsonBody.toRequestBody(json)
        val req = Request.Builder()
            .url(endpoint(path))
            .header("Authorization", "Bearer \$token")
            .post(body)
            .build()
        http.newCall(req).execute().use { resp ->
            check(resp.isSuccessful) { "POST \$path failed: \${resp.code}" }
            return resp.body?.string() ?: ""
        }
    }
}
