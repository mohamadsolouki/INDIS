import SwiftUI

/// ZK credential verification screen.
///
/// Two modes:
///   - **Present**: citizen generates a ZK proof from a local credential and
///     displays it as a QR code for a verifier terminal to scan.
///   - **Scan**: citizen scans a verifier's challenge QR, generates a proof,
///     and POSTs to /v1/verifier/verify.
///
/// PRD FR-013: verifiers see ONLY a boolean result — never raw citizen data.
struct VerifyView: View {

    @EnvironmentObject private var appState: AppState
    @State private var mode: VerifyMode = .present
    @State private var selectedCredential: CredentialRecord?
    @State private var selectedPredicate: ZKProofManager.Predicate = .ageOver18
    @State private var qrImage: UIImage?
    @State private var generating = false
    @State private var errorMessage = ""

    // Scan mode state
    @State private var showScanner = false
    @State private var verifyResult: Bool?
    @State private var verifying = false

    private let zkManager = ZKProofManager()
    private var credStore: GatewayCredentialRepository {
        GatewayCredentialRepository(api: GatewayAPIClient(baseURL: appState.gatewayURL))
    }

    enum VerifyMode { case present, scan }

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 0.06, green: 0.08, blue: 0.14).ignoresSafeArea()

                VStack(spacing: 0) {
                    // Mode toggle
                    Picker("", selection: $mode) {
                        Text("ارائه مدرک").tag(VerifyMode.present)
                        Text("اسکن تأیید").tag(VerifyMode.scan)
                    }
                    .pickerStyle(.segmented)
                    .padding()

                    if mode == .present {
                        PresentModeView(
                            credentials: CredentialStore.shared.loadAll(),
                            selectedCredential: $selectedCredential,
                            selectedPredicate: $selectedPredicate,
                            qrImage: $qrImage,
                            generating: $generating,
                            errorMessage: $errorMessage,
                            zkManager: zkManager
                        )
                    } else {
                        ScanModeView(
                            showScanner: $showScanner,
                            verifyResult: $verifyResult,
                            verifying: $verifying,
                            appState: appState
                        )
                    }
                }
            }
            .navigationTitle("تأیید هویت")
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}

// MARK: — Present mode

private struct PresentModeView: View {

    let credentials: [CredentialRecord]
    @Binding var selectedCredential: CredentialRecord?
    @Binding var selectedPredicate: ZKProofManager.Predicate
    @Binding var qrImage: UIImage?
    @Binding var generating: Bool
    @Binding var errorMessage: String
    let zkManager: ZKProofManager

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                Text("انتخاب مدرک و ادعا")
                    .font(.subheadline).foregroundColor(Color.white.opacity(0.7))

