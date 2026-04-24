import Combine
import CoreLocation
import SwiftUI

class LocationMonitor: NSObject, ObservableObject, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    private let apiService: APIService
    private let minimumDistance: Double = 1000 // 1 km
    private let minimumInterval: TimeInterval = 900 // 15 min

    @AppStorage("lastSentLat") private var lastSentLat: Double = 0
    @AppStorage("lastSentLon") private var lastSentLon: Double = 0
    @AppStorage("lastSentTime") private var lastSentTime: Double = 0

    init(apiService: APIService) {
        self.apiService = apiService
        super.init()
        manager.delegate = self
    }

    func startIfAuthorized() {
        let status = manager.authorizationStatus
        guard status == .authorizedWhenInUse || status == .authorizedAlways else { return }
        manager.startMonitoringSignificantLocationChanges()
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        let lastSent = CLLocation(latitude: lastSentLat, longitude: lastSentLon)
        let timeSince = Date().timeIntervalSince1970 - lastSentTime

        guard location.distance(from: lastSent) > minimumDistance,
              timeSince > minimumInterval else { return }

        lastSentLat = location.coordinate.latitude
        lastSentLon = location.coordinate.longitude
        lastSentTime = Date().timeIntervalSince1970

        Task {
            let geocoder = CLGeocoder()
            let name = try? await geocoder.reverseGeocodeLocation(location).first?.locality
            try? await apiService.updateLocation(
                latitude: location.coordinate.latitude,
                longitude: location.coordinate.longitude,
                name: name
            )
        }
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {}
}
