import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService: AuthService
    @StateObject private var notificationService: NotificationService
    @StateObject private var api: APIService
    @StateObject private var tracker: EventTracker
    @AppStorage("onboardingComplete") private var onboardingComplete = false
    @Environment(\.scenePhase) private var scenePhase

    init() {
        let auth = AuthService()
        let apiSvc = APIService(authService: auth)
        _authService = StateObject(wrappedValue: auth)
        _notificationService = StateObject(wrappedValue: NotificationService())
        _api = StateObject(wrappedValue: apiSvc)
        _tracker = StateObject(wrappedValue: EventTracker { events in
            try? await apiSvc.postEventsBatch(events)
        })
    }

    var body: some Scene {
        WindowGroup {
            if authService.isSignedIn {
                FeedView(
                    authService: authService,
                    apiService: api,
                    notificationService: notificationService
                )
                .environmentObject(api)
                .environmentObject(tracker)
                .onChange(of: scenePhase) { _, phase in
                    if phase == .background {
                        Task { await tracker.flush() }
                    }
                }
                .fullScreenCover(isPresented: Binding(
                    get: { !onboardingComplete },
                    set: { if !$0 { onboardingComplete = true } }
                )) {
                    OnboardingView(apiService: api) {
                        onboardingComplete = true
                    }
                }
            } else {
                LoginView(authService: authService)
            }
        }
    }
}
