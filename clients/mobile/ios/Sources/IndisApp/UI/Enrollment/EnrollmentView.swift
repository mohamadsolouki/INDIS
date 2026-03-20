import SwiftUI

/// Multi-step enrollment flow.
///
/// Step 1: pathway selection (Standard / Enhanced / Social Attestation)
/// Step 2: document capture (national ID card scan)
/// Step 3: biometric capture (face + optional fingerprint)
/// Step 4: waiting for gateway approval
///
/// PRD FR-003: three enrollment pathways.
struct EnrollmentView: View {

    @EnvironmentObject private var appState: AppState
    @Environment(\.dismiss) private var dismiss

    @State private var step: EnrollmentStep = .pathway
    @State private var selectedPathway: EnrollmentPathway = .standard
    @State private var enrollmentId: String = ""
    @State private var enrollmentStatus: String = ""
    @State private var errorMessage: String = ""

    private var repo: EnrollmentRepository {
        EnrollmentRepository(api: GatewayAPIClient(baseURL: appState.gatewayURL))
    }

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 0.06, green: 0.08, blue: 0.14).ignoresSafeArea()

                VStack(spacing: 0) {
                    // Progress bar
                    EnrollmentProgressBar(step: step)
                        .padding(.horizontal)
                        .padding(.top, 8)

                    // Step content
                    Group {
                        switch step {
                        case .pathway:
                            PathwaySelectionView(selected: $selectedPathway) {
                                step = .document
                            }
                        case .document:
                            DocumentStepView {
                                step = .biometric
                            }
                        case .biometric:
                            BiometricStepView { face, fingerprint in
                                Task { await submitEnrollment(face: face, fingerprint: fingerprint) }
                            }
                        case .waiting:
                            EnrollmentWaitingView(
                                enrollmentId: enrollmentId,
                                status: enrollmentStatus
                            ) { newStatus in
                                if newStatus == "approved" { dismiss() }
                                else { enrollmentStatus = newStatus }
                            }
                        }
                    }
                    .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    .animation(.easeInOut(duration: 0.3), value: step)
                }
            }
            .navigationTitle("ثبت‌نام")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("انصراف") { dismiss() }
                        .foregroundColor(.white)
                }
            }
        }
    }

    private func submitEnrollment(face: Data, fingerprint: Data) async {
        do {
            let did  = EncryptedWalletStore.shared.get(forKey: .did) ?? ""
            let id   = try await repo.startEnrollment(nationalId: did, pathway: selectedPathway)
            enrollmentId = id
            let result = try await repo.submitBiometric(enrollmentId: id, faceImageData: face, fingerprintData: fingerprint)
            enrollmentStatus = result.status
            await MainActor.run { step = .waiting }
        } catch {
            await MainActor.run { errorMessage = error.localizedDescription }
        }
    }
}

// MARK: — Step enum

enum EnrollmentStep: Int, CaseIterable {
    case pathway  = 0
    case document = 1
    case biometric = 2
    case waiting  = 3

    var label: String {
        switch self {
        case .pathway:   return "مسیر"
        case .document:  return "مدارک"
        case .biometric: return "بیومتریک"
        case .waiting:   return "تأیید"
        }
    }
}

// MARK: — Progress bar

private struct EnrollmentProgressBar: View {
    let step: EnrollmentStep

    var body: some View {
        HStack(spacing: 0) {
            ForEach(EnrollmentStep.allCases, id: \.rawValue) { s in
                VStack(spacing: 4) {
                    Circle()
                        .fill(s.rawValue <= step.rawValue ? Color.blue : Color.white.opacity(0.2))
                        .frame(width: 24, height: 24)
                        .overlay(
                            Text(PersianNumerals.format(s.rawValue + 1))
                                .font(.caption2).bold()
                                .foregroundColor(.white)
                        )
                    Text(s.label)
                        .font(.caption2)
                        .foregroundColor(s.rawValue <= step.rawValue ? .white : Color.white.opacity(0.4))
                }
                if s != EnrollmentStep.allCases.last {
                    Rectangle()
                        .fill(s.rawValue < step.rawValue ? Color.blue : Color.white.opacity(0.2))
                        .frame(height: 2)
                        .frame(maxWidth: .infinity)
                }
            }
        }
        .padding(.vertical, 12)
    }
}

// MARK: — Pathway selection

private struct PathwaySelectionView: View {
    @Binding var selected: EnrollmentPathway
    let onNext: () -> Void

    private let options: [(pathway: EnrollmentPathway, icon: String, title: String, desc: String)] = [
        (.standard, "doc.text.fill",          "استاندارد",       "اسکن مدارک + بیومتریک"),
        (.enhanced, "building.columns.fill",  "پیشرفته",         "تطبیق با ثبت احوال"),
        (.social,   "person.3.fill",          "تأیید اجتماعی",   "۳+ گواه + بیومتریک"),
    ]

    var body: some View {
        VStack(spacing: 20) {
            Text("مسیر ثبت‌نام را انتخاب کنید")
                .font(.title3).bold()
                .foregroundColor(.white)
                .padding(.top, 24)

            ForEach(options, id: \.pathway.rawValue) { option in
                PathwayCard(
                    icon: option.icon,
                    title: option.title,
                    desc: option.desc,
                    isSelected: selected == option.pathway
                ) {
                    selected = option.pathway
                }
            }

            Spacer()

            Button("ادامه", action: onNext)
                .frame(maxWidth: .infinity)
                .padding()
                .background(Color.blue)
                .foregroundColor(.white)
                .cornerRadius(14)
                .padding(.horizontal)
                .padding(.bottom, 32)
        }
        .padding(.horizontal)
    }
}

private struct PathwayCard: View {
    let icon: String
    let title: String
    let desc: String
    let isSelected: Bool
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 16) {
                Image(systemName: icon)
                    .resizable()
                    .scaledToFit()
                    .frame(width: 28, height: 28)
                    .foregroundColor(isSelected ? .blue : Color.white.opacity(0.6))
                VStack(alignment: .leading, spacing: 4) {
                    Text(title).font(.headline).foregroundColor(.white)
                    Text(desc).font(.caption).foregroundColor(Color.white.opacity(0.6))
                }
                Spacer()
                if isSelected {
                    Image(systemName: "checkmark.circle.fill").foregroundColor(.blue)
                }
            }
            .padding()
            .background(isSelected ? Color.blue.opacity(0.15) : Color.white.opacity(0.07))
            .cornerRadius(12)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? Color.blue : Color.clear, lineWidth: 1.5)
            )
        }
    }
}

// MARK: — Waiting view

private struct EnrollmentWaitingView: View {
    let enrollmentId: String
    let status: String
    let onStatusUpdate: (String) -> Void

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            if status == "approved" {
                Image(systemName: "checkmark.seal.fill")
                    .resizable().scaledToFit().frame(width: 80)
                    .foregroundColor(.green)
                Text("ثبت‌نام تأیید شد").font(.title2).bold().foregroundColor(.white)
            } else if status == "rejected" {
                Image(systemName: "xmark.seal.fill")
                    .resizable().scaledToFit().frame(width: 80)
                    .foregroundColor(.red)
                Text("ثبت‌نام رد شد").font(.title2).bold().foregroundColor(.white)
            } else {
                ProgressView().scaleEffect(2).tint(.blue)
                Text("در حال بررسی…").font(.title3).foregroundColor(.white)
                Text("شناسه: \(enrollmentId)")
                    .font(.caption).foregroundColor(Color.white.opacity(0.5))
            }
            Spacer()
        }
    }
}
