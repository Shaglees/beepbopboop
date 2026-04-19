import SwiftUI

struct FeedView: View {
    @StateObject private var forYouVM: FeedListViewModel
    @StateObject private var communityVM: FeedListViewModel
    @StateObject private var personalVM: FeedListViewModel
    @StateObject private var savedVM: FeedListViewModel
    @State private var selectedTab = 0
    @State private var showSettings = false
    @State private var isHeaderVisible = true
    @State private var hasRequestedNotifications = false
    @Namespace private var tabGlass
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
        _communityVM = StateObject(wrappedValue: FeedListViewModel(feedType: .community, apiService: apiService))
        _personalVM = StateObject(wrappedValue: FeedListViewModel(feedType: .personal, apiService: apiService))
        _savedVM = StateObject(wrappedValue: FeedListViewModel(feedType: .saved, apiService: apiService))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Custom collapsible header
                if isHeaderVisible {
                    VStack(spacing: 0) {
                        titleBar
                        tabBar
                    }
                    .transition(.move(edge: .top).combined(with: .opacity))
                }

                // Paged content
                TabView(selection: $selectedTab) {
                    FeedListView(viewModel: forYouVM, isHeaderVisible: $isHeaderVisible, onSettingsTapped: { showSettings = true })
                        .tag(0)
                        .task { if forYouVM.posts.isEmpty && !forYouVM.isLoading { await forYouVM.refresh() } }

                    FeedListView(viewModel: communityVM, isHeaderVisible: $isHeaderVisible, onSettingsTapped: { showSettings = true })
                        .tag(1)
                        .task { if communityVM.posts.isEmpty && !communityVM.isLoading { await communityVM.refresh() } }

                    FeedListView(viewModel: personalVM, isHeaderVisible: $isHeaderVisible, onSettingsTapped: { showSettings = true })
                        .tag(2)
                        .task { if personalVM.posts.isEmpty && !personalVM.isLoading { await personalVM.refresh() } }

                    FeedListView(viewModel: savedVM, isHeaderVisible: $isHeaderVisible, onSettingsTapped: { showSettings = true })
                        .tag(3)
                        .task { if savedVM.posts.isEmpty && !savedVM.isLoading { await savedVM.refresh() } }
                }
                .tabViewStyle(.page(indexDisplayMode: .never))
            }
            .toolbar(.hidden, for: .navigationBar)
            .animation(.easeInOut(duration: 0.25), value: isHeaderVisible)
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
        guard ns.authorizationStatus == .notDetermined else { return }
        _ = await ns.requestAuthorization()
    }

    // MARK: - Title Bar

    private var titleBar: some View {
        HStack {
            Button {
                showSettings = true
            } label: {
                Image(systemName: "gearshape")
            }
            .buttonStyle(.glass)

            Spacer()

            Text("BeepBopBoop")
                .font(.headline.weight(.bold))

            Spacer()

            Button("Sign Out") {
                authService.signOut()
            }
            .font(.subheadline)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
    }

    // MARK: - Tab Bar

    private var tabBar: some View {
        GlassEffectContainer(spacing: 4) {
            HStack(spacing: 4) {
                tabButton("For You", tag: 0)
                tabButton("Community", tag: 1)
                tabButton("Personal", tag: 2)
                tabButton("Saved", tag: 3, systemImage: "bookmark")
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
        }
    }

    private func tabButton(_ title: String, tag: Int, systemImage: String? = nil) -> some View {
        Button {
            withAnimation(.bouncy) {
                selectedTab = tag
            }
        } label: {
            HStack(spacing: 4) {
                if let systemImage {
                    Image(systemName: selectedTab == tag ? systemImage + ".fill" : systemImage)
                        .font(.subheadline)
                }
                Text(title)
                    .font(.subheadline.weight(selectedTab == tag ? .semibold : .regular))
            }
            .foregroundStyle(selectedTab == tag ? .primary : .secondary)
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .glassEffect(
                selectedTab == tag ? .regular.tint(.accentColor).interactive() : .regular,
                in: .capsule
            )
        }
        .buttonStyle(.plain)
    }
}
