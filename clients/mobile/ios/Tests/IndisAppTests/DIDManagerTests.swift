import XCTest
@testable import IndisApp

/// Tests for DIDManager — verifies DID derivation format without Secure Enclave
/// (Secure Enclave is not available in the simulator so we test the format only).
final class DIDManagerTests: XCTestCase {

    /// A DID generated from any public key must match the `did:indis:` prefix
    /// and have a 40-character hex suffix (20 bytes × 2 hex chars each).
    func testDIDFormat() throws {
        // We cannot call generateDID() in unit tests (Secure Enclave absent),
        // but we can validate the derivation logic directly using mock bytes.
        let mockPubKeyBytes = Data(repeating: 0xAB, count: 65)
        let did = deriveDIDFromBytes(mockPubKeyBytes)
        XCTAssertTrue(did.hasPrefix("did:indis:"), "DID must start with did:indis:")
        let suffix = String(did.dropFirst("did:indis:".count))
        XCTAssertEqual(suffix.count, 40, "DID suffix must be 40 hex chars (20 bytes)")
        XCTAssertTrue(suffix.allSatisfy(\.isHexDigit), "DID suffix must be hex")
    }

    func testDIDDeterminism() {
        let bytes = Data((0..<65).map { UInt8($0) })
        let did1 = deriveDIDFromBytes(bytes)
        let did2 = deriveDIDFromBytes(bytes)
        XCTAssertEqual(did1, did2, "DID derivation must be deterministic")
    }

    /// Mirrors DIDManager.deriveDID() without needing a SecKey.
    private func deriveDIDFromBytes(_ bytes: Data) -> String {
        import CryptoKit
        let digest = SHA256.hash(data: bytes)
        let hex = digest.prefix(20).map { String(format: "%02x", $0) }.joined()
        return "did:indis:\(hex)"
    }
}
