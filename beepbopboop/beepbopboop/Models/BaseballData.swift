import Foundation
import SwiftUI

// MARK: - Baseball Box Score Data

struct BaseballData: Codable {
    let sport: String
    let league: String
    let status: String          // "Final", "F/10" for extras
    let innings: Int?
    let extraInnings: Bool
    let home: TeamInfo
    let away: TeamInfo
    let winningPitcher: PitcherLine?
    let losingPitcher: PitcherLine?
    let savePitcher: SavePitcher?
    let keyBatter: BatterLine?
    let headline: String?
    let venue: String?
}

struct PitcherLine: Codable {
    let name: String
    let record: String          // "4-1"
    let era: String             // "2.34"
    let inningsPitched: Double  // 7.0, 5.1 = 5⅓
    let strikeouts: Int

    var formattedIP: String {
        let whole = Int(inningsPitched)
        let fraction = inningsPitched - Double(whole)
        if fraction < 0.05 { return "\(whole).0" }
        if fraction < 0.15 { return "\(whole).1" }
        if fraction < 0.25 { return "\(whole).2" }
        return String(format: "%.1f", inningsPitched)
    }
}

struct SavePitcher: Codable {
    let name: String
    let saves: Int
}

struct BatterLine: Codable {
    let name: String
    let team: String
    let hr: Int?
    let rbi: Int?
    let avg: String?
    let hits: Int?
    let atBats: Int?

    var summaryText: String {
        var parts: [String] = []
        if let hr, hr > 0 { parts.append(hr == 1 ? "HR" : "\(hr) HR") }
        if let rbi, rbi > 0 { parts.append("\(rbi) RBI") }
        if let hits, let ab = atBats { parts.append("\(hits)-\(ab)") }
        if let avg { parts.append(avg) }
        return parts.joined(separator: ", ")
    }
}
