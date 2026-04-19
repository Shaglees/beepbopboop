import Combine
import EventKit
import MapKit
import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel: SettingsViewModel
    @Environment(\.dismiss) private var dismiss
    @AppStorage("onboardingComplete") private var onboardingComplete = false
    @State private var showUpdateInterests = false
    private let apiService: APIService

    init(apiService: APIService, notificationService: NotificationService? = nil, calendarService: CalendarService) {
        self.apiService = apiService
        _viewModel = StateObject(wrappedValue: SettingsViewModel(
            apiService: apiService,
            notificationService: notificationService,
            calendarService: calendarService
        ))
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Your Feed") {
                    if let summary = viewModel.weightsSummary {
                        if summary.dataPoints < 10 {
                            Label("Still learning — keep scrolling and react to posts", systemImage: "brain")
                                .foregroundStyle(.secondary)
                                .font(.subheadline)
                        } else {
                            VStack(alignment: .leading, spacing: 6) {
                                Text("You engage most with:")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                                Text(summary.topLabels.prefix(3).joined(separator: " · "))
                                    .font(.subheadline)
                                    .fontWeight(.medium)
                            }
                            .padding(.vertical, 2)
                            Text("Based on \(summary.dataPoints) interactions")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    } else {
                        Text("Reacting to posts helps your feed improve")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }

                Section("Location") {
                    TextField("Search for a city or place", text: $viewModel.searchText)
                        .textContentType(.addressCity)
                        .autocorrectionDisabled()

                    if !viewModel.searchResults.isEmpty {
                        ForEach(viewModel.searchResults, id: \.self) { result in
                            Button {
                                Task { await viewModel.selectSearchResult(result) }
                            } label: {
                                VStack(alignment: .leading) {
                                    Text(result.title)
                                        .foregroundColor(.primary)
                                    if !result.subtitle.isEmpty {
                                        Text(result.subtitle)
                                            .font(.caption)
                                            .foregroundColor(.secondary)
                                    }
                                }
                            }
                        }
                    }

                    if let location = viewModel.selectedLocationName {
                        HStack {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.green)
                            Text(location)
                        }
                    }
                }

                Section("Radius") {
                    Picker("Radius", selection: $viewModel.selectedRadius) {
                        Text("10 km").tag(10.0)
                        Text("25 km").tag(25.0)
                        Text("50 km").tag(50.0)
                        Text("100 km").tag(100.0)
                    }
                    .pickerStyle(.segmented)
                }

                Section("Sports") {
                    NavigationLink("Sports & Teams") {
                        SportsSettingsView(followedTeams: $viewModel.followedTeams)
                    }
                }

                Section("Tune your feed") {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Text("📍")
                            Text("More local")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Spacer()
                            Text("More global")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Text("🌍")
                        }
                        Slider(value: $viewModel.geoBias, in: 0...1) { editing in
                            if !editing { viewModel.scheduleWeightsSave() }
                        }
                    }

                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Text("⚡")
                            Text("Live & timely")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Spacer()
                            Text("Evergreen")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Text("📚")
                        }
                        Slider(value: $viewModel.freshnessBias, in: 0...1) { editing in
                            if !editing { viewModel.scheduleWeightsSave() }
                        }
                    }

                    HStack {
                        Button("Reset to defaults") {
                            viewModel.resetWeightsToDefaults()
                        }
                        .font(.caption)
                        .foregroundColor(.secondary)

                        Spacer()

                        if viewModel.feedUpdated {
                            HStack(spacing: 4) {
                                Image(systemName: "checkmark.circle.fill")
                                    .foregroundColor(.green)
                                    .font(.caption)
                                Text("Feed updated")
                                    .font(.caption)
                                    .foregroundColor(.green)
                            }
                            .transition(.opacity)
                        }
                    }
                    .animation(.easeInOut(duration: 0.3), value: viewModel.feedUpdated)
                }

                Section("Calendar") {
                    Toggle("Anticipatory feed", isOn: $viewModel.calendarEnabled)
                        .onChange(of: viewModel.calendarEnabled) { _, enabled in
                            if enabled {
                                Task { await viewModel.requestCalendarAccessAndSync() }
                            }
                        }
                    if viewModel.calendarEnabled {
                        Text("Your upcoming events help surface relevant content before you need it.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                Section("Notifications") {
                    Toggle("Daily digest", isOn: $viewModel.notificationsEnabled)

                    if viewModel.notificationsEnabled {
                        Picker("Delivery time", selection: $viewModel.digestHour) {
                            ForEach(0..<24, id: \.self) { hour in
                                Text(hourLabel(hour)).tag(hour)
                            }
                        }

                        if viewModel.notificationsDenied {
                            HStack(spacing: 6) {
                                Image(systemName: "exclamationmark.triangle")
                                    .foregroundStyle(.orange)
                                Text("Allow notifications in Settings to receive digests.")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }

                Section {
                    Button {
                        Task { await viewModel.save() }
                    } label: {
                        if viewModel.isSaving {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            Text("Save")
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(viewModel.isSaving)
                }

                if let error = viewModel.errorMessage {
                    Section {
                        Text(error)
                            .foregroundColor(.red)
                    }
                }

                if viewModel.didSave {
                    Section {
                        HStack {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.green)
                            Text("Settings saved")
                        }
                    }
                }

                interestsSection
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                        .buttonStyle(.glass)
                }
            }
            .task { await viewModel.loadSettings() }
            .sheet(isPresented: $showUpdateInterests) {
                OnboardingView(apiService: apiService) {
                    showUpdateInterests = false
                }
            }
        }
    }

    private var interestsSection: some View {
        Section("Interests") {
            Button {
                showUpdateInterests = true
            } label: {
                Label("Update interests", systemImage: "sparkles")
            }
        }
    }

    private func hourLabel(_ hour: Int) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "h a"
        var components = DateComponents()
        components.hour = hour
        components.minute = 0
        if let date = Calendar.current.date(from: components) {
            return formatter.string(from: date)
        }
        return "\(hour):00"
    }
}

