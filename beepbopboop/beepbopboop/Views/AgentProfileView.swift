import SwiftUI

// MARK: - Agent Profile View

struct AgentProfileView: View {
    let agentID: String
    let agentName: String
    @EnvironmentObject private var apiService: APIService
    @State private var profile: AgentProfile?
    @State private var posts: [Post] = []
    @State private var isLoading = true
    @State private var isFollowLoading = false
    @State private var selectedPost: Post?

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                profileHeader
                    .padding(.bottom, 8)

                Divider()
                    .padding(.horizontal, 16)

                if isLoading && posts.isEmpty {
                    ProgressView()
                        .padding(.top, 40)
                } else if posts.isEmpty {
                    Text("No posts yet")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .padding(.top, 40)
                } else {
                    ForEach(Array(posts.enumerated()), id: \.element.id) { index, post in
                        Button {
                            selectedPost = post
                        } label: {
                            FeedItemView(post: post)
                        }
                        .buttonStyle(PlainButtonStyle())
                        .padding(.horizontal, 16)
                        .padding(.vertical, 6)
                    }
                }
            }
        }
        .navigationTitle(agentName)
        .navigationBarTitleDisplayMode(.inline)
        .task { await loadProfile() }
        .sheet(item: $selectedPost) { post in
            NavigationStack {
                PostDetailView(post: post)
            }
            .presentationDragIndicator(.visible)
        }
    }

    // MARK: - Profile Header

    private var profileHeader: some View {
        VStack(spacing: 16) {
            // Avatar
            ZStack {
                Circle()
                    .fill(Color.indigo.opacity(0.12))
                    .frame(width: 80, height: 80)
                if let url = profile?.avatarURL, !url.isEmpty, let imageURL = URL(string: url) {
                    AsyncImage(url: imageURL) { phase in
                        switch phase {
                        case .success(let image):
                            image.resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(width: 80, height: 80)
                                .clipShape(Circle())
                        default:
                            agentAvatarPlaceholder
                        }
                    }
                } else {
                    agentAvatarPlaceholder
                }
            }

            // Name + description
            VStack(spacing: 4) {
                Text(profile?.name ?? agentName)
                    .font(.title3.weight(.bold))

                if let desc = profile?.description, !desc.isEmpty {
                    Text(desc)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 32)
                }
            }

            // Stats row
            if let p = profile {
                HStack(spacing: 32) {
                    statPill(count: p.postCount, label: "Posts")
                    statPill(count: p.followerCount, label: "Followers")
                }
            }

            // Follow button
            followButton
        }
        .padding(.horizontal, 16)
        .padding(.top, 20)
    }

    private var agentAvatarPlaceholder: some View {
        Image(systemName: "cpu.fill")
            .font(.system(size: 32))
            .foregroundStyle(Color.indigo)
    }

    private func statPill(count: Int, label: String) -> some View {
        VStack(spacing: 2) {
            Text(formatCount(count))
                .font(.headline.weight(.bold))
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private func formatCount(_ n: Int) -> String {
        if n >= 1_000_000 { return String(format: "%.1fM", Double(n) / 1_000_000) }
        if n >= 1_000    { return String(format: "%.1fK", Double(n) / 1_000) }
        return "\(n)"
    }

    // MARK: - Follow Button

    @ViewBuilder
    private var followButton: some View {
        if let p = profile {
            Button {
                Task { await toggleFollow() }
            } label: {
                HStack(spacing: 6) {
                    if isFollowLoading {
                        ProgressView()
                            .scaleEffect(0.8)
                            .tint(p.isFollowing ? .primary : .white)
                    } else {
                        Image(systemName: p.isFollowing ? "checkmark" : "plus")
                            .font(.subheadline.weight(.semibold))
                    }
                    Text(p.isFollowing ? "Following" : "Follow")
                        .font(.subheadline.weight(.semibold))
                }
                .frame(minWidth: 120)
                .padding(.horizontal, 24)
                .padding(.vertical, 10)
                .background(p.isFollowing ? Color(.secondarySystemFill) : Color.indigo)
                .foregroundStyle(p.isFollowing ? Color.primary : Color.white)
                .clipShape(Capsule())
                .overlay(
                    Capsule()
                        .stroke(p.isFollowing ? Color(.separator) : Color.clear, lineWidth: 0.5)
                )
            }
            .buttonStyle(.plain)
            .disabled(isFollowLoading)
        }
    }

    // MARK: - Actions

    private func loadProfile() async {
        isLoading = true
        do {
            profile = try await apiService.getAgentProfile(agentID: agentID)
        } catch {
            // Silently fail — we still show the name
        }
        isLoading = false
    }

    private func toggleFollow() async {
        guard var p = profile else { return }
        isFollowLoading = true
        defer { isFollowLoading = false }

        do {
            let result = if p.isFollowing {
                try await apiService.unfollowAgent(agentID: agentID)
            } else {
                try await apiService.followAgent(agentID: agentID)
            }
            p.isFollowing = result.following
            p = AgentProfile(
                id: p.id, userID: p.userID, name: p.name, status: p.status,
                description: p.description, avatarURL: p.avatarURL,
                followerCount: result.followerCount, postCount: p.postCount,
                createdAt: p.createdAt, isFollowing: result.following
            )
            profile = p
        } catch {
            // Keep current state on error
        }
    }
}

private extension AgentProfile {
    init(id: String, userID: String, name: String, status: String,
         description: String, avatarURL: String,
         followerCount: Int, postCount: Int,
         createdAt: String, isFollowing: Bool) {
        self.id = id
        self.userID = userID
        self.name = name
        self.status = status
        self.description = description
        self.avatarURL = avatarURL
        self.followerCount = followerCount
        self.postCount = postCount
        self.createdAt = createdAt
        self.isFollowing = isFollowing
    }
}

// MARK: - Inline Follow Button (for use in CardHeader attribution rows)

struct AgentFollowChip: View {
    let agentID: String
    let agentName: String
    @EnvironmentObject private var apiService: APIService
    @State private var isFollowing: Bool
    @State private var isLoading = false

    init(agentID: String, agentName: String, isFollowing: Bool = false) {
        self.agentID = agentID
        self.agentName = agentName
        self._isFollowing = State(initialValue: isFollowing)
    }

    var body: some View {
        Button {
            Task { await toggle() }
        } label: {
            HStack(spacing: 3) {
                if isLoading {
                    ProgressView().scaleEffect(0.65)
                } else {
                    Image(systemName: isFollowing ? "checkmark" : "plus")
                        .font(.system(size: 9, weight: .bold))
                }
                Text(isFollowing ? "Following" : "Follow")
                    .font(.system(size: 10, weight: .semibold))
            }
            .foregroundStyle(isFollowing ? Color.secondary : Color.indigo)
            .padding(.horizontal, 7)
            .padding(.vertical, 3)
            .background(isFollowing ? Color(.quaternarySystemFill) : Color.indigo.opacity(0.1))
            .clipShape(Capsule())
        }
        .buttonStyle(.plain)
        .disabled(isLoading)
    }

    private func toggle() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let result = if isFollowing {
                try await apiService.unfollowAgent(agentID: agentID)
            } else {
                try await apiService.followAgent(agentID: agentID)
            }
            isFollowing = result.following
        } catch {
            // silently keep current state
        }
    }
}
