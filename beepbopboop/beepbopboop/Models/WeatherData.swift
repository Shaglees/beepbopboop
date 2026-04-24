import Foundation
import SwiftUI

struct WeatherData: Codable {
    let current: CurrentWeather
    let hourly: [HourlyForecast]
    let daily: [DailyForecast]
    let location: WeatherLocation

    struct CurrentWeather: Codable {
        let tempC: Double
        let feelsLikeC: Double
        let humidity: Int
        let windSpeedKmh: Double
        let uvIndex: Double
        let isDay: Bool
        let condition: String
        let conditionCode: Int

        enum CodingKeys: String, CodingKey {
            case tempC = "temp_c"
            case feelsLikeC = "feels_like_c"
            case humidity
            case windSpeedKmh = "wind_speed_kmh"
            case uvIndex = "uv_index"
            case isDay = "is_day"
            case condition
            case conditionCode = "condition_code"
        }
    }

    struct HourlyForecast: Codable, Identifiable {
        var id: String { time }
        let time: String
        let tempC: Double
        let condition: String
        let conditionCode: Int
        let precipProbability: Int

        enum CodingKeys: String, CodingKey {
            case time
            case tempC = "temp_c"
            case condition
            case conditionCode = "condition_code"
            case precipProbability = "precip_probability"
        }

        var hourLabel: String {
            // "2026-04-16T14:00" → "2pm"
            guard time.count >= 13 else { return time }
            let hourStr = String(time.dropFirst(11).prefix(2))
            guard let hour = Int(hourStr) else { return hourStr }
            if hour == 0 { return "12am" }
            if hour == 12 { return "12pm" }
            return hour < 12 ? "\(hour)am" : "\(hour - 12)pm"
        }
    }

    struct DailyForecast: Codable, Identifiable {
        var id: String { date }
        let date: String
        let highC: Double
        let lowC: Double
        let condition: String
        let conditionCode: Int
        let sunrise: String
        let sunset: String
        let precipProbability: Int

        enum CodingKeys: String, CodingKey {
            case date
            case highC = "high_c"
            case lowC = "low_c"
            case condition
            case conditionCode = "condition_code"
            case sunrise, sunset
            case precipProbability = "precip_probability"
        }

        var dayLabel: String {
            // "2026-04-16" → "Thu"
            let formatter = DateFormatter()
            formatter.dateFormat = "yyyy-MM-dd"
            guard let d = formatter.date(from: date) else { return date }
            if Calendar.current.isDateInToday(d) { return "Today" }
            if Calendar.current.isDateInTomorrow(d) { return "Tomorrow" }
            formatter.dateFormat = "EEE"
            return formatter.string(from: d)
        }
    }

    struct WeatherLocation: Codable {
        let latitude: Double
        let longitude: Double
        let timezone: String
    }
}

// MARK: - Visual Properties

extension WeatherData {

    /// SF Symbol name for a WMO weather code.
    static func icon(for code: Int, isDay: Bool = true) -> String {
        switch code {
        case 0:       return isDay ? "sun.max.fill" : "moon.stars.fill"
        case 1:       return isDay ? "sun.min.fill" : "moon.fill"
        case 2:       return isDay ? "cloud.sun.fill" : "cloud.moon.fill"
        case 3:       return "cloud.fill"
        case 45, 48:  return "cloud.fog.fill"
        case 51, 53, 55, 56, 57:
                      return "cloud.drizzle.fill"
        case 61, 63, 80, 81:
                      return "cloud.rain.fill"
        case 65, 82:  return "cloud.heavyrain.fill"
        case 66, 67:  return "thermometer.snowflake"
        case 71, 73, 77, 85:
                      return "cloud.snow.fill"
        case 75, 86:  return "cloud.snow.fill"
        case 95:      return "cloud.bolt.fill"
        case 96, 99:  return "cloud.bolt.rain.fill"
        default:      return "cloud.sun.fill"
        }
    }

    /// Gradient colors based on condition + day/night.
    static func gradient(for code: Int, isDay: Bool) -> [Color] {
        if !isDay {
            switch code {
            case 0, 1:    return [Color(hex: 0x0F1B2D), Color(hex: 0x1A2A4A)]
            case 2, 3:    return [Color(hex: 0x1A2233), Color(hex: 0x2C3E50)]
            case 45, 48:  return [Color(hex: 0x1C1C2E), Color(hex: 0x2D2D44)]
            case 51...67: return [Color(hex: 0x141E30), Color(hex: 0x243B55)]
            case 71...86: return [Color(hex: 0x1A1A2E), Color(hex: 0x3D3D6B)]
            case 95...99: return [Color(hex: 0x0D0D1A), Color(hex: 0x2C1654)]
            default:      return [Color(hex: 0x141E30), Color(hex: 0x243B55)]
            }
        }
        switch code {
        case 0:       return [Color(hex: 0x56CCF2), Color(hex: 0x2F80ED)]        // Clear sky
        case 1:       return [Color(hex: 0x74B9FF), Color(hex: 0x0984E3)]        // Mainly clear
        case 2:       return [Color(hex: 0xA8D8EA), Color(hex: 0x6C9BCF)]        // Partly cloudy
        case 3:       return [Color(hex: 0x9BAEC1), Color(hex: 0x6B7F99)]        // Overcast
        case 45, 48:  return [Color(hex: 0xBDC3C7), Color(hex: 0x95A5A6)]        // Fog
        case 51, 53, 55:
                      return [Color(hex: 0x89ABD9), Color(hex: 0x5D7EA7)]        // Drizzle
        case 56, 57:  return [Color(hex: 0x8EB8D9), Color(hex: 0x6A8CAD)]        // Freezing drizzle
        case 61, 63, 80, 81:
                      return [Color(hex: 0x667DB6), Color(hex: 0x4A6FA5)]        // Rain
        case 65, 82:  return [Color(hex: 0x485B7C), Color(hex: 0x2C3E50)]        // Heavy rain
        case 66, 67:  return [Color(hex: 0x7F8C8D), Color(hex: 0x5D6D7E)]        // Freezing rain
        case 71, 73, 77, 85:
                      return [Color(hex: 0xD5DDE5), Color(hex: 0xA8B8CC)]        // Snow
        case 75, 86:  return [Color(hex: 0xC4D4E0), Color(hex: 0x8FA4BB)]        // Heavy snow
        case 95:      return [Color(hex: 0x3D3D6B), Color(hex: 0x1A1A2E)]        // Thunderstorm
        case 96, 99:  return [Color(hex: 0x2C1654), Color(hex: 0x0D0D1A)]        // Thunderstorm + hail
        default:      return [Color(hex: 0x74B9FF), Color(hex: 0x0984E3)]
        }
    }

    /// Accent color for text overlays.
    static func accentColor(for code: Int, isDay: Bool) -> Color {
        if !isDay { return .white }
        switch code {
        case 0, 1, 2:        return .white
        case 3, 45, 48:      return .white
        case 51...67:         return .white
        case 71...86:         return Color(hex: 0x2C3E50)
        case 95...99:         return Color(hex: 0xF1C40F)
        default:              return .white
        }
    }
}

// MARK: - Hex Color Helper

extension Color {
    init(hex: UInt, opacity: Double = 1.0) {
        self.init(
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: opacity
        )
    }
}
