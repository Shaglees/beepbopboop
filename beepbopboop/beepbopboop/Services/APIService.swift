import Foundation

enum FeedType {
    case forYou, community, personal

    var path: String {
        switch self {
        case .forYou: return "/feeds/foryou"
        case .community: return "/feeds/community"
        case .personal: return "/feeds/personal"
        }
    }
}

class APIService {
    private let baseURL: String
    private let authService: AuthService

    init(baseURL: String = Config.backendBaseURL, authService: AuthService) {
        self.baseURL = baseURL
        self.authService = authService
    }

    // MARK: - Legacy feed (backward compat)

    @MainActor
    func fetchFeed() async throws -> [Post] {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/feed") else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }

        let decoder = JSONDecoder()
        let safePosts = try decoder.decode([SafeDecodable<Post>].self, from: data)
        return safePosts.compactMap { $0.value }
    }

    // MARK: - Multi-feed with pagination

    @MainActor
    func fetchFeed(type feedType: FeedType, cursor: String? = nil, limit: Int = 20) async throws -> FeedResponse {
        let token = authService.getToken()
        var components = URLComponents(string: "\(baseURL)\(feedType.path)")
        var queryItems: [URLQueryItem] = []
        if let cursor = cursor {
            queryItems.append(URLQueryItem(name: "cursor", value: cursor))
        }
        queryItems.append(URLQueryItem(name: "limit", value: String(limit)))
        components?.queryItems = queryItems

        guard let url = components?.url else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 422 {
            throw APIError.locationRequired
        }

        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }

        return try JSONDecoder().decode(FeedResponse.self, from: data)
    }

    // MARK: - User Settings

    @MainActor
    func getSettings() async throws -> UserSettings {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/user/settings") else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }
        return try JSONDecoder().decode(UserSettings.self, from: data)
    }

    @MainActor
    func updateSettings(_ settings: UserSettings) async throws -> UserSettings {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/user/settings") else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(settings)

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }
        return try JSONDecoder().decode(UserSettings.self, from: data)
    }

    enum APIError: LocalizedError {
        case invalidURL
        case invalidResponse
        case httpError(Int)
        case locationRequired

        var errorDescription: String? {
            switch self {
            case .invalidURL: return "Invalid backend URL"
            case .invalidResponse: return "Invalid server response"
            case .httpError(let code): return "Server error: \(code)"
            case .locationRequired: return "Location required"
            }
        }
    }
}
