import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService = AuthService()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            if authService.isSignedIn {
                let api = APIService(authService: authService)
                let tracker = EventTracker { events in
                    try? await api.postEventsBatch(events)
                }
                FeedView(
                    authService: authService,
                    apiService: api
                )
                .environmentObject(api)
                .environmentObject(tracker)
                .onChange(of: scenePhase) { _, phase in
                    if phase == .background {
                        Task { await tracker.flush() }
                    }
                }
            } else {
                LoginView(authService: authService)
            }
        }
    }
}
