package org.indis.app.ui.enrollment

import android.os.Bundle
import android.view.View
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import androidx.core.view.isVisible
import androidx.fragment.app.commit
import com.google.android.material.snackbar.Snackbar
import org.indis.app.R

/**
 * Multi-step enrollment wizard.
 *
 * Step 0: Pathway selection  — Standard / Enhanced / Social Attestation
 * Step 1: Document capture   — [DocumentFragment]
 * Step 2: Biometric capture  — [BiometricFragment]
 * Step 3: Submitting…
 * Step 4: Success / Error
 *
 * PRD FR-003: Standard, Enhanced, and Social Attestation pathways.
 * PRD UC-001 / UC-002 / UC-003.
 */
class EnrollmentActivity : AppCompatActivity() {

    private val viewModel: EnrollmentViewModel by viewModels()

    private lateinit var tvStepIndicator: TextView
    private lateinit var btnNext: Button
    private lateinit var layoutPathway: LinearLayout

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_enrollment)

        tvStepIndicator = findViewById(R.id.tv_step_indicator)
        btnNext         = findViewById(R.id.btn_next)
        layoutPathway   = findViewById(R.id.layout_pathway_selector)

        setupPathwayButtons()
        observeViewModel()

        btnNext.setOnClickListener { onNextClicked() }
    }

    private fun setupPathwayButtons() {
        listOf(
            R.id.btn_pathway_standard  to EnrollmentViewModel.Pathway.STANDARD,
            R.id.btn_pathway_enhanced  to EnrollmentViewModel.Pathway.ENHANCED,
            R.id.btn_pathway_social    to EnrollmentViewModel.Pathway.SOCIAL,
        ).forEach { (id, pathway) ->
            findViewById<Button?>(id)?.setOnClickListener {
                viewModel.selectPathway(pathway)
            }
        }
    }

    private fun observeViewModel() {
        viewModel.step.observe(this) { step ->
            updateUiForStep(step)
        }

        viewModel.submitting.observe(this) { submitting ->
            btnNext.isEnabled = !submitting
            if (submitting) tvStepIndicator.text = getString(R.string.enrollment_submitting)
        }

        viewModel.enrollmentId.observe(this) { enrollmentId ->
            if (enrollmentId != null) {
                Snackbar.make(
                    btnNext,
                    "${getString(R.string.enrollment_success)}: $enrollmentId",
                    Snackbar.LENGTH_LONG,
                ).addCallback(object : Snackbar.Callback() {
                    override fun onDismissed(snackbar: Snackbar?, event: Int) = finish()
                }).show()
            }
        }

        viewModel.error.observe(this) { msg ->
            if (msg != null) {
                Snackbar.make(btnNext, msg, Snackbar.LENGTH_LONG)
                    .setAction(getString(R.string.retry)) { viewModel.retryFromDocument() }
                    .show()
            }
        }
    }

    private fun updateUiForStep(step: EnrollmentViewModel.Step) {
        val (stepNum, totalSteps, btnLabel, showPathway, fragment) = when (step) {
            EnrollmentViewModel.Step.PATHWAY  -> StepConfig(0, 3, null, showPathway = true, null)
            EnrollmentViewModel.Step.DOCUMENT -> StepConfig(1, 3, getString(R.string.enrollment_next), showPathway = false, DocumentFragment())
            EnrollmentViewModel.Step.BIOMETRIC-> StepConfig(2, 3, getString(R.string.enrollment_next), showPathway = false, BiometricFragment())
            EnrollmentViewModel.Step.SUBMIT   -> StepConfig(3, 3, getString(R.string.enrollment_submit), showPathway = false, null)
            EnrollmentViewModel.Step.SUCCESS  -> StepConfig(3, 3, null, showPathway = false, null)
            EnrollmentViewModel.Step.ERROR    -> StepConfig(3, 3, getString(R.string.retry), showPathway = false, null)
        }

        tvStepIndicator.text = if (stepNum > 0) "$stepNum / $totalSteps" else getString(R.string.enrollment_select_pathway)
        layoutPathway.isVisible = showPathway
        btnNext.isVisible = btnLabel != null
        if (btnLabel != null) btnNext.text = btnLabel

        if (fragment != null) {
            supportFragmentManager.commit { replace(R.id.fragment_container, fragment) }
        }
    }

    private fun onNextClicked() {
        when (viewModel.step.value) {
            EnrollmentViewModel.Step.SUBMIT -> viewModel.submitEnrollment()
            EnrollmentViewModel.Step.ERROR  -> viewModel.retryFromDocument()
            else -> {}
        }
    }

    /** Called by [DocumentFragment] after citizen captures their document. */
    fun onDocumentCaptured(imageB64: String) = viewModel.onDocumentCaptured(imageB64)

    /** Called by [BiometricFragment] after citizen captures their face. */
    fun onFaceCaptured(imageB64: String) = viewModel.onBiometricCaptured(imageB64)

    private data class StepConfig(
        val stepNum: Int,
        val totalSteps: Int,
        val btnLabel: String?,
        val showPathway: Boolean,
        val fragment: androidx.fragment.app.Fragment?,
    )
}
