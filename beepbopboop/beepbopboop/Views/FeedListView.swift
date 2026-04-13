import SwiftUI

struct FeedListView: View {
    @ObservedObject var viewModel: FeedListViewModel
    var onSettingsTapped: () -> Void

    var body: some View {
        Group {
            if viewModel.needsLocation {
                locationGateView
            } else if viewModel.isLoading && viewModel.posts.isEmpty {
                ProgressView("Loading feed...")
            } else if let error = viewModel.errorMessage, viewModel.posts.isEmpty {
                errorView(error)
            } else if viewModel.posts.isEmpty && !viewModel.isLoading {
                emptyView
            } else {
                feedList
            }
        }
    }

    // MARK: - Subviews

    private var feedList: some View {
        List {
            ForEach(viewModel.posts) { post in
                NavigationLink(destination: PostDetailView(post: post)) {
                    FeedItemView(post: post)
                }
                .listRowSeparator(.visible)
                .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 0))
                .onAppear {
                    if viewModel.shouldLoadMore(currentPost: post) {
                        Task { await viewModel.loadMore() }
                    }
                }
            }

            if viewModel.isLoading && !viewModel.posts.isEmpty {
                HStack {
                    Spacer()
                    ProgressView()
                        .padding()
                    Spacer()
                }
                .listRowSeparator(.hidden)
            }
        }
        .listStyle(.plain)
        .refreshable { await viewModel.refresh() }
    }

    private var locationGateView: some View {
        VStack(spacing: 16) {
            Image(systemName: "location.circle")
                .font(.system(size: 48))
                .foregroundColor(.blue)
            Text("Set Your Location")
                .font(.headline)
            Text("Set your location in settings to see posts from your community.")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
            Button("Open Settings") {
                onSettingsTapped()
            }
            .buttonStyle(.borderedProminent)
        }
        .padding()
    }

    private func errorView(_ error: String) -> some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundColor(.orange)
            Text(error)
                .multilineTextAlignment(.center)
            Button("Retry") { Task { await viewModel.refresh() } }
                .buttonStyle(.bordered)
        }
        .padding()
    }

    private var emptyView: some View {
        VStack(spacing: 12) {
            Image(systemName: "tray")
                .font(.largeTitle)
                .foregroundColor(.secondary)
            Text("No posts yet")
                .foregroundColor(.secondary)
            Text(viewModel.emptyMessage)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
}
