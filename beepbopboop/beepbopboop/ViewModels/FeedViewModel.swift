import Combine
import Foundation

@MainActor
class FeedViewModel: ObservableObject {
    @Published var posts: [Post] = []
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?

    private let apiService: APIService

    init(apiService: APIService) {
        self.apiService = apiService
    }

    func loadFeed() async {
        isLoading = true
        errorMessage = nil
        do {
            posts = try await apiService.fetchFeed()
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }
}
