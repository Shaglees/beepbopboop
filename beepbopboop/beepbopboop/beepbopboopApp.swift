import EventKit
import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService = AuthService()
    @AppStorage("onboardingComplete") private var onboardingComplete = false
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var notificationService = NotificationService()
    @StateObject private var calendarService = CalendarService()

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
                .onChange(of: scenePhase) { _, phase in
                    if phase == .background {
                        Task { await tracker.flush() }
                    }
                    if phase == .active {
                        Task { await syncCalendarIfEnabled(api: api) }
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
                .task { await syncCalendarIfEnabled(api: api) }
            } else {
                LoginView(authService: authService)
            }
        }
    }

    private func syncCalendarIfEnabled(api: APIService) async {
        guard calendarService.authorizationStatus == .fullAccess else { return }
        let events = calendarService.fetchUpcomingEvents()
        guard !events.isEmpty else { return }
        try? await api.syncCalendarEvents(events)
    }
}
