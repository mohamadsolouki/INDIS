package org.indis.app.ui.wallet

import android.content.Intent
import android.os.Bundle
import android.view.View
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import android.widget.ProgressBar
import android.widget.LinearLayout
import android.widget.TextView
import com.google.android.material.snackbar.Snackbar
import org.indis.app.R
import org.indis.app.domain.biometric.BiometricAuthHelper

/**
 * Credential wallet screen.
 *
 * Loads stored W3C Verifiable Credentials from the encrypted local database
 * (offline) and syncs from the gateway (online) via [WalletViewModel].
 *
 * Gated behind [BiometricAuthHelper] when biometric hardware is enrolled,
 * preventing shoulder-surfing access to credential data.
 *
 * PRD FR-006: offline credential presentation.
 */
class WalletActivity : AppCompatActivity() {

    private val viewModel: WalletViewModel by viewModels()

    private lateinit var progressBar: ProgressBar
    private lateinit var layoutEmpty: LinearLayout
    private lateinit var recyclerView: RecyclerView
    private lateinit var tvError: TextView

    private val bioHelper by lazy { BiometricAuthHelper(this) }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_wallet)

        progressBar  = findViewById(R.id.progress_wallet)
        layoutEmpty  = findViewById(R.id.layout_empty)
        recyclerView = findViewById(R.id.recycler_credentials)
        tvError      = findViewById<TextView?>(R.id.tv_wallet_error) ?: TextView(this)

        recyclerView.layoutManager = LinearLayoutManager(this)

        observeViewModel()

        // Gate wallet access with biometric unlock.
        if (bioHelper.canAuthenticate()) {
            lockAndAuthenticate()
        } else {
            viewModel.loadCredentials()
        }
    }

    private fun lockAndAuthenticate() {
        // Blur / hide content until auth succeeds.
        recyclerView.alpha = 0f

        bioHelper.authenticate(
            title    = getString(R.string.biometric_title_wallet),
            subtitle = getString(R.string.biometric_subtitle_wallet),
            onSuccess = {
                recyclerView.animate().alpha(1f).setDuration(200).start()
                viewModel.loadCredentials()
            },
            onError = { msg ->
                Snackbar.make(recyclerView, msg, Snackbar.LENGTH_LONG).show()
                finish()
            },
        )
    }

    private fun observeViewModel() {
        viewModel.isLoading.observe(this) { loading ->
            progressBar.visibility = if (loading) View.VISIBLE else View.GONE
        }

        viewModel.credentials.observe(this) { list ->
            if (list.isNullOrEmpty()) {
                layoutEmpty.visibility  = View.VISIBLE
                recyclerView.visibility = View.GONE
            } else {
                layoutEmpty.visibility  = View.GONE
                recyclerView.visibility = View.VISIBLE
                recyclerView.adapter = CredentialCardAdapter(list) { card ->
                    openDetail(card)
                }
            }
        }

        viewModel.error.observe(this) { msg ->
            if (msg != null) {
                Snackbar.make(recyclerView, msg, Snackbar.LENGTH_LONG)
                    .setAction("تلاش مجدد") { viewModel.refresh() }
                    .show()
            }
        }
    }

    private fun openDetail(card: CredentialCard) {
        val intent = Intent(this, CredentialDetailActivity::class.java).apply {
            putExtra(CredentialDetailActivity.EXTRA_CREDENTIAL_ID, card.id)
            putExtra(CredentialDetailActivity.EXTRA_CREDENTIAL_TYPE, card.type)
            putExtra(CredentialDetailActivity.EXTRA_ISSUED_AT, card.issuedAt)
            putExtra(CredentialDetailActivity.EXTRA_EXPIRES_AT, card.expiresAt)
            putExtra(CredentialDetailActivity.EXTRA_IS_REVOKED, card.isRevoked)
        }
        startActivity(intent)
    }
}
