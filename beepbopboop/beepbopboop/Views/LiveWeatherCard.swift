import SwiftUI

struct LiveWeatherCard: View {
    let weather: WeatherData
    @State private var animateParticles = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private var code: Int { weather.current.conditionCode }
    private var isDay: Bool { weather.current.isDay }
    private var gradientColors: [Color] { WeatherData.gradient(for: code, isDay: isDay) }
    private var textColor: Color { WeatherData.accentColor(for: code, isDay: isDay) }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Hero section with gradient + icon + temp
            ZStack(alignment: .bottomLeading) {
                // Background gradient
                LinearGradient(
                    colors: gradientColors,
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )

                // Particle overlay
                if !reduceMotion {
                    particleOverlay
                }

                // Decorative large icon in background
                Image(systemName: WeatherData.icon(for: code, isDay: isDay))
                    .font(.system(size: 120, weight: .thin))
                    .foregroundStyle(textColor.opacity(0.12))
                    .offset(x: 140, y: -20)

                // Content overlay
                VStack(alignment: .leading, spacing: 6) {
                    // Header
                    HStack(spacing: 6) {
                        Circle()
                            .fill(.cyan)
                            .frame(width: 8, height: 8)
                        Text("Weather")
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(textColor.opacity(0.8))
                        Text("NOW")
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(textColor)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(textColor.opacity(0.15))
                            .cornerRadius(4)
                        Spacer()
                    }

                    Spacer()

                    // Icon + temperature
                    HStack(alignment: .top, spacing: 12) {
                        Image(systemName: WeatherData.icon(for: code, isDay: isDay))
                            .font(.system(size: 44))
                            .foregroundStyle(iconPrimary, iconSecondary)
                            .symbolRenderingMode(.palette)
                            .modifier(WeatherSymbolAnimation(code: code, reduceMotion: reduceMotion))

                        VStack(alignment: .leading, spacing: 2) {
                            Text("\(Int(weather.current.tempC.rounded()))°")
                                .font(.system(size: 56, weight: .thin, design: .rounded))
                                .foregroundStyle(textColor)
                            Text(weather.current.condition)
                                .font(.headline.weight(.semibold))
                                .foregroundStyle(textColor.opacity(0.9))
                        }

                        Spacer()

                        // Feels like + wind
                        VStack(alignment: .trailing, spacing: 4) {
                            Spacer()
                            Label("Feels \(Int(weather.current.feelsLikeC.rounded()))°", systemImage: "thermometer.medium")
                                .font(.caption.weight(.medium))
                                .foregroundStyle(textColor.opacity(0.7))
                            Label("\(Int(weather.current.windSpeedKmh.rounded())) km/h", systemImage: "wind")
                                .font(.caption.weight(.medium))
                                .foregroundStyle(textColor.opacity(0.7))
                            if weather.current.uvIndex >= 3 {
                                Label("UV \(Int(weather.current.uvIndex.rounded()))", systemImage: "sun.max.trianglebadge.exclamationmark")
                                    .font(.caption.weight(.medium))
                                    .foregroundStyle(.yellow.opacity(0.9))
                            }
                        }
                    }
                }
                .padding(16)
            }
            .frame(height: 200)

