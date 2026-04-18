import Combine
import MapKit
import SwiftUI

// MARK: - Interest Category

struct InterestCategory: Identifiable {
    let id: String
    let title: String
    let icon: String
    let color: Color
    let labelWeights: [String: Double]
    let typeWeights: [String: Double]

    static let all: [InterestCategory] = [
        InterestCategory(
            id: "sports", title: "Sports", icon: "sportscourt.fill", color: .blue,
            labelWeights: ["sports": 0.8, "nba": 0.5, "nhl": 0.5, "nfl": 0.5, "mlb": 0.5],
            typeWeights: [:]
        ),
        InterestCategory(
            id: "weather", title: "Weather", icon: "cloud.sun.fill", color: .cyan,
            labelWeights: ["weather": 0.6],
            typeWeights: [:]
        ),
        InterestCategory(
            id: "events", title: "Local Events", icon: "calendar", color: .purple,
            labelWeights: ["event": 0.7],
            typeWeights: ["event": 0.5]
        ),
        InterestCategory(
            id: "fashion", title: "Fashion", icon: "tshirt.fill", color: Color(red: 0.878, green: 0.251, blue: 0.984),
            labelWeights: ["fashion": 0.8, "outfit": 0.6],
            typeWeights: [:]
        ),
        InterestCategory(
            id: "food", title: "Food & Drinks", icon: "fork.knife", color: .green,
            labelWeights: ["food": 0.7, "place": 0.5],
            typeWeights: [:]
        ),
        InterestCategory(
            id: "news", title: "News", icon: "newspaper.fill", color: .orange,
            labelWeights: ["article": 0.6],
            typeWeights: ["article": 0.4]
        ),
        InterestCategory(
            id: "deals", title: "Deals", icon: "tag.fill", color: .pink,
            labelWeights: ["deal": 0.7],
            typeWeights: [:]
        ),
        InterestCategory(
            id: "tech", title: "Tech", icon: "cpu.fill", color: .indigo,
            labelWeights: ["tech": 0.7],
            typeWeights: [:]
        ),
    ]
}

// MARK: - ViewModel

@MainActor
class OnboardingViewModel: NSObject, ObservableObject, MKLocalSearchCompleterDelegate {
    @Published var selectedInterests: Set<String> = []
    @Published var geoBias: Double = 0.5
    @Published var searchText = "" {
        didSet { completer.queryFragment = searchText }
    }
    @Published var searchResults: [MKLocalSearchCompletion] = []
    @Published var selectedLocationName: String?
    @Published var selectedLatitude: Double?
    @Published var selectedLongitude: Double?
    @Published var isSaving = false
    @Published var errorMessage: String?

    private let apiService: APIService
    private let completer = MKLocalSearchCompleter()

    init(apiService: APIService) {
        self.apiService = apiService
        super.init()
        completer.delegate = self
        completer.resultTypes = .address
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

    func saveAndComplete(onComplete: @escaping () -> Void) async {
        isSaving = true
        errorMessage = nil

        do {
            var labelWeights: [String: Double] = [:]
            var typeWeights: [String: Double] = [:]

            for categoryId in selectedInterests {
                if let category = InterestCategory.all.first(where: { $0.id == categoryId }) {
                    for (key, value) in category.labelWeights {
                        labelWeights[key] = (labelWeights[key] ?? 0) + value
                    }
                    for (key, value) in category.typeWeights {
                        typeWeights[key] = (typeWeights[key] ?? 0) + value
                    }
                }
            }

            if !labelWeights.isEmpty || !typeWeights.isEmpty {
                let weightsRequest = UserWeightsRequest(
                    labelWeights: labelWeights,
                    typeWeights: typeWeights,
                    freshnessBias: 0.7,
                    geoBias: geoBias
                )
                try await apiService.updateWeights(weightsRequest)
            }

            if let lat = selectedLatitude, let lon = selectedLongitude {
                let settings = UserSettings(
                    locationName: selectedLocationName,
                    latitude: lat,
                    longitude: lon,
                    radiusKm: 25.0
                )
                _ = try await apiService.updateSettings(settings)

                UserDefaults.standard.set(selectedLocationName, forKey: "settings_locationName")
                UserDefaults.standard.set(lat, forKey: "settings_latitude")
                UserDefaults.standard.set(lon, forKey: "settings_longitude")
                UserDefaults.standard.set(25.0, forKey: "settings_radiusKm")
            }

            onComplete()
        } catch {
            errorMessage = error.localizedDescription
            isSaving = false
        }
    }

    nonisolated func completerDidUpdateResults(_ completer: MKLocalSearchCompleter) {
        Task { @MainActor in
            searchResults = completer.results
        }
    }

    nonisolated func completer(_ completer: MKLocalSearchCompleter, didFailWithError error: Error) {}
}

// MARK: - OnboardingView

struct OnboardingView: View {
    @StateObject private var viewModel: OnboardingViewModel
    @State private var currentPage = 0
    let onComplete: () -> Void

