import SwiftUI

struct FeedView: View {
    @StateObject private var forYouVM: FeedListViewModel
    @StateObject private var followingVM: FeedListViewModel
    @StateObject private var communityVM: FeedListViewModel
    @State private var selectedFeed: FeedSection = .forYou
    @State private var showSettings = false
    @State private var hasRequestedNotifications = false
    private let authService: AuthService
    private let apiService: APIService
    private let notificationService: NotificationService?
    private let calendarService: CalendarService

    init(authService: AuthService, apiService: APIService, notificationService: NotificationService? = nil, calendarService: CalendarService) {
        self.authService = authService
        self.apiService = apiService
        self.notificationService = notificationService
        self.calendarService = calendarService
        _forYouVM = StateObject(wrappedValue: FeedListViewModel(feedType: .forYou, apiService: apiService))
        _followingVM = StateObject(wrappedValue: FeedListViewModel(feedType: .following, apiService: apiService))
        _communityVM = StateObject(wrappedValue: FeedListViewModel(feedType: .community, apiService: apiService))
    }

    var body: some View {
        NavigationStack {
            GeometryReader { proxy in
                VStack(spacing: 0) {
                    VStack(spacing: 0) {
                        titleBar
                        tabBar
                    }
                    .padding(.top, proxy.safeAreaInsets.top)

                    selectedFeedList
                        .task(id: selectedFeed) {
                            await refreshSelectedFeedIfNeeded()
                        }
                }
                .background(BBBDesign.background.ignoresSafeArea())
                .ignoresSafeArea(.container, edges: .top)
            }
            .toolbar(.hidden, for: .navigationBar)
            .sheet(isPresented: $showSettings) {
                SettingsView(apiService: apiService, notificationService: notificationService, calendarService: calendarService)
                    .onDisappear {
                        // Refresh geo-dependent feeds after settings change
                        Task {
                            await forYouVM.refresh()
                            await communityVM.refresh()
                        }
                    }
            }
            .task {
                await requestNotificationsAfterFirstLoad()
            }
        }
    }

    // MARK: - Notifications

    private func requestNotificationsAfterFirstLoad() async {
        guard let ns = notificationService, !hasRequestedNotifications else { return }
        // Wait until the feed has loaded at least once before prompting
        while forYouVM.posts.isEmpty && forYouVM.isLoading {
            try? await Task.sleep(nanoseconds: 500_000_000)
        }
        guard !forYouVM.posts.isEmpty else { return }
        hasRequestedNotifications = true
        await ns.checkStatus()
        if ns.authorizationStatus == .notDetermined {
            _ = await ns.requestAuthorization()
        }
        if ns.authorizationStatus == .authorized {
            let digestHour = UserDefaults.standard.object(forKey: "settings_digestHour") as? Int ?? 8
            if let posts = try? await apiService.fetchDigestPosts() {
                await ns.scheduleDailyDigest(posts: posts, digestHour: digestHour)
            }
        }
    }

    @ViewBuilder
    private var selectedFeedList: some View {
        switch selectedFeed {
        case .forYou:
            FeedListView(viewModel: forYouVM, onSettingsTapped: { showSettings = true })
        case .following:
            FeedListView(viewModel: followingVM, onSettingsTapped: { showSettings = true })
        case .community:
            FeedListView(viewModel: communityVM, onSettingsTapped: { showSettings = true })
        }
    }

    private func refreshSelectedFeedIfNeeded() async {
        let viewModel: FeedListViewModel
        switch selectedFeed {
        case .forYou:
            viewModel = forYouVM
        case .following:
            viewModel = followingVM
        case .community:
            viewModel = communityVM
        }

        if viewModel.posts.isEmpty && !viewModel.isLoading {
            await viewModel.refresh()
        }
    }

    // MARK: - Title Bar

    private var titleBar: some View {
        HStack {
            Button {
                showSettings = true
            } label: {
                Image(systemName: "gearshape")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(BBBDesign.ink2)
                    .frame(width: 44, height: 36)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Settings")

            Spacer()

            Text("BeepBopBoop")
                .font(.system(size: 20, weight: .semibold, design: .serif))
                .tracking(-0.2)
                .foregroundStyle(BBBDesign.ink)

            Spacer()

            Button {
                authService.signOut()
            } label: {
                Image(systemName: "rectangle.portrait.and.arrow.right")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(BBBDesign.ink2)
                    .frame(width: 44, height: 36)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Sign out")
        }
        .padding(.horizontal, 16)
        .padding(.top, 10)
        .padding(.bottom, 2)
        .background(.ultraThinMaterial)
    }

    // MARK: - Tab Bar

    private var tabBar: some View {
        GeometryReader { proxy in
            HStack(spacing: 0) {
                ForEach(FeedSection.allCases) { section in
                    tabButton(section)
                        .frame(width: proxy.size.width / CGFloat(FeedSection.allCases.count))
                }
            }
        }
        .frame(height: 54)
        .padding(.horizontal, 22)
        .padding(.top, 2)
        .padding(.bottom, 11)
        .background(.ultraThinMaterial)
        .overlay(alignment: .bottom) {
            Rectangle()
                .fill(BBBDesign.line)
                .frame(height: 1)
        }
    }

    private func tabButton(_ section: FeedSection) -> some View {
        let isSelected = selectedFeed == section

        return Button {
            withAnimation(.easeOut(duration: 0.18)) {
                selectedFeed = section
            }
        } label: {
            VStack(spacing: 5) {
                Text(section.title)
                    .font(.system(size: 14, weight: isSelected ? .semibold : .regular))
                    .tracking(-0.1)

                Capsule()
                    .fill(isSelected ? BBBDesign.clay : Color.clear)
                    .frame(width: 22, height: 2)
            }
                .frame(maxWidth: .infinity, minHeight: 44)
                .tracking(-0.1)
                .foregroundStyle(isSelected ? BBBDesign.ink : BBBDesign.ink3)
                .padding(.vertical, 7)
                .contentShape(Rectangle())
                .accessibilityAddTraits(isSelected ? .isSelected : [])
        }
        .buttonStyle(.plain)
    }
}

private enum FeedSection: CaseIterable, Identifiable, Hashable {
    case forYou
    case following
    case community

    var id: Self { self }

    var title: String {
        switch self {
        case .forYou:
            return "For you"
        case .following:
            return "Following"
        case .community:
            return "Community"
        }
    }
}
