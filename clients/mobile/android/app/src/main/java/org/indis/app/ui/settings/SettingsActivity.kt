package org.indis.app.ui.settings

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.view.ViewGroup
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.CompoundButton
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.Spinner
import android.widget.Switch
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import org.indis.app.R
import org.indis.app.ui.home.OnboardingActivity
import org.indis.app.ui.wallet.PrivacyCenterActivity

/**
 * Settings screen — language, Persian numerals toggle, gateway URL, logout.
 *
 * All preferences are stored in "indis_prefs" SharedPreferences so other
 * parts of the app can read them without passing a Settings reference.
 */
class SettingsActivity : AppCompatActivity() {

    private val LANGUAGES = listOf(
        "fa"  to "فارسی",
        "en"  to "English",
        "ckb" to "کوردی سۆرانی",
        "kmr" to "Kurdî Kurmancî",
        "ar"  to "العربية",
        "az"  to "Azərbaycan",
    )

    private fun prefs() = getSharedPreferences("indis_prefs", Context.MODE_PRIVATE)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val root = ScrollView(this)
        val container = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(48, 64, 48, 48)
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT,
            )
        }
        root.addView(container)

        // ── Title ─────────────────────────────────────────────────────────────
        container.addView(TextView(this).apply {
            text = getString(R.string.settings_title)
            textSize = 22f
            setPadding(0, 0, 0, 32)
        })

        // ── Language picker ───────────────────────────────────────────────────
        container.addView(sectionLabel(getString(R.string.settings_language)))

        val langSpinner = Spinner(this)
        val langAdapter = ArrayAdapter(this, android.R.layout.simple_spinner_item, LANGUAGES.map { it.second })
        langAdapter.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item)
        langSpinner.adapter = langAdapter

        val currentLang = prefs().getString("language", "fa") ?: "fa"
        langSpinner.setSelection(LANGUAGES.indexOfFirst { it.first == currentLang }.coerceAtLeast(0))

        langSpinner.onItemSelectedListener = object : AdapterView.OnItemSelectedListener {
            override fun onItemSelected(parent: AdapterView<*>?, view: android.view.View?, pos: Int, id: Long) {
                val tag = LANGUAGES[pos].first
                prefs().edit().putString("language", tag).apply()
                AppCompatDelegate.setApplicationLocales(LocaleListCompat.forLanguageTags(tag))
            }
            override fun onNothingSelected(parent: AdapterView<*>?) = Unit
        }
        container.addView(langSpinner)
        container.addView(divider())

        // ── Persian numerals ──────────────────────────────────────────────────
        container.addView(sectionLabel(getString(R.string.settings_persian_numerals)))
        val numSwitch = Switch(this).apply {
            isChecked = prefs().getBoolean("persian_numerals", true)
        }
        numSwitch.setOnCheckedChangeListener { _: CompoundButton, checked: Boolean ->
            prefs().edit().putBoolean("persian_numerals", checked).apply()
        }
        container.addView(numSwitch)
        container.addView(divider())

        // ── Gateway URL ───────────────────────────────────────────────────────
        container.addView(sectionLabel(getString(R.string.settings_gateway_url)))
        val gatewayEdit = EditText(this).apply {
            setText(prefs().getString("gateway_url", "http://10.0.2.2:8080"))
            hint = "http://10.0.2.2:8080"
        }
        container.addView(gatewayEdit)
        val saveGateway = Button(this).apply { text = getString(R.string.settings_version).let { "حفظ" } }
        saveGateway.text = "ذخیره آدرس"
        saveGateway.setOnClickListener {
            val url = gatewayEdit.text.toString().trim()
            if (url.isNotEmpty()) {
                prefs().edit().putString("gateway_url", url).apply()
                Toast.makeText(this, "ذخیره شد", Toast.LENGTH_SHORT).show()
            }
        }
        container.addView(saveGateway)
        container.addView(divider())

        // ── Privacy Center shortcut ───────────────────────────────────────────
        val privacyBtn = Button(this).apply {
            text = getString(R.string.settings_privacy_center)
            setOnClickListener {
                startActivity(Intent(this@SettingsActivity, PrivacyCenterActivity::class.java))
            }
        }
        container.addView(privacyBtn)
        container.addView(divider())

        // ── App version ───────────────────────────────────────────────────────
        val versionName = runCatching {
            packageManager.getPackageInfo(packageName, 0).versionName
        }.getOrDefault("0.1.0")
        container.addView(TextView(this).apply {
            text = "${getString(R.string.settings_version)}: $versionName"
            textSize = 13f
            setTextColor(resources.getColor(android.R.color.darker_gray, theme))
            setPadding(0, 8, 0, 8)
        })
        container.addView(divider())

        // ── Logout ────────────────────────────────────────────────────────────
        val logoutBtn = Button(this).apply {
            text = getString(R.string.settings_logout)
            setBackgroundColor(resources.getColor(android.R.color.holo_red_light, theme))
            setTextColor(resources.getColor(android.R.color.white, theme))
        }
        logoutBtn.setOnClickListener {
            AlertDialog.Builder(this)
                .setMessage(getString(R.string.settings_logout_confirm))
                .setPositiveButton(android.R.string.ok) { _, _ -> logout() }
                .setNegativeButton(android.R.string.cancel, null)
                .show()
        }
        container.addView(logoutBtn)

        setContentView(root)
    }

    private fun logout() {
        prefs().edit()
            .remove("jwt_token")
            .remove("did")
            .remove("onboarding_seen")
            .apply()
        val intent = Intent(this, OnboardingActivity::class.java)
        intent.flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
        startActivity(intent)
        finish()
    }

    private fun sectionLabel(text: String) = TextView(this).apply {
        this.text = text
        textSize = 14f
        setPadding(0, 16, 0, 4)
        setTextColor(resources.getColor(android.R.color.darker_gray, theme))
    }

    private fun divider() = android.view.View(this).apply {
        layoutParams = LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 1).also {
            it.setMargins(0, 16, 0, 16)
        }
        setBackgroundColor(resources.getColor(android.R.color.darker_gray, theme))
        alpha = 0.2f
    }
}
