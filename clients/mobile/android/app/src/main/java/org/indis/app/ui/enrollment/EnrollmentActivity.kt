package org.indis.app.ui.enrollment

import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.fragment.app.commit
import org.indis.app.R
import org.indis.app.data.network.GatewayApiClient
import org.indis.app.data.repository.EnrollmentRepository

/**
 * Multi-step enrollment wizard.
 *
 * Step 1: Document capture  — [DocumentFragment]
 * Step 2: Biometric capture — [BiometricFragment]
 *
 * On completion the [EnrollmentRepository] submits the enrollment package
 * to the gateway (POST /v1/enrollment/submit).
 *
 * PRD UC-001 (standard enrollment pathway).
 */
class EnrollmentActivity : AppCompatActivity() {

    private lateinit var tvStepIndicator: TextView
    private lateinit var btnNext: Button

    private val totalSteps = 2
    private var currentStep = 1

    private var docImageB64: String? = null
    private var faceImageB64: String? = null

    private val gatewayClient by lazy {
        GatewayApiClient("http://10.0.2.2:8080") // emulator localhost
    }
    private val enrollmentRepo by lazy { EnrollmentRepository(gatewayClient) }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_enrollment)

        tvStepIndicator = findViewById(R.id.tv_step_indicator)
        btnNext         = findViewById(R.id.btn_next)

        if (savedInstanceState == null) showStep(1)

        btnNext.setOnClickListener { advanceStep() }
    }

    private fun showStep(step: Int) {
        currentStep = step
        tvStepIndicator.text = "$step / $totalSteps"

        supportFragmentManager.commit {
            replace(
                R.id.fragment_container,
                if (step == 1) DocumentFragment() else BiometricFragment(),
            )
        }

        btnNext.text = if (step < totalSteps)
            getString(R.string.enrollment_next)
        else
            getString(R.string.enrollment_submit)
    }

    private fun advanceStep() {
        if (currentStep < totalSteps) {
            showStep(currentStep + 1)
        } else {
            submitEnrollment()
        }
    }

    private fun submitEnrollment() {
        val token = getSharedPreferences("indis_prefs", MODE_PRIVATE)
            .getString("jwt_token", "") ?: ""

        enrollmentRepo.submit(
            docImageB64  = docImageB64 ?: "",
            faceImageB64 = faceImageB64 ?: "",
            token        = token,
            onSuccess    = { finish() },
            onError      = { /* TODO: show error snackbar */ },
        )
    }

    /** Called by [DocumentFragment] after the citizen captures their document. */
    fun onDocumentCaptured(imageB64: String) {
        docImageB64 = imageB64
    }

    /** Called by [BiometricFragment] after the citizen captures their face. */
    fun onFaceCaptured(imageB64: String) {
        faceImageB64 = imageB64
    }
}
