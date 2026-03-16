import Combine
import Foundation

@MainActor
class AuthService: ObservableObject {
    @Published var isSignedIn: Bool = false
    @Published var userIdentifier: String = ""

    func signIn(identifier: String) {
        userIdentifier = identifier
        isSignedIn = true
    }

    func signOut() {
        userIdentifier = ""
        isSignedIn = false
    }

    /// Returns the token for API calls.
    /// In dev mode, this is just the user identifier string.
    func getToken() -> String {
        return userIdentifier
    }
}
