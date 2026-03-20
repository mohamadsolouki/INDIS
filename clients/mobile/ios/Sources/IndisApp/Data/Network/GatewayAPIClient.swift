import Foundation

/// Errors thrown by GatewayAPIClient.
enum GatewayError: LocalizedError {
    case httpError(Int, String)
    case decodingError(String)
    case networkUnavailable

    var errorDescription: String? {
        switch self {
        case .httpError(let code, let body): return "HTTP \(code): \(body)"
        case .decodingError(let msg):        return "Decode error: \(msg)"
        case .networkUnavailable:            return "شبکه در دسترس نیست"
        }
    }
}

/// URLSession-based async HTTP client for all INDIS gateway calls.
///
/// Auth: every request includes `Authorization: Bearer <token>` when a token is provided.
/// Mirrors the Android GatewayApiClient (OkHttp) in behaviour and API surface.
actor GatewayAPIClient {

    private let baseURL: String
    private let session: URLSession

    init(baseURL: String = "http://localhost:8080") {
        self.baseURL = baseURL.hasSuffix("/") ? String(baseURL.dropLast()) : baseURL
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest  = 30
        config.timeoutIntervalForResource = 60
        self.session = URLSession(configuration: config)
    }

    // MARK: — Core methods

    /// HTTP GET. Returns decoded Decodable or throws GatewayError.
    func get<T: Decodable>(_ path: String, token: String, as type: T.Type = T.self) async throws -> T {
        let request = buildRequest(path: path, method: "GET", body: nil, token: token)
        return try await execute(request)
    }

    /// HTTP POST with Encodable body. Returns decoded Decodable or throws GatewayError.
    func post<B: Encodable, T: Decodable>(_ path: String, body: B, token: String, as type: T.Type = T.self) async throws -> T {
        let data = try JSONEncoder().encode(body)
        let request = buildRequest(path: path, method: "POST", body: data, token: token)
        return try await execute(request)
    }

    /// HTTP PUT with Encodable body.
    func put<B: Encodable, T: Decodable>(_ path: String, body: B, token: String, as type: T.Type = T.self) async throws -> T {
        let data = try JSONEncoder().encode(body)
        let request = buildRequest(path: path, method: "PUT", body: data, token: token)
        return try await execute(request)
    }

    // MARK: — Private helpers

    private func buildRequest(path: String, method: String, body: Data?, token: String) -> URLRequest {
        let url = URL(string: "\(baseURL)/\(path.hasPrefix("/") ? String(path.dropFirst()) : path)")!
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if !token.isEmpty {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        req.httpBody = body
        return req
    }

    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        do {
            let (data, response) = try await session.data(for: request)
            guard let http = response as? HTTPURLResponse else {
                throw GatewayError.networkUnavailable
            }
            guard (200..<300).contains(http.statusCode) else {
                let body = String(data: data, encoding: .utf8) ?? ""
                throw GatewayError.httpError(http.statusCode, body)
            }
            do {
                return try JSONDecoder().decode(T.self, from: data)
            } catch {
                throw GatewayError.decodingError(error.localizedDescription)
            }
        } catch let error as GatewayError {
            throw error
        } catch {
            throw GatewayError.networkUnavailable
        }
    }
}
