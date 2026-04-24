import EventKit
import SwiftUI

@main
struct beepbopboopApp: App {
    @StateObject private var authService: AuthService
    @StateObject private var notificationService: NotificationService
    @StateObject private var api: APIService
    @StateObject private var tracker: EventTracker
    @StateObject private var calendarService: CalendarService
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
        _calendarService = StateObject(wrappedValue: CalendarService())
    }

    var body: some Scene {
        WindowGroup {
            if authService.isSignedIn {
                FeedView(
                    authService: authService,
                    apiService: api,
                    notificationService: notificationService,
                    calendarService: calendarService
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
                .task(id: authService.isSignedIn) {
                    guard authService.isSignedIn, !onboardingComplete else { return }
                    if let profile = try? await api.getProfile(),
                       profile.profileInitialized {
                        onboardingComplete = true
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

    private func syncCalendarIfEnabled(api: APIService) async {
        guard calendarService.authorizationStatus == .fullAccess else { return }
        let events = calendarService.fetchUpcomingEvents()
        guard !events.isEmpty else { return }
        try? await api.syncCalendarEvents(events)
    }
}