    init(apiService: APIService, onComplete: @escaping () -> Void) {
        _viewModel = StateObject(wrappedValue: OnboardingViewModel(apiService: apiService))
        self.onComplete = onComplete
    }

    var body: some View {
        VStack(spacing: 0) {
            headerBar

            if currentPage == 0 {
                interestsPage
                    .transition(.asymmetric(
                        insertion: .move(edge: .trailing).combined(with: .opacity),
                        removal: .move(edge: .leading).combined(with: .opacity)
                    ))
            } else if currentPage == 1 {
                geoBiasPage
                    .transition(.asymmetric(
                        insertion: .move(edge: .trailing).combined(with: .opacity),
                        removal: .move(edge: .leading).combined(with: .opacity)
                    ))
            } else {
                locationPage
                    .transition(.asymmetric(
                        insertion: .move(edge: .trailing).combined(with: .opacity),
                        removal: .move(edge: .leading).combined(with: .opacity)
                    ))
            }

            bottomBar
        }
        .animation(.spring(response: 0.4, dampingFraction: 0.9), value: currentPage)
    }

    // MARK: - Header

    private var headerBar: some View {
        HStack {
            HStack(spacing: 6) {
                ForEach(0..<3) { index in
                    Circle()
                        .fill(index == currentPage ? Color.primary : Color.secondary.opacity(0.3))
                        .frame(
                            width: index == currentPage ? 8 : 6,
                            height: index == currentPage ? 8 : 6
                        )
                        .animation(.spring(response: 0.3), value: currentPage)
                }
            }

            Spacer()

            Button(currentPage < 2 ? "Skip" : "Skip") {
                if currentPage < 2 {
                    currentPage += 1
                } else {
                    Task { await viewModel.saveAndComplete(onComplete: onComplete) }
                }
            }
            .font(.subheadline.weight(.medium))
            .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 8)
    }

    // MARK: - Bottom bar

    private var bottomBar: some View {
        VStack(spacing: 8) {
            if let error = viewModel.errorMessage {
                VStack(spacing: 4) {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(.red)
                        .multilineTextAlignment(.center)
                    Button("Continue anyway") { onComplete() }
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal, 20)
            }

            Button {
                if currentPage < 2 {
                    currentPage += 1
                } else {
                    Task { await viewModel.saveAndComplete(onComplete: onComplete) }
                }
            } label: {
                Group {
                    if viewModel.isSaving {
                        ProgressView().tint(Color(.systemBackground))
                    } else {
                        Text(currentPage < 2 ? "Next" : "Get Started")
                            .fontWeight(.semibold)
                    }
                }
                .frame(maxWidth: .infinity)
                .frame(height: 50)
                .background(Color.primary)
                .foregroundStyle(Color(.systemBackground))
                .clipShape(RoundedRectangle(cornerRadius: 14))
            }
            .disabled(viewModel.isSaving)
            .padding(.horizontal, 20)
            .padding(.bottom, 32)
        }
    }

    // MARK: - Screen 1: Interests

    private var interestsPage: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("What do you\ncare about?")
                        .font(.largeTitle.weight(.bold))
                    Text("Pick interests to personalise your feed.")
                        .font(.body)
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal, 20)
                .padding(.top, 8)

