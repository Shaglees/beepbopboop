import SwiftUI

struct FeedListView: View {
    @ObservedObject var viewModel: FeedListViewModel
    var onSettingsTapped: () -> Void
    @Namespace private var zoomNamespace
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        Group {
            if viewModel.needsLocation {
                locationGateView
            } else if viewModel.isLoading && viewModel.posts.isEmpty {
                skeletonLoadingView
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
            ForEach(Array(viewModel.posts.enumerated()), id: \.element.id) { index, post in
                NavigationLink {
                    PostDetailView(post: post)
                        .navigationTransition(.zoom(sourceID: post.id, in: zoomNamespace))
                } label: {
                    FeedItemView(post: post)
                        .matchedTransitionSource(id: post.id, in: zoomNamespace)
                }
                .buttonStyle(CardPressStyle())
                .listRowSeparator(.hidden)
                .listRowInsets(EdgeInsets(top: 6, leading: 16, bottom: 6, trailing: 16))
                .listRowBackground(Color.clear)
                .modifier(StaggeredAppearance(index: index, reduceMotion: reduceMotion))
                .onAppear {
                    if viewModel.shouldLoadMore(currentPost: post) {
                        Task { await viewModel.loadMore() }
                    }
                }
            }

            if viewModel.isLoading && !viewModel.posts.isEmpty {
                SkeletonCard()
                    .listRowSeparator(.hidden)
                    .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 0))
            }
        }
        .listStyle(.plain)
        .scrollEdgeEffectStyle(.soft, for: .top)
        .refreshable { await viewModel.refresh() }
    }

    private var skeletonLoadingView: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(0..<4, id: \.self) { _ in
                    SkeletonCard()
                    Divider()
                }
            }
        }
    }

    private var locationGateView: some View {
        VStack(spacing: 16) {
            Image(systemName: "location.circle")
                .font(.system(size: 48))
                .foregroundColor(.blue)
                .symbolEffect(.pulse, isActive: true)
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
                .symbolEffect(.wiggle, isActive: true)
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
                .symbolEffect(.breathe, isActive: true)
            Text("No posts yet")
                .foregroundColor(.secondary)
            Text(viewModel.emptyMessage)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
}

// MARK: - Card Press Style

private struct CardPressStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.97 : 1.0)
            .animation(.spring(response: 0.3, dampingFraction: 0.6), value: configuration.isPressed)
    }
}

// MARK: - Staggered Card Entrance Animation

private struct StaggeredAppearance: ViewModifier {
    let index: Int
    let reduceMotion: Bool
    @State private var appeared = false

    func body(content: Content) -> some View {
        content
            .opacity(appeared ? 1 : 0)
            .offset(y: appeared ? 0 : 16)
            .onAppear {
                if reduceMotion {
                    appeared = true
                } else {
                    withAnimation(.snappy(duration: 0.35).delay(Double(min(index, 8)) * 0.05)) {
                        appeared = true
                    }
                }
            }
    }
}

// MARK: - Skeleton Loading Card

private struct SkeletonCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 6) {
                Circle()
                    .fill(Color(.systemGray5))
                    .frame(width: 8, height: 8)
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))
                    .frame(width: 90, height: 10)
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))
                    .frame(width: 30, height: 10)
            }
            RoundedRectangle(cornerRadius: 4)
                .fill(Color(.systemGray5))
                .frame(height: 16)
            RoundedRectangle(cornerRadius: 4)
                .fill(Color(.systemGray5))
                .frame(height: 12)
                .padding(.trailing, 60)
            RoundedRectangle(cornerRadius: 4)
                .fill(Color(.systemGray5))
                .frame(height: 12)
                .padding(.trailing, 120)
            HStack {
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))
                    .frame(width: 60, height: 10)
                Spacer()
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .phaseAnimator([false, true]) { content, phase in
            content.opacity(phase ? 0.4 : 0.8)
        } animation: { _ in .easeInOut(duration: 0.8) }
    }
}
