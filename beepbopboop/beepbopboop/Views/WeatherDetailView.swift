import SwiftUI
import MapKit

struct WeatherDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: WeatherData? { post.weatherData }

    private func conditionGradient(code: Int, isDay: Bool) -> LinearGradient {
        let colors: [Color]
        switch code {
        case 1000: colors = isDay ? [.blue, Color(red: 0.4, green: 0.7, blue: 1)] : [Color(red: 0.05, green: 0.05, blue: 0.2), .indigo]
        case 1003, 1006: colors = [Color(red: 0.45, green: 0.6, blue: 0.8), Color(red: 0.6, green: 0.7, blue: 0.85)]
        case 1009: colors = [Color(red: 0.5, green: 0.5, blue: 0.55), Color(red: 0.65, green: 0.65, blue: 0.7)]
        case 1063, 1180...1201: colors = [Color(red: 0.25, green: 0.35, blue: 0.55), Color(red: 0.4, green: 0.5, blue: 0.7)]
        case 1210...1225: colors = [Color(red: 0.65, green: 0.75, blue: 0.9), Color(red: 0.8, green: 0.85, blue: 0.95)]
        default: colors = [Color(red: 0.35, green: 0.55, blue: 0.75), Color(red: 0.55, green: 0.7, blue: 0.9)]
        }
        return LinearGradient(colors: colors, startPoint: .top, endPoint: .bottom)
    }

    private func weatherIcon(code: Int) -> String {
        switch code {
        case 1000: return "sun.max.fill"
        case 1003: return "cloud.sun.fill"
        case 1006: return "cloud.fill"
        case 1009: return "smoke.fill"
        case 1030, 1135, 1147: return "cloud.fog.fill"
        case 1063, 1150...1153: return "cloud.drizzle.fill"
        case 1180...1201: return "cloud.rain.fill"
        case 1210...1225: return "cloud.snow.fill"
        case 1273...1282: return "cloud.bolt.rain.fill"
        default: return "cloud.sun.fill"
        }
    }

    var body: some View {
        GeometryReader { proxy in
            ScrollView {
                VStack(spacing: 0) {
                    hero(safeTop: proxy.safeAreaInsets.top)

                    VStack(alignment: .leading, spacing: 20) {
                        // Hourly forecast
                        if let hourly = data?.hourly, !hourly.isEmpty {
                            VStack(alignment: .leading, spacing: 10) {
                                Text("HOURLY")
                                    .font(.caption.weight(.bold))
                                    .foregroundStyle(.secondary)

                                ScrollView(.horizontal, showsIndicators: false) {
                                    HStack(spacing: 12) {
                                        ForEach(hourly.prefix(12), id: \.time) { h in
                                            VStack(spacing: 6) {
                                                Text(h.hourLabel)
                                                    .font(.caption)
                                                    .foregroundStyle(.secondary)
                                                Image(systemName: weatherIcon(code: h.conditionCode))
                                                    .font(.title3)
                                                    .symbolRenderingMode(.hierarchical)
                                                Text("\(Int(h.tempC))°")
                                                    .font(.subheadline.weight(.medium))
                                                if h.precipProbability > 20 {
                                                    Text("\(Int(h.precipProbability))%")
                                                        .font(.caption2)
                                                        .foregroundStyle(.blue)
                                                }
                                            }
                                            .padding(.horizontal, 10)
                                            .padding(.vertical, 12)
                                            .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 12))
                                        }
                                    }
                                    .padding(.horizontal, 16)
                                }
                                .padding(.horizontal, -16)
                            }
                        }

                        // Daily forecast
                        if let daily = data?.daily, !daily.isEmpty {
                            VStack(alignment: .leading, spacing: 10) {
                                Text("7-DAY FORECAST")
                                    .font(.caption.weight(.bold))
                                    .foregroundStyle(.secondary)

                                VStack(spacing: 0) {
                                    ForEach(daily.prefix(7), id: \.date) { day in
                                        HStack {
                                            Text(day.dayLabel)
                                                .font(.subheadline.weight(.medium))
                                                .lineLimit(1)
                                                .minimumScaleFactor(0.82)
                                                .frame(width: 74, alignment: .leading)
                                            Image(systemName: weatherIcon(code: day.conditionCode))
                                                .symbolRenderingMode(.hierarchical)
                                                .frame(width: 28)
                                            if day.precipProbability > 20 {
                                                Text("\(Int(day.precipProbability))%")
                                                    .font(.caption)
                                                    .foregroundStyle(.blue)
                                            }
                                            Spacer()
                                            Text("\(Int(day.lowC))°")
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                                .frame(width: 32)
                                            // Temp range bar
                                            GeometryReader { geo in
                                                let allHighs = daily.map { $0.highC }
                                                let allLows = daily.map { $0.lowC }
                                                let minT = allLows.min() ?? day.lowC
                                                let maxT = allHighs.max() ?? day.highC
                                                let range = maxT - minT
                                                let startFrac = range > 0 ? (day.lowC - minT) / range : 0
                                                let endFrac = range > 0 ? (day.highC - minT) / range : 1
                                                ZStack(alignment: .leading) {
                                                    Capsule().fill(Color.secondary.opacity(0.2)).frame(height: 4)
                                                    Capsule()
                                                        .fill(LinearGradient(colors: [.blue, .orange], startPoint: .leading, endPoint: .trailing))
                                                        .frame(width: geo.size.width * CGFloat(endFrac - startFrac), height: 4)
                                                        .offset(x: geo.size.width * CGFloat(startFrac))
                                                }
                                            }
                                            .frame(width: 60, height: 4)
                                            Text("\(Int(day.highC))°")
                                                .font(.subheadline.weight(.medium))
                                                .frame(width: 32, alignment: .trailing)
                                        }
                                        .padding(.vertical, 10)
                                        if day.date != daily.prefix(7).last?.date {
                                            Divider()
                                        }
                                    }
                                }
                                .padding(12)
                                .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 12))
                            }
                        }

                        // Current details grid
                        if let current = data?.current {
                            VStack(alignment: .leading, spacing: 10) {
                                Text("DETAILS")
                                    .font(.caption.weight(.bold))
                                    .foregroundStyle(.secondary)

                                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                                    WeatherStatCell(icon: "humidity.fill", label: "Humidity", value: "\(current.humidity)%")
                                    WeatherStatCell(icon: "wind", label: "Wind", value: "\(Int(current.windSpeedKmh)) km/h")
                                    WeatherStatCell(icon: "sun.max.fill", label: "UV Index", value: "\(current.uvIndex)")
                                    WeatherStatCell(icon: "thermometer.medium", label: "Feels Like", value: "\(Int(current.feelsLikeC))°C")
                                }
                            }
                        }

                        Divider()
                        PostDetailEngagementBar(post: post)
                    }
                    .padding(16)
                }
            }
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.hidden, for: .navigationBar)
    }

    private func hero(safeTop: CGFloat) -> some View {
        ZStack(alignment: .topTrailing) {
            conditionGradient(
                code: data?.current.conditionCode ?? 1000,
                isDay: data?.current.isDay ?? true
            )

            Button { dismiss() } label: {
                Image(systemName: "xmark")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(.white.opacity(0.92))
                    .frame(width: 44, height: 44)
                    .background(.white.opacity(0.18), in: Circle())
                    .overlay {
                        Circle().stroke(.white.opacity(0.28), lineWidth: 1)
                    }
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Close")
            .padding(.top, safeTop + 12)
            .padding(.trailing, 18)

            VStack(spacing: 10) {
                Text(post.title)
                    .font(.title3.weight(.semibold))
                    .multilineTextAlignment(.center)
                    .foregroundStyle(.white.opacity(0.92))
                    .lineLimit(2)
                    .minimumScaleFactor(0.82)

                Image(systemName: weatherIcon(code: data?.current.conditionCode ?? 1000))
                    .font(.system(size: 62))
                    .foregroundStyle(.white)
                    .symbolRenderingMode(.hierarchical)

                if let current = data?.current {
                    Text("\(Int(current.tempC))°")
                        .font(.system(size: 76, weight: .thin))
                        .foregroundStyle(.white)
                    Text(current.condition)
                        .font(.title3)
                        .foregroundStyle(.white.opacity(0.86))
                    Text("Feels like \(Int(current.feelsLikeC))°")
                        .font(.subheadline)
                        .foregroundStyle(.white.opacity(0.74))
                }
            }
            .padding(.horizontal, 72)
            .padding(.top, safeTop + 58)
            .padding(.bottom, 30)
        }
        .frame(minHeight: safeTop + 330)
    }
}

// MARK: - Weather Stat Cell

private struct WeatherStatCell: View {
    let icon: String
    let label: String
    let value: String

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: icon)
                .font(.title3)
                .foregroundStyle(.secondary)
                .frame(width: 28)
            VStack(alignment: .leading, spacing: 2) {
                Text(label).font(.caption).foregroundStyle(.secondary)
                Text(value).font(.subheadline.weight(.medium))
            }
            Spacer()
        }
        .padding(12)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
    }
}
