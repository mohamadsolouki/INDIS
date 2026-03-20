package org.indis.app.ui.home

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import org.indis.app.R

/**
 * Shown once on first launch.  Records completion in shared preferences
 * so [IndisApplication] can redirect to [MainActivity] on subsequent launches.
 *
 * PRD FR-001: citizen enrollment entry-point.
 */
class OnboardingActivity : AppCompatActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_onboarding)

        val btnStart = findViewById<Button>(R.id.btn_start_enrollment)
        val btnAlreadyHave = findViewById<TextView>(R.id.tv_already_enrolled)

        btnStart.setOnClickListener {
            markOnboardingSeen()
            startActivity(Intent(this, org.indis.app.ui.enrollment.EnrollmentActivity::class.java))
            finish()
        }

        btnAlreadyHave.setOnClickListener {
            markOnboardingSeen()
            startActivity(Intent(this, MainActivity::class.java))
            finish()
        }
    }

    private fun markOnboardingSeen() {
        getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putBoolean(KEY_SEEN, true)
            .apply()
    }

    companion object {
        const val PREFS = "indis_prefs"
        const val KEY_SEEN = "onboarding_seen"

        /** Returns true when the user has already completed onboarding. */
        fun isCompleted(context: Context): Boolean =
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .getBoolean(KEY_SEEN, false)
    }
}
