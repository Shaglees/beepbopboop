import CoreLocation
import MapKit
import SwiftUI

struct OnboardingLocationView: View {
    @Binding var profile: UserProfileIdentity
    let onNext: () -> Void
    @State private var locationManager = LocationHelper()
    @StateObject private var completer = LocationSearchCompleter()

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("Where are you based?")
                .font(.system(size: 28, weight: .bold, design: .serif))

            VStack(spacing: 4) {
                TextField("Search city or neighborhood", text: $completer.searchText)
                    .textFieldStyle(.roundedBorder)
                    .padding(.horizontal, 40)

                if !completer.results.isEmpty {
                    VStack(spacing: 0) {
                        ForEach(completer.results, id: \.self) { result in
                            Button {
                                Task {
                                    if let loc = await completer.select(result) {
                                        profile.homeLocation = loc.name
                                        profile.homeLat = loc.lat
                                        profile.homeLon = loc.lon
                                    }
                                }
                            } label: {
                                VStack(alignment: .leading) {
                                    Text(result.title).font(.system(size: 15))
                                    Text(result.subtitle).font(.system(size: 12)).foregroundStyle(.secondary)
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.vertical, 8).padding(.horizontal, 16)
                            }
                            .buttonStyle(.plain)
                            Divider()
                        }
                    }
                    .background(Color(.systemBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                    .shadow(radius: 2)
                    .padding(.horizontal, 40)
                }
            }

            if !profile.homeLocation.isEmpty {
                Label(profile.homeLocation, systemImage: "mappin.circle.fill")
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(.secondary)
            }

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
                .disabled(profile.homeLat == nil)
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
