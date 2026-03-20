package org.indis.app.ui.wallet

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.indis.app.data.network.GatewayApiClient
import org.indis.app.data.repository.GatewayCredentialRepository

/**
 * ViewModel for [WalletActivity].
 *
 * Holds credential list state across configuration changes and exposes
 * loading / error states as [LiveData] for the UI to observe.
 *
 * PRD FR-006: credentials are loaded from the offline Room cache first,
 * then refreshed from the gateway when network is available.
 */
class WalletViewModel(app: Application) : AndroidViewModel(app) {

    private val _credentials = MutableLiveData<List<CredentialCard>>()
    val credentials: LiveData<List<CredentialCard>> = _credentials

    private val _isLoading = MutableLiveData(false)
    val isLoading: LiveData<Boolean> = _isLoading

    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error

    private val repo by lazy {
        val prefs = app.getSharedPreferences("indis_prefs", Application.MODE_PRIVATE)
        val url = prefs.getString("gateway_url", "http://10.0.2.2:8080") ?: "http://10.0.2.2:8080"
        GatewayCredentialRepository(context = app.applicationContext, api = GatewayApiClient(url))
    }

    fun loadCredentials() {
        _isLoading.value = true
        _error.value = null
        viewModelScope.launch {
            runCatching {
                withContext(Dispatchers.IO) { repo.listCredentials() }
            }.onSuccess { list ->
                _credentials.value = list
            }.onFailure { err ->
                _error.value = err.message ?: "خطا در بارگذاری اعتبارنامه‌ها"
                // Still show cached data if available
                runCatching {
                    withContext(Dispatchers.IO) { repo.listCredentialsCached() }
                }.onSuccess { cached ->
                    if (!cached.isNullOrEmpty()) _credentials.value = cached
                }
            }
            _isLoading.value = false
        }
    }

    fun refresh() = loadCredentials()
}
