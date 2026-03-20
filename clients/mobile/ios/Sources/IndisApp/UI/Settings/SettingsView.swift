import SwiftUI

/// Settings screen — mirrors Android SettingsActivity.
///
/// Sections:
///   - Language selection (fa / en / ckb / kmr / ar / az)
///   - Persian numerals toggle
///   - Gateway URL configuration
///   - Privacy Center shortcut
///   - App info (version + DID)
///   - Logout
struct SettingsView: View {

    @EnvironmentObject private var appState: AppState
    @State private var showLogoutConfirmation = false
    @State private var showPrivacy = false
    @State private var editingURL = false
    @State private var draftURL = ""

    private let locales: [(code: String, label: String)] = [
        ("fa",  "فارسی"),
        ("en",  "English"),
        ("ckb", "کوردی سۆرانی"),
        ("kmr", "Kurdî Kurmancî"),
        ("ar",  "العربية"),
        ("az",  "Azərbaycanca"),
    ]

    private var appVersion: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build   = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(version) (\(build))"
    }

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 0.06, green: 0.08, blue: 0.14).ignoresSafeArea()

                List {
                    // MARK: Language
                    Section {
                        Picker("زبان", selection: Binding(
                            get: { appState.selectedLocale },
                            set: { appState.saveSettings(gatewayURL: appState.gatewayURL, locale: $0, persianNumerals: appState.usePersianNumerals) }
                        )) {
                            ForEach(locales, id: \.code) { locale in
                                Text(locale.label).tag(locale.code)
                            }
                        }
                    } header: {
                        Text("زبان").settingHeader()
                    }
                    .listRowBackground(rowBackground)

                    // MARK: Display
                    Section {
                        Toggle(isOn: Binding(
                            get: { appState.usePersianNumerals },
                            set: { appState.saveSettings(gatewayURL: appState.gatewayURL, locale: appState.selectedLocale, persianNumerals: $0) }
                        )) {
                            Label("اعداد فارسی", systemImage: "textformat.123")
                                .foregroundColor(.white)
                        }
                        .tint(.blue)
                    } header: {
                        Text("نمایش").settingHeader()
                    }
                    .listRowBackground(rowBackground)

                    // MARK: Network
                    Section {
                        VStack(alignment: .leading, spacing: 8) {
                            Label("آدرس Gateway", systemImage: "network")
                                .foregroundColor(.white)
                            if editingURL {
                                TextField("http://...", text: $draftURL)
                                    .textFieldStyle(.plain)
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundColor(.white)
                                    .environment(\.layoutDirection, .leftToRight)
                                    .onSubmit {
                                        appState.saveSettings(gatewayURL: draftURL, locale: appState.selectedLocale, persianNumerals: appState.usePersianNumerals)
                                        editingURL = false
                                    }
                            } else {
                                Text(appState.gatewayURL)
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundColor(Color.white.opacity(0.5))
                                    .onTapGesture {
                                        draftURL = appState.gatewayURL
                                        editingURL = true
                                    }
                            }
                        }
                        .padding(.vertical, 4)
                    } header: {
                        Text("شبکه").settingHeader()
                    }
                    .listRowBackground(rowBackground)

                    // MARK: Privacy
                    Section {
                        Button {
                            showPrivacy = true
                        } label: {
                            Label("مرکز حریم خصوصی", systemImage: "lock.shield")
                                .foregroundColor(.white)
                        }
                    } header: {
                        Text("حریم خصوصی").settingHeader()
                    }
                    .listRowBackground(rowBackground)

                    // MARK: About
                    Section {
                        HStack {
                            Label("نسخه", systemImage: "info.circle")
                                .foregroundColor(.white)
                            Spacer()
                            Text(appVersion)
                                .font(.caption)
                                .foregroundColor(Color.white.opacity(0.5))
                        }
                        VStack(alignment: .leading, spacing: 4) {
                            Label("شناسه دستگاه (DID)", systemImage: "key.fill")
                                .foregroundColor(.white)
                                .font(.subheadline)
                            Text(appState.did)
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundColor(Color.white.opacity(0.4))
                                .lineLimit(2)
                        }
                        .padding(.vertical, 4)
                    } header: {
                        Text("درباره").settingHeader()
                    }
                    .listRowBackground(rowBackground)

                    // MARK: Logout
                    Section {
                        Button(role: .destructive) {
                            showLogoutConfirmation = true
                        } label: {
                            Label("خروج از حساب", systemImage: "rectangle.portrait.and.arrow.right")
                        }
                    }
                    .listRowBackground(rowBackground)
                }
                .listStyle(.insetGrouped)
                .scrollContentBackground(.hidden)
            }
            .navigationTitle("تنظیمات")
            .navigationBarTitleDisplayMode(.large)
            .confirmationDialog(
                "آیا مطمئن هستید که می‌خواهید خارج شوید؟",
                isPresented: $showLogoutConfirmation,
                titleVisibility: .visible
            ) {
                Button("خروج", role: .destructive) { appState.logout() }
                Button("انصراف", role: .cancel) {}
            }
            .sheet(isPresented: $showPrivacy) {
                PrivacyCenterView()
            }
        }
    }

    private var rowBackground: some View {
        Color.white.opacity(0.07)
    }
}

// MARK: — Helper modifier

private extension Text {
    func settingHeader() -> some View {
        self
            .font(.caption)
            .foregroundColor(Color.white.opacity(0.5))
            .textCase(nil)
    }
}
