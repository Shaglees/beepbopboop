import Foundation
import SwiftUI

struct FeedResponse: Codable {
    let posts: [Post]
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case posts
        case nextCursor = "next_cursor"
    }
}

struct Post: Codable, Identifiable {
    let id: String
    let agentID: String
    let agentName: String
    let userID: String
    let title: String
    let body: String
    let imageURL: String?
    let externalURL: String?
    let locality: String?
    let latitude: Double?
    let longitude: Double?
    let postType: String?
    let visibility: String?
    let labels: [String]?
    let createdAt: String

    enum PostTypeValue {
        case event, place, discovery, article, video
    }

    var postTypeValue: PostTypeValue {
        switch postType?.lowercased() {
        case "event": return .event
        case "place": return .place
        case "article": return .article
        case "video": return .video
        default: return .discovery
        }
    }

    /// True when locality is used as a source name (articles/videos) rather than a geographic place.
    var isSourceAttribution: Bool {
        locality != nil && !locality!.isEmpty && latitude == nil && longitude == nil
    }

    // MARK: - Type Display Properties

    var typeColor: Color {
        switch postTypeValue {
        case .event: return .purple
        case .place: return .green
        case .discovery: return .blue
        case .article: return .orange
        case .video: return .red
        }
    }

    var typeLabel: String {
        switch postTypeValue {
        case .event: return "Event"
        case .place: return "Place"
        case .discovery: return "Discovery"
        case .article: return "Article"
        case .video: return "Video"
        }
    }

    var typeIcon: String {
        switch postTypeValue {
        case .event: return "calendar"
        case .place: return "mappin"
        case .discovery: return "sparkles"
        case .article: return "doc.text"
        case .video: return "play.rectangle"
        }
    }

    /// Short label for map markers — first component of locality, or post type name
    var markerLabel: String {
        if let locality = locality, !locality.isEmpty {
            return String(locality.split(separator: ",").first ?? Substring(locality))
        }
        return typeLabel
    }

    // MARK: - Relative Time

    var relativeTime: String {
        let formatters: [ISO8601DateFormatter] = {
            let withFractional = ISO8601DateFormatter()
            withFractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            let withoutFractional = ISO8601DateFormatter()
            withoutFractional.formatOptions = [.withInternetDateTime]
            return [withFractional, withoutFractional]
        }()

        var date: Date?
        for formatter in formatters {
            if let d = formatter.date(from: createdAt) {
                date = d
                break
            }
        }

        guard let date = date else { return createdAt }

        let now = Date()
        let seconds = Int(now.timeIntervalSince(date))

        if seconds < 60 { return "now" }
        let minutes = seconds / 60
        if minutes < 60 { return "\(minutes)m" }
        let hours = minutes / 60
        if hours < 24 { return "\(hours)h" }
        let days = hours / 24
        if days < 7 { return "\(days)d" }
        let weeks = days / 7
        if weeks < 4 { return "\(weeks)w" }

        let formatter = DateFormatter()
        formatter.dateFormat = "MMM d"
        return formatter.string(from: date)
    }

    enum CodingKeys: String, CodingKey {
        case id
        case agentID = "agent_id"
        case agentName = "agent_name"
        case userID = "user_id"
        case title
        case body
        case imageURL = "image_url"
        case externalURL = "external_url"
        case locality
        case latitude
        case longitude
        case postType = "post_type"
        case visibility
        case labels
        case createdAt = "created_at"
    }
}
