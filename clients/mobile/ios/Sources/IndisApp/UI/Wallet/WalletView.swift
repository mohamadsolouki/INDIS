import SwiftUI

/// Credential wallet screen — lists all locally cached W3C Verifiable Credentials.
///
/// Pulls from GatewayCredentialRepository (network → local cache fallback).
/// PRD FR-006: credentials must be available offline for 72 hours.
struct WalletView: View {

    @EnvironmentObject private var appState: AppState
    @State private var credentials: [CredentialRecord] = []
    @State private var loading = true
    @State private var errorMessage = ""
    @State private var showPrivacy = false

    private var repo: GatewayCredentialRepository {
        GatewayCredentialRepository(api: GatewayAPIClient(baseURL: appState.gatewayURL))
    }

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 0.06, green: 0.08, blue: 0.14).ignoresSafeArea()

                Group {
                    if loading {
                        ProgressView("در حال بارگذاری…")
                            .tint(.blue)
                            .foregroundColor(Color.white.opacity(0.7))
                    } else if credentials.isEmpty {
                        VStack(spacing: 16) {
                            Image(systemName: "wallet.pass")
                                .resizable().scaledToFit().frame(width: 60)
                                .foregroundColor(Color.white.opacity(0.3))
                            Text("هیچ مدرکی یافت نشد")
                                .foregroundColor(Color.white.opacity(0.5))
                        }
                    } else {
                        ScrollView {
                            LazyVStack(spacing: 12) {
                                ForEach(credentials) { cred in
                                    CredentialCardView(record: cred)
                                }
                            }
                            .padding(.horizontal)
                            .padding(.top, 4)
                        }
                    }
                }
            }
            .navigationTitle("کیف پول")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showPrivacy = true
                    } label: {
                        Image(systemName: "lock.shield")
                            .foregroundColor(.white)
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        Task { await refresh() }
                    } label: {
                        Image(systemName: "arrow.clockwise")
                            .foregroundColor(.white)
                    }
                }
            }
            .sheet(isPresented: $showPrivacy) {
                PrivacyCenterView()
            }
            .task { await refresh() }
        }
    }

    private func refresh() async {
        loading = true
        credentials = await repo.listCredentials()
        loading = false
    }
}
