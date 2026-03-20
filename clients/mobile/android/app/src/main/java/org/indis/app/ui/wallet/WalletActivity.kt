package org.indis.app.ui.wallet

import android.os.Bundle
import android.view.View
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import android.widget.ProgressBar
import android.widget.LinearLayout
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.indis.app.R
import org.indis.app.data.local.EncryptedWalletDatabase
import org.indis.app.data.repository.CredentialRepository

/**
 * Credential wallet screen.
 *
 * Loads stored W3C Verifiable Credentials from the encrypted local database
 * and displays them in a RecyclerView.  Syncs from the gateway when online
 * via [CredentialRepository].
 *
 * PRD FR-006: offline credential presentation.
 */
class WalletActivity : AppCompatActivity() {

    private lateinit var progressBar: ProgressBar
    private lateinit var layoutEmpty: LinearLayout
    private lateinit var recyclerView: RecyclerView

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_wallet)

        progressBar   = findViewById(R.id.progress_wallet)
        layoutEmpty   = findViewById(R.id.layout_empty)
        recyclerView  = findViewById(R.id.recycler_credentials)

        recyclerView.layoutManager = LinearLayoutManager(this)

        loadCredentials()
    }

    private fun loadCredentials() {
        lifecycleScope.launch {
            val credentials = withContext(Dispatchers.IO) {
                EncryptedWalletDatabase.getInstance(this@WalletActivity)
                    .credentialDao()
                    .getAllCredentials()
            }

            progressBar.visibility = View.GONE

            if (credentials.isEmpty()) {
                layoutEmpty.visibility = View.VISIBLE
            } else {
                recyclerView.visibility = View.VISIBLE
                recyclerView.adapter = CredentialAdapter(credentials)
            }
        }
    }
}
