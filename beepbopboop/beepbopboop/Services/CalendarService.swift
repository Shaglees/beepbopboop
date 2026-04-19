import Combine
import EventKit
import Foundation

struct CalendarEventPayload: Encodable {
    let id: String
    let title: String
    let startTime: String
    let endTime: String?
    let location: String?
    let notes: String?

    enum CodingKeys: String, CodingKey {
        case id, title, location, notes
        case startTime = "start_time"
        case endTime = "end_time"
    }
}

@MainActor
class CalendarService: ObservableObject {
    @Published var authorizationStatus: EKAuthorizationStatus = .notDetermined

    private let store = EKEventStore()
    private let iso8601 = ISO8601DateFormatter()

    init() {
        authorizationStatus = EKEventStore.authorizationStatus(for: .event)
    }

    func requestAccess() async -> Bool {
        do {
            let granted = try await store.requestFullAccessToEvents()
            authorizationStatus = EKEventStore.authorizationStatus(for: .event)
            return granted
        } catch {
            authorizationStatus = EKEventStore.authorizationStatus(for: .event)
            return false
        }
    }

    func fetchUpcomingEvents(days: Int = 14) -> [CalendarEventPayload] {
        guard authorizationStatus == .fullAccess else { return [] }

        let now = Date()
        let end = Calendar.current.date(byAdding: .day, value: days, to: now) ?? now

        let predicate = store.predicateForEvents(withStart: now, end: end, calendars: nil)
        let events = store.events(matching: predicate)

        return events.compactMap { event in
            guard !event.isAllDay && event.title != nil else { return nil }
            return CalendarEventPayload(
                id: event.eventIdentifier,
                title: event.title ?? "(No title)",
                startTime: iso8601.string(from: event.startDate),
                endTime: event.endDate.map { iso8601.string(from: $0) },
                location: event.location,
                notes: event.notes
            )
        }
    }
}