                LazyVGrid(
                    columns: [GridItem(.flexible()), GridItem(.flexible()), GridItem(.flexible())],
                    spacing: 12
                ) {
                    ForEach(InterestCategory.all) { category in
                        InterestTile(
                            category: category,
                            isSelected: viewModel.selectedInterests.contains(category.id)
                        ) {
                            if viewModel.selectedInterests.contains(category.id) {
                                viewModel.selectedInterests.remove(category.id)
                            } else {
                                viewModel.selectedInterests.insert(category.id)
                            }
                        }
                    }
                }
                .padding(.horizontal, 20)
            }
            .padding(.bottom, 20)
        }
    }

    // MARK: - Screen 2: Geo Bias

    private var geoBiasPage: some View {
        VStack(alignment: .leading, spacing: 0) {
            VStack(alignment: .leading, spacing: 6) {
                Text("How local do\nyou want it?")
                    .font(.largeTitle.weight(.bold))
                Text("Tune how far BeepBopBoop looks for content.")
                    .font(.body)
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, 20)
            .padding(.top, 8)
            .padding(.bottom, 32)

            VStack(spacing: 20) {
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("📍")
                            .font(.system(size: 28))
                        Text("Hyperlocal")
                            .font(.subheadline.weight(.semibold))
                        Text("Your\nneighbourhood")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    Spacer()

                    VStack(alignment: .trailing, spacing: 4) {
                        Text("🌆")
                            .font(.system(size: 28))
                        Text("City-wide")
                            .font(.subheadline.weight(.semibold))
                        Text("Your\nbroader area")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.trailing)
                    }
                }

                Slider(value: $viewModel.geoBias, in: 0.1...0.9)
                    .tint(.primary)

                Text(geoBiasDescription)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .animation(.none, value: viewModel.geoBias)
            }
            .padding(20)
            .background(
                RoundedRectangle(cornerRadius: 16)
                    .fill(Color(.secondarySystemGroupedBackground))
            )
            .padding(.horizontal, 20)

            Spacer()
        }
    }

    private var geoBiasDescription: String {
        switch viewModel.geoBias {
        case ..<0.3: return "Tightly focused on your immediate neighbourhood"
        case ..<0.6: return "Balanced mix of local and nearby content"
        default: return "Broad city-wide coverage"
        }
    }

    // MARK: - Screen 3: Location

    private var locationPage: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Where are\nyou based?")
                        .font(.largeTitle.weight(.bold))
                    Text("BeepBopBoop surfaces what's happening near you.")
                        .font(.body)
                        .foregroundStyle(.secondary)
                }
                .padding(.top, 8)

                VStack(spacing: 8) {
                    TextField("Search for your city or neighbourhood", text: $viewModel.searchText)
                        .textContentType(.addressCity)
                        .autocorrectionDisabled()
                        .padding(14)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 12))

                    if !viewModel.searchResults.isEmpty {
                        VStack(spacing: 0) {
                            ForEach(Array(viewModel.searchResults.prefix(5).enumerated()), id: \.offset) { index, result in
                                if index > 0 {
                                    Divider().padding(.leading, 44)
                                }
                                Button {
                                    Task { await viewModel.selectSearchResult(result) }
                                } label: {
                                    HStack(spacing: 10) {
                                        Image(systemName: "mappin.circle.fill")
                                            .foregroundStyle(.secondary)
                                            .font(.title3)
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text(result.title)
                                                .font(.subheadline)
                                                .foregroundStyle(.primary)
                                            if !result.subtitle.isEmpty {
                                                Text(result.subtitle)
                                                    .font(.caption)
                                                    .foregroundStyle(.secondary)
                                            }
                                        }
                                        Spacer()
                                    }
                                    .padding(.horizontal, 14)
                                    .padding(.vertical, 10)
                                }
                            }
                        }
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                    }

                    if let location = viewModel.selectedLocationName {
                        HStack(spacing: 10) {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundStyle(.green)
                                .font(.title3)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Location set")
                                    .font(.subheadline.weight(.medium))
                                Text(location)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                        }
                        .padding(14)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                    }
                }

                Text("You can update this anytime in Settings.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 20)
        }
    }
}

// MARK: - Interest Tile

struct InterestTile: View {
    let category: InterestCategory
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 8) {
                ZStack {
                    Circle()
                        .fill(category.color.opacity(isSelected ? 0.22 : 0.1))
                        .frame(width: 48, height: 48)
                    Image(systemName: category.icon)
                        .font(.system(size: 20))
                        .foregroundStyle(category.color)
                }

                Text(category.title)
                    .font(.caption.weight(.medium))
                    .foregroundStyle(isSelected ? .primary : .secondary)
                    .multilineTextAlignment(.center)
                    .lineLimit(2)
                    .minimumScaleFactor(0.8)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .padding(.horizontal, 6)
            .background(
                RoundedRectangle(cornerRadius: 16)
                    .fill(isSelected
                        ? category.color.opacity(0.1)
                        : Color(.secondarySystemGroupedBackground))
                    .overlay(
                        RoundedRectangle(cornerRadius: 16)
                            .strokeBorder(
                                isSelected ? category.color.opacity(0.45) : Color.clear,
                                lineWidth: 1.5
                            )
                    )
            )
        }
        .buttonStyle(.plain)
        .animation(.spring(response: 0.25, dampingFraction: 0.7), value: isSelected)
    }
}
