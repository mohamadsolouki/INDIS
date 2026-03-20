import SwiftUI

/// Document capture step in the enrollment flow.
///
/// Instructs the user to photograph the front and back of their national ID card.
/// On a real device this would invoke `UIImagePickerController` / `VNDocumentCameraViewController`.
/// Here we use a placeholder tap-to-capture UI that accepts the step so the full
/// flow can be exercised before the real camera integration is wired.
struct DocumentStepView: View {

    let onNext: () -> Void

    @State private var frontCaptured = false
    @State private var backCaptured  = false

    var body: some View {
        VStack(spacing: 24) {
            Text("عکس کارت ملی")
                .font(.title3).bold()
                .foregroundColor(.white)
                .padding(.top, 24)

            Text("لطفاً روی کارت ملی خود را اسکن کنید")
                .font(.subheadline)
                .foregroundColor(Color.white.opacity(0.6))
                .multilineTextAlignment(.center)

            HStack(spacing: 16) {
                DocumentSideCard(
                    side: "روی کارت",
                    icon: "creditcard.fill",
                    captured: frontCaptured
                ) {
                    frontCaptured = true
                }
                DocumentSideCard(
                    side: "پشت کارت",
                    icon: "creditcard",
                    captured: backCaptured
                ) {
                    backCaptured = true
                }
            }
            .padding(.horizontal)

            Spacer()

            Button("ادامه") { onNext() }
                .disabled(!frontCaptured || !backCaptured)
                .frame(maxWidth: .infinity)
                .padding()
                .background(frontCaptured && backCaptured ? Color.blue : Color.gray)
                .foregroundColor(.white)
                .cornerRadius(14)
                .padding(.horizontal)
                .padding(.bottom, 32)
        }
        .padding(.horizontal)
    }
}

private struct DocumentSideCard: View {
    let side: String
    let icon: String
    let captured: Bool
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            VStack(spacing: 12) {
                ZStack {
                    RoundedRectangle(cornerRadius: 12)
                        .fill(captured ? Color.green.opacity(0.15) : Color.white.opacity(0.07))
                        .frame(height: 130)
                    if captured {
                        Image(systemName: "checkmark.circle.fill")
                            .resizable().scaledToFit().frame(width: 40)
                            .foregroundColor(.green)
                    } else {
                        Image(systemName: icon)
                            .resizable().scaledToFit().frame(width: 40)
                            .foregroundColor(Color.white.opacity(0.5))
                    }
                }
                Text(captured ? "ثبت شد ✓" : side)
                    .font(.caption)
                    .foregroundColor(captured ? .green : Color.white.opacity(0.7))
            }
        }
    }
}
