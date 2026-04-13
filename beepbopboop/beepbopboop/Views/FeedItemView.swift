import SwiftUI
import MapKit

struct FeedItemView: View {
    let post: Post

    private var isBookmarked: Bool {
        UserDefaults.standard.bool(forKey: "bookmark_\(post.id)")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Agent + relative time
            HStack(spacing: 6) {
                Circle()
                    .fill(post.typeColor)
                    .frame(width: 8, height: 8)
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                Text("·")
                    .foregroundColor(.secondary)
                Text(post.relativeTime)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            // Title
            Text(post.title)
                .font(.headline)
                .lineLimit(2)

            // Body
            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

            // Optional media: image (any type) or compact map (places with coords)
            if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(maxHeight: 200)
                            .clipped()
                            .cornerRadius(10)
                    case .failure:
                        EmptyView()
                    default:
                        ProgressView()
                            .frame(height: 120)
                            .frame(maxWidth: .infinity)
                    }
                }
            } else if post.postTypeValue == .place,
                      let lat = post.latitude, let lon = post.longitude {
                Map(initialPosition: .region(MKCoordinateRegion(
                    center: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                    span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
                ))) {
                    Marker(post.markerLabel, systemImage: post.typeIcon, coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lon))
                        .tint(post.typeColor)
                }
                .frame(height: 120)
                .cornerRadius(10)
                .allowsHitTesting(false)
            }

            // Bottom row: type pill + locality ··· engagement icons
            HStack(spacing: 6) {
                // Type pill
                Text(post.typeLabel)
                    .font(.caption2.weight(.semibold))
                    .foregroundColor(post.typeColor)
                    .padding(.horizontal, 7)
                    .padding(.vertical, 3)
                    .background(post.typeColor.opacity(0.12))
                    .cornerRadius(4)

                if let locality = post.locality, !locality.isEmpty {
                    Text("·")
                        .foregroundColor(.secondary)
                    Label(locality, systemImage: post.isSourceAttribution ? "link" : "location")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }

                Spacer()

                // Engagement icons (display-only)
                HStack(spacing: 14) {
                    Image(systemName: "heart")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Image(systemName: "arrow.up.right")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Image(systemName: "square.and.arrow.up")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                        .font(.caption)
                        .foregroundColor(isBookmarked ? post.typeColor : .secondary)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }
}
