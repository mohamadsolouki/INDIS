import Foundation
import Combine

/// Central observable state shared across the entire app via @EnvironmentObject.
///
/// Holds auth status, the citizen DID, JWT token, and the gateway base URL.
/// All persistent values are backed by EncryptedWalletStore (Keychain).
final class AppState: ObservableObject {

    @Published var isAuthenticated: Bool = false
    @Published var did: String = ""
    @Published var jwtToken: String = ""
    @Published var gatewayURL: String = "http://localhost:8080"
    @Published var usePersianNumerals: Bool = true
    @Published var selectedLocale: String = "fa"

    private let store = EncryptedWalletStore.shared

    init() {
        loadFromStore()
        RevocationCacheService.shared.scheduleIfNeeded()
    }

    // MARK: — Auth

    func login(did: String, token: String) {
        self.did = did
        self.jwtToken = token
        store.set(did, forKey: .did)
        store.set(token, forKey: .jwtToken)
        isAuthenticated = true
    }

    func logout() {
        store.delete(key: .did)
        store.delete(key: .jwtToken)
        did = ""
        jwtToken = ""
        isAuthenticated = false
    }

    func saveSettings(gatewayURL: String, locale: String, persianNumerals: Bool) {
        self.gatewayURL = gatewayURL
        self.selectedLocale = locale
        self.usePersianNumerals = persianNumerals
        store.set(gatewayURL, forKey: .gatewayURL)
        store.set(locale, forKey: .locale)
        UserDefaults.standard.set(persianNumerals, forKey: "persian_numerals")
    }

    // MARK: — Private

    private func loadFromStore() {
        did = store.get(forKey: .did) ?? ""
        jwtToken = store.get(forKey: .jwtToken) ?? ""
        isAuthenticated = !did.isEmpty && !jwtToken.isEmpty
        gatewayURL = store.get(forKey: .gatewayURL) ?? "http://localhost:8080"
        selectedLocale = store.get(forKey: .locale) ?? "fa"
        usePersianNumerals = UserDefaults.standard.object(forKey: "persian_numerals") as? Bool ?? true
    }
}
