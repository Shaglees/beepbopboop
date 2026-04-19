import Foundation

struct AgentProfile: Codable, Identifiable {
    let id: String
    let userID: String
    let name: String
    let status: String
    let description: String
    let avatarURL: String
    let followerCount: Int
    let postCount: Int
    let createdAt: String
    var isFollowing: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case userID = "user_id"
        case name
        case status
        case description
        case avatarURL = "avatar_url"
        case followerCount = "follower_count"
        case postCount = "post_count"
        case createdAt = "created_at"
        case isFollowing = "is_following"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        userID = try c.decode(String.self, forKey: .userID)
        name = try c.decode(String.self, forKey: .name)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "active"
        description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        avatarURL = try c.decodeIfPresent(String.self, forKey: .avatarURL) ?? ""
        followerCount = try c.decodeIfPresent(Int.self, forKey: .followerCount) ?? 0
        postCount = try c.decodeIfPresent(Int.self, forKey: .postCount) ?? 0
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        isFollowing = try c.decodeIfPresent(Bool.self, forKey: .isFollowing) ?? false
    }
}

struct FollowResponse: Codable {
    let following: Bool
    let followerCount: Int

    enum CodingKeys: String, CodingKey {
        case following
        case followerCount = "follower_count"
    }
}
