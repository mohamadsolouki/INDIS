import Foundation

/// Protocol for identity registration operations.
protocol IdentityRepository {
    /// Registers a national ID with the gateway, derives a DID, and stores credentials.
    /// Returns the registered DID on success.
    func enrollNationalId(_ nationalId: String) async throws -> String
}

/// Concrete implementation backed by GatewayAPIClient + Secure Enclave DID.
///
/// Flow:
///  1. DIDManager.generateDID() — Secure Enclave P-256 key → deterministic DID suffix.
///  2. POST /v1/identity/register — anchors DID on the backend.
///  3. Persist DID + JWT to EncryptedWalletStore (Keychain).
///
/// PRD FR-001: device-bound P-256 Secure Enclave key generates a deterministic DID suffix.
final class GatewayIdentityRepository: IdentityRepository {

    private let api: GatewayAPIClient
    private let didManager: DIDManager
    private let store: EncryptedWalletStore

    init(api: GatewayAPIClient, didManager: DIDManager = DIDManager(), store: EncryptedWalletStore = .shared) {
        self.api = api
        self.didManager = didManager
        self.store = store
    }

    func enrollNationalId(_ nationalId: String) async throws -> String {
        // 1. Reuse existing DID or generate a fresh one.
        let existingDID = store.get(forKey: .did)
        let did = try existingDID ?? didManager.generateDID()

        // 2. Register with the gateway.
        let request = RegisterIdentityRequest(national_id: nationalId, did: did)
        let response: RegisterIdentityResponse = try await api.post(
            "/v1/identity/register",
            body: request,
            token: ""
        )

        // 3. Persist.
        store.set(response.did, forKey: .did)
        store.set(response.token, forKey: .jwtToken)

        return response.did
    }
}
