import Combine
import MapKit
import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel: SettingsViewModel
    @Environment(\.dismiss) private var dismiss

    init(apiService: APIService) {
        _viewModel = StateObject(wrappedValue: SettingsViewModel(apiService: apiService))
    }

    var body: some View {
        NavigationStack {
            Form {
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
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
            .task { await viewModel.loadSettings() }
        }
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
    @Published var isSaving = false
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var didSave = false

    private let apiService: APIService
    private let completer = MKLocalSearchCompleter()

    init(apiService: APIService) {
        self.apiService = apiService
        super.init()
        completer.delegate = self
        completer.resultTypes = .address
    }

    func loadSettings() async {
        isLoading = true
        do {
            let settings = try await apiService.getSettings()
            selectedLocationName = settings.locationName
            selectedLatitude = settings.latitude
            selectedLongitude = settings.longitude
            selectedRadius = settings.radiusKm
            if selectedRadius <= 0 { selectedRadius = 25.0 }
        } catch {
            // First time — use defaults
        }
        isLoading = false
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
            radiusKm: selectedRadius
        )

        do {
            let saved = try await apiService.updateSettings(settings)
            selectedLocationName = saved.locationName
            selectedLatitude = saved.latitude
            selectedLongitude = saved.longitude
            selectedRadius = saved.radiusKm

            // Cache in UserDefaults for quick access
            UserDefaults.standard.set(saved.locationName, forKey: "settings_locationName")
            if let lat = saved.latitude { UserDefaults.standard.set(lat, forKey: "settings_latitude") }
            if let lon = saved.longitude { UserDefaults.standard.set(lon, forKey: "settings_longitude") }
            UserDefaults.standard.set(saved.radiusKm, forKey: "settings_radiusKm")

            didSave = true
        } catch {
            errorMessage = error.localizedDescription
        }

        isSaving = false
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
