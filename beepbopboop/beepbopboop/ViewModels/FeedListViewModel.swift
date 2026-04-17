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
    private static let scheduledPollingInterval: TimeInterval = 120

    /// Whether any post in the feed has a live game.
    var hasLiveGames: Bool {
        posts.contains { post in
            if let game = post.gameData { return game.isLive }
            return false
        }
    }

    /// Whether any post in the feed has a scheduled (upcoming) game.
    private var hasScheduledGames: Bool {
        posts.contains { post in
            if let game = post.gameData { return !game.isFinal && !game.isLive }
            return false
        }
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
    func restartPollingIfNeeded() {
        pollingTask?.cancel()
        pollingTask = nil

        if hasLiveGames {
            isPollingLive = true
            startPolling(interval: Self.livePollingInterval)
        } else if hasScheduledGames {
            isPollingLive = false
            startPolling(interval: Self.scheduledPollingInterval)
        } else {
            isPollingLive = false
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
