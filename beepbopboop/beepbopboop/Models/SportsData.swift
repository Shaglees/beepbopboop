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
    }

    /// Whether the game is finished.
    var isFinal: Bool {
        let s = status.lowercased()
        return s.hasPrefix("final") || s == "ft" || s == "full time"
    }

    /// Status pill color.
    var statusColor: Color {
        if isLive { return .red }
        if isFinal { return .green }
        let s = status.lowercased()
        if s.contains("ot") || s.contains("overtime") || s.contains("so") { return .orange }
        return .secondary
    }

    /// Formatted game time for matchup cards.
    var formattedGameTime: String? {
        guard let gt = gameTime else { return nil }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime]
        guard let date = iso.date(from: gt) else { return gt }
        let f = DateFormatter()
        f.dateFormat = "h:mm a"
        return f.string(from: date)
    }

    var formattedGameDate: String? {
        guard let gt = gameTime else { return nil }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime]
        guard let date = iso.date(from: gt) else { return nil }
        let f = DateFormatter()
        f.dateFormat = "EEEE, MMM d"
        return f.string(from: date)
    }

    /// Countdown string like "IN 3 HOURS" or "TOMORROW" for matchup cards.
    var countdown: String? {
        guard let gt = gameTime else { return nil }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime]
        guard let date = iso.date(from: gt) else { return nil }
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
