import CoreLocation
import SwiftUI

struct OnboardingLocationView: View {
    @Binding var profile: UserProfileIdentity
    let onNext: () -> Void
    @State private var locationManager = LocationHelper()

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("Where are you based?")
                .font(.system(size: 28, weight: .bold, design: .serif))
            TextField("City or neighborhood", text: $profile.homeLocation)
                .textFieldStyle(.roundedBorder)
                .padding(.horizontal, 40)
            Text("Timezone: \(profile.timezone)")
                .font(.system(size: 13, design: .monospaced))
                .foregroundStyle(.secondary)
            Button("Use my location") {
                locationManager.requestLocation { lat, lon, name, tz in
                    profile.homeLat = lat
                    profile.homeLon = lon
                    if let name { profile.homeLocation = name }
                    if let tz { profile.timezone = tz }
                }
            }
            .buttonStyle(.bordered)
            Spacer()
            Button("Continue") { onNext() }
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
        }
        .onAppear {
            let tz = TimeZone.current
            let seconds = tz.secondsFromGMT()
            let hours = seconds / 3600
            let mins = abs(seconds % 3600) / 60
            if mins == 0 {
                profile.timezone = "UTC\(hours >= 0 ? "+" : "")\(hours)"
            } else {
                profile.timezone = "UTC\(hours >= 0 ? "+" : "")\(hours):\(String(format: "%02d", mins))"
            }
        }
    }
}

class LocationHelper: NSObject, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    private var completion: ((Double, Double, String?, String?) -> Void)?

    func requestLocation(completion: @escaping (Double, Double, String?, String?) -> Void) {
        self.completion = completion
        manager.delegate = self
        manager.requestWhenInUseAuthorization()
        manager.requestLocation()
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let loc = locations.first else { return }
        let geocoder = CLGeocoder()
        geocoder.reverseGeocodeLocation(loc) { placemarks, _ in
            let name = placemarks?.first?.locality
            let tz = placemarks?.first?.timeZone
            var tzString: String? = nil
            if let tz {
                let s = tz.secondsFromGMT()
                let h = s / 3600
                let m = abs(s % 3600) / 60
                tzString = m == 0 ? "UTC\(h >= 0 ? "+" : "")\(h)" : "UTC\(h >= 0 ? "+" : "")\(h):\(String(format: "%02d", m))"
            }
            self.completion?(loc.coordinate.latitude, loc.coordinate.longitude, name, tzString)
        }
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {}
}
