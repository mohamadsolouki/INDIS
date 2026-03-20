import SwiftUI

/// Root navigation after login: bottom tab bar with 4 tabs.
///
/// Mirrors Android MainActivity's bottom navigation:
///   Home → Wallet → Verify → Settings
struct MainTabView: View {

    @EnvironmentObject private var appState: AppState

    var body: some View {
        TabView {
            HomeTab()
                .tabItem {
                    Label("خانه", systemImage: "house.fill")
                }

            WalletView()
                .tabItem {
                    Label("کیف پول", systemImage: "wallet.pass.fill")
                }

            VerifyView()
                .tabItem {
                    Label("تأیید", systemImage: "qrcode.viewfinder")
                }

            SettingsView()
                .tabItem {
                    Label("تنظیمات", systemImage: "gearshape.fill")
                }
        }
        .accentColor(.blue)
    }
}

// MARK: — Home tab

private struct HomeTab: View {

    @EnvironmentObject private var appState: AppState

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 0.06, green: 0.08, blue: 0.14).ignoresSafeArea()

                VStack(spacing: 24) {
                    // DID card
                    VStack(alignment: .leading, spacing: 8) {
                        Text("هویت دیجیتال شما")
                            .font(.caption)
                            .foregroundColor(Color.white.opacity(0.6))
                        Text(appState.did)
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(.white)
                            .lineLimit(2)
                    }
                    .padding()
                    .background(Color.white.opacity(0.07))
                    .cornerRadius(14)
                    .padding(.horizontal)

                    // Quick-action grid
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 16) {
                        QuickActionCard(icon: "plus.circle.fill", title: "ثبت‌نام", color: .blue) {
                            // Enrollment is handled via EnrollmentView sheet
                        }
                        QuickActionCard(icon: "wallet.pass.fill", title: "کیف پول", color: .green) {}
                        QuickActionCard(icon: "qrcode.viewfinder", title: "ارائه مدرک", color: .orange) {}
                        QuickActionCard(icon: "lock.shield.fill", title: "حریم خصوصی", color: .purple) {}
                    }
                    .padding(.horizontal)

                    Spacer()
                }
                .padding(.top)
            }
            .navigationTitle("INDIS")
            .navigationBarTitleDisplayMode(.large)
        }
    }
}

private struct QuickActionCard: View {
    let icon: String
    let title: String
    let color: Color
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 12) {
                Image(systemName: icon)
                    .resizable()
                    .scaledToFit()
                    .frame(width: 32, height: 32)
                    .foregroundColor(color)
                Text(title)
                    .font(.subheadline).bold()
                    .foregroundColor(.white)
            }
            .frame(maxWidth: .infinity)
            .padding(20)
            .background(Color.white.opacity(0.07))
            .cornerRadius(16)
        }
    }
}
