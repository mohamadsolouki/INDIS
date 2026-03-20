import Foundation

/// Protocol for credential wallet operations.
protocol CredentialRepository {
    func listCredentials() async -> [CredentialRecord]
}

/// Concrete implementation backed by GatewayAPIClient + CredentialStore (local JSON cache).
///
/// Network strategy:
///   1. Try GET /v1/identity/{did}/credentials from the gateway.
///   2. On success: upsert every credential into CredentialStore and return the list.
///   3. On network failure: return cached records (offline fallback).
///
/// PRD FR-006: credentials must be available offline for up to 72 hours.
final class GatewayCredentialRepository: CredentialRepository {

    private let api: GatewayAPIClient
    private let localStore: CredentialStore
    private let walletStore: EncryptedWalletStore

    init(api: GatewayAPIClient,
         localStore: CredentialStore = .shared,
         walletStore: EncryptedWalletStore = .shared) {
        self.api = api
        self.localStore = localStore
        self.walletStore = walletStore
    }

    func listCredentials() async -> [CredentialRecord] {
        let did   = walletStore.get(forKey: .did) ?? ""
        let token = walletStore.get(forKey: .jwtToken) ?? ""

        if !did.isEmpty, !token.isEmpty {
            do {
                let remote: CredentialListResponse = try await api.get(
                    "/v1/identity/\(did)/credentials",
                    token: token
                )
                let records = remote.credentials.map { $0.toRecord() }
                records.forEach { localStore.upsert($0) }
            } catch {
                // Swallow network errors — fall through to local cache.
            }
        }
        return localStore.loadAll()
    }
}

private extension RemoteCredential {
    func toRecord() -> CredentialRecord {
        CredentialRecord(
            id:             id,
            credentialType: type,
            issuer:         issuer,
            issuedAt:       issuedAt,
            expiresAt:      expiresAt,
            vcJson:         vc ?? "{}",
            status:         status
        )
    }
}
