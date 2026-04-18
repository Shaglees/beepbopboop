import SwiftUI
import MapKit

struct PlaceDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var hasCoordinates: Bool {
        post.latitude != nil && post.longitude != nil
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Full-bleed hero image with gradient overlay
                if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                    GeometryReader { geo in
                        AsyncImage(url: url) { phase in
                            switch phase {
                            case .success(let img):
                                img.resizable()
                                    .aspectRatio(contentMode: .fill)
                                    .frame(width: geo.size.width, height: 280)
                                    .clipped()
                                    .overlay(alignment: .bottom) {
                                        LinearGradient(
                                            colors: [.clear, .black.opacity(0.7)],
                                            startPoint: .top, endPoint: .bottom
                                        )
                                        .frame(height: 120)
                                    }
                                    .overlay(alignment: .bottomLeading) {
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text(post.title)
                                                .font(.title2.weight(.bold))
                                                .foregroundStyle(.white)
                                            if let locality = post.locality, !locality.isEmpty {
                                                Text(locality)
                                                    .font(.subheadline)
                                                    .foregroundStyle(.white.opacity(0.8))
                                            }
                                        }
                                        .padding(16)
                                    }
                            case .failure: EmptyView()
                            default:
                                Color.secondary.opacity(0.2)
                                    .frame(height: 280)
                                    .overlay(ProgressView())
                            }
                        }
                    }
                    .frame(height: 280)
                } else {
                    // No image: just a colored header
                    ZStack {
                        LinearGradient(
                            colors: [post.hintColor, post.hintColor.opacity(0.6)],
                            startPoint: .topLeading, endPoint: .bottomTrailing
                        )
                        .frame(height: 160)
                        VStack(spacing: 8) {
                            Image(systemName: "mappin.circle.fill")
                                .font(.system(size: 48))
                                .foregroundStyle(.white)
                            Text(post.title)
                                .font(.title2.weight(.bold))
                                .foregroundStyle(.white)
                                .multilineTextAlignment(.center)
                                .padding(.horizontal, 20)
                        }
                    }
                }

                VStack(alignment: .leading, spacing: 18) {
                    // Action buttons
                    HStack(spacing: 12) {
                        if let lat = post.latitude, let lon = post.longitude,
                           let url = URL(string: "maps://?daddr=\(lat),\(lon)") {
                            Link(destination: url) {
                                Label("Directions", systemImage: "arrow.triangle.turn.up.right.circle.fill")
                                    .font(.subheadline.weight(.semibold))
                                    .padding(.horizontal, 16)
                                    .padding(.vertical, 10)
                                    .background(post.hintColor, in: Capsule())
                                    .foregroundStyle(.white)
                            }
                        }
                        if let extURL = post.externalURL, !extURL.isEmpty, let url = URL(string: extURL) {
                            Link(destination: url) {
                                Label("Open", systemImage: "globe")
                                    .font(.subheadline.weight(.semibold))
                                    .padding(.horizontal, 16)
                                    .padding(.vertical, 10)
                                    .background(Color.secondary.opacity(0.15), in: Capsule())
                                    .foregroundStyle(.primary)
                            }
                        }
                    }

                    // Body text
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(4)
                    }

                    // Map
                    if let lat = post.latitude, let lon = post.longitude {
                        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lon)
                        Map(initialPosition: .region(MKCoordinateRegion(
                            center: coord,
                            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                        ))) {
                            Marker(post.title, coordinate: coord)
                                .tint(post.hintColor)
                        }
                        .frame(height: 200)
                        .cornerRadius(14)
                        .disabled(true)
                    }

                    // Agent + time
                    HStack(spacing: 6) {
                        Circle().fill(post.hintColor).frame(width: 8, height: 8)
                        Text(post.agentName).font(.caption).foregroundStyle(.secondary)
                        Text("·").foregroundStyle(.secondary)
                        Text(post.relativeTime).font(.caption).foregroundStyle(.secondary)
                    }

                    Divider()
                    PostDetailEngagementBar(post: post)
                }
                .padding(16)
            }
        }
        .ignoresSafeArea(edges: .top)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill").foregroundStyle(.secondary)
                }
            }
        }
    }
}