            // Hourly forecast strip
            if !weather.hourly.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 16) {
                        ForEach(weather.hourly.prefix(8)) { hour in
                            VStack(spacing: 6) {
                                Text(hour.hourLabel)
                                    .font(.caption2.weight(.medium))
                                    .foregroundStyle(.secondary)
                                Image(systemName: WeatherData.icon(for: hour.conditionCode, isDay: isDay))
                                    .font(.system(size: 18))
                                    .foregroundStyle(iconPrimary(for: hour.conditionCode), iconSecondary(for: hour.conditionCode))
                                    .symbolRenderingMode(.palette)
                                    .frame(height: 22)
                                Text("\(Int(hour.tempC.rounded()))°")
                                    .font(.subheadline.weight(.semibold))
                                if hour.precipProbability > 0 {
                                    Text("\(hour.precipProbability)%")
                                        .font(.caption2)
                                        .foregroundStyle(.cyan)
                                } else {
                                    Text(" ")
                                        .font(.caption2)
                                }
                            }
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 12)
                }
                .background(Color(.secondarySystemGroupedBackground))
            }

            // 5-day forecast
            if weather.daily.count > 1 {
                VStack(spacing: 0) {
                    ForEach(weather.daily.prefix(5)) { day in
                        HStack(spacing: 0) {
                            Text(day.dayLabel)
                                .font(.subheadline.weight(.medium))
                                .frame(width: 50, alignment: .leading)

                            Image(systemName: WeatherData.icon(for: day.conditionCode))
                                .font(.system(size: 16))
                                .foregroundStyle(iconPrimary(for: day.conditionCode), iconSecondary(for: day.conditionCode))
                                .symbolRenderingMode(.palette)
                                .frame(width: 28)

                            if day.precipProbability > 0 {
                                Text("\(day.precipProbability)%")
                                    .font(.caption2)
                                    .foregroundStyle(.cyan)
                                    .frame(width: 32, alignment: .leading)
                            } else {
                                Spacer()
                                    .frame(width: 32)
                            }

                            Spacer()

                            Text("\(Int(day.lowC.rounded()))°")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                                .frame(width: 32, alignment: .trailing)

                            // Temperature bar
                            tempBar(low: day.lowC, high: day.highC)
                                .frame(width: 80, height: 4)
                                .padding(.horizontal, 8)

                            Text("\(Int(day.highC.rounded()))°")
                                .font(.subheadline.weight(.medium))
                                .frame(width: 32, alignment: .leading)
                        }
                        .padding(.horizontal, 16)
                        .padding(.vertical, 8)
                    }
                }
                .background(Color(.secondarySystemGroupedBackground))
            }
        }
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .shadow(color: .black.opacity(0.12), radius: 12, x: 0, y: 4)
        .onAppear { animateParticles = true }
    }

    // MARK: - Temperature Bar

    private func tempBar(low: Double, high: Double) -> some View {
        let allLows = weather.daily.map(\.lowC)
        let allHighs = weather.daily.map(\.highC)
        let minTemp = allLows.min() ?? low
        let maxTemp = allHighs.max() ?? high
        let range = max(maxTemp - minTemp, 1)

        let startFraction = (low - minTemp) / range
        let endFraction = (high - minTemp) / range

        return GeometryReader { geo in
            ZStack(alignment: .leading) {
                Capsule()
                    .fill(Color(.systemGray5))
                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [.cyan, .orange],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .frame(width: geo.size.width * (endFraction - startFraction))
                    .offset(x: geo.size.width * startFraction)
            }
        }
    }

    // MARK: - Icon Colors

    private var iconPrimary: Color {
        iconPrimary(for: code)
    }

    private var iconSecondary: Color {
        iconSecondary(for: code)
    }

    private func iconPrimary(for code: Int) -> Color {
        switch code {
        case 0, 1:     return .yellow
        case 2:        return .white
        case 3:        return Color(.systemGray3)
        case 45, 48:   return Color(.systemGray4)
        case 51...67:  return Color(.systemGray3)
        case 71...86:  return Color(.systemGray3)
        case 95...99:  return Color(.systemGray3)
        default:       return .cyan
        }
    }

    private func iconSecondary(for code: Int) -> Color {
        switch code {
        case 0, 1:     return .orange
        case 2:        return .yellow
        case 3:        return Color(.systemGray4)
        case 45, 48:   return Color(.systemGray5)
        case 51...57:  return .cyan
        case 61...67:  return .blue
        case 71...86:  return .white
        case 80...82:  return .cyan
        case 95...99:  return .yellow
        default:       return .yellow
        }
    }

    // MARK: - Particle Overlay

    @ViewBuilder
    private var particleOverlay: some View {
        switch code {
        case 51...67, 80...82:
            // Rain particles
            TimelineView(.animation(minimumInterval: 0.05)) { _ in
                Canvas { context, size in
                    for i in 0..<40 {
                        let seed = Double(i) * 137.508
                        let x = (seed.truncatingRemainder(dividingBy: size.width))
                        let phase = Date.now.timeIntervalSinceReferenceDate * 120 + seed
                        let y = phase.truncatingRemainder(dividingBy: Double(size.height))
                        let dropHeight: CGFloat = code >= 65 ? 14 : 8
                        context.opacity = 0.3
                        context.fill(
                            Path(CGRect(x: x, y: y, width: 1.5, height: dropHeight)),
                            with: .color(.white)
                        )
                    }
                }
            }
            .allowsHitTesting(false)
        case 71...86:
            // Snow particles
            TimelineView(.animation(minimumInterval: 0.05)) { _ in
                Canvas { context, size in
                    for i in 0..<25 {
                        let seed = Double(i) * 137.508
                        let phase = Date.now.timeIntervalSinceReferenceDate * 20 + seed
                        let x = (seed + sin(phase * 0.5) * 30).truncatingRemainder(dividingBy: Double(size.width))
                        let y = (phase * 0.7).truncatingRemainder(dividingBy: Double(size.height))
                        let radius: CGFloat = CGFloat(2 + (seed.truncatingRemainder(dividingBy: 3)))
                        context.opacity = 0.5
                        context.fill(
                            Path(ellipseIn: CGRect(x: x, y: y, width: radius, height: radius)),
                            with: .color(.white)
                        )
                    }
                }
            }
            .allowsHitTesting(false)
        case 95...99:
            // Lightning flash
            TimelineView(.animation(minimumInterval: 0.1)) { timeline in
                let flash = Int(timeline.date.timeIntervalSinceReferenceDate * 10) % 80 == 0
                Rectangle()
                    .fill(.white.opacity(flash ? 0.3 : 0))
                    .allowsHitTesting(false)
            }
        default:
            EmptyView()
        }
    }
}

// MARK: - Weather Symbol Animation

private struct WeatherSymbolAnimation: ViewModifier {
    let code: Int
    let reduceMotion: Bool

    func body(content: Content) -> some View {
        if reduceMotion {
            content
        } else {
            switch code {
            case 51...67, 80...82:
                content.symbolEffect(.variableColor.iterative, isActive: true)
            case 0, 1:
                content.symbolEffect(.pulse, isActive: true)
            case 95...99:
                content.symbolEffect(.bounce, isActive: true)
            default:
                content.symbolEffect(.breathe, isActive: true)
            }
        }
    }
}
