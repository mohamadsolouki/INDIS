import SwiftUI

/// Biometric capture step: face photo + optional fingerprint scan.
///
/// Production: uses AVCaptureSession for live face capture and Core NFC / Touch ID
/// for fingerprint. Dev placeholder uses tap-to-simulate for both.
struct BiometricStepView: View {

    let onComplete: (Data, Data) -> Void

    @State private var faceCaptured        = false
    @State private var fingerprintCaptured = false
    @State private var isCapturingFace     = false

    var body: some View {
        VStack(spacing: 24) {
            Text("بیومتریک")
                .font(.title3).bold()
                .foregroundColor(.white)
                .padding(.top, 24)

            Text("چهره و اثر انگشت خود را ثبت کنید")
                .font(.subheadline)
                .foregroundColor(Color.white.opacity(0.6))

            // Face capture
            BiometricCaptureTile(
                icon:     "face.smiling.inverse",
                title:    "اسکن چهره",
                captured: faceCaptured,
                color:    .blue
            ) {
                isCapturingFace = true
                // Simulate async camera capture
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                    faceCaptured = true
                    isCapturingFace = false
                }
            }
            .padding(.horizontal)

            // Fingerprint capture
            BiometricCaptureTile(
                icon:     "hand.point.up.left.fill",
                title:    "اثر انگشت",
                captured: fingerprintCaptured,
                color:    .purple
            ) {
                fingerprintCaptured = true
            }
            .padding(.horizontal)

            Spacer()

            Button("تأیید و ادامه") {
                let mockFace        = Data(repeating: 0xFF, count: 512)
                let mockFingerprint = Data(repeating: 0xAA, count: 256)
                onComplete(mockFace, mockFingerprint)
            }
            .disabled(!faceCaptured || !fingerprintCaptured)
            .frame(maxWidth: .infinity)
            .padding()
            .background(faceCaptured && fingerprintCaptured ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(14)
            .padding(.horizontal)
            .padding(.bottom, 32)
        }
    }
}

private struct BiometricCaptureTile: View {
    let icon: String
    let title: String
    let captured: Bool
    let color: Color
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 16) {
                ZStack {
                    Circle()
                        .fill(captured ? Color.green.opacity(0.15) : color.opacity(0.15))
                        .frame(width: 56, height: 56)
                    Image(systemName: captured ? "checkmark.circle.fill" : icon)
                        .resizable().scaledToFit().frame(width: 28)
                        .foregroundColor(captured ? .green : color)
                }
                VStack(alignment: .leading, spacing: 4) {
                    Text(title).font(.headline).foregroundColor(.white)
                    Text(captured ? "ثبت شد" : "برای شروع لمس کنید")
                        .font(.caption)
                        .foregroundColor(captured ? .green : Color.white.opacity(0.5))
                }
                Spacer()
            }
            .padding()
            .background(Color.white.opacity(0.07))
            .cornerRadius(14)
        }
    }
}
