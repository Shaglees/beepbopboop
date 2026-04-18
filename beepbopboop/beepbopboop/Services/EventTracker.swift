import Combine
import Foundation

class EventTracker: ObservableObject {
    let objectWillChange = ObservableObjectPublisher()

    struct PendingEvent: Codable, Equatable {
        let postID: String
        let eventType: String
        let dwellMs: Int

        enum CodingKeys: String, CodingKey {
            case postID = "post_id"
            case eventType = "event_type"
            case dwellMs = "dwell_ms"
        }
    }

    private(set) var buffer: [PendingEvent] = []
    private var timers: [String: Date] = [:]
    private var viewedInSession: Set<String> = []
    private var viewTasks: [String: Task<Void, Never>] = [:]

    let flushThreshold: Int
    private let onFlush: @MainActor ([PendingEvent]) async -> Void

    private static let minDwellMs = 500
    private static let viewThresholdMs = 1000
    private static let dwellThresholdMs = 3000

    init(flushThreshold: Int = 10, onFlush: @escaping @MainActor ([PendingEvent]) async -> Void) {
        self.flushThreshold = flushThreshold
        self.onFlush = onFlush
    }

    @MainActor func cardAppeared(postID: String) {
        timers[postID] = Date()

        guard !viewedInSession.contains(postID) else { return }

        let task = Task { @MainActor [weak self] in
            do {
                try await Task.sleep(for: .milliseconds(Self.viewThresholdMs))
            } catch {
                return
            }
            guard let self, self.timers[postID] != nil else { return }
            self.viewedInSession.insert(postID)
            self.enqueue(PendingEvent(postID: postID, eventType: "view", dwellMs: Self.viewThresholdMs))
        }
        viewTasks[postID] = task
    }

    @MainActor func cardDisappeared(postID: String) {
        viewTasks[postID]?.cancel()
        viewTasks[postID] = nil

        guard let start = timers.removeValue(forKey: postID) else { return }
        let dwellMs = Int(Date().timeIntervalSince(start) * 1000)

        guard dwellMs >= Self.minDwellMs else { return }
        if dwellMs >= Self.dwellThresholdMs {
            enqueue(PendingEvent(postID: postID, eventType: "dwell", dwellMs: dwellMs))
        }
    }

    @MainActor func fireEvent(postID: String, type: String) {
        enqueue(PendingEvent(postID: postID, eventType: type, dwellMs: 0))
    }

    @MainActor func flush() async {
        guard !buffer.isEmpty else { return }
        let events = buffer
        buffer.removeAll()
        await onFlush(events)
    }

    @MainActor private func enqueue(_ event: PendingEvent) {
        buffer.append(event)
        if buffer.count >= flushThreshold {
            Task { await flush() }
        }
    }
}
