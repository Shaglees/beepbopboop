import Foundation

enum Config {
    static let backendBaseURL: String = {
        ProcessInfo.processInfo.environment["BACKEND_URL"] ?? "http://localhost:8080"
    }()
}
