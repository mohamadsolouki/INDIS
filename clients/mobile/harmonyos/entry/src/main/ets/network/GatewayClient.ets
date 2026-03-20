import http from '@ohos.net.http'
import hilog from '@ohos.hilog'
import { AppStorage } from '../model/AppStorage'

/**
 * HTTP client for all INDIS gateway API calls.
 *
 * Uses HarmonyOS @ohos.net.http with 30s timeout.
 * Auth header is injected automatically when a token is available.
 * Mirrors Android GatewayApiClient (OkHttp) and iOS GatewayAPIClient (URLSession).
 */
export class GatewayClient {

  private readonly baseURL: string

  constructor(baseURL?: string) {
    this.baseURL = (baseURL ?? AppStorage.getGatewayURL()).replace(/\/$/, '')
  }

  // ── Public API ────────────────────────────────────────────────────────────

  async get<T>(path: string, token?: string): Promise<T> {
    return this.request<T>('GET', path, undefined, token)
  }

  async post<T>(path: string, body: object, token?: string): Promise<T> {
    return this.request<T>('POST', path, body, token)
  }

  async put<T>(path: string, body: object, token?: string): Promise<T> {
    return this.request<T>('PUT', path, body, token)
  }

  // ── Private ───────────────────────────────────────────────────────────────

  private async request<T>(method: string, path: string, body?: object, token?: string): Promise<T> {
    const url = `${this.baseURL}/${path.replace(/^\//, '')}`
    const authToken = token ?? AppStorage.getToken()

    const httpRequest = http.createHttp()
    try {
      const response = await httpRequest.request(url, {
        method: method as http.RequestMethod,
        header: {
          'Content-Type': 'application/json',
          ...(authToken ? { 'Authorization': `Bearer ${authToken}` } : {}),
        },
        extraData: body ? JSON.stringify(body) : undefined,
        connectTimeout: 10000,
        readTimeout: 30000,
      })

      if (response.responseCode < 200 || response.responseCode >= 300) {
        throw new Error(`HTTP ${response.responseCode}: ${response.result}`)
      }
      return JSON.parse(response.result as string) as T
    } finally {
      httpRequest.destroy()
    }
  }
}
