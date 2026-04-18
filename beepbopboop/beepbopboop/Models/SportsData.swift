import Foundation
import SwiftUI

// MARK: - Scoreboard / Matchup Data

struct GameData: Codable {
    let sport: String?
    let league: String?
    let status: String             // "Final", "Live 2nd", "Scheduled", "OT", etc.
    let gameTime: String?          // ISO-8601 for scheduled games
    let home: TeamInfo
    let away: TeamInfo
    let headline: String?          // "Miller 2G 1A · Demko 31 saves"
    let venue: String?
    let broadcast: String?
    let series: String?            // "Game 3 · Series tied 1-1"
    // Soccer-specific fields (all optional for backward compatibility)
    let leagueShortName: String?
    let leagueColor: String?
    let matchday: String?
    let competition: String?
    let goalScorers: [GoalScorer]?
    let yellowCards: Int?
    let redCards: Int?
}

struct GoalScorer: Codable {
    let player: String
    let team: String        // matches TeamInfo.abbr
    let minute: Int
    let assist: String?
}

struct TeamInfo: Codable {
    let name: String
    let abbr: String
    let score: Int?
    let record: String?
    let color: String?             // Hex string e.g. "#00205B"

    var swiftUIColor: Color {
        guard let hex = color else { return .gray }
        return Color(hexString: hex)
    }
}

// MARK: - Standings Data

struct StandingsData: Codable {
    let league: String
    let leagueColor: String?
    let date: String               // "2026-04-16"
    let games: [StandingsGame]
    let headline: String?
}

struct StandingsGame: Codable, Identifiable {
    var id: String { "\(home)-\(away)-\(homeScore)-\(awayScore)" }
    let home: String               // Abbreviation
    let away: String
    let homeScore: Int
    let awayScore: Int
    let status: String
    let homeColor: String?
    let awayColor: String?

    var homeSwiftUIColor: Color {
        guard let hex = homeColor else { return .gray }
        return Color(hexString: hex)
    }

    var awaySwiftUIColor: Color {
        guard let hex = awayColor else { return .gray }
        return Color(hexString: hex)
    }
}

// MARK: - Hex String → Color

extension Color {
    init(hexString: String) {
        let cleaned = hexString.trimmingCharacters(in: .whitespaces).replacingOccurrences(of: "#", with: "")
        guard cleaned.count == 6, let val = UInt(cleaned, radix: 16) else {
            self = .gray
            return
        }
        self.init(hex: val)
    }
}

// MARK: - Sport Icon

extension GameData {
    var sportIcon: String {
        switch sport?.lowercased() {
        case "hockey":     return "figure.hockey"
        case "baseball":   return "figure.baseball"
        case "basketball": return "figure.basketball"
        case "soccer", "football":
                           return "figure.soccer"
        case "mma":        return "figure.martial.arts"
        case "golf":       return "figure.golf"
        case "tennis":     return "figure.tennis"
        default:           return "sportscourt"
        }
    }

    /// Whether the game is currently live.
    var isLive: Bool {
        let s = status.lowercased()
        return s.hasPrefix("live") || s.contains("period") || s.contains("quarter")
            || s.contains("half") || s.contains("inning")
            || s == "ht" || s == "1h" || s == "2h" || s.hasPrefix("et")
    }

    /// Whether the game is finished.
    var isFinal: Bool {
        let s = status.lowercased()
        return s.hasPrefix("final") || s == "ft" || s == "full time"
    }

    /// League accent color for soccer branding strips.
    var leagueAccentColor: Color {
        guard let hex = leagueColor else { return .white.opacity(0.3) }
        let c = Color(hexString: hex)
        return c == .gray ? Color(red: 0.6, green: 0.6, blue: 0.9) : c
    }

    /// Status pill color.
    var statusColor: Color {
        if isLive { return .red }
        if isFinal { return .green }
        let s = status.lowercased()
        if s.contains("ot") || s.contains("overtime") || s.contains("so") { return .orange }
        return .secondary
    }

    /// Parse game time ISO-8601 string into a Date.
    var gameDate: Date? {
        guard let gt = gameTime else { return nil }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = iso.date(from: gt) { return d }
        iso.formatOptions = [.withInternetDateTime]
        return iso.date(from: gt)
    }

    /// Formatted game time with timezone abbreviation (e.g. "7:30 PM EDT").
    var formattedGameTime: String? {
        guard let date = gameDate else { return gameTime }
        let f = DateFormatter()
        f.dateFormat = "h:mm a zzz"
        return f.string(from: date)
    }

    /// Formatted game time without timezone (for compact cards).
    var formattedGameTimeShort: String? {
        guard let date = gameDate else { return gameTime }
        let f = DateFormatter()
        f.dateFormat = "h:mm a"
        return f.string(from: date)
    }

    var formattedGameDate: String? {
        guard let date = gameDate else { return nil }
        let f = DateFormatter()
        f.dateFormat = "EEEE, MMM d"
        return f.string(from: date)
    }

    /// Countdown string like "IN 3 HOURS" or "TOMORROW" for matchup cards.
    var countdown: String? {
        guard let date = gameDate else { return nil }
        let seconds = Int(date.timeIntervalSince(Date()))
        guard seconds > 0 else { return nil }
        let minutes = seconds / 60
        let hours = minutes / 60
        let days = hours / 24
        if minutes < 60 { return "IN \(minutes) MIN" }
        if hours < 24 { return "IN \(hours) HOUR\(hours == 1 ? "" : "S")" }
        if days == 1 { return "TOMORROW" }
        if days < 7 { return "IN \(days) DAYS" }
        return nil
    }
}
