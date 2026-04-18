import SwiftUI
import MapKit
import Combine

struct CalendarDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss
    @State private var now = Date()

    private let timer = Timer.publish(every: 1, on: .main, in: .common).autoconnect()

    private var eventDate: Date? {
        let detector = try? NSDataDetector(types: NSTextCheckingResult.CheckingType.date.rawValue)
        let range = NSRange(post.title.startIndex..., in: post.title)
        if let match = detector?.firstMatch(in: post.title, range: range) {
            return match.date
        }
        // Try body
        let bRange = NSRange(post.body.startIndex..., in: post.body)
        return detector?.firstMatch(in: post.body, range: bRange)?.date
    }

    private var countdown: String? {
        guard let date = eventDate else { return nil }
        let diff = date.timeIntervalSince(now)
        guard diff > 0 else { return nil }
        let days = Int(diff) / 86400
        let hours = (Int(diff) % 86400) / 3600
        let minutes = (Int(diff) % 3600) / 60
        if days > 0 { return "In \(days)d \(hours)h" }
        if hours > 0 { return "In \(hours)h \(minutes)m" }
        return "In \(minutes) minutes"
    }

    private var monthDayYear: (month: String, day: String, weekday: String)? {
        guard let date = eventDate else { return nil }
        let mf = DateFormatter(); mf.dateFormat = "MMM"
        let df = DateFormatter(); df.dateFormat = "d"
        let wf = DateFormatter(); wf.dateFormat = "EEEE"
        return (mf.string(from: date), df.string(from: date), wf.string(from: date))
    }

    private var timeString: String? {
        guard let date = eventDate else { return nil }
        let tf = DateFormatter(); tf.dateFormat = "h:mm a"
        return tf.string(from: date)
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                // Hero — ticket stub aesthetic
                VStack(spacing: 0) {
                    LinearGradient(
                        colors: [post.hintColor, post.hintColor.opacity(0.7)],
                        startPoint: .topLeading, endPoint: .bottomTrailing
                    )
                    .frame(height: 220)
                    .overlay {
                        VStack(spacing: 8) {
                            if let mdy = monthDayYear {
                                Text(mdy.month.uppercased())
                                    .font(.title3.weight(.bold))
                                    .foregroundStyle(.white.opacity(0.8))
                                    .kerning(3)
                                Text(mdy.day)
                                    .font(.system(size: 80, weight: .black))
                                    .foregroundStyle(.white)
                                    .lineLimit(1)
                                Text(mdy.weekday)
                                    .font(.headline.weight(.medium))
                                    .foregroundStyle(.white.opacity(0.85))
                                if let t = timeString {
                                    Text(t)
                                        .font(.body)
                                        .foregroundStyle(.white.opacity(0.75))
                                }
                            } else {
                                Image(systemName: "calendar")
                                    .font(.system(size: 60))
                                    .foregroundStyle(.white)
                                Text(post.title)
                                    .font(.title2.weight(.bold))
                                    .foregroundStyle(.white)
                                    .multilineTextAlignment(.center)
                                    .padding(.horizontal, 20)
                            }
                        }
                    }

                    // Dashed perforated divider
                    Canvas { ctx, size in
                        let dashWidth: CGFloat = 8
                        let gapWidth: CGFloat = 5
                        var x: CGFloat = 0
                        while x < size.width {
                            let rect = CGRect(x: x, y: size.height / 2 - 1, width: dashWidth, height: 2)
                            ctx.fill(Path(rect), with: .color(.secondary.opacity(0.4)))
                            x += dashWidth + gapWidth
                        }
                    }
                    .frame(height: 12)
                }

                VStack(alignment: .leading, spacing: 20) {
                    // Countdown
                    if let cd = countdown {
                        HStack {
                            Image(systemName: "timer")
                                .foregroundStyle(post.hintColor)
                            Text(cd)
                                .font(.headline.weight(.semibold))
                                .foregroundStyle(post.hintColor)
                            Spacer()
                        }
                        .padding(12)
                        .background(post.hintColor.opacity(0.1), in: RoundedRectangle(cornerRadius: 10))
                    }

                    // Title & body
                    Text(post.title)
                        .font(.title2.weight(.bold))

                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .foregroundStyle(.secondary)
                            .lineSpacing(4)
                    }

                    // Map if coordinates available
                    if let lat = post.latitude, let lon = post.longitude {
                        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lon)
                        Map(initialPosition: .region(MKCoordinateRegion(
                            center: coord,
                            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                        ))) {
                            Marker(post.title, coordinate: coord)
                                .tint(post.hintColor)
                        }
                        .frame(height: 180)
                        .cornerRadius(12)
                        .disabled(true)
                    }

                    // CTAs
                    VStack(spacing: 10) {
                        if let lat = post.latitude, let lon = post.longitude,
                           let url = URL(string: "maps://?daddr=\(lat),\(lon)") {
                            Link(destination: url) {
                                Label("Get Directions", systemImage: "map")
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 14)
                                    .background(post.hintColor, in: RoundedRectangle(cornerRadius: 12))
                                    .foregroundStyle(.white)
                                    .font(.subheadline.weight(.semibold))
                            }
                        }
                        if let extURL = post.externalURL, !extURL.isEmpty, let url = URL(string: extURL) {
                            Link(destination: url) {
                                Label("Open Event", systemImage: "arrow.up.right.square")
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 14)
                                    .background(Color.secondary.opacity(0.15), in: RoundedRectangle(cornerRadius: 12))
                                    .foregroundStyle(.primary)
                                    .font(.subheadline.weight(.semibold))
                            }
                        }
                    }

                    Divider()
                    PostDetailEngagementBar(post: post)
                }
                .padding(16)
            }
        }
        .navigationTitle("Event")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill").foregroundStyle(.secondary)
                }
            }
        }
        .onReceive(timer) { now = $0 }
    }
}
