import SwiftUI
import MapKit

struct FeedItemView: View {
    let post: Post

    private var isBookmarked: Bool {
        UserDefaults.standard.bool(forKey: "bookmark_\(post.id)")
    }

    var body: some View {
        switch post.displayHintValue {
        case .weather:
            WeatherCard(post: post, isBookmarked: isBookmarked)
        case .brief, .digest:
            CompactCard(post: post, isBookmarked: isBookmarked)
        case .calendar, .event:
            DateCard(post: post, isBookmarked: isBookmarked)
        case .deal:
            DealCard(post: post, isBookmarked: isBookmarked)
        case .place:
            PlaceCard(post: post, isBookmarked: isBookmarked)
        default:
            StandardCard(post: post, isBookmarked: isBookmarked)
        }
    }
}

// MARK: - Shared Components

private struct CardHeader: View {
    let post: Post

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(post.hintColor)
                .frame(width: 8, height: 8)
            Text(post.agentName)
                .font(.subheadline.weight(.medium))
            Text("·")
                .foregroundColor(.secondary)
            Text(post.relativeTime)
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
    }
}

private struct CardFooter: View {
    let post: Post
    let isBookmarked: Bool

    var body: some View {
        HStack(spacing: 6) {
            // Hint pill (use hint instead of type when they differ)
            Text(post.hintLabel)
                .font(.caption2.weight(.semibold))
                .foregroundColor(post.hintColor)
                .padding(.horizontal, 7)
                .padding(.vertical, 3)
                .background(post.hintColor.opacity(0.12))
                .cornerRadius(4)

            // Show type pill too when hint differs from type
            if post.displayHintValue != .card && post.hintLabel.lowercased() != (post.postType ?? "").lowercased() {
                Text(post.typeLabel)
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }

            if let locality = post.locality, !locality.isEmpty {
                Text("·")
                    .foregroundColor(.secondary)
                Label(locality, systemImage: post.isSourceAttribution ? "link" : "location")
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }

            Spacer()

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
                    .foregroundColor(isBookmarked ? post.hintColor : .secondary)
            }
        }
    }
}

// MARK: - Standard Card (card, article, comparison)

private struct StandardCard: View {
    let post: Post
    let isBookmarked: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            Text(post.title)
                .font(.headline)
                .lineLimit(2)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

            // Article hint: hero image
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

            // Comparison badge
            if post.displayHintValue == .comparison {
                HStack(spacing: 4) {
                    Image(systemName: "arrow.left.arrow.right")
                    Text("Comparison")
                }
                .font(.caption2.weight(.medium))
                .foregroundColor(.mint)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(.mint.opacity(0.1))
                .cornerRadius(6)
            }

            CardFooter(post: post, isBookmarked: isBookmarked)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }
}

// MARK: - Weather Card

private struct WeatherCard: View {
    let post: Post
    let isBookmarked: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            HStack(alignment: .top, spacing: 12) {
                Image(systemName: "cloud.sun.fill")
                    .font(.system(size: 32))
                    .foregroundStyle(.cyan, .yellow)
                    .frame(width: 44)

                VStack(alignment: .leading, spacing: 4) {
                    Text(post.title)
                        .font(.headline)
                        .lineLimit(2)
                    Text(post.body)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(4)
                }
            }

