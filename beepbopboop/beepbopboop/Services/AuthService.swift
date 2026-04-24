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

    @Published var profileInitialized: Bool = false
    @Published var isLoadingProfile: Bool = false

    func checkProfile(apiService: APIService) async {
        isLoadingProfile = true
        defer { isLoadingProfile = false }
        do {
            let profile = try await apiService.getProfile()
            profileInitialized = profile.profileInitialized
        } catch {
            profileInitialized = false
        }
    }
}
