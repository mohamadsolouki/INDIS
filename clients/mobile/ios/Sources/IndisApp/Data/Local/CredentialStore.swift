import Foundation

/// Local model for a cached Verifiable Credential.
struct CredentialRecord: Codable, Identifiable {
    let id: String
    let credentialType: String
    let issuer: String
    let issuedAt: String
    let expiresAt: String
    let vcJson: String
    let status: String      // "active" | "revoked" | "expired"

    var isRevoked: Bool { status == "revoked" }
    var isExpired: Bool {
        guard let date = ISO8601DateFormatter().date(from: expiresAt) else { return false }
        return date < Date()
    }
}

/// File-backed credential cache using JSON serialisation.
///
/// On iOS, AES-256 Data Protection (`NSFileProtectionCompleteUntilFirstUserAuthentication`)
/// is enforced by the OS on all app data directory files, providing equivalent
/// protection to Android's EncryptedSharedPreferences / Room encryption.
///
/// PRD FR-006: credentials must be available offline for 72 hours.
final class CredentialStore {

    static let shared = CredentialStore()

    private let fileURL: URL = {
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        return docs.appendingPathComponent("indis_credentials.json")
    }()

    private init() {}

    // MARK: — Public API

    func saveAll(_ records: [CredentialRecord]) {
        guard let data = try? JSONEncoder().encode(records) else { return }
        try? data.write(to: fileURL, options: [.completeFileProtection, .atomic])
    }

    func loadAll() -> [CredentialRecord] {
        guard let data = try? Data(contentsOf: fileURL),
              let records = try? JSONDecoder().decode([CredentialRecord].self, from: data)
        else { return [] }
        return records
    }

    func upsert(_ record: CredentialRecord) {
        var current = loadAll()
        current.removeAll { $0.id == record.id }
        current.insert(record, at: 0)
        saveAll(current)
    }

    func clear() {
        try? FileManager.default.removeItem(at: fileURL)
    }
}
