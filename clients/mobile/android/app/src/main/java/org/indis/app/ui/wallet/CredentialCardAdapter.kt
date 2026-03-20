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
 * Displays a [CredentialCard] per row — credential type, expiry, and revocation
 * status badge. Uses the stock two-line list item for simplicity; a custom card
 * layout can replace this in the UI-polish phase.
 */
class CredentialCardAdapter(
    private val items: List<CredentialCard>,
) : RecyclerView.Adapter<CredentialCardAdapter.ViewHolder>() {

    class ViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        val tvTitle: TextView   = view.findViewById(android.R.id.text1)
        val tvDetail: TextView  = view.findViewById(android.R.id.text2)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(android.R.layout.simple_list_item_2, parent, false)
        return ViewHolder(view)
    }

    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val card = items[position]
        holder.tvTitle.text = card.title
        if (card.revoked) {
            holder.tvDetail.text = "ابطال‌شده"
            holder.tvDetail.setTextColor(Color.parseColor("#C23030"))
        } else {
            holder.tvDetail.text = "انقضا: ${card.expiresAt}"
            holder.tvDetail.setTextColor(Color.parseColor("#555555"))
        }
    }

    override fun getItemCount(): Int = items.size
}
