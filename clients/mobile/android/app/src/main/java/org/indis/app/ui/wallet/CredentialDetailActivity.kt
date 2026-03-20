package org.indis.app.ui.wallet

import android.graphics.Bitmap
import android.os.Bundle
import android.view.View
import android.widget.Button
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import com.google.android.material.snackbar.Snackbar
import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.qrcode.QRCodeWriter
import org.indis.app.R
import org.json.JSONObject

/**
 * Detail screen for a single verifiable credential.
 *
 * Shows credential type, issued/expires dates, revocation status, and
 * provides an "offline QR" button that encodes a minimal ZK presentation
 * for the verifier terminal to scan (PRD FR-013: boolean result only).
 */
class CredentialDetailActivity : AppCompatActivity() {

    companion object {
        const val EXTRA_CREDENTIAL_ID   = "credential_id"
        const val EXTRA_CREDENTIAL_TYPE = "credential_type"
        const val EXTRA_ISSUED_AT       = "issued_at"
        const val EXTRA_EXPIRES_AT      = "expires_at"
        const val EXTRA_IS_REVOKED      = "is_revoked"
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val credId   = intent.getStringExtra(EXTRA_CREDENTIAL_ID)   ?: ""
        val credType = intent.getStringExtra(EXTRA_CREDENTIAL_TYPE) ?: ""
        val issuedAt = intent.getStringExtra(EXTRA_ISSUED_AT)       ?: ""
        val expiresAt= intent.getStringExtra(EXTRA_EXPIRES_AT)      ?: ""
        val isRevoked= intent.getBooleanExtra(EXTRA_IS_REVOKED, false)

        // Build layout programmatically (consistent with existing UI approach).
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(48, 64, 48, 48)
        }
        setContentView(root)

        root.addView(textView(credType, 22f, true))
        root.addView(dividerView())
        root.addView(labelValueRow(getString(R.string.credential_id_label), credId))
        root.addView(labelValueRow(getString(R.string.issued_at_label), issuedAt))
        root.addView(labelValueRow(getString(R.string.expires_at_label), expiresAt))

        val tvStatus = textView(
            if (isRevoked) "● ابطال‌شده" else "● معتبر",
            14f, bold = true
        ).apply {
            setTextColor(if (isRevoked) 0xFFC23030.toInt() else 0xFF0F9960.toInt())
        }
        root.addView(tvStatus)
        root.addView(dividerView())

        if (!isRevoked) {
            val imgQr = ImageView(this).apply {
                visibility = View.GONE
                setPadding(0, 24, 0, 0)
            }
            root.addView(imgQr)

            val btnQr = Button(this).apply {
                text = getString(R.string.show_offline_qr)
                setBackgroundColor(0xFF1A56DB.toInt())
                setTextColor(0xFFFFFFFF.toInt())
            }
            btnQr.setOnClickListener {
                try {
                    val payload = buildQrPayload(credId, credType)
                    val bmp = encodeQR(payload, 400)
                    imgQr.setImageBitmap(bmp)
                    imgQr.visibility = View.VISIBLE
                    btnQr.text = getString(R.string.hide_qr)
                    if (imgQr.visibility == View.VISIBLE && imgQr.drawable != null) {
                        btnQr.setOnClickListener {
                            imgQr.visibility = View.GONE
                            btnQr.text = getString(R.string.show_offline_qr)
                        }
                    }
                } catch (e: Exception) {
                    Snackbar.make(root, e.message ?: "خطا", Snackbar.LENGTH_SHORT).show()
                }
            }
            root.addView(btnQr)
        }
    }

    /** Encodes a boolean-only ZK presentation QR payload (PRD FR-013). */
    private fun buildQrPayload(credId: String, credType: String): String {
        val did = getSharedPreferences("indis_prefs", MODE_PRIVATE)
            .getString("did", "") ?: ""
        return JSONObject().apply {
            put("type",    "INDIS_CREDENTIAL_PRESENTATION")
            put("cred_id", credId.takeLast(16)) // partial ID only — not full PII
            put("cred_type", credType)
            put("did_hint", did.takeLast(8))
            put("valid",   true)               // boolean result — PRD FR-013
            put("ts",      System.currentTimeMillis())
        }.toString()
    }

    private fun encodeQR(content: String, sizePx: Int): Bitmap {
        val writer = QRCodeWriter()
        val hints  = mapOf(EncodeHintType.CHARACTER_SET to "UTF-8")
        val matrix = writer.encode(content, BarcodeFormat.QR_CODE, sizePx, sizePx, hints)
        val bitmap = Bitmap.createBitmap(sizePx, sizePx, Bitmap.Config.RGB_565)
        for (x in 0 until sizePx) {
            for (y in 0 until sizePx) {
                bitmap.setPixel(x, y, if (matrix[x, y]) 0xFF000000.toInt() else 0xFFFFFFFF.toInt())
            }
        }
        return bitmap
    }

    private fun textView(text: String, sp: Float, bold: Boolean = false) =
        TextView(this).apply {
            this.text = text
            textSize = sp
            if (bold) setTypeface(typeface, android.graphics.Typeface.BOLD)
            setPadding(0, 8, 0, 8)
        }

    private fun labelValueRow(label: String, value: String): LinearLayout {
        return LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            setPadding(0, 6, 0, 6)
            addView(textView("$label: ", 13f, bold = true).apply {
                setTextColor(0xFF64748B.toInt())
            })
            addView(textView(value, 13f).apply {
                typeface = android.graphics.Typeface.MONOSPACE
            })
        }
    }

    private fun dividerView() = View(this).apply {
        layoutParams = LinearLayout.LayoutParams(
            LinearLayout.LayoutParams.MATCH_PARENT, 1
        ).apply { setMargins(0, 12, 0, 12) }
        setBackgroundColor(0xFFE2E8F0.toInt())
    }
}
