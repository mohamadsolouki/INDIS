import Foundation
import Security
import CryptoKit

/// Generates and manages the citizen's device-bound DID.
///
/// Uses the iOS Secure Enclave to generate a P-256 (secp256r1) key pair.
/// The private key never leaves the Secure Enclave hardware.
/// The DID suffix is derived by SHA-256 hashing the raw public key bytes
/// and taking the first 20 bytes as a hex string — matching Android's derivation.
///
/// PRD FR-001: device-bound key; private key never transmitted.
final class DIDManager {

    private let keyTag = "org.indis.app.device_signing_key"

    // MARK: — Public API

    /// Generates a DID backed by a Secure Enclave key.
    /// If a key with the expected tag already exists it is reused.
    func generateDID() throws -> String {
        let publicKey = try loadOrCreateKey()
        return try deriveDID(from: publicKey)
    }

    /// Returns the existing DID if the key is already present, nil otherwise.
    func existingDID() -> String? {
        guard let key = loadExistingKey() else { return nil }
        return try? deriveDID(from: key)
    }

    // MARK: — Private

    private func loadOrCreateKey() throws -> SecKey {
        if let existing = loadExistingKey() { return existing }
        return try createKey()
    }

    private func loadExistingKey() -> SecKey? {
        let query: [CFString: Any] = [
            kSecClass:              kSecClassKey,
            kSecAttrKeyType:        kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrApplicationTag: keyTag.data(using: .utf8)!,
            kSecReturnRef:          true,
        ]
        var result: AnyObject?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess else { return nil }
        // We need the public key.
        guard let privateKey = result as? SecKey,
              let publicKey = SecKeyCopyPublicKey(privateKey) else { return nil }
        return publicKey
    }

    private func createKey() throws -> SecKey {
        var error: Unmanaged<CFError>?

        // Access control: Secure Enclave, biometry or device passcode.
        let access = SecAccessControlCreateWithFlags(
            kCFAllocatorDefault,
            kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
            .privateKeyUsage,
            &error
        )
        guard let access, error == nil else {
            throw DIDError.keyGenerationFailed(error!.takeRetainedValue().localizedDescription)
        }

        let attributes: [CFString: Any] = [
            kSecAttrKeyType:          kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits:    256,
            kSecAttrTokenID:          kSecAttrTokenIDSecureEnclave,
            kSecPrivateKeyAttrs: [
                kSecAttrIsPermanent:    true,
                kSecAttrApplicationTag: keyTag.data(using: .utf8)!,
                kSecAttrAccessControl:  access,
            ] as [CFString: Any],
        ]

        guard let privateKey = SecKeyCreateRandomKey(attributes as CFDictionary, &error),
              let publicKey = SecKeyCopyPublicKey(privateKey) else {
            throw DIDError.keyGenerationFailed(error?.takeRetainedValue().localizedDescription ?? "unknown")
        }
        return publicKey
    }

    private func deriveDID(from publicKey: SecKey) throws -> String {
        var error: Unmanaged<CFError>?
        guard let pubData = SecKeyCopyExternalRepresentation(publicKey, &error) as Data? else {
            throw DIDError.keyExportFailed
        }
        // SHA-256 of the raw public key bytes; take first 20 bytes as hex suffix.
        let digest = SHA256.hash(data: pubData)
        let hex = digest.prefix(20).map { String(format: "%02x", $0) }.joined()
        return "did:indis:\(hex)"
    }
}

enum DIDError: LocalizedError {
    case keyGenerationFailed(String)
    case keyExportFailed

    var errorDescription: String? {
        switch self {
        case .keyGenerationFailed(let msg): return "Key generation failed: \(msg)"
        case .keyExportFailed:              return "Failed to export public key"
        }
    }
}
