package org.indis.app.ui.enrollment

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.indis.app.data.network.GatewayApiClient
import org.indis.app.data.repository.EnrollmentRepository

/**
 * ViewModel for [EnrollmentActivity].
 *
 * Manages the multi-step enrollment wizard state: pathway selection,
 * document/biometric capture, and submission result.
 *
 * PRD FR-003: Standard, Enhanced, and Social Attestation pathways.
 */
class EnrollmentViewModel(app: Application) : AndroidViewModel(app) {

    enum class Pathway { STANDARD, ENHANCED, SOCIAL }
    enum class Step { PATHWAY, DOCUMENT, BIOMETRIC, SUBMIT, SUCCESS, ERROR }

    private val _step = MutableLiveData(Step.PATHWAY)
    val step: LiveData<Step> = _step

    private val _pathway = MutableLiveData(Pathway.STANDARD)
    val pathway: LiveData<Pathway> = _pathway

    private val _documentImageBase64 = MutableLiveData<String?>()
    private val _biometricImageBase64 = MutableLiveData<String?>()

    private val _submitting = MutableLiveData(false)
    val submitting: LiveData<Boolean> = _submitting

    private val _enrollmentId = MutableLiveData<String?>()
    val enrollmentId: LiveData<String?> = _enrollmentId

    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error

    private val repo by lazy {
        val prefs = app.getSharedPreferences("indis_prefs", Application.MODE_PRIVATE)
        val url = prefs.getString("gateway_url", "http://10.0.2.2:8080") ?: "http://10.0.2.2:8080"
        EnrollmentRepository(api = GatewayApiClient(url))
    }

    fun selectPathway(p: Pathway) {
        _pathway.value = p
        _step.value = Step.DOCUMENT
    }

    fun onDocumentCaptured(base64: String) {
        _documentImageBase64.value = base64
        _step.value = Step.BIOMETRIC
    }

    fun onBiometricCaptured(base64: String) {
        _biometricImageBase64.value = base64
        _step.value = Step.SUBMIT
    }

    fun submitEnrollment() {
        val docB64  = _documentImageBase64.value ?: return
        val faceB64 = _biometricImageBase64.value ?: return
        val pathwayStr = when (_pathway.value) {
            Pathway.ENHANCED -> "enhanced"
            Pathway.SOCIAL   -> "social"
            else             -> "standard"
        }

        _submitting.value = true
        _error.value = null

        viewModelScope.launch {
            runCatching {
                withContext(Dispatchers.IO) {
                    repo.submitEnrollment(
                        documentBase64 = docB64,
                        faceBase64 = faceB64,
                        pathway = pathwayStr,
                    )
                }
            }.onSuccess { enrollmentId ->
                _enrollmentId.value = enrollmentId
                _step.value = Step.SUCCESS
            }.onFailure { err ->
                _error.value = err.message ?: "خطا در ارسال ثبت‌نام"
                _step.value = Step.ERROR
            }
            _submitting.value = false
        }
    }

    fun retryFromDocument() {
        _error.value = null
        _step.value = Step.DOCUMENT
    }
}
