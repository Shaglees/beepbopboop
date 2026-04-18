struct FeedWeights: Codable {
    var labelWeights: [String: Double]?
    var typeWeights: [String: Double]?
    var freshnessBias: Double
    var geoBias: Double

    static let defaults = FeedWeights(freshnessBias: 0.8, geoBias: 0.5)

    enum CodingKeys: String, CodingKey {
        case labelWeights = "label_weights"
        case typeWeights = "type_weights"
        case freshnessBias = "freshness_bias"
        case geoBias = "geo_bias"
    }
}

// Wrapper matching GET /user/weights response envelope
struct UserWeightsResponse: Codable {
    let userId: String?
    let weights: FeedWeights?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case weights
    }
}
