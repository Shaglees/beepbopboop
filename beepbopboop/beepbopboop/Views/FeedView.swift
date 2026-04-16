import SwiftUI

struct FeedView: View {
    @StateObject private var forYouVM: FeedListViewModel
    @StateObject private var communityVM: FeedListViewModel
    @StateObject private var personalVM: FeedListViewModel
    @State private var selectedTab = 0
    @State private var showSettings = false
    @State private var isHeaderVisible = true
    @Namespace private var tabGlass
    private let authService: AuthService
    private let apiService: APIService

    init(authService: AuthService, apiService: APIService) {
        self.authService = authService
        self.apiService = apiService
        _forYouVM = StateObject(wrappedValue: FeedListViewModel(feedType: .forYou, apiService: apiService))
        _communityVM = StateObject(wrappedValue: FeedListViewModel(feedType: .community, apiService: apiService))
        _personalVM = StateObject(wrappedValue: FeedListViewModel(feedType: .personal, apiService: apiService))
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
                }
                .tabViewStyle(.page(indexDisplayMode: .never))
            }
            .toolbar(.hidden, for: .navigationBar)
            .animation(.easeInOut(duration: 0.25), value: isHeaderVisible)
            .sheet(isPresented: $showSettings) {
                SettingsView(apiService: apiService)
                    .onDisappear {
                        // Refresh geo-dependent feeds after settings change
                        Task {
                            await forYouVM.refresh()
                            await communityVM.refresh()
                        }
                    }
            }
        }
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
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
        }
    }

    private func tabButton(_ title: String, tag: Int) -> some View {
        Button {
            withAnimation(.bouncy) {
                selectedTab = tag
            }
        } label: {
            Text(title)
                .font(.subheadline.weight(selectedTab == tag ? .semibold : .regular))
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
