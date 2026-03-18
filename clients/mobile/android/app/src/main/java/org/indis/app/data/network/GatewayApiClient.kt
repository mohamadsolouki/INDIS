package org.indis.app.data.network

class GatewayApiClient(private val baseUrl: String) {
    fun endpoint(path: String): String = "${baseUrl.trimEnd('/')}/${path.trimStart('/')}"
}
