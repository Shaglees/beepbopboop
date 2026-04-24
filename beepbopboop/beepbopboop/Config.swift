import Foundation

enum Config {
    static let backendBaseURL: String = {
        ProcessInfo.processInfo.environment["BACKEND_URL"] ?? "http://192.168.2.52:8080"
    }()
}
