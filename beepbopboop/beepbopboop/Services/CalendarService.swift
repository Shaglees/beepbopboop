import Combine
import EventKit
import Foundation

/// Reads upcoming calendar events using EventKit and extracts the minimal
/// data needed for anticipatory feed personalisation. Raw event text is
/// never sent beyond the structured payload the server extracts intents from.
@MainActor
class CalendarService: ObservableObject {
    private let store = EKEventStore()

    @Published var authorizationStatus: EKAuthorizationStatus = .notDetermined

    init() {
        authorizationStatus = EKEventStore.authorizationStatus(for: .event)
    }

    // MARK: - Authorization

    /// Requests calendar read access. Returns true if access was granted.
    func requestAccess() async -> Bool {
        if #available(iOS 17.0, *) {
            do {
                let granted = try await store.requestFullAccessToEvents()
                authorizationStatus = EKEventStore.authorizationStatus(for: .event)
                return granted
            } catch {
                return false
            }
        } else {
            return await withCheckedContinuation { continuation in
                store.requestAccess(to: .event) { [weak self] granted, _ in
                    Task { @MainActor in
                        self?.authorizationStatus = EKEventStore.authorizationStatus(for: .event)
                        continuation.resume(returning: granted)
                    }
                }
            }
        }
    }

    var hasAccess: Bool {
        switch authorizationStatus {
        case .authorized, .fullAccess, .writeOnly:
            return true
        default:
            return false
        }
    }

    // MARK: - Event Fetching

    /// Fetches upcoming events in the next `days` days from all calendars.
    func fetchUpcomingEvents(days: Int = 7) -> [CalendarEventPayload] {
        guard hasAccess else { return [] }

        let now = Date()
        guard let end = Calendar.current.date(byAdding: .day, value: days, to: now) else {
            return []
        }

        let predicate = store.predicateForEvents(withStart: now, end: end, calendars: nil)
        let ekEvents = store.events(matching: predicate)

        return ekEvents.compactMap { event in
            guard let start = event.startDate, let end = event.endDate else { return nil }
            // Skip all-day events that are longer than 3 days (recurring birthdays etc.)
            if event.isAllDay && end.timeIntervalSince(start) > 3 * 24 * 3600 { return nil }

            return CalendarEventPayload(
                title: event.title ?? "",
                startTime: start,
                endTime: end,
                location: event.location ?? "",
                notes: "" // deliberately omit notes to minimise data sent
            )
        }
    }
}

// MARK: - Payload types

/// Minimal event data sent to POST /user/calendar-context.
struct CalendarEventPayload: Codable {
    let title: String
    let startTime: Date
    let endTime: Date
    let location: String
    let notes: String

    enum CodingKeys: String, CodingKey {
        case title
        case startTime = "start_time"
        case endTime = "end_time"
        case location
        case notes
    }
}

struct CalendarContextRequest: Codable {
    let events: [CalendarEventPayload]
}

struct CalendarContextResponse: Codable {
    let intentsExtracted: Int

    enum CodingKeys: String, CodingKey {
        case intentsExtracted = "intents_extracted"
    }
}

struct UserIntent: Codable, Identifiable {
    let id: String
    let intentType: String
    let signalType: String
    let activeFrom: Date
    let activeUntil: Date

    enum CodingKeys: String, CodingKey {
        case id
        case intentType = "intent_type"
        case signalType = "signal_type"
        case activeFrom = "active_from"
        case activeUntil = "active_until"
    }
}
