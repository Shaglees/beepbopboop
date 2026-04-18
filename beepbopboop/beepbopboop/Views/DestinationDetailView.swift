import SwiftUI
import MapKit

struct DestinationDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: TravelData? { post.travelData }

    private static let accent = Color(red: 0.024, green: 0.714, blue: 0.831)

    // MARK: - Hero URL resolution

    private var heroURL: URL? {
        if let raw = data?.heroImageUrl, !raw.isEmpty, let url = URL(string: raw) { return url }
        if let img = post.heroImage, let url = URL(string: img.url) { return url }
        if let raw = post.imageURL, !raw.isEmpty, let url = URL(string: raw) { return url }
        return nil
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: - Full-bleed hero (280pt)
                heroSection

                // MARK: - Content
                VStack(alignment: .leading, spacing: 20) {

                    // Agent + time line
                    agentLine

                    // Weather widget
                    if data?.currentTempC != nil || data?.currentCondition != nil {
                        weatherWidget
                    }

                    // Known-for tags
                    if let known = data?.knownFor, !known.isEmpty {
                        knownForSection(known)
                    }

                    // Info grid (best time, flight, currency, visa, timezone)
                    if let d = data {
                        infoGrid(d)
                    }

                    Divider()

                    // Body text
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .foregroundStyle(.primary)
                            .lineSpacing(4)
                    }

                    // Map preview
                    if let d = data {
                        mapSection(lat: d.latitude, lon: d.longitude, label: d.city)
                    }

                    // Weekend forecast card
                    if let forecast = data?.weekendForecast, !forecast.isEmpty {
                        forecastCard(forecast)
                    }

                    // Wiki "Learn More" button
                    if let wikiStr = data?.wikiUrl, !wikiStr.isEmpty, let url = URL(string: wikiStr) {
                        wikiButton(url: url)
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
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    // MARK: - Hero Section

    @ViewBuilder
    private var heroSection: some View {
        if let url = heroURL {
            GeometryReader { geo in
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img):
                        img.resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: geo.size.width, height: 280)
                            .clipped()
                            .overlay {
                                LinearGradient(
                                    colors: [.clear, Color(.systemBackground)],
                                    startPoint: .center,
                                    endPoint: .bottom
                                )
                            }
                            .overlay(alignment: .bottomLeading) {
                                heroLocationLabel
                                    .padding(16)
                            }
                    case .failure:
                        placeholderHero(width: geo.size.width)
                    default:
                        Color.secondary.opacity(0.2)
                            .frame(width: geo.size.width, height: 280)
                            .overlay(ProgressView())
                    }
                }
            }
            .frame(height: 280)
        } else {
            GeometryReader { geo in
                ZStack(alignment: .bottomLeading) {
                    LinearGradient(
                        colors: [Self.accent, Self.accent.opacity(0.5), Color.cyan.opacity(0.3)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                    .frame(width: geo.size.width, height: 280)

                    Image(systemName: "airplane")
                        .font(.system(size: 100, weight: .ultraLight))
                        .foregroundStyle(.white.opacity(0.15))
                        .frame(width: geo.size.width, height: 280)

                    LinearGradient(
                        colors: [.clear, Color(.systemBackground)],
                        startPoint: .center,
                        endPoint: .bottom
                    )
                    .frame(width: geo.size.width, height: 280)

                    heroLocationLabel
                        .padding(16)
                }
            }
            .frame(height: 280)
        }
    }

    @ViewBuilder
    private func placeholderHero(width: CGFloat) -> some View {
        ZStack(alignment: .bottomLeading) {
            LinearGradient(
                colors: [Self.accent, Self.accent.opacity(0.5), Color.cyan.opacity(0.3)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .frame(width: width, height: 280)

            Image(systemName: "airplane")
                .font(.system(size: 100, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.15))
                .frame(width: width, height: 280)

            LinearGradient(
                colors: [.clear, Color(.systemBackground)],
                startPoint: .center,
                endPoint: .bottom
            )
            .frame(width: width, height: 280)

            heroLocationLabel
                .padding(16)
        }
    }

    @ViewBuilder
    private var heroLocationLabel: some View {
        if let d = data {
            VStack(alignment: .leading, spacing: 4) {
                Text(d.city)
                    .font(.system(size: 32, weight: .bold))
                    .foregroundStyle(.white)
                    .shadow(radius: 4)
                Text(d.country)
                    .font(.title3)
                    .foregroundStyle(.white.opacity(0.85))
                    .shadow(radius: 3)
            }
        }
    }

    // MARK: - Agent Line

    private var agentLine: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(Self.accent)
                .frame(width: 10, height: 10)
            Text(post.agentName)
                .font(.subheadline.weight(.medium))
            Text("·")
                .foregroundStyle(.secondary)
            Text(post.relativeTime)
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
    }

    // MARK: - Weather Widget

    @ViewBuilder
    private var weatherWidget: some View {
        HStack(spacing: 12) {
            let iconName = weatherIcon(for: data?.currentCondition ?? "")
            Image(systemName: iconName)
                .font(.system(size: 28, weight: .medium))
                .foregroundStyle(weatherIconColor(for: data?.currentCondition ?? ""))
                .frame(width: 36, height: 36)

            VStack(alignment: .leading, spacing: 2) {
                if let temp = data?.currentTempC {
                    Text("\(Int(temp.rounded()))°C")
                        .font(.title2.weight(.bold))
                        .foregroundStyle(.primary)
                }
                if let condition = data?.currentCondition, !condition.isEmpty {
                    Text(condition)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer()

            Text("Current weather")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(Self.accent.opacity(0.08), in: RoundedRectangle(cornerRadius: 14))
        .overlay(
            RoundedRectangle(cornerRadius: 14)
                .stroke(Self.accent.opacity(0.2), lineWidth: 1)
        )
    }

    private func weatherIcon(for condition: String) -> String {
        let lower = condition.lowercased()
        if lower.contains("thunder") || lower.contains("storm") { return "cloud.bolt.rain.fill" }
        if lower.contains("snow") || lower.contains("blizzard") { return "cloud.snow.fill" }
        if lower.contains("heavy rain") || lower.contains("downpour") { return "cloud.heavyrain.fill" }
        if lower.contains("rain") || lower.contains("drizzle") || lower.contains("shower") { return "cloud.rain.fill" }
        if lower.contains("fog") || lower.contains("mist") { return "cloud.fog.fill" }
        if lower.contains("partly cloudy") || lower.contains("partly sunny") { return "cloud.sun.fill" }
        if lower.contains("cloudy") || lower.contains("overcast") { return "cloud.fill" }
        if lower.contains("clear") || lower.contains("sunny") { return "sun.max.fill" }
        if lower.contains("wind") { return "wind" }
        // Fall back to condition code if available
        if let code = data?.currentConditionCode {
            return WeatherData.icon(for: code, isDay: true)
        }
        return "cloud.sun.fill"
    }

    private func weatherIconColor(for condition: String) -> Color {
        let lower = condition.lowercased()
        if lower.contains("thunder") || lower.contains("storm") { return .yellow }
        if lower.contains("snow") || lower.contains("blizzard") { return .cyan }
        if lower.contains("rain") || lower.contains("drizzle") { return .blue }
        if lower.contains("fog") || lower.contains("mist") { return .gray }
        if lower.contains("cloudy") || lower.contains("overcast") { return .gray }
        if lower.contains("clear") || lower.contains("sunny") { return .orange }
        return Self.accent
    }

    // MARK: - Known-For Tags

    private func knownForSection(_ items: [String]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("KNOWN FOR")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(Self.accent)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(items, id: \.self) { item in
                        Text(item)
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(Self.accent)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(Self.accent.opacity(0.1), in: Capsule())
                            .overlay(
                                Capsule()
                                    .stroke(Self.accent.opacity(0.3), lineWidth: 1)
                            )
                    }
                }
                .padding(.vertical, 2)
            }
        }
    }

    // MARK: - Info Grid

    @ViewBuilder
    private func infoGrid(_ d: TravelData) -> some View {
        let items = infoItems(for: d)
        if !items.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                Text("TRAVEL INFO")
                    .font(.system(size: 11, weight: .bold))
                    .tracking(1.5)
                    .foregroundStyle(.secondary)

                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                    ForEach(items, id: \.title) { item in
                        infoCard(item)
                    }
                }
            }
        }
    }

    private struct InfoItem {
        let icon: String
        let title: String
        let value: String
        let iconColor: Color
    }

    private func infoItems(for d: TravelData) -> [InfoItem] {
        var items: [InfoItem] = []

        if let best = d.bestTimeToVisit, !best.isEmpty {
            items.append(InfoItem(
                icon: "calendar.badge.clock",
                title: "Best Time",
                value: best,
                iconColor: Self.accent
            ))
        }

        if let price = d.flightPriceFrom, !price.isEmpty {
            let note = d.flightPriceNote.map { " \($0)" } ?? ""
            items.append(InfoItem(
                icon: "airplane",
                title: "Flights",
                value: price + note,
                iconColor: Color(hex: 0xFBBF24)
            ))
        }

        if let currency = d.currency, !currency.isEmpty {
            items.append(InfoItem(
                icon: "dollarsign.circle",
                title: "Currency",
                value: currency,
                iconColor: .green
            ))
        }

        if let tz = d.timeZone, !tz.isEmpty {
            items.append(InfoItem(
                icon: "clock.fill",
                title: "Timezone",
                value: tz,
                iconColor: .purple
            ))
        }

        if let required = d.visaRequired {
            items.append(InfoItem(
                icon: required ? "exclamationmark.shield.fill" : "checkmark.seal.fill",
                title: "Visa",
                value: required ? "Visa Required" : "No Visa Required",
                iconColor: required ? .red : .green
            ))
        }

        return items
    }

    private func infoCard(_ item: InfoItem) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 6) {
                Image(systemName: item.icon)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(item.iconColor)
                Text(item.title)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
            }
            Text(item.value)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(.primary)
                .lineLimit(2)
                .fixedSize(horizontal: false, vertical: true)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
    }

    // MARK: - Map Preview

    private func mapSection(lat: Double, lon: Double, label: String) -> some View {
        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lon)
        return VStack(alignment: .leading, spacing: 10) {
            Text("LOCATION")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            Map(initialPosition: .region(MKCoordinateRegion(
                center: coord,
                span: MKCoordinateSpan(latitudeDelta: 0.5, longitudeDelta: 0.5)
            ))) {
                Marker(label, systemImage: "airplane", coordinate: coord)
                    .tint(Self.accent)
            }
            .frame(height: 160)
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(Self.accent.opacity(0.15), lineWidth: 1)
            )
        }
    }

    // MARK: - Weekend Forecast Card

    private func forecastCard(_ forecast: String) -> some View {
        HStack(spacing: 10) {
            Image(systemName: "cloud.sun.fill")
                .font(.system(size: 18, weight: .medium))
                .foregroundStyle(Self.accent)

            VStack(alignment: .leading, spacing: 2) {
                Text("WEEKEND FORECAST")
                    .font(.system(size: 10, weight: .bold))
                    .tracking(1.2)
                    .foregroundStyle(Self.accent)
                Text(forecast)
                    .font(.subheadline)
                    .foregroundStyle(.primary)
            }

            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(Self.accent.opacity(0.06), in: RoundedRectangle(cornerRadius: 12))
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(Self.accent.opacity(0.15), lineWidth: 1)
        )
    }

    // MARK: - Wiki Button

    private func wikiButton(url: URL) -> some View {
        Link(destination: url) {
            Label("Learn More on Wikipedia", systemImage: "globe")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(Self.accent)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .background(Self.accent.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(Self.accent.opacity(0.3), lineWidth: 1)
                )
        }
    }
}
