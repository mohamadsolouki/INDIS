package org.indis.app.ui.wallet

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import org.indis.app.data.local.CredentialEntity

/**
 * RecyclerView adapter for the credential wallet list.
 *
 * Each card displays the credential type, issuer, and expiry date.
 * Uses a simple card layout defined in list_item_credential.xml.
 */
class CredentialAdapter(
    private val items: List<CredentialEntity>,
) : RecyclerView.Adapter<CredentialAdapter.ViewHolder>() {

    class ViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        val tvType: TextView     = view.findViewById(android.R.id.text1)
        val tvIssuer: TextView   = view.findViewById(android.R.id.text2)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(android.R.layout.simple_list_item_2, parent, false)
        return ViewHolder(view)
    }

    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val cred = items[position]
        holder.tvType.text   = cred.credentialType
        holder.tvIssuer.text = cred.issuer
    }

    override fun getItemCount(): Int = items.size
}
