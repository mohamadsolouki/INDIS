package org.indis.app.ui.wallet

import android.graphics.Color
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView

/**
 * RecyclerView adapter for the credential wallet list.
 *
 * Displays a [CredentialCard] per row with credential type, expiry, and
 * revocation status. Calls [onItemClick] when a row is tapped so
 * [WalletActivity] can open [CredentialDetailActivity].
 */
class CredentialCardAdapter(
    private val items: List<CredentialCard>,
    private val onItemClick: (CredentialCard) -> Unit = {},
) : RecyclerView.Adapter<CredentialCardAdapter.ViewHolder>() {

    class ViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        val tvTitle: TextView  = view.findViewById(android.R.id.text1)
        val tvDetail: TextView = view.findViewById(android.R.id.text2)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(android.R.layout.simple_list_item_2, parent, false)
        return ViewHolder(view)
    }

    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val card = items[position]
        holder.tvTitle.text = card.title.ifBlank { card.type }
        if (card.isRevoked) {
            holder.tvDetail.text = "ابطال‌شده"
            holder.tvDetail.setTextColor(Color.parseColor("#C23030"))
        } else {
            holder.tvDetail.text = "انقضا: ${card.expiresAt}"
            holder.tvDetail.setTextColor(Color.parseColor("#555555"))
        }
        holder.itemView.setOnClickListener { onItemClick(card) }
    }

    override fun getItemCount(): Int = items.size
}
