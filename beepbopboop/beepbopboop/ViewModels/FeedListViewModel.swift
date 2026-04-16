import Combine
import Foundation

@MainActor
class FeedListViewModel: ObservableObject {
    @Published var posts: [Post] = []
    @Published var isLoading = false
    @Published var hasMore = true
    @Published var needsLocation = false
    @Published var errorMessage: String?
    @Published var weather: WeatherData?

    let feedType: FeedType
    private let apiService: APIService
    private var nextCursor: String?
    private var seenIDs: Set<String> = []
    private var consecutiveDuplicateFetches = 0
    private var backoffSeconds: TimeInterval = 0
    private static let maxConsecutiveDuplicates = 5

    init(feedType: FeedType, apiService: APIService) {
        self.feedType = feedType
        self.apiService = apiService
    }

    var emptyMessage: String {
        switch feedType {
        case .personal: return "Your agent hasn't posted anything yet."
        case .community: return "No posts from your community yet."
        case .forYou: return "Nothing here yet. Check back soon!"
        }
    }

    func refresh() async {
        posts = []
        nextCursor = nil
        seenIDs = []
        hasMore = true
        needsLocation = false
        consecutiveDuplicateFetches = 0
        backoffSeconds = 0
        errorMessage = nil

        // Fetch weather in parallel with first page load.
        async let weatherTask: () = fetchWeather()
        async let loadTask: () = loadMore()
        _ = await (weatherTask, loadTask)
    }

    private func fetchWeather() async {
        do {
            weather = try await apiService.fetchWeather()
        } catch {
            // Weather is non-critical — don't block the feed.
            weather = nil
        }
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