@MainActor
class SettingsViewModel: NSObject, ObservableObject, MKLocalSearchCompleterDelegate {
    @Published var searchText = "" {
        didSet { completer.queryFragment = searchText }
    }
    @Published var searchResults: [MKLocalSearchCompletion] = []
    @Published var selectedLocationName: String?
    @Published var selectedLatitude: Double?
    @Published var selectedLongitude: Double?
    @Published var selectedRadius: Double = 25.0
    @Published var notificationsEnabled: Bool = true
    @Published var digestHour: Int = 8
    @Published var notificationsDenied: Bool = false
    @Published var isSaving = false
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var didSave = false
    @Published var weightsSummary: WeightsSummary?
    @Published var followedTeams: Set<String> = []
    @Published var geoBias: Double = 0.5
    @Published var freshnessBias: Double = 0.8
    @Published var feedUpdated = false
    @Published var calendarEnabled: Bool = false

    private var weightsSaveTask: Task<Void, Never>?
    private var badgeTask: Task<Void, Never>?
    private var cachedWeights: FeedWeights = .defaults
    private let apiService: APIService
    private let notificationService: NotificationService?
    private let calendarService: CalendarService
    private let completer = MKLocalSearchCompleter()

    init(apiService: APIService, notificationService: NotificationService? = nil, calendarService: CalendarService) {
        self.apiService = apiService
        self.notificationService = notificationService
        self.calendarService = calendarService
        super.init()
        completer.delegate = self
        completer.resultTypes = .address
    }

    func loadSettings() async {
        isLoading = true
        async let settingsLoad: () = loadSettingsOnly()
        async let weightsLoad: () = loadWeights()
        _ = await (settingsLoad, weightsLoad)
        isLoading = false
    }

    private func loadSettingsOnly() async {
        do {
            let settings = try await apiService.getSettings()
            selectedLocationName = settings.locationName
            selectedLatitude = settings.latitude
            selectedLongitude = settings.longitude
            selectedRadius = settings.radiusKm
            if selectedRadius <= 0 { selectedRadius = 25.0 }
            followedTeams = Set(settings.followedTeams ?? [])
            notificationsEnabled = settings.notificationsEnabled
            digestHour = settings.digestHour
            calendarEnabled = settings.calendarEnabled
        } catch {
            // First time — use defaults
        }
        weightsSummary = try? await apiService.getWeightsSummary()
        if let ns = notificationService {
            await ns.checkStatus()
            notificationsDenied = ns.authorizationStatus == .denied
        }
    }

    func requestCalendarAccessAndSync() async {
        let granted = await calendarService.requestAccess()
        guard granted else {
            calendarEnabled = false
            return
        }
        let events = calendarService.fetchUpcomingEvents()
        try? await apiService.syncCalendarEvents(events)
    }

