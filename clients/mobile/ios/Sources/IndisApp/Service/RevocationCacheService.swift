import Foundation
import BackgroundTasks

/// Background revocation list cache service.
///
/// Fetches GET /v1/credential/revocations every 6 hours (when network is available)
/// and stores the result in UserDefaults. The cached list is valid for 72 hours,
/// enabling offline ZK proof generation without a live network connection.
///
/// Uses BGAppRefreshTask (iOS 13+) for background refresh.
/// The BGTaskScheduler identifier "org.indis.app.revocation-refresh" must be
/// registered in Info.plist under BGTaskSchedulerPermittedIdentifiers.
///
/// PRD FR-006: revocation list must be cached for 72h offline operation.
final class RevocationCacheService {

    static let shared = RevocationCacheService()

    private let taskIdentifier = "org.indis.app.revocation-refresh"
    private let cacheKey       = "indis.revocation_list_json"
    private let fetchedAtKey   = "indis.revocation_list_fetched_at"
    private let maxAgeSeconds: TimeInterval = 72 * 60 * 60   // 72 hours
    private let refreshInterval: TimeInterval = 6 * 60 * 60  // 6 hours

    private init() {}

    // MARK: — Public API

    /// Call once from AppState.init() to register the background task handler
    /// and schedule the first refresh.
    func scheduleIfNeeded() {
        registerBackgroundTask()
        scheduleAppRefresh()
        // Also do an eager refresh if the cache is stale or absent.
        if cachedRevocations() == nil {
            Task { await refresh() }
        }
    }

    /// Returns the cached revocation list JSON, or nil if absent / older than 72h.
    func cachedRevocations() -> [String]? {
        let defaults = UserDefaults.standard
        let fetchedAt = defaults.double(forKey: fetchedAtKey)
        guard fetchedAt > 0,
              Date().timeIntervalSince1970 - fetchedAt < maxAgeSeconds,
              let json = defaults.string(forKey: cacheKey),
              let data = json.data(using: .utf8),
              let resp = try? JSONDecoder().decode(RevocationListResponse.self, from: data)
        else { return nil }
        return resp.revoked_credential_ids
    }

    // MARK: — Background task

    private func registerBackgroundTask() {
        BGTaskScheduler.shared.register(forTaskWithIdentifier: taskIdentifier, using: nil) { [weak self] task in
            self?.handleAppRefresh(task: task as! BGAppRefreshTask)
        }
    }

    private func handleAppRefresh(task: BGAppRefreshTask) {
        scheduleAppRefresh()   // Re-schedule for next time.
        let operation = Task { await refresh() }
        task.expirationHandler = { operation.cancel() }
        Task {
            await operation.value
            task.setTaskCompleted(success: true)
        }
    }

    private func scheduleAppRefresh() {
        let request = BGAppRefreshTaskRequest(identifier: taskIdentifier)
        request.earliestBeginDate = Date(timeIntervalSinceNow: refreshInterval)
        try? BGTaskScheduler.shared.submit(request)
    }

    // MARK: — Fetch

    @discardableResult
    func refresh() async -> Bool {
        let store = EncryptedWalletStore.shared
        let gatewayURL = store.get(forKey: .gatewayURL) ?? "http://localhost:8080"
        let token      = store.get(forKey: .jwtToken)   ?? ""

        do {
            let api: GatewayAPIClient = GatewayAPIClient(baseURL: gatewayURL)
            let resp: RevocationListResponse = try await api.get("/v1/credential/revocations", token: token)
            let json = (try? String(data: JSONEncoder().encode(resp), encoding: .utf8)) ?? "[]"
            UserDefaults.standard.set(json, forKey: cacheKey)
            UserDefaults.standard.set(Date().timeIntervalSince1970, forKey: fetchedAtKey)
            return true
        } catch {
            return false   // Keep stale cache in place.
        }
    }
}
