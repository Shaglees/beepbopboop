import Foundation

struct UserWeightsRequest: Encodable {
    let labelWeights: [String: Double]
    let typeWeights: [String: Double]
    let freshnessBias: Double
    let geoBias: Double

    enum CodingKeys: String, CodingKey {
        case labelWeights = "label_weights"
        case typeWeights = "type_weights"
        case freshnessBias = "freshness_bias"
        case geoBias = "geo_bias"
    }
}