            CardFooter(post: post, isBookmarked: isBookmarked)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(
            LinearGradient(
                colors: [.cyan.opacity(0.08), .orange.opacity(0.05)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
    }
}

// MARK: - Compact Card (brief, digest)

private struct CompactCard: View {
    let post: Post
    let isBookmarked: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            CardHeader(post: post)

            Text(post.title)
                .font(.subheadline.weight(.semibold))
                .lineLimit(1)

            // Show body as compact bullets — split on newlines
            let lines = post.body.components(separatedBy: "\n").filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
            VStack(alignment: .leading, spacing: 2) {
                ForEach(Array(lines.prefix(5).enumerated()), id: \.offset) { index, line in
                    HStack(alignment: .top, spacing: 6) {
                        if post.displayHintValue == .digest {
                            Text("\(index + 1).")
                                .font(.caption2.weight(.bold))
                                .foregroundColor(post.hintColor)
                                .frame(width: 16, alignment: .trailing)
                        } else {
                            Text("•")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                        Text(line.trimmingCharacters(in: .whitespaces))
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    }
                }
                if lines.count > 5 {
                    Text("+\(lines.count - 5) more")
                        .font(.caption2)
                        .foregroundColor(.tertiary)
                }
            }

            CardFooter(post: post, isBookmarked: isBookmarked)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }
}

// MARK: - Date Card (calendar, event)

private struct DateCard: View {
    let post: Post
    let isBookmarked: Bool

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            // Date badge
            VStack(spacing: 2) {
                Text(dateParts.month)
                    .font(.caption2.weight(.bold))
                    .foregroundColor(post.hintColor)
                    .textCase(.uppercase)
                Text(dateParts.day)
                    .font(.title2.weight(.bold))
                    .foregroundColor(.primary)
            }
            .frame(width: 48, height: 52)
            .background(post.hintColor.opacity(0.1))
            .cornerRadius(8)

            VStack(alignment: .leading, spacing: 6) {
                CardHeader(post: post)

                Text(post.title)
                    .font(.headline)
                    .lineLimit(2)

                Text(post.body)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineLimit(3)

                // Event: show location + external link
                if post.displayHintValue == .event {
                    if let locality = post.locality, !locality.isEmpty {
                        Label(locality, systemImage: "location")
                            .font(.caption)
                            .foregroundColor(post.hintColor)
                    }
                    if let extURL = post.externalURL, !extURL.isEmpty {
                        Label("Get Tickets", systemImage: "arrow.up.right.square")
                            .font(.caption.weight(.medium))
                            .foregroundColor(post.hintColor)
                    }
                }

                CardFooter(post: post, isBookmarked: isBookmarked)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    private var dateParts: (month: String, day: String) {
        let formatters: [ISO8601DateFormatter] = {
            let f1 = ISO8601DateFormatter()
            f1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            let f2 = ISO8601DateFormatter()
            f2.formatOptions = [.withInternetDateTime]
            return [f1, f2]
        }()
        var date = Date()
        for f in formatters {
            if let d = f.date(from: post.createdAt) { date = d; break }
        }
        let cal = Calendar.current
        let monthF = DateFormatter()
        monthF.dateFormat = "MMM"
        return (monthF.string(from: date), "\(cal.component(.day, from: date))")
    }
}

// MARK: - Deal Card

private struct DealCard: View {
    let post: Post
    let isBookmarked: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Deal accent banner
            HStack(spacing: 6) {
                Image(systemName: "tag.fill")
                    .font(.caption)
                Text("DEAL")
                    .font(.caption.weight(.black))
                Spacer()
                Text(post.relativeTime)
                    .font(.caption2)
                    .foregroundColor(.white.opacity(0.8))
            }
            .foregroundColor(.white)
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
            .background(
                LinearGradient(
                    colors: [.pink, .orange],
                    startPoint: .leading,
                    endPoint: .trailing
                )
            )
            .cornerRadius(8)

            Text(post.title)
                .font(.headline)
                .lineLimit(2)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

            if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(maxHeight: 160)
                            .clipped()
                            .cornerRadius(10)
                    case .failure:
                        EmptyView()
                    default:
                        ProgressView()
                            .frame(height: 100)
                            .frame(maxWidth: .infinity)
                    }
                }
            }

            CardFooter(post: post, isBookmarked: isBookmarked)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }
}

// MARK: - Place Card

private struct PlaceCard: View {
    let post: Post
    let isBookmarked: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Map first if coordinates available
            if let lat = post.latitude, let lon = post.longitude {
                Map(initialPosition: .region(MKCoordinateRegion(
                    center: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                    span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
                ))) {
                    Marker(post.markerLabel, systemImage: "mappin", coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lon))
                        .tint(.green)
                }
                .frame(height: 140)
                .cornerRadius(10)
                .allowsHitTesting(false)
            } else if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(maxHeight: 140)
                            .clipped()
                            .cornerRadius(10)
                    case .failure:
                        EmptyView()
                    default:
                        ProgressView()
                            .frame(height: 100)
                            .frame(maxWidth: .infinity)
                    }
                }
            }

            CardHeader(post: post)

            Text(post.title)
                .font(.headline)
                .lineLimit(2)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

            CardFooter(post: post, isBookmarked: isBookmarked)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }
}
