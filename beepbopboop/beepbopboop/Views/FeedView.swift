import SwiftUI

struct FeedView: View {
    @StateObject private var viewModel: FeedViewModel
    private let authService: AuthService

    init(authService: AuthService, apiService: APIService) {
        self.authService = authService
        _viewModel = StateObject(wrappedValue: FeedViewModel(apiService: apiService))
    }

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.isLoading && viewModel.posts.isEmpty {
                    ProgressView("Loading feed...")
                } else if let error = viewModel.errorMessage, viewModel.posts.isEmpty {
                    VStack(spacing: 12) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.largeTitle)
                            .foregroundColor(.orange)
                        Text(error)
                            .multilineTextAlignment(.center)
                        Button("Retry") { Task { await viewModel.loadFeed() } }
                            .buttonStyle(.bordered)
                    }
                    .padding()
                } else if viewModel.posts.isEmpty {
                    VStack(spacing: 12) {
                        Image(systemName: "tray")
                            .font(.largeTitle)
                            .foregroundColor(.secondary)
                        Text("No posts yet")
                            .foregroundColor(.secondary)
                        Text("Your agent hasn't posted anything yet.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                } else {
                    List(viewModel.posts) { post in
                        NavigationLink(destination: PostDetailView(post: post)) {
                            FeedItemView(post: post)
                        }
                        .listRowSeparator(.hidden)
                        .listRowInsets(EdgeInsets(top: 4, leading: 16, bottom: 4, trailing: 16))
                    }
                    .listStyle(.plain)
                    .refreshable { await viewModel.loadFeed() }
                }
            }
            .navigationTitle("BeepBopBoop")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {
                        authService.signOut()
                    }) {
                        Text("Sign Out")
                    }
                }
            }
            .task { await viewModel.loadFeed() }
        }
    }
}
