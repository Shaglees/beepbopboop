import Foundation
import SwiftUI

// MARK: - Music Data (Album + Concert)

struct MusicData: Codable {
    let type: String                // "album" | "concert"

    // Album fields
    let spotifyId: String?
    let title: String?
    let artist: String
    let artistSpotifyId: String?
    let albumType: String?          // "album" | "single" | "ep"
    let coverUrl: String?
    let releaseDate: String?
    let trackCount: Int?
    let label: String?
    let lastfmListeners: Int?
    let lastfmPlaycount: Int?
    let tags: [String]?
    let spotifyUrl: String?
    let previewUrl: String?

    // Concert fields
    let songkickId: Int?
    let venue: String?
    let venueAddress: String?
    let date: String?
    let doorsTime: String?
    let startTime: String?
    let ticketUrl: String?
    let onSale: Bool?
    let priceRange: String?
    let latitude: Double?
    let longitude: Double?

    var isAlbum: Bool { type == "album" }
    var isConcert: Bool { type == "concert" }

    var albumTypeDisplay: String {
        switch albumType?.lowercased() {
        case "single": return "SINGLE"
        case "ep":     return "EP"
        default:       return "ALBUM"
        }
    }

    var formattedListeners: String? {
        guard let count = lastfmListeners else { return nil }
        if count >= 1_000_000 {
            return String(format: "%.1fM", Double(count) / 1_000_000)
        } else if count >= 1_000 {
            return String(format: "%.0fK", Double(count) / 1_000)
        }
        return "\(count)"
    }

    var formattedPlaycount: String? {
        guard let count = lastfmPlaycount else { return nil }
        if count >= 1_000_000 {
            return String(format: "%.0fM", Double(count) / 1_000_000)
        } else if count >= 1_000 {
            return String(format: "%.0fK", Double(count) / 1_000)
        }
        return "\(count)"
    }

    var formattedDate: String? {
        guard let dateStr = date else { return nil }
        let parser = DateFormatter()
        parser.dateFormat = "yyyy-MM-dd"
        guard let parsed = parser.date(from: dateStr) else { return dateStr }
        let display = DateFormatter()
        display.dateFormat = "MMM d"
        return display.string(from: parsed)
    }

    var formattedReleaseDate: String? {
        guard let dateStr = releaseDate else { return nil }
        let parser = DateFormatter()
        parser.dateFormat = "yyyy-MM-dd"
        guard let parsed = parser.date(from: dateStr) else { return dateStr }
        let display = DateFormatter()
        display.dateFormat = "MMM d, yyyy"
        return display.string(from: parsed)
    }

    var monthAbbrev: String? {
        guard let dateStr = date else { return nil }
        let parser = DateFormatter()
        parser.dateFormat = "yyyy-MM-dd"
        guard let parsed = parser.date(from: dateStr) else { return nil }
        let display = DateFormatter()
        display.dateFormat = "MMM"
        return display.string(from: parsed).uppercased()
    }

    var dayNumber: String? {
        guard let dateStr = date else { return nil }
        let parser = DateFormatter()
        parser.dateFormat = "yyyy-MM-dd"
        guard let parsed = parser.date(from: dateStr) else { return nil }
        let display = DateFormatter()
        display.dateFormat = "d"
        return display.string(from: parsed)
    }
}

