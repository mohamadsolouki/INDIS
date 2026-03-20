import Foundation

/// Protocol for enrollment workflow operations.
protocol EnrollmentRepositoryProtocol {
    /// Starts an enrollment session and returns the enrollment ID.
    func startEnrollment(nationalId: String, pathway: EnrollmentPathway) async throws -> String
    /// Submits biometric data and returns the updated status.
    func submitBiometric(enrollmentId: String, faceImageData: Data, fingerprintData: Data) async throws -> EnrollmentStatusResponse
    /// Polls the status of an in-progress enrollment.
    func checkStatus(enrollmentId: String) async throws -> EnrollmentStatusResponse
}

/// Three enrollment pathways supported by the INDIS system (PRD §FR-003).
enum EnrollmentPathway: String {
    case standard = "standard"   // Document scans + biometrics
    case enhanced = "enhanced"   // Civil registry cross-check
    case social   = "social"     // 3+ community co-attestors + biometrics
}

/// Concrete enrollment repository backed by GatewayAPIClient.
final class EnrollmentRepository: EnrollmentRepositoryProtocol {

    private let api: GatewayAPIClient
    private let walletStore: EncryptedWalletStore

    init(api: GatewayAPIClient, walletStore: EncryptedWalletStore = .shared) {
        self.api = api
        self.walletStore = walletStore
    }

    func startEnrollment(nationalId: String, pathway: EnrollmentPathway) async throws -> String {
        let did   = walletStore.get(forKey: .did) ?? ""
        let token = walletStore.get(forKey: .jwtToken) ?? ""
        let request = StartEnrollmentRequest(national_id: nationalId, did: did, pathway: pathway.rawValue)
        let response: StartEnrollmentResponse = try await api.post(
            "/v1/enrollment/start",
            body: request,
            token: token
        )
        return response.enrollment_id
    }

    func submitBiometric(enrollmentId: String, faceImageData: Data, fingerprintData: Data) async throws -> EnrollmentStatusResponse {
        let token = walletStore.get(forKey: .jwtToken) ?? ""
        let request = SubmitBiometricRequest(
            face_image_b64:   faceImageData.base64EncodedString(),
            fingerprint_b64:  fingerprintData.base64EncodedString()
        )
        return try await api.post(
            "/v1/enrollment/\(enrollmentId)/biometric",
            body: request,
            token: token
        )
    }

    func checkStatus(enrollmentId: String) async throws -> EnrollmentStatusResponse {
        let token = walletStore.get(forKey: .jwtToken) ?? ""
        return try await api.get("/v1/enrollment/\(enrollmentId)/status", token: token)
    }
}
