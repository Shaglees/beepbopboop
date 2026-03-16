import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService = AuthService()

    var body: some Scene {
        WindowGroup {
            if authService.isSignedIn {
                FeedView(
                    authService: authService,
                    apiService: APIService(authService: authService)
                )
            } else {
                LoginView(authService: authService)
            }
        }
    }
}
