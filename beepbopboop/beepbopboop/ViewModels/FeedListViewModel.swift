import Combine
import Foundation

@MainActor
class FeedListViewModel: ObservableObject {
    @Published var posts: [Post] = []
    @Published var isLoading = false
    @Published var hasMore = true
    @Published var needsLocation = false
    @Published var errorMessage: String?
    @Published var isPollingLive = false
    let feedType: FeedType
    private let apiService: APIService
    private var nextCursor: String?
    private var seenIDs: Set<String> = []
    private var consecutiveDuplicateFetches = 0
    private var backoffSeconds: TimeInterval = 0
    private static let maxConsecutiveDuplicates = 5

    // MARK: - Live Score Polling

    private var pollingTask: Task<Void, Never>?
    private static let livePollingInterval: TimeInterval = 30
    private static let nearGamePollingInterval: TimeInterval = 30
    private static let approachingGamePollingInterval: TimeInterval = 120

    /// Time before game start to begin fast polling (15 minutes).
    private static let pregameWindow: TimeInterval = 15 * 60
    /// Time before game start to begin slow polling (2 hours).
    private static let approachingWindow: TimeInterval = 2 * 60 * 60

    /// Whether any post in the feed has a live game.
    var hasLiveGames: Bool {
        posts.contains { post in
            if let game = post.gameData { return game.isLive }
            return false
        }
    }

    /// Earliest upcoming game date across all posts in the feed.
    private var earliestScheduledGameDate: Date? {
        let now = Date()
        return posts.compactMap { post -> Date? in
            guard let game = post.gameData,
                  !game.isFinal && !game.isLive,
                  let date = game.gameDate,
                  date > now else { return nil }
            return date
        }.min()
    }

    /// Whether any post in the feed has a scheduled (upcoming) game.
    private var hasScheduledGames: Bool {
        earliestScheduledGameDate != nil
    }

    init(feedType: FeedType, apiService: APIService) {
        self.feedType = feedType
        self.apiService = apiService
    }

    deinit {
        pollingTask?.cancel()
    }

    var emptyMessage: String {
        switch feedType {
        case .personal: return "Your agent hasn't posted anything yet."
        case .community: return "No posts from your community yet."
        case .forYou: return "Nothing here yet. Check back soon!"
        case .saved: return "Nothing saved yet — tap the bookmark icon on any post."
        }
    }

    // MARK: - Feed Loading

    func refresh() async {
        posts = []
        nextCursor = nil
        seenIDs = []
        hasMore = true
        needsLocation = false
        consecutiveDuplicateFetches = 0
        backoffSeconds = 0
        errorMessage = nil

        await loadMore()

        // After initial load, start polling if there are live/upcoming games
        restartPollingIfNeeded()
    }

