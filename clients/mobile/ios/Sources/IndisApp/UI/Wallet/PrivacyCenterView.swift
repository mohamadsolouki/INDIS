import SwiftUI

/// Privacy Center — three-tab screen mirroring Android's PrivacyCenterActivity.
///
/// Tab 1: Disclosure history (GET /v1/privacy/history)
/// Tab 2: Consent rules (GET /v1/privacy/consent)
/// Tab 3: Data export (POST /v1/privacy/export)
struct PrivacyCenterView: View {

    @EnvironmentObject private var appState: AppState
    @Environment(\.dismiss) private var dismiss
    @State private var selectedTab = 0

    private var api: GatewayAPIClient {
        GatewayAPIClient(baseURL: appState.gatewayURL)
    }

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 0.06, green: 0.08, blue: 0.14).ignoresSafeArea()

                VStack(spacing: 0) {
                    // Tab picker
                    Picker("", selection: $selectedTab) {
                        Text("تاریخچه").tag(0)
                        Text("رضایت").tag(1)
                        Text("خروجی").tag(2)
                    }
                    .pickerStyle(.segmented)
                    .padding()

                    TabView(selection: $selectedTab) {
                        PrivacyHistoryTab(api: api, token: appState.jwtToken).tag(0)
                        PrivacyConsentTab(api: api, token: appState.jwtToken).tag(1)
                        PrivacyExportTab(api: api, token: appState.jwtToken).tag(2)
                    }
                    .tabViewStyle(.page(indexDisplayMode: .never))
                }
            }
            .navigationTitle("مرکز حریم خصوصی")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("بستن") { dismiss() }
                        .foregroundColor(.white)
                }
            }
        }
    }
}

// MARK: — History tab

private struct PrivacyHistoryTab: View {

    let api: GatewayAPIClient
    let token: String

    @State private var events: [PrivacyEvent] = []
    @State private var loading = true

    var body: some View {
        Group {
            if loading {
                ProgressView().tint(.blue)
            } else if events.isEmpty {
                emptyView(icon: "doc.text", message: "هیچ رویدادی ثبت نشده")
            } else {
                List(events) { event in
                    PrivacyEventRow(event: event)
                        .listRowBackground(Color.white.opacity(0.06))
                }
                .listStyle(.plain)
                .scrollContentBackground(.hidden)
            }
        }
        .task {
            do {
                let resp: PrivacyHistoryResponse = try await api.get("/v1/privacy/history", token: token)
                events = resp.events
            } catch {}
            loading = false
        }
    }
}

private struct PrivacyEventRow: View {
    let event: PrivacyEvent

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(localizedEventType(event.event_type))
                    .font(.subheadline).bold().foregroundColor(.white)
                Spacer()
                Text(PersianCalendar.formatISO(event.timestamp))
                    .font(.caption).foregroundColor(Color.white.opacity(0.5))
            }
            if let verifier = event.verifier_id {
                Text("تأیید‌کننده: \(verifier)")
                    .font(.caption).foregroundColor(Color.white.opacity(0.6))
            }
            if let predicate = event.predicate {
                Text("ادعا: \(predicate)")
                    .font(.caption).foregroundColor(Color.blue.opacity(0.8))
            }
        }
        .padding(.vertical, 4)
    }

    private func localizedEventType(_ type: String) -> String {
        switch type {
        case "disclosure": return "افشای اطلاعات"
        case "consent":    return "رضایت"
        case "export":     return "صدور خروجی"
        default:           return type
        }
    }
}

// MARK: — Consent tab

private struct PrivacyConsentTab: View {

    let api: GatewayAPIClient
    let token: String

    @State private var rules: [ConsentRule] = []
    @State private var loading = true

    var body: some View {
        Group {
            if loading {
                ProgressView().tint(.blue)
            } else if rules.isEmpty {
                emptyView(icon: "hand.raised", message: "هیچ قانون رضایتی ثبت نشده")
            } else {
                List(rules) { rule in
                    ConsentRuleRow(rule: rule)
                        .listRowBackground(Color.white.opacity(0.06))
                }
                .listStyle(.plain)
                .scrollContentBackground(.hidden)
            }
        }
        .task {
            do {
                let resp: ConsentRulesResponse = try await api.get("/v1/privacy/consent", token: token)
                rules = resp.rules
            } catch {}
            loading = false
        }
    }
}

private struct ConsentRuleRow: View {
    let rule: ConsentRule

    var body: some View {
        HStack {
            Image(systemName: rule.granted ? "checkmark.circle.fill" : "xmark.circle.fill")
                .foregroundColor(rule.granted ? .green : .red)
            VStack(alignment: .leading, spacing: 4) {
                Text(rule.attribute).font(.subheadline).foregroundColor(.white)
                Text(rule.verifier_id).font(.caption).foregroundColor(Color.white.opacity(0.5))
            }
        }
    }
}

// MARK: — Export tab

private struct PrivacyExportTab: View {

    let api: GatewayAPIClient
    let token: String

    @State private var format = "json"
    @State private var exporting = false
    @State private var downloadURL = ""
    @State private var errorMessage = ""

    var body: some View {
        VStack(spacing: 24) {
            Image(systemName: "square.and.arrow.up")
                .resizable().scaledToFit().frame(width: 56)
                .foregroundColor(.blue)
                .padding(.top, 32)

            Text("صدور اطلاعات شخصی")
                .font(.title3).bold().foregroundColor(.white)
            Text("کلیه اطلاعاتی که INDIS درباره شما دارد را دریافت کنید.")
                .font(.subheadline)
                .foregroundColor(Color.white.opacity(0.6))
                .multilineTextAlignment(.center)
                .padding(.horizontal, 24)

            Picker("فرمت", selection: $format) {
                Text("JSON").tag("json")
                Text("PDF").tag("pdf")
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, 24)

            if !downloadURL.isEmpty {
                Link("دانلود فایل", destination: URL(string: downloadURL)!)
                    .font(.headline)
                    .foregroundColor(.blue)
            }

            if !errorMessage.isEmpty {
                Text(errorMessage).font(.caption).foregroundColor(.red).padding(.horizontal)
            }

            Button {
                Task { await requestExport() }
            } label: {
                if exporting {
                    ProgressView().tint(.white)
                } else {
                    Text("درخواست خروجی")
                }
            }
            .disabled(exporting)
            .frame(maxWidth: .infinity)
            .padding()
            .background(Color.blue)
            .foregroundColor(.white)
            .cornerRadius(14)
            .padding(.horizontal, 24)

            Spacer()
        }
    }

    private func requestExport() async {
        exporting = true
        errorMessage = ""
        do {
            let resp: ExportDataResponse = try await api.post(
                "/v1/privacy/export",
                body: ExportDataRequest(format: format),
                token: token
            )
            downloadURL = resp.download_url
        } catch {
            errorMessage = error.localizedDescription
        }
        exporting = false
    }
}

// MARK: — Helpers

private func emptyView(icon: String, message: String) -> some View {
    VStack(spacing: 16) {
        Image(systemName: icon)
            .resizable().scaledToFit().frame(width: 50)
            .foregroundColor(Color.white.opacity(0.3))
        Text(message)
            .foregroundColor(Color.white.opacity(0.5))
    }
    .frame(maxWidth: .infinity, maxHeight: .infinity)
}
