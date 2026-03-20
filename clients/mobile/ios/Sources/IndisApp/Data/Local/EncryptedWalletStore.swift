import Foundation
import Security

/// Keys used in the Keychain and UserDefaults.
enum StoreKey: String {
    case did          = "indis.did"
    case jwtToken     = "indis.jwt_token"
    case gatewayURL   = "indis.gateway_url"
    case locale       = "indis.locale"
}

/// Persistent, encrypted storage for INDIS credentials.
///
/// Sensitive values (DID, JWT token) are stored in the iOS Keychain using
/// `kSecClassGenericPassword` with `kSecAttrAccessibleAfterFirstUnlock`.
/// Non-sensitive settings (gateway URL, locale) use Keychain too for consistency.
///
/// Mirrors Android's SharedPreferences + EncryptedSharedPreferences usage.
final class EncryptedWalletStore {

    static let shared = EncryptedWalletStore()
    private let service = "org.indis.app"

    private init() {}

    // MARK: — Public API

    func set(_ value: String, forKey key: StoreKey) {
        let data = Data(value.utf8)
        let query: [CFString: Any] = [
            kSecClass:           kSecClassGenericPassword,
            kSecAttrService:     service,
            kSecAttrAccount:     key.rawValue,
            kSecValueData:       data,
            kSecAttrAccessible:  kSecAttrAccessibleAfterFirstUnlock,
        ]
        SecItemDelete(query as CFDictionary)
        SecItemAdd(query as CFDictionary, nil)
    }

    func get(forKey key: StoreKey) -> String? {
        let query: [CFString: Any] = [
            kSecClass:          kSecClassGenericPassword,
            kSecAttrService:    service,
            kSecAttrAccount:    key.rawValue,
            kSecReturnData:     true,
            kSecMatchLimit:     kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    func delete(key: StoreKey) {
        let query: [CFString: Any] = [
            kSecClass:       kSecClassGenericPassword,
            kSecAttrService: service,
            kSecAttrAccount: key.rawValue,
        ]
        SecItemDelete(query as CFDictionary)
    }
}
