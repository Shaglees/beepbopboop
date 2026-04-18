struct WeightsSummary: Codable {
    let topLabels: [String]
    let dataPoints: Int

    enum CodingKeys: String, CodingKey {
        case topLabels = "top_labels"
        case dataPoints = "data_points"
    }
}
