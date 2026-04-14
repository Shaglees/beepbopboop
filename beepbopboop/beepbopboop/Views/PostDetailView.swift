import SwiftUI
import MapKit
import UIKit

struct PostDetailView: View {
    let post: Post
    @AppStorage private var isBookmarked: Bool

    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    private var bodyLines: [String] {
        post.body.components(separatedBy: "\n").filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Hint-specific header
                hintHeader

                VStack(alignment: .leading, spacing: 16) {
                    // Agent + relative time
                    HStack(spacing: 6) {
                        Circle()
                            .fill(post.hintColor)
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

                    // Body — hint-aware rendering
                    bodyContent

                    // Image (if available)
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

                    // Metadata
                    GlassEffectContainer(spacing: 8) {
                        HStack(spacing: 8) {
                            Text(post.typeLabel)
                                .font(.caption2.weight(.semibold))
                                .foregroundColor(post.typeColor)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .glassEffect(.regular.tint(post.typeColor), in: .capsule)

                            if post.displayHintValue != .card && post.hintLabel.lowercased() != (post.postType ?? "").lowercased() {
                                Label(post.hintLabel, systemImage: post.hintIcon)
                                    .font(.caption2.weight(.semibold))
                                    .foregroundColor(post.hintColor)
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .glassEffect(.regular.tint(post.hintColor), in: .capsule)
                            }

                            if let locality = post.locality, !locality.isEmpty {
                                localityLink
                            }
                        }

                        Label(formattedDate, systemImage: "clock")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    Divider()

                    // Engagement bar
                    engagementBar
                }
                .padding()
            }
        }
        .navigationTitle(post.hintLabel)
        .navigationBarTitleDisplayMode(.inline)
    }

    // MARK: - Body Content (hint-aware)

    @ViewBuilder
    private var bodyContent: some View {
        switch post.displayHintValue {
        case .digest:
            digestBody
        case .brief:
            briefBody
        case .weather:
            weatherBody
        default:
            LinkableText(post.body, font: .preferredFont(forTextStyle: .body))
        }
    }

