import SwiftUI

struct FeedView: View {
    @StateObject private var forYouVM: FeedListViewModel
    @StateObject private var communityVM: FeedListViewModel
    @StateObject private var personalVM: FeedListViewModel
    @State private var selectedTab = 0
    @State private var showSettings = false
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
                // Pill tab bar
                tabBar

                // Paged content
                TabView(selection: $selectedTab) {
                    FeedListView(viewModel: forYouVM, onSettingsTapped: { showSettings = true })
                        .tag(0)
                        .task { if forYouVM.posts.isEmpty && !forYouVM.isLoading { await forYouVM.refresh() } }

                    FeedListView(viewModel: communityVM, onSettingsTapped: { showSettings = true })
                        .tag(1)
                        .task { if communityVM.posts.isEmpty && !communityVM.isLoading { await communityVM.refresh() } }

                    FeedListView(viewModel: personalVM, onSettingsTapped: { showSettings = true })
                        .tag(2)
                        .task { if personalVM.posts.isEmpty && !personalVM.isLoading { await personalVM.refresh() } }
                }
                .tabViewStyle(.page(indexDisplayMode: .never))
            }
            .navigationTitle("BeepBopBoop")
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        showSettings = true
                    } label: {
                        Image(systemName: "gearshape")
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Sign Out") {
                        authService.signOut()
                    }
                }
            }
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

    // MARK: - Tab Bar

    private var tabBar: some View {
        HStack(spacing: 4) {
            tabButton("For You", tag: 0)
            tabButton("Community", tag: 1)
            tabButton("Personal", tag: 2)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
    }

    private func tabButton(_ title: String, tag: Int) -> some View {
        Button {
            withAnimation(.easeInOut(duration: 0.2)) {
                selectedTab = tag
            }
        } label: {
            Text(title)
                .font(.subheadline.weight(selectedTab == tag ? .semibold : .regular))
                .foregroundColor(selectedTab == tag ? .white : .primary)
                .padding(.horizontal, 16)
                .padding(.vertical, 8)
                .background(
                    Capsule()
                        .fill(selectedTab == tag ? Color.accentColor : Color(.systemGray5))
                )
        }
        .buttonStyle(.plain)
    }
}
