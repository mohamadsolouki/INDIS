import XCTest
@testable import IndisApp

final class CredentialStoreTests: XCTestCase {

    private let store = CredentialStore.shared

    override func setUp() {
        super.setUp()
        store.clear()
    }

    func testSaveAndLoad() {
        let record = CredentialRecord(
            id:             "cred-001",
            credentialType: "NationalIdCredential",
            issuer:         "did:indis:issuer",
            issuedAt:       "2025-01-01T00:00:00Z",
            expiresAt:      "2030-01-01T00:00:00Z",
            vcJson:         "{}",
            status:         "active"
        )
        store.upsert(record)
        let loaded = store.loadAll()
        XCTAssertEqual(loaded.count, 1)
        XCTAssertEqual(loaded[0].id, "cred-001")
    }

    func testUpsertUpdatesExisting() {
        let original = CredentialRecord(id: "c1", credentialType: "T", issuer: "I",
                                        issuedAt: "2025-01-01T00:00:00Z",
                                        expiresAt: "2030-01-01T00:00:00Z",
                                        vcJson: "{}", status: "active")
        let updated  = CredentialRecord(id: "c1", credentialType: "T", issuer: "I",
                                        issuedAt: "2025-01-01T00:00:00Z",
                                        expiresAt: "2030-01-01T00:00:00Z",
                                        vcJson: "{}", status: "revoked")
        store.upsert(original)
        store.upsert(updated)
        XCTAssertEqual(store.loadAll().count, 1)
        XCTAssertEqual(store.loadAll()[0].status, "revoked")
    }

    func testRevokedFlag() {
        let revoked = CredentialRecord(id: "r1", credentialType: "T", issuer: "I",
                                       issuedAt: "2025-01-01T00:00:00Z",
                                       expiresAt: "2030-01-01T00:00:00Z",
                                       vcJson: "{}", status: "revoked")
        XCTAssertTrue(revoked.isRevoked)
    }
}
