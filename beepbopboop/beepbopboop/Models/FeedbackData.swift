import Foundation

// MARK: - FeedbackData (parsed from external_url for display_hint: "feedback")

struct FeedbackOption: Codable, Identifiable {
    let key: String
    let label: String
    var id: String { key }
}

struct SurveyQuestion: Codable, Identifiable {
    let key: String
    let text: String
    let type: String  // "poll", "freeform", "rating"
    let options: [FeedbackOption]?
    var id: String { key }
}

struct FeedbackData: Codable {
    let feedbackType: String   // "poll", "survey", "freeform", "rating"
    let question: String
    let reason: String?
    let options: [FeedbackOption]?
    let minValue: Double?
    let maxValue: Double?
    let questions: [SurveyQuestion]?

    enum CodingKeys: String, CodingKey {
        case feedbackType = "feedback_type"
        case question, reason, options
        case minValue = "min_value"
        case maxValue = "max_value"
        case questions
    }
}

// MARK: - SurveyAnswer (per-question answer for survey submissions)

struct SurveyAnswer: Codable {
    let question: String
    let type: String
    var selected: [String]?
    var text: String?
    var value: Double?
}

// MARK: - FeedbackResponse (submitted to / received from backend)

struct FeedbackResponse: Codable {
    let type: String
    var selected: [String]?   // poll
    var text: String?          // freeform
    var value: Double?         // rating
    var answers: [SurveyAnswer]?   // survey: per-question answers
}

// MARK: - FeedbackSummary (GET /posts/{id}/responses)

struct FeedbackSummary: Codable {
    let postID: String
    let totalResponses: Int
    let myResponse: FeedbackResponse?
    let tally: [String: Int]?
    let avgRating: Double?

    enum CodingKeys: String, CodingKey {
        case postID = "post_id"
        case totalResponses = "total_responses"
        case myResponse = "my_response"
        case tally
        case avgRating = "avg_rating"
    }
}
