import Foundation

struct MediaData: Codable {
    let tmdbId: Int
    let type: String            // "movie" | "show"
    let title: String
    let year: Int?
    let posterUrl: String?
    let backdropUrl: String?
    let tagline: String?
    let tmdbRating: Double?     // 0.0–10.0
    let tmdbVoteCount: Int?
    let rtScore: Int?           // 0–100 Tomatometer
    let rtAudienceScore: Int?
    let runtime: Int?           // minutes (movies)
    let releaseDate: String?
    let genres: [String]
    let director: String?       // movies
    let creator: String?        // shows
    let cast: [String]
    let streaming: [String]     // flat-rate subscription platforms
    let rentBuy: [String]
    let inTheatres: Bool
    let onTheAir: Bool?         // shows currently airing
    let status: String          // "upcoming" | "in_theatres" | "available"
    let network: String?        // shows
    let seasons: Int?           // shows

    enum CodingKeys: String, CodingKey {
        case tmdbId, type, title, year, posterUrl, backdropUrl, tagline
        case tmdbRating, tmdbVoteCount, rtScore, rtAudienceScore
        case runtime, releaseDate, genres, director, creator, cast
        case streaming, rentBuy, inTheatres, onTheAir, status, network, seasons
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        tmdbId = try c.decode(Int.self, forKey: .tmdbId)
        type = try c.decodeIfPresent(String.self, forKey: .type) ?? "movie"
        title = try c.decode(String.self, forKey: .title)
        year = try c.decodeIfPresent(Int.self, forKey: .year)
        posterUrl = try c.decodeIfPresent(String.self, forKey: .posterUrl)
        backdropUrl = try c.decodeIfPresent(String.self, forKey: .backdropUrl)
        tagline = try c.decodeIfPresent(String.self, forKey: .tagline)
        tmdbRating = try c.decodeIfPresent(Double.self, forKey: .tmdbRating)
        tmdbVoteCount = try c.decodeIfPresent(Int.self, forKey: .tmdbVoteCount)
        rtScore = try c.decodeIfPresent(Int.self, forKey: .rtScore)
        rtAudienceScore = try c.decodeIfPresent(Int.self, forKey: .rtAudienceScore)
        runtime = try c.decodeIfPresent(Int.self, forKey: .runtime)
        releaseDate = try c.decodeIfPresent(String.self, forKey: .releaseDate)
        genres = (try? c.decodeIfPresent([String].self, forKey: .genres)) ?? []
        director = try c.decodeIfPresent(String.self, forKey: .director)
        creator = try c.decodeIfPresent(String.self, forKey: .creator)
        cast = (try? c.decodeIfPresent([String].self, forKey: .cast)) ?? []
        streaming = (try? c.decodeIfPresent([String].self, forKey: .streaming)) ?? []
        rentBuy = (try? c.decodeIfPresent([String].self, forKey: .rentBuy)) ?? []
        inTheatres = (try? c.decodeIfPresent(Bool.self, forKey: .inTheatres)) ?? false
        onTheAir = try c.decodeIfPresent(Bool.self, forKey: .onTheAir)
        status = (try? c.decodeIfPresent(String.self, forKey: .status)) ?? "available"
        network = try c.decodeIfPresent(String.self, forKey: .network)
        seasons = try c.decodeIfPresent(Int.self, forKey: .seasons)
    }
}
