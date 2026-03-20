import Foundation

// MARK: — Identity

struct RegisterIdentityRequest: Encodable {
    let national_id: String
    let did: String
}

struct RegisterIdentityResponse: Decodable {
    let did: String
    let token: String
}

// MARK: — Credentials

struct CredentialListResponse: Decodable {
    let credentials: [RemoteCredential]
}

struct RemoteCredential: Decodable {
    let id: String
    let type: String
    let issuer: String
    let issuedAt: String
    let expiresAt: String
    let vc: String?
    let status: String
}

// MARK: — Enrollment

struct StartEnrollmentRequest: Encodable {
    let national_id: String
    let did: String
    let pathway: String   // "standard" | "enhanced" | "social"
}

struct StartEnrollmentResponse: Decodable {
    let enrollment_id: String
    let status: String
}

struct SubmitBiometricRequest: Encodable {
    let face_image_b64: String
    let fingerprint_b64: String
}

struct EnrollmentStatusResponse: Decodable {
    let enrollment_id: String
    let status: String        // "pending" | "approved" | "rejected"
    let message: String?
}

// MARK: — Revocation

struct RevocationListResponse: Decodable {
    let revoked_credential_ids: [String]
}

// MARK: — Privacy

struct PrivacyHistoryResponse: Decodable {
    let events: [PrivacyEvent]
}

struct PrivacyEvent: Decodable, Identifiable {
    let id: String
    let event_type: String
    let verifier_id: String?
    let timestamp: String
    let predicate: String?
}

struct ConsentRulesResponse: Decodable {
    let rules: [ConsentRule]
}

struct ConsentRule: Decodable, Identifiable {
    let id: String
    let verifier_id: String
    let attribute: String
    let granted: Bool
    let expires_at: String?
}

struct ExportDataRequest: Encodable {
    let format: String   // "json" | "pdf"
}

struct ExportDataResponse: Decodable {
    let download_url: String
    let expires_at: String
}

// MARK: — Verifier

struct VerifyProofRequest: Encodable {
    let verifier_id: String
    let proof_b64: String
    let nonce: String
    let predicate: String
    let credential_type: String
    let proof_system: String
    let public_inputs_b64: String
}

struct VerifyProofResponse: Decodable {
    let valid: Bool
    let message: String?
}