    func loadMore() async {
        guard !isLoading, hasMore else { return }
        guard consecutiveDuplicateFetches < Self.maxConsecutiveDuplicates else {
            hasMore = false
            return
        }

        if backoffSeconds > 0 {
            try? await Task.sleep(nanoseconds: UInt64(backoffSeconds * 1_000_000_000))
        }

        isLoading = true
        errorMessage = nil

        do {
            let response = try await apiService.fetchFeed(type: feedType, cursor: nextCursor)

            var newPosts: [Post] = []
            for post in response.posts {
                if !seenIDs.contains(post.id) {
                    seenIDs.insert(post.id)
                    newPosts.append(post)
                }
            }

            // Cap seenIDs
            if seenIDs.count > 2000 {
                seenIDs = Set(seenIDs.suffix(2000))
            }

            if newPosts.isEmpty && !response.posts.isEmpty {
                consecutiveDuplicateFetches += 1
                backoffSeconds = min(backoffSeconds == 0 ? 30 : backoffSeconds * 2, 300)
            } else {
                consecutiveDuplicateFetches = 0
                backoffSeconds = 0
            }

            posts.append(contentsOf: newPosts)

            if let cursor = response.nextCursor {
                nextCursor = cursor
            } else {
                // End of feed — loop from top
                nextCursor = nil
            }

            if response.posts.count == 0 {
                hasMore = false
            }

            needsLocation = false
        } catch let error as APIService.APIError where error == .locationRequired {
            needsLocation = true
            hasMore = false
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func shouldLoadMore(currentPost: Post) -> Bool {
        guard let index = posts.firstIndex(where: { $0.id == currentPost.id }) else { return false }
        return index >= posts.count - 3
    }

    // MARK: - Live Score Polling

    /// Start or restart polling based on current feed state.
    ///
    /// Polling tiers (from most to least urgent):
    /// 1. **Live game** → poll every 30s for score updates.
    /// 2. **Game starting within 15 min** → poll every 30s (about to go live).
    /// 3. **Game starting within 2 hours** → poll every 2 min.
    /// 4. **Game more than 2 hours away** → sleep until 15 min before the
    ///    earliest game, then re-evaluate (no wasted network requests).
    /// 5. **All games final / no sports posts** → stop polling entirely.
    func restartPollingIfNeeded() {
        pollingTask?.cancel()
        pollingTask = nil

        if hasLiveGames {
            // Tier 1: live game — fast polling
            isPollingLive = true
            startPolling(interval: Self.livePollingInterval)
            return
        }

        guard let earliest = earliestScheduledGameDate else {
            // Tier 5: nothing upcoming — stop
            isPollingLive = false
            return
        }

        let timeUntilGame = earliest.timeIntervalSince(Date())

        if timeUntilGame <= Self.pregameWindow {
            // Tier 2: game imminent — fast polling (it may flip to "Live" any moment)
            isPollingLive = false
            startPolling(interval: Self.nearGamePollingInterval)
        } else if timeUntilGame <= Self.approachingWindow {
            // Tier 3: game within 2 hours — moderate polling
            isPollingLive = false
            startPolling(interval: Self.approachingGamePollingInterval)
        } else {
            // Tier 4: game is far away — sleep until the pregame window, then re-evaluate
            isPollingLive = false
            let sleepUntil = timeUntilGame - Self.pregameWindow
            scheduleWakeUp(after: sleepUntil)
        }
    }

    /// Stop polling (e.g. when view disappears).
    func stopPolling() {
        pollingTask?.cancel()
        pollingTask = nil
        isPollingLive = false
    }

    private func startPolling(interval: TimeInterval) {
        pollingTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: UInt64(interval * 1_000_000_000))
                guard !Task.isCancelled else { break }
                await self?.refreshSportsPosts()
            }
        }
    }

    /// Sleep for `delay` seconds, then do a single refresh and re-evaluate polling tier.
    /// Used when the next game is far away — avoids polling every 2 min for hours.
    private func scheduleWakeUp(after delay: TimeInterval) {
        pollingTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
            guard !Task.isCancelled else { return }
            await self?.refreshSportsPosts()
        }
    }

    /// Refresh only the feed to pick up updated sports scores.
    /// Re-fetches the first page and merges updated posts into the existing feed.
    private func refreshSportsPosts() async {
        do {
            let response = try await apiService.fetchFeed(type: feedType, cursor: nil, limit: 20)

            // Build a lookup of fresh posts by ID
            var freshByID: [String: Post] = [:]
            for post in response.posts {
                freshByID[post.id] = post
            }

            // Update existing posts in-place where we have fresh data
            var updated = false
            for i in posts.indices {
                if let fresh = freshByID[posts[i].id] {
                    // Only update if the post data actually changed
                    if posts[i].externalURL != fresh.externalURL
                        || posts[i].title != fresh.title
                        || posts[i].body != fresh.body {
                        posts[i] = fresh
                        updated = true
                    }
                }
            }

            // Also insert any brand-new posts we haven't seen
            for post in response.posts {
                if !seenIDs.contains(post.id) {
                    seenIDs.insert(post.id)
                    posts.insert(post, at: 0)
                    updated = true
                }
            }

            // Adjust polling frequency based on updated game states
            if updated {
                restartPollingIfNeeded()
            }
        } catch {
            // Silent failure for background polling — don't disrupt the UI
        }
    }
}

extension APIService.APIError: Equatable {
    static func == (lhs: APIService.APIError, rhs: APIService.APIError) -> Bool {
        switch (lhs, rhs) {
        case (.invalidURL, .invalidURL): return true
        case (.invalidResponse, .invalidResponse): return true
        case (.locationRequired, .locationRequired): return true
        case (.httpError(let a), .httpError(let b)): return a == b
        default: return false
        }
    }
}
