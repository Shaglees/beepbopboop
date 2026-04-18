import Foundation
import SwiftUI

// MARK: - Video Game Data (game_release / game_review display hints)

struct VideoGameData: Codable {
    let igdbId: Int?
    let steamAppId: Int?
    let title: String
    let coverUrl: String?
    let rating: Int?                    // 0–100 aggregated Metacritic score
    let ratingCount: Int?
    let releaseDate: String?            // "YYYY-MM-DD"
    let status: String                  // "upcoming" | "released" | "early_access"
    let platforms: [String]
    let genres: [String]
    let developer: String?
    let publisher: String?
    let steamPrice: String?
    let steamDiscount: Int?             // % discount, 0 if none
    let steamPositiveRatio: Double?     // 0.0–1.0 from Steam review data
    let storeUrl: String?
}

// MARK: - Computed properties

extension VideoGameData {
    var isUpcoming: Bool { status == "upcoming" }
    var isEarlyAccess: Bool { status == "early_access" }

    /// Formatted release date for display (e.g. "May 30, 2026").
    var formattedReleaseDate: String? {
        guard let dateStr = releaseDate else { return nil }
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        guard let date = formatter.date(from: dateStr) else { return dateStr }
        formatter.dateFormat = "MMM d, yyyy"
        return formatter.string(from: date)
    }

    /// Days until release (positive) or days since release (negative).
    var daysUntilRelease: Int? {
        guard let dateStr = releaseDate else { return nil }
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = TimeZone(identifier: "UTC")
        guard let date = formatter.date(from: dateStr) else { return nil }
        return Calendar.current.dateComponents([.day], from: Date(), to: date).day
    }

    /// Human-readable countdown string for upcoming releases.
    var releaseCountdown: String? {
        guard isUpcoming, let days = daysUntilRelease else { return nil }
        if days <= 0 { return "OUT NOW" }
        if days == 1 { return "RELEASES TOMORROW" }
        if days < 7 { return "RELEASES IN \(days) DAYS" }
        if days < 30 { return "RELEASES IN \(days / 7) WEEK\(days / 7 == 1 ? "" : "S")" }
        return "RELEASES \(formattedReleaseDate ?? releaseDate ?? "")"
    }

    /// Score circle color — green 75+, yellow 60–74, red below 60.
    var ratingColor: Color {
        guard let r = rating else { return .secondary }
        if r >= 75 { return Color(red: 0.2, green: 0.78, blue: 0.35) }
        if r >= 60 { return Color(red: 0.96, green: 0.77, blue: 0.19) }
        return Color(red: 0.93, green: 0.26, blue: 0.21)
    }

    /// Primary genre for accent color selection.
    var accentColor: Color {
        let genre = genres.first?.lowercased() ?? ""
        if genre.contains("rpg") || genre.contains("role") { return Color(red: 0.58, green: 0.27, blue: 0.96) }
        if genre.contains("shooter") || genre.contains("fps") || genre.contains("action") { return Color(red: 0.93, green: 0.35, blue: 0.18) }
        if genre.contains("sport") || genre.contains("racing") { return Color(red: 0.13, green: 0.70, blue: 0.37) }
        if genre.contains("strategy") || genre.contains("simulation") { return Color(red: 0.18, green: 0.52, blue: 0.93) }
        if genre.contains("horror") || genre.contains("survival") { return Color(red: 0.65, green: 0.10, blue: 0.10) }
        return Color(red: 0.25, green: 0.47, blue: 0.85)
    }

    /// Platform badge color for known platforms.
    static func platformColor(_ platform: String) -> Color {
        switch platform.lowercased() {
        case "ps5", "ps4", "playstation 5", "playstation 4":
            return Color(red: 0.0, green: 0.19, blue: 0.53)
        case "xbox series x", "xbox one", "xbox":
            return Color(red: 0.063, green: 0.486, blue: 0.063)
        case "switch", "nintendo switch":
            return Color(red: 0.90, green: 0.10, blue: 0.22)
        default:
            return Color(white: 0.35)
        }
    }

    /// Abbreviated platform name for badge display.
    static func platformAbbr(_ platform: String) -> String {
        switch platform.lowercased() {
        case "playstation 5": return "PS5"
        case "playstation 4": return "PS4"
        case "xbox series x", "xbox series x|s": return "Xbox"
        case "pc (windows)", "pc": return "PC"
        case "nintendo switch": return "Switch"
        case "macos", "mac": return "Mac"
        case "ios": return "iOS"
        case "android": return "Android"
        default: return platform
        }
    }

    /// Steam positive review percentage string (e.g. "84%").
    var steamPositivePercent: String? {
        guard let ratio = steamPositiveRatio else { return nil }
        return "\(Int(ratio * 100))%"
    }

    /// Formatted Steam review count from ratingCount (e.g. "12.4k").
    var formattedReviewCount: String? {
        guard let count = ratingCount else { return nil }
        if count >= 1000 { return String(format: "%.1fk", Double(count) / 1000.0) }
        return "\(count)"
    }

    /// Price with discount applied, e.g. "$29.99 (-40%)".
    var displayPrice: String? {
        guard let price = steamPrice else { return nil }
        if let disc = steamDiscount, disc > 0 {
            return "\(price) (-\(disc)%)"
        }
        return price
    }
}
