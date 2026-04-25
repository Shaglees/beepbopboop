import Combine
import Foundation

@MainActor
class ContentMixViewModel: ObservableObject {
    @Published var targets: [String: Double] = [:]
    @Published var omega: String = ""
    @Published var pinned: Set<String> = []
    @Published var autoAdjust: Bool = true
    @Published var actual30d: [String: Double] = [:]
    @Published var status: [String: String] = [:]
    @Published var isLoading = false
    @Published var error: String?

    private let apiService: APIService
    private var saveTask: Task<Void, Never>?

    static let verticalInfo: [(key: String, emoji: String, name: String)] = [
        ("sports", "🏀", "Sports"),
        ("food", "🍕", "Food"),
        ("music", "🎵", "Music"),
        ("travel", "✈️", "Travel"),
        ("science", "🔬", "Science"),
        ("gaming", "🎮", "Gaming"),
        ("creators", "🎨", "Creators"),
        ("fashion", "👗", "Fashion"),
        ("movies", "🎬", "Movies"),
        ("pets", "🐾", "Pets"),
        ("news", "📰", "News"),
    ]

    static let verticalColors: [String: String] = [
        "sports": "#4CAF50", "food": "#FF9800", "music": "#2196F3",
        "travel": "#9C27B0", "science": "#F44336", "gaming": "#00BCD4",
        "creators": "#795548", "fashion": "#E91E63", "movies": "#607D8B",
        "pets": "#8BC34A", "news": "#FF5722",
    ]

    init(apiService: APIService) {
        self.apiService = apiService
    }

    func load() async {
        isLoading = true
        error = nil
        do {
            let spread = try await apiService.fetchSpreadTargets()
            targets = spread.targets
            omega = spread.omega
            pinned = Set(spread.pinned)
            autoAdjust = spread.autoAdjust
            actual30d = spread.actual30d
            status = spread.status
        } catch {
            self.error = "Failed to load content mix"
        }
        isLoading = false
    }

    func scheduleSave() {
        saveTask?.cancel()
        saveTask = Task {
            try? await Task.sleep(nanoseconds: 500_000_000)
            guard !Task.isCancelled else { return }
            await save()
        }
    }

    func save() async {
        let req = PutSpreadRequest(
            targets: targets,
            omega: omega,
            pinned: Array(pinned),
            autoAdjust: autoAdjust
        )
        do {
            try await apiService.updateSpreadTargets(req)
        } catch {
            self.error = "Failed to save"
        }
    }

    func togglePin(_ vertical: String) {
        let wasPinned = pinned.contains(vertical)
        if wasPinned {
            pinned.remove(vertical)
        } else {
            pinned.insert(vertical)
        }
        Task {
            let req = PutSpreadRequest(
                targets: targets,
                omega: omega,
                pinned: Array(pinned),
                autoAdjust: autoAdjust
            )
            do {
                try await apiService.updateSpreadTargets(req)
            } catch {
                // Rollback on failure
                if wasPinned {
                    pinned.insert(vertical)
                } else {
                    pinned.remove(vertical)
                }
                self.error = "Failed to save"
            }
        }
    }

    func updateWeight(_ vertical: String, newWeight: Double) {
        let oldWeight = targets[vertical] ?? 0
        let diff = newWeight - oldWeight
        targets[vertical] = newWeight

        // Re-normalize non-pinned, non-changed verticals proportionally.
        let adjustable = targets.keys.filter { $0 != vertical && !pinned.contains($0) }
        let adjustableSum = adjustable.reduce(0.0) { $0 + (targets[$1] ?? 0) }

        if adjustableSum > 0 {
            for key in adjustable {
                let proportion = (targets[key] ?? 0) / adjustableSum
                targets[key] = max(0, (targets[key] ?? 0) - diff * proportion)
            }
        }

        // Normalize to exactly 1.0.
        let total = targets.values.reduce(0, +)
        if total > 0 {
            for key in targets.keys {
                targets[key] = (targets[key] ?? 0) / total
            }
        }
    }
}
