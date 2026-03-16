import Foundation

class APIService {
    private let baseURL: String
    private let authService: AuthService

    init(baseURL: String = Config.backendBaseURL, authService: AuthService) {
        self.baseURL = baseURL
        self.authService = authService
    }

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
        return try decoder.decode([Post].self, from: data)
    }

    enum APIError: LocalizedError {
        case invalidURL
        case invalidResponse
        case httpError(Int)

        var errorDescription: String? {
            switch self {
            case .invalidURL: return "Invalid backend URL"
            case .invalidResponse: return "Invalid server response"
            case .httpError(let code): return "Server error: \(code)"
            }
        }
    }
}
