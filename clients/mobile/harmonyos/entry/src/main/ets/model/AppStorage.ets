import preferences from '@ohos.data.preferences'
import hilog from '@ohos.hilog'

const PREF_NAME = 'indis_prefs'

/**
 * Persistent application state backed by HarmonyOS Preferences.
 *
 * Provides typed accessors for DID, JWT, gateway URL, locale, and display settings.
 * Mirrors Android SharedPreferences / iOS EncryptedWalletStore usage patterns.
 *
 * PRD FR-001: stores device-bound DID after Secure Element key derivation.
 */
export class AppStorage {

  private static prefs: preferences.Preferences | null = null

  static load(context: object): void {
    try {
      AppStorage.prefs = preferences.getPreferencesSync(context, { name: PREF_NAME })
    } catch (e) {
      hilog.error(0x0000, 'INDIS', 'AppStorage.load failed: %{public}s', String(e))
    }
  }

  // ── Auth ──────────────────────────────────────────────────────────────────

  static getDID(): string {
    return AppStorage.prefs?.getSync('did', '') as string ?? ''
  }

  static getToken(): string {
    return AppStorage.prefs?.getSync('jwt_token', '') as string ?? ''
  }

  static isAuthenticated(): boolean {
    return AppStorage.getDID() !== '' && AppStorage.getToken() !== ''
  }

  static login(did: string, token: string): void {
    AppStorage.prefs?.putSync('did', did)
    AppStorage.prefs?.putSync('jwt_token', token)
    AppStorage.prefs?.flush()
  }

  static logout(): void {
    AppStorage.prefs?.deleteSync('did')
    AppStorage.prefs?.deleteSync('jwt_token')
    AppStorage.prefs?.flush()
  }

  // ── Settings ──────────────────────────────────────────────────────────────

  static getGatewayURL(): string {
    return AppStorage.prefs?.getSync('gateway_url', 'http://10.0.2.2:8080') as string
  }

  static setGatewayURL(url: string): void {
    AppStorage.prefs?.putSync('gateway_url', url)
    AppStorage.prefs?.flush()
  }

  static getLocale(): string {
    return AppStorage.prefs?.getSync('locale', 'fa') as string
  }

  static setLocale(locale: string): void {
    AppStorage.prefs?.putSync('locale', locale)
    AppStorage.prefs?.flush()
  }

  static usePersianNumerals(): boolean {
    return AppStorage.prefs?.getSync('persian_numerals', true) as boolean
  }

  static setPersianNumerals(value: boolean): void {
    AppStorage.prefs?.putSync('persian_numerals', value)
    AppStorage.prefs?.flush()
  }

  // ── Revocation cache ──────────────────────────────────────────────────────

  static getRevocationList(): string | null {
    const json = AppStorage.prefs?.getSync('revocation_list_json', '') as string
    const fetchedAt = AppStorage.prefs?.getSync('revocation_list_fetched_at', 0) as number
    const maxAge = 72 * 60 * 60 * 1000
    if (!json || Date.now() - fetchedAt > maxAge) return null
    return json
  }

  static saveRevocationList(json: string): void {
    AppStorage.prefs?.putSync('revocation_list_json', json)
    AppStorage.prefs?.putSync('revocation_list_fetched_at', Date.now())
    AppStorage.prefs?.flush()
  }
}
