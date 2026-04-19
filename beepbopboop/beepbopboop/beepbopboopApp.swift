import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService = AuthService()
    @StateObject private var calendarService = CalendarService()
    @AppStorage("onboardingComplete") private var onboardingComplete = false
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var notificationService = NotificationService()

    var body: some Scene {
        WindowGroup {
            if authService.isSignedIn {
                let api = APIService(authService: authService)
                let tracker = EventTracker { events in
                    try? await api.postEventsBatch(events)
                }
                FeedView(
                    authService: authService,
                    apiService: api,
                    notificationService: notificationService
                )
                .environmentObject(api)
                .environmentObject(tracker)
                .environmentObject(calendarService)
                .onChange(of: scenePhase) { _, phase in
                    if phase == .background {
                        Task { await tracker.flush() }
                    }
                    if phase == .active {
                        // Re-sync calendar context whenever the app comes to foreground
                        // so intent signals stay fresh without a separate background worker.
                        Task {
                            await syncCalendarIfAuthorised(api: api)
                        }
                    }
                }
                .task {
                    // Initial sync on first launch after sign-in.
                    await syncCalendarIfAuthorised(api: api)
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

    /// Syncs calendar events silently if the user has already granted calendar access.
    /// Does NOT prompt for permission — that happens from SettingsView.
    private func syncCalendarIfAuthorised(api: APIService) async {
        guard calendarService.hasAccess else { return }
        let events = calendarService.fetchUpcomingEvents(days: 7)
        await api.syncCalendarContext(events)
    }
}
