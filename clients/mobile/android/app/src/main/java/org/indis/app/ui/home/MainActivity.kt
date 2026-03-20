package org.indis.app.ui.home

import android.content.Intent
import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import com.google.android.material.bottomnavigation.BottomNavigationView
import org.indis.app.R
import org.indis.app.ui.enrollment.EnrollmentActivity
import org.indis.app.ui.settings.SettingsActivity
import org.indis.app.ui.verify.VerifyActivity
import org.indis.app.ui.wallet.WalletActivity

/**
 * Main entry-point for the INDIS citizen app.
 *
 * Hosts a [BottomNavigationView] that switches between the four primary
 * destinations: wallet, enrollment, verification, and settings.
 * Each destination is an independent Activity so deep-links work without
 * a fragment back-stack dependency.
 */
class MainActivity : AppCompatActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val nav = findViewById<BottomNavigationView>(R.id.bottom_nav)
        // Wallet is the default landing after onboarding.
        nav.selectedItemId = R.id.nav_wallet

        nav.setOnItemSelectedListener { item ->
            when (item.itemId) {
                R.id.nav_wallet -> {
                    startActivity(Intent(this, WalletActivity::class.java))
                    true
                }
                R.id.nav_enroll -> {
                    startActivity(Intent(this, EnrollmentActivity::class.java))
                    true
                }
                R.id.nav_verify -> {
                    startActivity(Intent(this, VerifyActivity::class.java))
                    true
                }
                R.id.nav_settings -> {
                    startActivity(Intent(this, SettingsActivity::class.java))
                    true
                }
                else -> false
            }
        }
    }
}
