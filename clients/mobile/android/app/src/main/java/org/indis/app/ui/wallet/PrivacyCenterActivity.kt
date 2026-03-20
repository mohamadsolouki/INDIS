package org.indis.app.ui.wallet

import android.content.Context
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.LinearLayout
import android.widget.ListView
import android.widget.TabHost
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.indis.app.R
import org.indis.app.data.network.GatewayApiClient
import org.json.JSONArray
import org.json.JSONObject

/**
 * Privacy Control Center — lets the citizen review and control their data.
 *
 * Three tabs:
 *  • تاریخچه  (History)  — recent verification events (GET /v1/privacy/history)
 *  • رضایت    (Consent)  — per-verifier consent rules  (GET /v1/privacy/consent)
 *  • خروجی   (Export)   — request a GDPR-style data export (POST /v1/privacy/export)
 *
 * PRD FR-008: citizen controls which verifiers may access which credential types.
 */
class PrivacyCenterActivity : AppCompatActivity() {

    private val api by lazy {
        GatewayApiClient(gatewayBaseUrl())
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Build layout programmatically — avoids adding another XML file.
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT,
            )
        }

        // Title bar
        val title = TextView(this).apply {
            text = getString(R.string.privacy_center_title)
            textSize = 20f
            setPadding(32, 48, 32, 16)
            setTextColor(resources.getColor(android.R.color.black, theme))
        }
        root.addView(title)

        val tabHost = TabHost(this).apply {
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT,
            )
        }
        root.addView(tabHost)

        tabHost.setup()

        // ── History tab ──────────────────────────────────────────────────────
        val historyList = ListView(this).apply { id = View.generateViewId() }
        tabHost.addTab(
            tabHost.newTabSpec("history")
                .setIndicator(getString(R.string.privacy_history_tab))
                .setContent { historyList },
        )

        // ── Consent tab ──────────────────────────────────────────────────────
        val consentList = ListView(this).apply { id = View.generateViewId() }
        tabHost.addTab(
            tabHost.newTabSpec("consent")
                .setIndicator(getString(R.string.privacy_consent_tab))
                .setContent { consentList },
        )

        // ── Export tab ───────────────────────────────────────────────────────
        val exportLayout = LinearLayout(this).apply {
            id = View.generateViewId()
            orientation = LinearLayout.VERTICAL
            setPadding(32, 32, 32, 32)
        }
        val exportDesc = TextView(this).apply {
            text = getString(R.string.privacy_export_desc)
            textSize = 14f
            setPadding(0, 0, 0, 24)
        }
        val exportBtn = Button(this).apply {
            text = getString(R.string.privacy_export_btn)
        }
        exportLayout.addView(exportDesc)
        exportLayout.addView(exportBtn)

        tabHost.addTab(
            tabHost.newTabSpec("export")
                .setIndicator(getString(R.string.privacy_export_tab))
                .setContent { exportLayout },
        )

        setContentView(root)

        // ── Load data ────────────────────────────────────────────────────────
        loadHistory(historyList)
        loadConsent(consentList)

        exportBtn.setOnClickListener { requestExport() }
    }

    private fun loadHistory(listView: ListView) {
        lifecycleScope.launch {
            val items = withContext(Dispatchers.IO) {
                runCatching {
                    val raw = api.get("/v1/privacy/history", token())
                    val arr: JSONArray = JSONObject(raw).optJSONArray("events") ?: JSONArray()
                    (0 until arr.length()).map { i ->
                        val ev = arr.getJSONObject(i)
                        "${ev.optString("timestamp")} — ${ev.optString("verifier_name")} — ${ev.optString("credential_type")}"
                    }
                }.getOrElse { listOf(getString(R.string.privacy_load_error)) }
            }
            listView.adapter = ArrayAdapter(this@PrivacyCenterActivity, android.R.layout.simple_list_item_1, items)
        }
    }

    private fun loadConsent(listView: ListView) {
        lifecycleScope.launch {
            val items = withContext(Dispatchers.IO) {
                runCatching {
                    val raw = api.get("/v1/privacy/consent", token())
                    val arr: JSONArray = JSONObject(raw).optJSONArray("rules") ?: JSONArray()
                    (0 until arr.length()).map { i ->
                        val r = arr.getJSONObject(i)
                        "${r.optString("verifier_id")} | ${r.optString("credential_type")} | ${r.optString("rule")}"
                    }
                }.getOrElse { listOf(getString(R.string.privacy_load_error)) }
            }
            listView.adapter = ArrayAdapter(this@PrivacyCenterActivity, android.R.layout.simple_list_item_1, items)
        }
    }

    private fun requestExport() {
        lifecycleScope.launch {
            val ok = withContext(Dispatchers.IO) {
                runCatching {
                    val did = did()
                    api.post("/v1/privacy/export", """{"did":"$did"}""", token())
                    true
                }.getOrElse { false }
            }
            Toast.makeText(
                this@PrivacyCenterActivity,
                if (ok) getString(R.string.privacy_export_requested) else getString(R.string.privacy_load_error),
                Toast.LENGTH_SHORT,
            ).show()
        }
    }

    private fun prefs() = getSharedPreferences("indis_prefs", Context.MODE_PRIVATE)
    private fun token(): String = prefs().getString("jwt_token", "") ?: ""
    private fun did(): String = prefs().getString("did", "") ?: ""
    private fun gatewayBaseUrl(): String =
        prefs().getString("gateway_url", "http://10.0.2.2:8080") ?: "http://10.0.2.2:8080"
}
