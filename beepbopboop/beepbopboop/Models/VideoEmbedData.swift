import Foundation

/// Structured payload in `external_url` for `display_hint: video_embed`.
struct VideoEmbedData: Codable {
    let provider: String
    let videoId: String?
    let watchUrl: String?
    let embedUrl: String
    let thumbnailUrl: String?
    let channelTitle: String?
    let supportsPreviewCap: Bool?

    enum CodingKeys: String, CodingKey {
        case provider
        case videoId = "video_id"
        case watchUrl = "watch_url"
        case embedUrl = "embed_url"
        case thumbnailUrl = "thumbnail_url"
        case channelTitle = "channel_title"
        case supportsPreviewCap = "supports_preview_cap"
    }

    var previewCapEnabled: Bool { supportsPreviewCap == true }
}
