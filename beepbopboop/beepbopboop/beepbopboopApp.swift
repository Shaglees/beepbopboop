import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService = AuthService()
    @StateObject private var notificationService = NotificationService()

    var body: some Scene {
        WindowGroup {
            if authService.isSignedIn {
                let api = APIService(authService: authService)
                FeedView(
                    authService: authService,
                    apiService: api,
                    notificationService: notificationService
                )
                .environmentObject(api)
            } else {
                LoginView(authService: authService)
            }
        }
    }
}
