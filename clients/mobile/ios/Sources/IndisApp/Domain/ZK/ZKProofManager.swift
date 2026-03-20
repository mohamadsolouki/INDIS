import Foundation

/// Result of an on-device ZK proof generation.
struct ZKProofPayload: Codable {
    /// Base64-encoded Groth16 proof bytes.
    let proof: String
    /// Random nonce preventing proof replay.
    let nonce: String
    /// Predicate that was proven (e.g. "age_over_18").
    let predicate: String
    /// Proof system used.
    let proofSystem: String

    /// Encodes the payload as a QR-ready JSON string.
    var qrJSON: String {
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        return (try? String(data: encoder.encode(self), encoding: .utf8)) ?? "{}"
    }
}

/// On-device ZK proof generation manager.
///
/// Production: bridges to the zkproof Rust crate via a C FFI / Swift Package plugin.
/// Development placeholder: returns a deterministic mock proof so the full UI flow
/// can be exercised before the native Rust bridge is compiled.
///
/// PRD FR-006: proof generation must work fully offline using locally cached credentials.
final class ZKProofManager {

    enum Predicate: String, CaseIterable {
        case ageOver18          = "age_over_18"
        case citizenshipIR      = "citizenship_ir"
        case voterEligible      = "voter_eligible"
        case credentialValid    = "credential_valid"
    }

    // MARK: — Public API

    /// Generates a ZK proof for the given predicate over the supplied credential JSON.
    ///
    /// - Parameters:
    ///   - predicate: The claim being proven.
    ///   - vcJson: Raw W3C VC JSON from the local credential wallet.
    ///   - revocationList: Cached revocation list to include in the proof circuit.
    func generateProof(
        predicate: Predicate,
        vcJson: String,
        revocationList: [String] = []
    ) async throws -> ZKProofPayload {
        // TODO: replace with Rust FFI bridge via `indis_zkproof` crate.
        // The bridge will be compiled using `swift-bridge` or `uniffi`.
        return await Task.detached(priority: .userInitiated) {
            self.mockProof(predicate: predicate)
        }.value
    }

    // MARK: — Private

    private func mockProof(predicate: Predicate) -> ZKProofPayload {
        let nonce = UUID().uuidString.replacingOccurrences(of: "-", with: "").lowercased()
        let mockBytes = Data(repeating: 0xAB, count: 128)
        return ZKProofPayload(
            proof:       mockBytes.base64EncodedString(),
            nonce:       nonce,
            predicate:   predicate.rawValue,
            proofSystem: "groth16"
        )
    }
}