    func selectSearchResult(_ result: MKLocalSearchCompletion) async {
        let request = MKLocalSearch.Request(completion: result)
        do {
            let response = try await MKLocalSearch(request: request).start()
            if let item = response.mapItems.first {
                selectedLocationName = [result.title, result.subtitle]
                    .filter { !$0.isEmpty }
                    .joined(separator: ", ")
                let coordinate = item.location.coordinate
                selectedLatitude = coordinate.latitude
                selectedLongitude = coordinate.longitude
                searchText = ""
                searchResults = []
            }
        } catch {
            errorMessage = "Could not resolve location"
        }
    }

    func save() async {
        guard selectedLatitude != nil, selectedLongitude != nil else {
            errorMessage = "Please select a location first"
            return
        }

        isSaving = true
        errorMessage = nil
        didSave = false

        let settings = UserSettings(
            locationName: selectedLocationName,
            latitude: selectedLatitude,
            longitude: selectedLongitude,
            radiusKm: selectedRadius,
            followedTeams: followedTeams.isEmpty ? nil : Array(followedTeams),
            notificationsEnabled: notificationsEnabled,
            digestHour: digestHour,
            calendarEnabled: calendarEnabled
        )

        do {
            let saved = try await apiService.updateSettings(settings)
            selectedLocationName = saved.locationName
            selectedLatitude = saved.latitude
            selectedLongitude = saved.longitude
            selectedRadius = saved.radiusKm
            followedTeams = Set(saved.followedTeams ?? [])
            notificationsEnabled = saved.notificationsEnabled
            digestHour = saved.digestHour

            // Cache in UserDefaults for quick access
            UserDefaults.standard.set(saved.locationName, forKey: "settings_locationName")
            if let lat = saved.latitude { UserDefaults.standard.set(lat, forKey: "settings_latitude") }
            if let lon = saved.longitude { UserDefaults.standard.set(lon, forKey: "settings_longitude") }
            UserDefaults.standard.set(saved.radiusKm, forKey: "settings_radiusKm")
            UserDefaults.standard.set(saved.notificationsEnabled, forKey: "settings_notificationsEnabled")
            UserDefaults.standard.set(saved.digestHour, forKey: "settings_digestHour")

            if let ns = notificationService {
                if saved.notificationsEnabled {
                    _ = await ns.requestAuthorization()
                    notificationsDenied = ns.authorizationStatus == .denied
                } else {
                    ns.cancelDailyDigest()
                }
            }

            didSave = true
        } catch {
            errorMessage = error.localizedDescription
        }

        isSaving = false
    }

    private func loadWeights() async {
        do {
            let weights = try await apiService.getWeights()
            cachedWeights = weights
            geoBias = weights.geoBias
            freshnessBias = weights.freshnessBias
        } catch {
            // Use defaults on failure
        }
    }

    func scheduleWeightsSave() {
        weightsSaveTask?.cancel()
        let geo = geoBias
        let fresh = freshnessBias
        weightsSaveTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 500_000_000)
            guard !Task.isCancelled, let self else { return }
            await self.saveWeights(geo: geo, freshness: fresh)
        }
    }

    func resetWeightsToDefaults() {
        geoBias = FeedWeights.defaults.geoBias
        freshnessBias = FeedWeights.defaults.freshnessBias
        scheduleWeightsSave()
    }

    private func saveWeights(geo: Double, freshness: Double) async {
        let weights = FeedWeights(
            labelWeights: cachedWeights.labelWeights,
            typeWeights: cachedWeights.typeWeights,
            freshnessBias: freshness,
            geoBias: geo
        )
        do {
            try await apiService.updateWeights(weights)
            cachedWeights = weights
            await showFeedUpdatedBadge()
        } catch {
            // Silent — feed won't update until next adjustment
        }
    }

    private func showFeedUpdatedBadge() async {
        feedUpdated = true
        badgeTask?.cancel()
        badgeTask = Task {
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            guard !Task.isCancelled else { return }
            feedUpdated = false
        }
    }

    // MARK: - MKLocalSearchCompleterDelegate

    nonisolated func completerDidUpdateResults(_ completer: MKLocalSearchCompleter) {
        Task { @MainActor in
            searchResults = completer.results
        }
    }

    nonisolated func completer(_ completer: MKLocalSearchCompleter, didFailWithError error: Error) {
        // Silently ignore search errors
    }
}
