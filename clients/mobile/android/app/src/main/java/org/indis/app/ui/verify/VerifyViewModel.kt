package org.indis.app.ui.verify

import android.app.Application
import android.graphics.Bitmap
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.viewModelScope
import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.qrcode.QRCodeWriter
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.indis.app.domain.zk.ZKProofManager
import org.json.JSONObject
import java.util.UUID

/**
 * ViewModel for [VerifyActivity].
 *
 * Holds the generated QR bitmap across configuration changes.
 * Runs ZK proof generation on [Dispatchers.Default] to keep the UI thread free.
 *
 * PRD FR-007: ZK proof generation.
 * PRD FR-013: Verifier receives ONLY a boolean result — raw identity attributes
 * are never embedded in the QR payload.
 */
class VerifyViewModel(app: Application) : AndroidViewModel(app) {

    private val _qrBitmap = MutableLiveData<Bitmap?>()
    val qrBitmap: LiveData<Bitmap?> = _qrBitmap

    private val _isGenerating = MutableLiveData(false)
    val isGenerating: LiveData<Boolean> = _isGenerating

    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error

    private val zkManager = ZKProofManager()

    /** Returns true if a QR has been generated and is ready for display. */
    val hasQr: Boolean get() = _qrBitmap.value != null

    fun generateProof(predicate: String, did: String?) {
        _isGenerating.value = true
        _error.value = null
        _qrBitmap.value = null

        viewModelScope.launch {
            runCatching {
                withContext(Dispatchers.Default) {
                    // Stub: ZKProofManager.verifyProof returns true (boolean placeholder).
                    // TODO: replace with real Groth16/Bulletproofs proof bytes via JNI bridge.
                    val proofValid = zkManager.verifyProof(predicate.toByteArray())
                    val nonce     = UUID.randomUUID().toString().replace("-", "")

                    // PRD FR-013: encode ONLY boolean public signals — never raw attributes.
                    val payload = JSONObject().apply {
                        put("type",      "INDIS_ZK_PROOF")
                        put("predicate", predicate)
                        put("valid",     proofValid)   // boolean only
                        put("nonce",     nonce)
                        put("ts",        System.currentTimeMillis())
                        if (did != null) put("did_hint", did.takeLast(8)) // last 8 chars only
                    }.toString()

                    encodeQR(payload, 512)
                }
            }.onSuccess { bitmap ->
                _qrBitmap.value = bitmap
            }.onFailure { err ->
                _error.value = err.message ?: "خطا در تولید اثبات"
            }
            _isGenerating.value = false
        }
    }

    fun clearQr() {
        _qrBitmap.value = null
        _error.value = null
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
}