    private var digestBody: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(bodyLines.enumerated()), id: \.offset) { index, line in
                HStack(alignment: .top, spacing: 12) {
                    Text("\(index + 1)")
                        .font(.title3.weight(.bold).monospacedDigit())
                        .foregroundColor(.teal)
                        .frame(width: 28, alignment: .trailing)

                    Text(line.trimmingCharacters(in: .whitespaces))
                        .font(.body)
                        .foregroundColor(.primary)
                }
                .padding(.vertical, 10)

                if index < bodyLines.count - 1 {
                    Divider()
                        .padding(.leading, 40)
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    private var briefBody: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(bodyLines.enumerated()), id: \.offset) { index, line in
                HStack(alignment: .top, spacing: 12) {
                    Image(systemName: "circle")
                        .font(.system(size: 8))
                        .foregroundColor(.secondary)
                        .frame(width: 20, alignment: .center)
                        .padding(.top, 6)

                    Text(line.trimmingCharacters(in: .whitespaces))
                        .font(.body)
                        .foregroundColor(.primary)
                }
                .padding(.vertical, 10)

                if index < bodyLines.count - 1 {
                    Divider()
                        .padding(.leading, 32)
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    private var weatherBody: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(bodyLines.enumerated()), id: \.offset) { index, line in
                let parts = line.split(separator: ":", maxSplits: 1)
                if parts.count == 2 {
                    HStack {
                        Text(parts[0].trimmingCharacters(in: .whitespaces))
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                        Spacer()
                        Text(parts[1].trimmingCharacters(in: .whitespaces))
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.primary)
                    }
                    .padding(.vertical, 8)
                } else {
                    Text(line.trimmingCharacters(in: .whitespaces))
                        .font(.body)
                        .foregroundColor(.primary)
                        .padding(.vertical, 8)
                }

                if index < bodyLines.count - 1 {
                    Divider()
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    // MARK: - Hint-Specific Headers

    @ViewBuilder
    private var hintHeader: some View {
        switch post.displayHintValue {
        case .weather:
            weatherHeader
        case .deal:
            dealHeader
        case .calendar, .event:
            dateHeader.padding()
        case .comparison:
            comparisonHeader.padding(.horizontal).padding(.top)
        case .digest:
            digestHeader.padding(.horizontal).padding(.top)
        case .brief:
            briefHeader.padding(.horizontal).padding(.top)
        default:
            EmptyView()
        }
    }

    private var weatherHeader: some View {
        let weather = DetailWeatherInfo.detect(from: post.title + " " + post.body)
        return HStack(spacing: 16) {
            Image(systemName: weather.icon)
                .font(.system(size: 48))
                .foregroundStyle(weather.primaryColor, weather.secondaryColor)

            VStack(alignment: .leading, spacing: 4) {
                Text(weather.label)
                    .font(.title3.weight(.semibold))
                    .foregroundColor(.primary)
                if let locality = post.locality, !locality.isEmpty {
                    Label(locality, systemImage: "location")
                        .font(.subheadline)
                        .foregroundColor(.white.opacity(0.8))
                }
            }
            Spacer()
        }
        .padding(20)
        .background(
            LinearGradient(
                colors: [weather.primaryColor.opacity(0.25), weather.secondaryColor.opacity(0.15)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
    }

    private var dealHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "tag.fill")
                .font(.title2)
            Text("DEAL")
                .font(.title3.weight(.black))
            Spacer()
            if let locality = post.locality, !locality.isEmpty {
                Text(locality)
                    .font(.subheadline)
                    .foregroundColor(.white.opacity(0.8))
            }
        }
        .foregroundColor(.white)
        .padding(20)
        .background(
            LinearGradient(
                colors: [.pink, .orange],
                startPoint: .leading,
                endPoint: .trailing
            )
        )
    }

    private var dateHeader: some View {
        let parts = extractDateParts()
        return HStack(spacing: 16) {
            VStack(spacing: 2) {
                Text(parts.month)
                    .font(.caption.weight(.bold))
                    .foregroundColor(post.hintColor)
                    .textCase(.uppercase)
                Text(parts.day)
                    .font(.system(size: 36, weight: .bold))
                    .foregroundColor(.primary)
            }
            .frame(width: 72, height: 76)
            .background(post.hintColor.opacity(0.1))
            .cornerRadius(14)

            VStack(alignment: .leading, spacing: 6) {
                Text(post.displayHintValue == .event ? "Event" : "Calendar")
                    .font(.subheadline.weight(.semibold))
                    .foregroundColor(post.hintColor)
                if let locality = post.locality, !locality.isEmpty {
                    Label(locality, systemImage: "location")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                if post.displayHintValue == .event, let extURL = post.externalURL, !extURL.isEmpty, let url = URL(string: extURL) {
                    Link(destination: url) {
                        Label("Get Tickets", systemImage: "arrow.up.right.square")
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(post.hintColor)
                    }
                }
            }
            Spacer()
        }
    }

    private var comparisonHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "arrow.left.arrow.right")
                .font(.title3)
            Text("Comparison")
                .font(.headline.weight(.semibold))
            Spacer()
        }
        .foregroundColor(.mint)
        .padding(14)
        .background(.mint.opacity(0.1))
        .cornerRadius(12)
    }

    private var digestHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "list.bullet.rectangle.fill")
                .font(.title3)
            Text("Digest")
                .font(.headline.weight(.semibold))
            Spacer()
            Text("\(bodyLines.count) items")
                .font(.subheadline)
                .foregroundColor(.teal.opacity(0.7))
        }
        .foregroundColor(.teal)
        .padding(14)
        .background(.teal.opacity(0.1))
        .cornerRadius(12)
    }

    private var briefHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "checklist")
                .font(.title3)
            Text("Brief")
                .font(.headline.weight(.semibold))
            Spacer()
            Text("\(bodyLines.count) items")
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
        .foregroundColor(.primary)
        .padding(14)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    // MARK: - Engagement Bar

    private var engagementBar: some View {
        HStack(spacing: 0) {
            Button {
                withAnimation(.bouncy) {
                    isBookmarked.toggle()
                }
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
            } label: {
                Label(
                    isBookmarked ? "Bookmarked" : "Bookmark",
                    systemImage: isBookmarked ? "bookmark.fill" : "bookmark"
                )
                .font(.subheadline)
                .foregroundColor(isBookmarked ? post.typeColor : .secondary)
                .symbolEffect(.bounce, value: isBookmarked)
                .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)

            Spacer()

            ShareLink(item: shareText) {
                Label("Share", systemImage: "square.and.arrow.up")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            if let externalURL = post.externalURL, !externalURL.isEmpty, let url = URL(string: externalURL) {
                Spacer()
                    .frame(width: 20)
                Link(destination: url) {
                    Label("Open", systemImage: "arrow.up.right.square")
                        .font(.subheadline)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
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

    private var formattedDate: String {
        let formatters: [ISO8601DateFormatter] = {
            let f1 = ISO8601DateFormatter()
            f1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            let f2 = ISO8601DateFormatter()
            f2.formatOptions = [.withInternetDateTime]
            return [f1, f2]
        }()
        for f in formatters {
            if let date = f.date(from: post.createdAt) {
                let df = DateFormatter()
                df.dateStyle = .medium
                df.timeStyle = .short
                return df.string(from: date)
            }
        }
        return post.createdAt
    }

    private func extractDateParts() -> (month: String, day: String) {
        let detector = try? NSDataDetector(types: NSTextCheckingResult.CheckingType.date.rawValue)
        let range = NSRange(post.title.startIndex..., in: post.title)
        if let match = detector?.firstMatch(in: post.title, range: range), let date = match.date {
            let monthF = DateFormatter()
            monthF.dateFormat = "MMM"
            return (monthF.string(from: date), "\(Calendar.current.component(.day, from: date))")
        }

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
        let monthF = DateFormatter()
        monthF.dateFormat = "MMM"
        return (monthF.string(from: date), "\(Calendar.current.component(.day, from: date))")
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

// MARK: - Detail Weather Info

private struct DetailWeatherInfo {
    let icon: String
    let label: String
    let primaryColor: Color
    let secondaryColor: Color

    static func detect(from text: String) -> DetailWeatherInfo {
        let lower = text.lowercased()

        if lower.contains("snow") || lower.contains("blizzard") {
            return DetailWeatherInfo(icon: "cloud.snow.fill", label: "Snow", primaryColor: .gray, secondaryColor: .white)
        }
        if lower.contains("thunder") || lower.contains("lightning") || lower.contains("storm") {
            return DetailWeatherInfo(icon: "cloud.bolt.rain.fill", label: "Thunderstorm", primaryColor: .gray, secondaryColor: .yellow)
        }
        if lower.contains("heavy rain") || lower.contains("downpour") || lower.contains("torrential") {
            return DetailWeatherInfo(icon: "cloud.heavyrain.fill", label: "Heavy Rain", primaryColor: .gray, secondaryColor: .blue)
        }
        if lower.contains("rain") || lower.contains("drizzle") || lower.contains("shower") {
            return DetailWeatherInfo(icon: "cloud.rain.fill", label: "Rain", primaryColor: .gray, secondaryColor: .cyan)
        }
        if lower.contains("partly cloudy") || lower.contains("partly sunny") || lower.contains("mix of sun") {
            return DetailWeatherInfo(icon: "cloud.sun.fill", label: "Partly Cloudy", primaryColor: .cyan, secondaryColor: .yellow)
        }
        if lower.contains("overcast") || lower.contains("cloudy") {
            return DetailWeatherInfo(icon: "cloud.fill", label: "Cloudy", primaryColor: .gray, secondaryColor: .gray)
        }
        if lower.contains("fog") || lower.contains("mist") || lower.contains("haze") {
            return DetailWeatherInfo(icon: "cloud.fog.fill", label: "Fog", primaryColor: .gray, secondaryColor: .secondary)
        }
        if lower.contains("clear") || lower.contains("sunny") {
            return DetailWeatherInfo(icon: "sun.max.fill", label: "Clear", primaryColor: .yellow, secondaryColor: .orange)
        }
        if lower.contains("wind") || lower.contains("gusty") || lower.contains("breezy") {
            return DetailWeatherInfo(icon: "wind", label: "Windy", primaryColor: .cyan, secondaryColor: .gray)
        }
        return DetailWeatherInfo(icon: "cloud.sun.fill", label: "Partly Cloudy", primaryColor: .cyan, secondaryColor: .yellow)
    }
}
