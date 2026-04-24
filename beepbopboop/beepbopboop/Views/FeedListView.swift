import SwiftUI

struct FeedListView: View {
    @ObservedObject var viewModel: FeedListViewModel
    var onSettingsTapped: () -> Void
    @State private var selectedPost: Post?

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
        ScrollView {
            LazyVStack(spacing: 14) {
                ForEach(viewModel.posts) { post in
                    FeedCardRow(post: post) {
                        selectedPost = post
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .onAppear {
                        if viewModel.shouldLoadMore(currentPost: post) {
                            Task { await viewModel.loadMore() }
                        }
                    }
                }

                if viewModel.isLoading && !viewModel.posts.isEmpty {
                    SkeletonCard()
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
            .padding(.bottom, 28)
        }
        .background(BBBDesign.background)
        .scrollEdgeEffectStyle(.soft, for: .top)
        .refreshable { await viewModel.refresh() }
        .onAppear { viewModel.restartPollingIfNeeded() }
        .onDisappear { viewModel.stopPolling() }
        .sheet(item: $selectedPost) { post in
            NavigationStack {
                PostDetailView(post: post)
            }
            .presentationDragIndicator(.visible)
        }
    }

    private var skeletonLoadingView: some View {
        ScrollView {
            LazyVStack(spacing: 14) {
                ForEach(0..<4, id: \.self) { _ in
                    SkeletonCard()
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
        }
        .background(BBBDesign.background)
    }

    private var locationGateView: some View {
        VStack(spacing: 16) {
            Image(systemName: "location.circle")
                .font(.system(size: 48))
                .foregroundColor(BBBDesign.clay)
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
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(BBBDesign.background)
    }

    private func errorView(_ error: String) -> some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundColor(BBBDesign.clay)
                .symbolEffect(.wiggle, isActive: true)
            Text(error)
                .multilineTextAlignment(.center)
            Button("Retry") { Task { await viewModel.refresh() } }
                .buttonStyle(.bordered)
        }
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(BBBDesign.background)
    }

    @ViewBuilder
    private var emptyView: some View {
        Group {
            if viewModel.feedType == .personal {
                AgentEmptyStateView()
            } else if viewModel.feedType == .following {
                FollowingEmptyStateView()
            } else {
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
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(BBBDesign.background)
    }
}

// MARK: - Feed Card Row
//
// Wrapping the card in a Button { } made nested buttons unreliable: taps on
// the reaction Menu, ShareLink, or bookmark would sometimes bubble up to the
// outer card button (opening the detail sheet) or get swallowed outright
// depending on hit-testing order in that particular SwiftUI build.
//
// `onTapGesture` on a plain container behaves differently from a nested
// Button: SwiftUI hit-tests inner Buttons *first* and only fires the tap
// gesture on the parent when nothing inside claimed it. We keep the pressed
// scale animation via a separate, non-tap LongPressGesture that reads the
// pressing state but never produces a "tap" action of its own.

private struct FeedCardRow: View {
    let post: Post
    let onTap: () -> Void
    @State private var swipeReaction: String?
    @EnvironmentObject private var apiService: APIService

    var body: some View {
        FeedItemView(post: post)
            .contentShape(RoundedRectangle(cornerRadius: BBBDesign.cardRadius, style: .continuous))
            .overlay(alignment: .center) {
                // Flash overlay when a swipe reaction fires
                if let reaction = swipeReaction {
                    swipeFlash(reaction: reaction)
                }
            }
            .onTapGesture {
                onTap()
            }
            .background(
                // UIKit swipe recognizers don't interfere with scroll
                SwipeGestureView(
                    onSwipeLeft: { commitReaction("less") },
                    onSwipeRight: { commitReaction("more") }
                )
            )
    }

    @ViewBuilder
    private func swipeFlash(reaction: String) -> some View {
        let isMore = reaction == "more"
        HStack(spacing: 6) {
            Image(systemName: isMore ? "arrow.up.circle.fill" : "arrow.down.circle.fill")
                .font(.title2)
            Text(isMore ? "More" : "Less")
                .font(.subheadline.weight(.bold))
        }
        .foregroundColor(.white)
        .padding(.horizontal, 20)
        .padding(.vertical, 10)
        .background(Capsule().fill(isMore ? BBBDesign.reactionMore : BBBDesign.reactionLess))
        .transition(.scale.combined(with: .opacity))
        .allowsHitTesting(false)
    }

    private func commitReaction(_ key: String) {
        UIImpactFeedbackGenerator(style: .medium).impactOccurred()

        // Show flash
        withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
            swipeReaction = key
        }
        // Hide flash after delay
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.8) {
            withAnimation(.easeOut(duration: 0.3)) {
                swipeReaction = nil
            }
        }

        Task {
            try? await apiService.setReaction(postID: post.id, reaction: key)
        }
    }
}

// MARK: - UIKit Swipe Gesture (doesn't conflict with scroll)

private struct SwipeGestureView: UIViewRepresentable {
    let onSwipeLeft: () -> Void
    let onSwipeRight: () -> Void

    func makeUIView(context: Context) -> UIView {
        let view = UIView()
        view.backgroundColor = .clear

        let leftSwipe = UISwipeGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleSwipe(_:)))
        leftSwipe.direction = .left
        view.addGestureRecognizer(leftSwipe)

        let rightSwipe = UISwipeGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleSwipe(_:)))
        rightSwipe.direction = .right
        view.addGestureRecognizer(rightSwipe)

        return view
    }

    func updateUIView(_ uiView: UIView, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator(onSwipeLeft: onSwipeLeft, onSwipeRight: onSwipeRight)
    }

    class Coordinator: NSObject {
        let onSwipeLeft: () -> Void
        let onSwipeRight: () -> Void

        init(onSwipeLeft: @escaping () -> Void, onSwipeRight: @escaping () -> Void) {
            self.onSwipeLeft = onSwipeLeft
            self.onSwipeRight = onSwipeRight
        }

        @objc func handleSwipe(_ gesture: UISwipeGestureRecognizer) {
            switch gesture.direction {
            case .left: onSwipeLeft()
            case .right: onSwipeRight()
            default: break
            }
        }
    }
}

// MARK: - Skeleton Loading Card

private struct SkeletonCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Header: dot + name + pill + spacer + time
            HStack(spacing: 6) {
                Circle()
                    .fill(Color(.systemGray5))
                    .frame(width: 8, height: 8)
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))
                    .frame(width: 80, height: 10)
                RoundedRectangle(cornerRadius: 10)
                    .fill(Color(.systemGray5))
                    .frame(width: 44, height: 16)
                Spacer()
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))
                    .frame(width: 30, height: 10)
            }
            // Body lines
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
            // Footer: location + spacer + bookmark
            HStack {
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))
                    .frame(width: 60, height: 10)
                Spacer()
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))
                    .frame(width: 14, height: 14)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .phaseAnimator([false, true]) { content, phase in
            content.opacity(phase ? 0.4 : 0.8)
        } animation: { _ in .easeInOut(duration: 0.8) }
    }
}
