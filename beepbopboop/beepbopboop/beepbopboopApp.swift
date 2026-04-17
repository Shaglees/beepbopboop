import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService = AuthService()

    var body: some Scene {
        WindowGroup {
            if authService.isSignedIn {
                let api = APIService(authService: authService)
                FeedView(
                    authService: authService,
                    apiService: api
                )
                .environmentObject(api)
            } else {
                LoginView(authService: authService)
            }
        }
    }
}
