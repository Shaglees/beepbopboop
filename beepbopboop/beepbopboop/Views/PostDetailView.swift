import SwiftUI
import MapKit

struct PostDetailView: View {
    let post: Post
    @AppStorage private var isBookmarked: Bool

    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // Agent + relative time
                HStack(spacing: 6) {
                    Circle()
                        .fill(post.typeColor)
                        .frame(width: 10, height: 10)
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
                    .font(.title2)
                    .fontWeight(.bold)

                // Body — linkable text for tappable URLs and phone numbers
                LinkableText(post.body, font: .preferredFont(forTextStyle: .body))

                // Image (if available) — taps through to external link
                if let imageURL = post.imageURL, !imageURL.isEmpty, let imgSrc = URL(string: imageURL) {
                    if let externalURL = post.externalURL, !externalURL.isEmpty, let dest = URL(string: externalURL) {
                        Link(destination: dest) {
                            postImage(url: imgSrc)
                        }
                    } else {
                        postImage(url: imgSrc)
                    }
                }

                // Map (if coordinates available)
                if let lat = post.latitude, let lon = post.longitude {
                    Map(initialPosition: .region(MKCoordinateRegion(
                        center: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                        span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                    ))) {
                        Marker(post.markerLabel, systemImage: post.typeIcon, coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lon))
                            .tint(post.typeColor)
                    }
                    .frame(height: 200)
                    .cornerRadius(12)
                }

                // Metadata row: type pill + hint pill + locality + full date
                VStack(alignment: .leading, spacing: 8) {
                    HStack(spacing: 8) {
                        Text(post.typeLabel)
                            .font(.caption2.weight(.semibold))
                            .foregroundColor(post.typeColor)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(post.typeColor.opacity(0.12))
                            .cornerRadius(4)

                        // Show hint pill when it differs from type
                        if post.displayHintValue != .card && post.hintLabel.lowercased() != (post.postType ?? "").lowercased() {
                            Label(post.hintLabel, systemImage: post.hintIcon)
                                .font(.caption2.weight(.semibold))
                                .foregroundColor(post.hintColor)
                                .padding(.horizontal, 7)
                                .padding(.vertical, 3)
                                .background(post.hintColor.opacity(0.12))
                                .cornerRadius(4)
                        }

                        if let locality = post.locality, !locality.isEmpty {
                            localityLink
                        }
                    }

                    Label(post.createdAt, systemImage: "clock")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                Divider()

                // Engagement bar
                HStack(spacing: 0) {
                    // Bookmark toggle
                    Button {
                        isBookmarked.toggle()
                    } label: {
                        Label(
                            isBookmarked ? "Bookmarked" : "Bookmark",
                            systemImage: isBookmarked ? "bookmark.fill" : "bookmark"
                        )
                        .font(.subheadline)
                        .foregroundColor(isBookmarked ? post.typeColor : .secondary)
                    }
                    .buttonStyle(.plain)

                    Spacer()

                    // Share
                    ShareLink(item: shareText) {
                        Label("Share", systemImage: "square.and.arrow.up")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    // External link (when URL present)
                    if let externalURL = post.externalURL, !externalURL.isEmpty, let url = URL(string: externalURL) {
                        Spacer()
                            .frame(width: 20)
                        Link(destination: url) {
                            Label("Open", systemImage: "arrow.up.right.square")
                                .font(.subheadline)
                        }
                    }
                }

                Spacer()
            }
            .padding()
        }
        .navigationTitle(post.typeLabel)
        .navigationBarTitleDisplayMode(.inline)
    }

    // MARK: - Helpers

    @ViewBuilder
    private func postImage(url: URL) -> some View {
        AsyncImage(url: url) { phase in
            switch phase {
            case .success(let image):
                image
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxWidth: .infinity)
                    .cornerRadius(12)
            case .failure:
                EmptyView()
            default:
                ProgressView()
                    .frame(height: 200)
                    .frame(maxWidth: .infinity)
            }
        }
    }

    private var shareText: String {
        var text = post.title + "\n\n" + post.body
        if let url = post.externalURL, !url.isEmpty {
            text += "\n\n" + url
        }
        return text
    }

    @ViewBuilder
    private var localityLink: some View {
        if let locality = post.locality, !locality.isEmpty {
            if post.isSourceAttribution {
                if let ext = post.externalURL, !ext.isEmpty, let url = URL(string: ext) {
                    Link(destination: url) {
                        Label(locality, systemImage: "link")
                            .font(.subheadline)
                    }
                } else {
                    Label(locality, systemImage: "link")
                        .font(.subheadline)
                }
            } else if let lat = post.latitude, let lon = post.longitude,
                      let mapURL = URL(string: "https://maps.apple.com/?ll=\(lat),\(lon)&q=\(locality.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? locality)") {
                Link(destination: mapURL) {
                    Label(locality, systemImage: "location")
                        .font(.subheadline)
                }
            } else {
                Label(locality, systemImage: "location")
                    .font(.subheadline)
            }
        }
    }
}
