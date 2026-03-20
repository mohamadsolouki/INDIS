package org.indis.app.ui.verify

import android.os.Bundle
import android.view.View
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.Spinner
import android.widget.TextView
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import com.google.android.material.snackbar.Snackbar
import org.indis.app.R

/**
 * Verification screen — generates a ZK proof QR code for presentation
 * to a verifier terminal.
 *
 * The citizen selects a predicate (e.g. "age ≥ 18"), taps "Generate", and
 * [VerifyViewModel] produces a proof via [ZKProofManager].
 * The QR encodes ONLY a boolean result — never raw identity data (PRD FR-013).
 *
 * PRD FR-007 (ZK proof generation), FR-008 (selective disclosure).
 */
class VerifyActivity : AppCompatActivity() {

    private val viewModel: VerifyViewModel by viewModels()

    private lateinit var spinnerPredicate: Spinner
    private lateinit var btnGenerateProof: Button
    private lateinit var frameQr: FrameLayout
    private lateinit var imgQrCode: ImageView
    private lateinit var tvQrHint: TextView
    private lateinit var tvStatus: TextView

    private val predicates = listOf(
        "age_gte_18"       to "سن ≥ ۱۸ سال",
        "citizen"          to "تابعیت ایران",
        "voter_eligible"   to "واجد شرایط رأی‌گیری",
        "credential_valid" to "اعتبارنامه معتبر است",
    )

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_verify)

        spinnerPredicate = findViewById(R.id.spinner_predicate)
        btnGenerateProof = findViewById(R.id.btn_generate_proof)
        frameQr          = findViewById(R.id.frame_qr)
        imgQrCode        = findViewById(R.id.img_qr_code)
        tvQrHint         = findViewById(R.id.tv_qr_hint)
        tvStatus         = findViewById(R.id.tv_status)

        spinnerPredicate.adapter = ArrayAdapter(
            this,
            android.R.layout.simple_spinner_item,
            predicates.map { it.second },
        ).also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }

        btnGenerateProof.setOnClickListener { generateProof() }

        observeViewModel()
    }

    private fun generateProof() {
        val idx       = spinnerPredicate.selectedItemPosition
        val predicate = predicates[idx].first
        val did       = getSharedPreferences("indis_prefs", MODE_PRIVATE)
            .getString("did", null)

        viewModel.generateProof(predicate, did)
    }

    private fun observeViewModel() {
        viewModel.isGenerating.observe(this) { generating ->
            btnGenerateProof.isEnabled = !generating
            tvStatus.text = if (generating) getString(R.string.verify_generating) else ""
        }

        viewModel.qrBitmap.observe(this) { bitmap ->
            if (bitmap != null) {
                imgQrCode.setImageBitmap(bitmap)
                frameQr.visibility  = View.VISIBLE
                tvQrHint.visibility = View.VISIBLE
            } else {
                frameQr.visibility  = View.GONE
                tvQrHint.visibility = View.GONE
            }
        }

        viewModel.error.observe(this) { msg ->
            if (msg != null) {
                tvStatus.text = ""
                Snackbar.make(btnGenerateProof, msg, Snackbar.LENGTH_LONG).show()
            }
        }
    }
}
