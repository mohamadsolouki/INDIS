package org.indis.app.service

import android.content.Context
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import org.indis.app.data.network.GatewayApiClient
import java.util.concurrent.TimeUnit

/**
 * Background worker that refreshes the credential revocation list every 6 hours.
 *
 * Fetches GET /v1/credential/revocations from the INDIS gateway and stores
 * the result in SharedPreferences under the key "revocation_list_json".
 * This allows offline ZK proof generation and presentation to check revocation
 * without a live network connection for up to 72 hours (PRD FR-006).
 *
 * Schedule: NETWORK_CONNECTED, periodic every 6 hours, existing work replaced.
 */
class RevocationCacheWorker(
    context: Context,
    params: WorkerParameters,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val prefs = applicationContext.getSharedPreferences("indis_prefs", Context.MODE_PRIVATE)
        val gatewayUrl = prefs.getString("gateway_url", "http://10.0.2.2:8080") ?: "http://10.0.2.2:8080"
        val token = prefs.getString("jwt_token", "") ?: ""

        return try {
            val api = GatewayApiClient(gatewayUrl)
            val json = api.get("/v1/credential/revocations", token)
            prefs.edit()
                .putString("revocation_list_json", json)
                .putLong("revocation_list_fetched_at", System.currentTimeMillis())
                .apply()
            Result.success()
        } catch (e: Exception) {
            // Retry later; keep the stale cache in place.
            Result.retry()
        }
    }

    companion object {
        private const val WORK_NAME = "indis_revocation_cache"

        /**
         * Enqueues the periodic sync. Safe to call multiple times — uses
         * [ExistingPeriodicWorkPolicy.KEEP] so an in-progress sync is not interrupted.
         */
        fun schedule(context: Context) {
            val request = PeriodicWorkRequestBuilder<RevocationCacheWorker>(6, TimeUnit.HOURS)
                .setConstraints(
                    Constraints.Builder()
                        .setRequiredNetworkType(NetworkType.CONNECTED)
                        .build()
                )
                .build()

            WorkManager.getInstance(context).enqueueUniquePeriodicWork(
                WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                request,
            )
        }

        /**
         * Returns the cached revocation list JSON, or null if it has never been
         * fetched or if the cache is older than [maxAgeMs] (default 72 hours).
         */
        fun getCachedRevocations(context: Context, maxAgeMs: Long = 72 * 60 * 60 * 1000L): String? {
            val prefs = context.getSharedPreferences("indis_prefs", Context.MODE_PRIVATE)
            val fetchedAt = prefs.getLong("revocation_list_fetched_at", 0L)
            if (System.currentTimeMillis() - fetchedAt > maxAgeMs) return null
            return prefs.getString("revocation_list_json", null)
        }
    }
}
