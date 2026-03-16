import Foundation

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
    let postType: String?
    let createdAt: String

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
        case postType = "post_type"
        case createdAt = "created_at"
    }
}
