import Combine
import MapKit

@MainActor
class LocationSearchCompleter: NSObject, ObservableObject, MKLocalSearchCompleterDelegate {
    @Published var searchText = "" {
        didSet { completer.queryFragment = searchText }
    }
    @Published var results: [MKLocalSearchCompletion] = []
    private let completer = MKLocalSearchCompleter()

    override init() {
        super.init()
        completer.delegate = self
        completer.resultTypes = .address
    }

    func select(_ result: MKLocalSearchCompletion) async -> (name: String, lat: Double, lon: Double)? {
        let request = MKLocalSearch.Request(completion: result)
        guard let response = try? await MKLocalSearch(request: request).start(),
              let item = response.mapItems.first else { return nil }
        let name = [result.title, result.subtitle].filter { !$0.isEmpty }.joined(separator: ", ")
        searchText = ""
        results = []
        return (name, item.placemark.coordinate.latitude, item.placemark.coordinate.longitude)
    }

    nonisolated func completerDidUpdateResults(_ completer: MKLocalSearchCompleter) {
        Task { @MainActor in results = completer.results }
    }

    nonisolated func completer(_ completer: MKLocalSearchCompleter, didFailWithError error: Error) {}
}
