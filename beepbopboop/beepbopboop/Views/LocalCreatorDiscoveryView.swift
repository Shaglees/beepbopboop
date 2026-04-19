import Combine
import CoreLocation
import SwiftUI

// MARK: - LocalCreatorDiscoveryView

/// A dedicated browse screen for local creator discovery.
/// Requests location permission, sends coordinates to the backend,
/// and displays creator_spotlight posts grouped by designation type.
struct LocalCreatorDiscoveryView: View {
    @StateObject private var viewModel: LocalCreatorDiscoveryViewModel
    @EnvironmentObject private var apiService: APIService

    init(apiService: APIService) {
        _viewModel = StateObject(wrappedValue: LocalCreatorDiscoveryViewModel(apiService: apiService))
    }

    private let creatorIndigo = Color(red: 0.380, green: 0.333, blue: 0.933)

    var body: some View {
        NavigationStack {
            Group {
                switch viewModel.state {
                case .idle:
                    idleView
                case .requestingPermission:
                    permissionView
                case .loading:
                    loadingView
                case .loaded(let posts):
                    loadedView(posts)
                case .empty:
                    emptyView
                case .error(let msg):
                    errorView(msg)
                case .permissionDenied:
                    permissionDeniedView
                }
            }
            .navigationTitle("Local Creators")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    if viewModel.state.isLoaded {
                        Button {
                            Task { await viewModel.refresh() }
                        } label: {
                            Image(systemName: "arrow.clockwise")
                        }
                    }
                }
            }
        }
        .onAppear {
            if case .idle = viewModel.state {
                Task { await viewModel.start() }
            }
        }
    }

    // MARK: - State views

    private var idleView: some View {
        VStack(spacing: 20) {
            Image(systemName: "person.3.sequence")
                .font(.system(size: 56, weight: .ultraLight))
                .foregroundStyle(creatorIndigo.opacity(0.5))
            Text("Discover Local Creators")
                .font(.title2.weight(.semibold))
            Text("Find artists, musicians, writers and other creators in your neighbourhood.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
            Button("Get Started") {
                Task { await viewModel.start() }
            }
            .buttonStyle(.borderedProminent)
            .tint(creatorIndigo)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var permissionView: some View {
        VStack(spacing: 16) {
            ProgressView()
            Text("Requesting location…")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var loadingView: some View {
        VStack(spacing: 16) {
            ProgressView()
            Text("Searching for local creators…")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func loadedView(_ posts: [Post]) -> some View {
        ScrollView {
            LazyVStack(spacing: 12) {
                if let area = viewModel.areaName {
                    areaHeader(area)
                }
                ForEach(posts) { post in
                    NavigationLink {
                        PostDetailView(post: post)
                    } label: {
                        Group {
                            if let card = CreatorSpotlightCard(post: post) {
                                card
                            } else {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text(post.title)
                                        .font(.headline)
                                    Text(post.body)
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                        .lineLimit(2)
                                }
                                .padding()
                                .background(Color(.secondarySystemGroupedBackground))
                            }
                        }
                        .clipShape(RoundedRectangle(cornerRadius: 16))
                        .shadow(color: .black.opacity(0.1), radius: 8, x: 0, y: 4)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
        }
    }

    private func areaHeader(_ area: String) -> some View {
        HStack(spacing: 6) {
            Image(systemName: "location.fill")
                .font(.caption)
                .foregroundStyle(creatorIndigo)
            Text("Creators near \(area)")
                .font(.subheadline.weight(.medium))
                .foregroundStyle(.secondary)
            Spacer()
        }
        .padding(.horizontal, 4)
        .padding(.bottom, 4)
    }

    private var emptyView: some View {
        VStack(spacing: 16) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 48, weight: .ultraLight))
                .foregroundStyle(.secondary)
            Text("No creators found yet")
                .font(.title3.weight(.semibold))
            Text("We're still researching your area. Check back soon or try a wider radius.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func errorView(_ message: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: "exclamationmark.triangle")
                .font(.system(size: 48))
                .foregroundStyle(.orange)
            Text("Something went wrong")
                .font(.title3.weight(.semibold))
            Text(message)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
            Button("Try Again") {
                Task { await viewModel.refresh() }
            }
            .buttonStyle(.borderedProminent)
            .tint(creatorIndigo)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var permissionDeniedView: some View {
        VStack(spacing: 16) {
            Image(systemName: "location.slash")
                .font(.system(size: 48))
                .foregroundStyle(.secondary)
            Text("Location access needed")
                .font(.title3.weight(.semibold))
            Text("Enable location access in Settings to discover creators near you.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
            Button("Open Settings") {
                if let url = URL(string: UIApplication.openSettingsURLString) {
                    UIApplication.shared.open(url)
                }
            }
            .buttonStyle(.borderedProminent)
            .tint(creatorIndigo)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

// MARK: - ViewModel

@MainActor
final class LocalCreatorDiscoveryViewModel: NSObject, ObservableObject {
    @Published var state: DiscoveryState = .idle
    @Published var areaName: String?

    private let apiService: APIService
    private var locationManager: CLLocationManager?

    enum DiscoveryState {
        case idle
        case requestingPermission
        case loading
        case loaded([Post])
        case empty
        case error(String)
        case permissionDenied

        var isLoaded: Bool {
            if case .loaded = self { return true }
            return false
        }
    }

    init(apiService: APIService) {
        self.apiService = apiService
    }

    func start() async {
        let manager = CLLocationManager()
        self.locationManager = manager

        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            await fetchWithCurrentLocation(manager)
        case .notDetermined:
            state = .requestingPermission
            await requestPermissionAndFetch(manager)
        case .denied, .restricted:
            state = .permissionDenied
        @unknown default:
            state = .permissionDenied
        }
    }

    func refresh() async {
        guard let manager = locationManager else {
            await start()
            return
        }
        await fetchWithCurrentLocation(manager)
    }

    // MARK: - Private

    private func requestPermissionAndFetch(_ manager: CLLocationManager) async {
        // Use a continuation to bridge the CLLocationManager delegate callback.
        let status = await withCheckedContinuation { (continuation: CheckedContinuation<CLAuthorizationStatus, Never>) in
            let delegate = LocationPermissionDelegate(continuation: continuation)
            manager.delegate = delegate
            // Retain delegate strongly via associated object trick — store in ivar.
            objc_setAssociatedObject(manager, &AssociatedKeys.delegate, delegate, .OBJC_ASSOCIATION_RETAIN_NONATOMIC)
            manager.requestWhenInUseAuthorization()
        }

        switch status {
        case .authorizedWhenInUse, .authorizedAlways:
            await fetchWithCurrentLocation(manager)
        default:
            state = .permissionDenied
        }
    }

    private func fetchWithCurrentLocation(_ manager: CLLocationManager) async {
        state = .loading

        // Get current location via one-shot async wrapper.
        guard let location = await currentLocation(from: manager) else {
            state = .error("Could not determine your location.")
            return
        }

        let lat = location.coordinate.latitude
        let lon = location.coordinate.longitude

        // Send location to backend for background discovery.
        try? await apiService.updateUserLocation(latitude: lat, longitude: lon)

        // Fetch cached creator posts for this region.
        do {
            let posts = try await apiService.fetchLocalCreators(lat: lat, lon: lon, radius: nil)
            if posts.isEmpty {
                state = .empty
            } else {
                // Extract area name from the first creator post with a locality.
                areaName = posts.first(where: { $0.locality != nil })?.locality
                state = .loaded(posts)
            }
        } catch {
            state = .error(error.localizedDescription)
        }
    }

    private func currentLocation(from manager: CLLocationManager) async -> CLLocation? {
        // If we already have a recent location, use it.
        if let loc = manager.location, abs(loc.timestamp.timeIntervalSinceNow) < 60 {
            return loc
        }

        return await withCheckedContinuation { (continuation: CheckedContinuation<CLLocation?, Never>) in
            let delegate = LocationOneShotDelegate(continuation: continuation)
            manager.delegate = delegate
            objc_setAssociatedObject(manager, &AssociatedKeys.oneShotDelegate, delegate, .OBJC_ASSOCIATION_RETAIN_NONATOMIC)
            manager.requestLocation()
        }
    }
}

// MARK: - Location Delegates

private enum AssociatedKeys {
    static var delegate: UInt8 = 0
    static var oneShotDelegate: UInt8 = 0
}

private final class LocationPermissionDelegate: NSObject, CLLocationManagerDelegate {
    private let continuation: CheckedContinuation<CLAuthorizationStatus, Never>

    init(continuation: CheckedContinuation<CLAuthorizationStatus, Never>) {
        self.continuation = continuation
    }

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        let status = manager.authorizationStatus
        if status != .notDetermined {
            continuation.resume(returning: status)
        }
    }
}

private final class LocationOneShotDelegate: NSObject, CLLocationManagerDelegate {
    private let continuation: CheckedContinuation<CLLocation?, Never>
    private var resumed = false

    init(continuation: CheckedContinuation<CLLocation?, Never>) {
        self.continuation = continuation
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard !resumed else { return }
        resumed = true
        continuation.resume(returning: locations.last)
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        guard !resumed else { return }
        resumed = true
        continuation.resume(returning: nil)
    }
}
