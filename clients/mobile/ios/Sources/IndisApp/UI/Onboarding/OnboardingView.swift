import SwiftUI

/// First-launch onboarding screen.
///
/// Guides the citizen through three introduction cards then prompts them to
/// enter their national ID to register.  On successful registration the
/// AppState.isAuthenticated flag flips to true and ContentView switches to MainTabView.
struct OnboardingView: View {

    @EnvironmentObject private var appState: AppState
    @State private var currentPage = 0
    @State private var showRegistration = false

    private let pages: [(icon: String, title: String, body: String)] = [
        ("shield.lefthalf.filled",   "هویت دیجیتال ایرانی",   "کارت هویت ملی دیجیتال امن با رمزنگاری پیشرفته"),
        ("lock.icloud",              "کلید در دست شما",       "کلید خصوصی روی دستگاه شما می‌ماند — هیچ‌وقت منتقل نمی‌شود"),
        ("checkmark.seal.fill",      "اثبات بدون افشا",       "اثبات هویت با ZK-Proof — بدون نمایش اطلاعات خصوصی"),
    ]

    var body: some View {
        ZStack {
            Color(red: 0.08, green: 0.10, blue: 0.16).ignoresSafeArea()

            VStack(spacing: 0) {
                // Pages
                TabView(selection: $currentPage) {
                    ForEach(pages.indices, id: \.self) { i in
                        OnboardingPageView(
                            icon:  pages[i].icon,
                            title: pages[i].title,
                            body:  pages[i].body
                        )
                        .tag(i)
                    }
                }
                .tabViewStyle(.page(indexDisplayMode: .never))
                .frame(maxHeight: .infinity)

                // Dots
                HStack(spacing: 8) {
                    ForEach(pages.indices, id: \.self) { i in
                        Circle()
                            .fill(i == currentPage ? Color.blue : Color.white.opacity(0.3))
                            .frame(width: 8, height: 8)
                    }
                }
                .padding(.bottom, 24)

                // CTA
                VStack(spacing: 12) {
                    Button {
                        if currentPage < pages.count - 1 {
                            withAnimation { currentPage += 1 }
                        } else {
                            showRegistration = true
                        }
                    } label: {
                        Text(currentPage < pages.count - 1 ? "ادامه" : "شروع ثبت‌نام")
                            .font(.headline)
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color.blue)
                            .foregroundColor(.white)
                            .cornerRadius(14)
                    }
                    .padding(.horizontal, 24)

                    if currentPage == pages.count - 1 {
                        Button {
                            // Skip to dev login
                            appState.login(did: "did:indis:devdevdevdev", token: "dev-token")
                        } label: {
                            Text("ورود توسعه‌دهنده")
                                .font(.caption)
                                .foregroundColor(Color.white.opacity(0.3))
                        }
                    }
                }
                .padding(.bottom, 40)
            }
        }
        .sheet(isPresented: $showRegistration) {
            RegistrationSheet()
        }
    }
}

// MARK: — Page card

private struct OnboardingPageView: View {
    let icon: String
    let title: String
    let body: String

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Image(systemName: icon)
                .resizable()
                .scaledToFit()
                .frame(width: 90, height: 90)
                .foregroundColor(.blue)
            Text(title)
                .font(.title2).bold()
                .foregroundColor(.white)
                .multilineTextAlignment(.center)
            Text(body)
                .font(.body)
                .foregroundColor(Color.white.opacity(0.7))
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
            Spacer()
        }
    }
}

// MARK: — Registration sheet

private struct RegistrationSheet: View {

    @EnvironmentObject private var appState: AppState
    @Environment(\.dismiss) private var dismiss

    @State private var nationalId = ""
    @State private var loading = false
    @State private var errorMessage = ""

    private var repo: GatewayIdentityRepository {
        GatewayIdentityRepository(api: GatewayAPIClient(baseURL: appState.gatewayURL))
    }

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 0.08, green: 0.10, blue: 0.16).ignoresSafeArea()

                VStack(spacing: 24) {
                    Image(systemName: "person.crop.circle.badge.plus")
                        .resizable()
                        .scaledToFit()
                        .frame(width: 60)
                        .foregroundColor(.blue)
                        .padding(.top, 32)

                    Text("ثبت کد ملی")
                        .font(.title2).bold()
                        .foregroundColor(.white)

                    Text("کد ملی ۱۰ رقمی خود را وارد کنید")
                        .font(.subheadline)
                        .foregroundColor(Color.white.opacity(0.6))

                    TextField("کد ملی", text: $nationalId)
                        .keyboardType(.numberPad)
                        .padding()
                        .background(Color.white.opacity(0.08))
                        .cornerRadius(12)
                        .foregroundColor(.white)
                        .environment(\.layoutDirection, .leftToRight)
                        .padding(.horizontal, 24)

                    if !errorMessage.isEmpty {
                        Text(errorMessage)
                            .font(.caption)
                            .foregroundColor(.red)
                            .padding(.horizontal, 24)
                    }

                    Button {
                        Task { await register() }
                    } label: {
                        if loading {
                            ProgressView().tint(.white)
                        } else {
                            Text("ثبت‌نام")
                                .font(.headline)
                        }
                    }
                    .disabled(nationalId.count < 10 || loading)
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(nationalId.count >= 10 ? Color.blue : Color.gray)
                    .foregroundColor(.white)
                    .cornerRadius(14)
                    .padding(.horizontal, 24)

                    Spacer()
                }
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("انصراف") { dismiss() }
                        .foregroundColor(.white)
                }
            }
        }
    }

    private func register() async {
        loading = true
        errorMessage = ""
        do {
            let did = try await repo.enrollNationalId(nationalId)
            let token = EncryptedWalletStore.shared.get(forKey: .jwtToken) ?? ""
            await MainActor.run {
                appState.login(did: did, token: token)
                dismiss()
            }
        } catch {
            await MainActor.run {
                errorMessage = error.localizedDescription
                loading = false
            }
        }
    }
}