                // Credential picker
                if credentials.isEmpty {
                    Text("هیچ مدرکی در کیف پول یافت نشد")
                        .foregroundColor(Color.white.opacity(0.5))
                        .padding()
                } else {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("مدرک").font(.caption).foregroundColor(Color.white.opacity(0.5))
                        ForEach(credentials) { cred in
                            Button {
                                selectedCredential = cred
                                qrImage = nil
                            } label: {
                                HStack {
                                    Text(cred.credentialType).foregroundColor(.white)
                                    Spacer()
                                    if selectedCredential?.id == cred.id {
                                        Image(systemName: "checkmark.circle.fill").foregroundColor(.blue)
                                    }
                                }
                                .padding(12)
                                .background(selectedCredential?.id == cred.id ? Color.blue.opacity(0.15) : Color.white.opacity(0.07))
                                .cornerRadius(10)
                            }
                        }
                    }
                    .padding(.horizontal)
                }

                // Predicate picker
                VStack(alignment: .leading, spacing: 8) {
                    Text("ادعا").font(.caption).foregroundColor(Color.white.opacity(0.5))
                    ForEach(ZKProofManager.Predicate.allCases, id: \.rawValue) { pred in
                        Button {
                            selectedPredicate = pred
                            qrImage = nil
                        } label: {
                            HStack {
                                Text(localizedPredicate(pred)).foregroundColor(.white)
                                Spacer()
                                if selectedPredicate == pred {
                                    Image(systemName: "checkmark.circle.fill").foregroundColor(.blue)
                                }
                            }
                            .padding(12)
                            .background(selectedPredicate == pred ? Color.blue.opacity(0.15) : Color.white.opacity(0.07))
                            .cornerRadius(10)
                        }
                    }
                }
                .padding(.horizontal)

                // QR display
                if let qr = qrImage {
                    Image(uiImage: qr)
                        .interpolation(.none)
                        .resizable().scaledToFit()
                        .frame(width: 240, height: 240)
                        .cornerRadius(12)
                        .padding()
                }

                if !errorMessage.isEmpty {
                    Text(errorMessage).font(.caption).foregroundColor(.red).padding(.horizontal)
                }

                // Generate button
                Button {
                    Task { await generate() }
                } label: {
                    if generating {
                        ProgressView().tint(.white)
                    } else {
                        Text(qrImage == nil ? "تولید ZK Proof" : "تولید مجدد")
                    }
                }
                .disabled(selectedCredential == nil || generating)
                .frame(maxWidth: .infinity)
                .padding()
                .background(selectedCredential != nil ? Color.blue : Color.gray)
                .foregroundColor(.white)
                .cornerRadius(14)
                .padding(.horizontal)
                .padding(.bottom, 32)
            }
            .padding(.top)
        }
    }

    private func generate() async {
        guard let cred = selectedCredential else { return }
        generating = true
        errorMessage = ""
        do {
            let payload = try await zkManager.generateProof(
                predicate: selectedPredicate,
                vcJson: cred.vcJson
            )
            let qrData = Data(payload.qrJSON.utf8)
            if let filter = CIFilter(name: "CIQRCodeGenerator") {
                filter.setValue(qrData, forKey: "inputMessage")
                filter.setValue("M", forKey: "inputCorrectionLevel")
                if let ciImage = filter.outputImage {
                    let scaled = ciImage.transformed(by: CGAffineTransform(scaleX: 10, y: 10))
                    qrImage = UIImage(ciImage: scaled)
                }
            }
        } catch {
            errorMessage = error.localizedDescription
        }
        generating = false
    }

    private func localizedPredicate(_ p: ZKProofManager.Predicate) -> String {
        switch p {
        case .ageOver18:       return "سن بیش از ۱۸"
        case .citizenshipIR:   return "تابعیت ایران"
        case .voterEligible:   return "واجد شرایط رأی"
        case .credentialValid: return "مدرک معتبر"
        }
    }
}

// MARK: — Scan mode

private struct ScanModeView: View {

    @Binding var showScanner: Bool
    @Binding var verifyResult: Bool?
    @Binding var verifying: Bool
    let appState: AppState

    var body: some View {
        VStack(spacing: 24) {
            if verifying {
                ProgressView("در حال تأیید…").tint(.blue).foregroundColor(Color.white.opacity(0.7))
            } else if let result = verifyResult {
                // Full-screen result (PRD FR-013)
                VStack(spacing: 20) {
                    Image(systemName: result ? "checkmark.seal.fill" : "xmark.seal.fill")
                        .resizable().scaledToFit().frame(width: 100)
                        .foregroundColor(result ? .green : .red)
                    Text(result ? "تأیید شد ✓" : "رد شد ✗")
                        .font(.largeTitle).bold()
                        .foregroundColor(result ? .green : .red)
                    Button("اسکن مجدد") { verifyResult = nil }
                        .padding()
                        .background(Color.white.opacity(0.1))
                        .cornerRadius(12)
                        .foregroundColor(.white)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(result ? Color.green.opacity(0.08) : Color.red.opacity(0.08))
            } else {
                Image(systemName: "qrcode.viewfinder")
                    .resizable().scaledToFit().frame(width: 80)
                    .foregroundColor(Color.white.opacity(0.4))
                    .padding(.top, 40)
                Text("QR تأیید‌کننده را اسکن کنید")
                    .foregroundColor(Color.white.opacity(0.7))
                Button("شروع اسکن") { showScanner = true }
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .cornerRadius(14)
                    .padding(.horizontal)
                Spacer()
            }
        }
        .sheet(isPresented: $showScanner) {
            QRScannerView(
                onScan: { payload in
                    showScanner = false
                    verifying = true
                    Task { await verify(payload: payload) }
                },
                onError: { _ in showScanner = false }
            )
        }
    }

    private func verify(payload: String) async {
        defer { verifying = false }
        guard let data = payload.data(using: .utf8),
              let json = try? JSONDecoder().decode(VerifyProofRequest.self, from: data) else {
            verifyResult = false
            return
        }
        do {
            let api = GatewayAPIClient(baseURL: appState.gatewayURL)
            let resp: VerifyProofResponse = try await api.post(
                "/v1/verifier/verify",
                body: json,
                token: appState.jwtToken
            )
            verifyResult = resp.valid
        } catch {
            verifyResult = false
        }
    }
}
