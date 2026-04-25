import Foundation

enum MockPostFactory {
    static func post(for hint: String) -> Post {
        let json = postJSON(for: hint)
        let data = json.data(using: .utf8)!
        return try! JSONDecoder().decode(Post.self, from: data)
    }

    private static func postJSON(for hint: String) -> String {
        let base = """
        "id":"mock-\(hint)","agent_id":"mock-agent","agent_name":"Preview",
        "user_id":"mock-user","created_at":"2026-04-24T12:00:00Z"
        """

        switch hint {
        case "scoreboard":
            return """
            {\(base),"title":"Lakers 112 - Celtics 108","body":"LeBron James led with 32 points as the Lakers edge past Boston in a thriller.",
            "display_hint":"scoreboard",
            "external_url":"{\\"sport\\":\\"basketball\\",\\"league\\":\\"NBA\\",\\"status\\":\\"Final\\",\\"home\\":{\\"name\\":\\"Los Angeles Lakers\\",\\"abbr\\":\\"LAL\\",\\"score\\":112,\\"record\\":\\"48-22\\",\\"color\\":\\"#552583\\"},\\"away\\":{\\"name\\":\\"Boston Celtics\\",\\"abbr\\":\\"BOS\\",\\"score\\":108,\\"record\\":\\"50-20\\",\\"color\\":\\"#007A33\\"}}"}
            """
        case "matchup":
            return """
            {\(base),"title":"Warriors vs Nuggets","body":"Western Conference showdown at Chase Center.",
            "display_hint":"matchup",
            "external_url":"{\\"sport\\":\\"basketball\\",\\"league\\":\\"NBA\\",\\"status\\":\\"Scheduled\\",\\"gameTime\\":\\"2026-04-25T19:30:00Z\\",\\"home\\":{\\"name\\":\\"Golden State Warriors\\",\\"abbr\\":\\"GSW\\",\\"record\\":\\"44-26\\",\\"color\\":\\"#1D428A\\"},\\"away\\":{\\"name\\":\\"Denver Nuggets\\",\\"abbr\\":\\"DEN\\",\\"record\\":\\"46-24\\",\\"color\\":\\"#0E2240\\"},\\"venue\\":\\"Chase Center\\",\\"broadcast\\":\\"ESPN\\"}"}
            """
        case "player_spotlight":
            return """
            {\(base),"title":"Jokic Triple-Double Machine","body":"Nikola Jokic recorded his 25th triple-double of the season.",
            "display_hint":"player_spotlight",
            "external_url":"{\\"type\\":\\"game_recap\\",\\"sport\\":\\"basketball\\",\\"league\\":\\"NBA\\",\\"playerId\\":\\"jokic\\",\\"playerName\\":\\"Nikola Jokic\\",\\"team\\":\\"Denver Nuggets\\",\\"teamAbbr\\":\\"DEN\\",\\"teamColor\\":\\"#0E2240\\",\\"position\\":\\"C\\",\\"opponent\\":\\"Lakers\\",\\"gameResult\\":\\"W 118-105\\",\\"lastGameStats\\":{\\"points\\":28,\\"rebounds\\":14,\\"assists\\":11,\\"steals\\":2,\\"blocks\\":1},\\"seasonAverages\\":{\\"points\\":26.4,\\"rebounds\\":12.1,\\"assists\\":9.8},\\"storyline\\":\\"25th triple-double this season\\"}"}
            """
        case "restaurant":
            return """
            {\(base),"title":"Sushi Nakazawa","body":"Omakase experience in the West Village. Reservations recommended.",
            "display_hint":"restaurant",
            "external_url":"{\\"name\\":\\"Sushi Nakazawa\\",\\"rating\\":4.7,\\"reviewCount\\":892,\\"cuisine\\":[\\"Japanese\\",\\"Sushi\\",\\"Omakase\\"],\\"priceRange\\":\\"$$$$\\",\\"address\\":\\"23 Commerce St\\",\\"neighbourhood\\":\\"West Village\\",\\"isOpenNow\\":true,\\"latitude\\":40.7295,\\"longitude\\":-74.0037,\\"mustTry\\":[\\"Chef's Omakase\\",\\"Uni\\"],\\"pricePerHead\\":\\"$150-200\\",\\"newOpening\\":false}"}
            """
        case "deal":
            return """
            {\(base),"title":"Hades II","body":"Supergiant's roguelike sequel hits a new low price on Steam.",
            "display_hint":"deal",
            "external_url":"{\\"steamDiscount\\":40,\\"steamPrice\\":\\"$17.99\\",\\"steamOriginalPrice\\":\\"$29.99\\",\\"ends_in\\":\\"2 days\\"}"}
            """
        case "album":
            return """
            {\(base),"title":"In Rainbows","body":"Radiohead's genre-defying masterpiece still resonates.",
            "display_hint":"album",
            "external_url":"{\\"type\\":\\"album\\",\\"title\\":\\"In Rainbows\\",\\"artist\\":\\"Radiohead\\",\\"albumType\\":\\"album\\",\\"trackCount\\":10,\\"label\\":\\"XL Recordings\\",\\"tags\\":[\\"Alternative\\",\\"Art Rock\\",\\"Electronic\\"]}"}
            """
        case "concert":
            return """
            {\(base),"title":"Khruangbin Live","body":"Thai funk trio brings their hypnotic grooves to Brooklyn Steel.",
            "display_hint":"concert",
            "external_url":"{\\"type\\":\\"concert\\",\\"artist\\":\\"Khruangbin\\",\\"venue\\":\\"Brooklyn Steel\\",\\"date\\":\\"2026-06-15\\",\\"doorsTime\\":\\"7:00 PM\\",\\"startTime\\":\\"8:00 PM\\",\\"priceRange\\":\\"$45-75\\",\\"onSale\\":true}"}
            """
        case "science":
            return """
            {\(base),"title":"New Exoplanet in Habitable Zone","body":"Astronomers discover a rocky planet orbiting Proxima Centauri with potential for liquid water.",
            "display_hint":"science",
            "external_url":"{\\"category\\":\\"space\\",\\"source\\":\\"Nature\\",\\"headline\\":\\"Rocky Exoplanet Found in Proxima Centauri Habitable Zone\\",\\"institution\\":\\"ESO\\",\\"tags\\":[\\"exoplanet\\",\\"habitable zone\\",\\"astronomy\\"]}"}
            """
        case "destination":
            return """
            {\(base),"title":"Discover Lisbon","body":"Pastel-colored tiles, world-class seafood, and golden-hour light over the Tagus.",
            "display_hint":"destination",
            "external_url":"{\\"city\\":\\"Lisbon\\",\\"country\\":\\"Portugal\\",\\"latitude\\":38.7223,\\"longitude\\":-9.1393,\\"knownFor\\":[\\"Pastéis de Nata\\",\\"Tram 28\\",\\"Fado Music\\"],\\"bestTimeToVisit\\":\\"Apr–Oct\\",\\"flightPriceFrom\\":\\"$420\\",\\"currency\\":\\"EUR\\",\\"visaRequired\\":false}"}
            """
        case "fitness":
            return """
            {\(base),"title":"Morning HIIT Burn","body":"A 30-minute high-intensity workout targeting full body.",
            "display_hint":"fitness",
            "external_url":"{\\"type\\":\\"workout\\",\\"activity\\":\\"hiit\\",\\"level\\":\\"intermediate\\",\\"durationMin\\":30,\\"muscleGroups\\":[\\"Full Body\\"],\\"equipmentNeeded\\":[\\"None\\"],\\"caloriesBurn\\":\\"350-450\\",\\"exercises\\":[{\\"name\\":\\"Burpees\\",\\"sets\\":3,\\"reps\\":\\"12\\"},{\\"name\\":\\"Mountain Climbers\\",\\"sets\\":3,\\"reps\\":\\"20\\"}]}"}
            """
        case "pet_spotlight":
            return """
            {\(base),"title":"Meet Luna","body":"Sweet 2-year-old tabby looking for a forever home.",
            "display_hint":"pet_spotlight",
            "external_url":"{\\"type\\":\\"adoption\\",\\"name\\":\\"Luna\\",\\"species\\":\\"cat\\",\\"breed\\":\\"Tabby\\",\\"age\\":\\"2 years\\",\\"gender\\":\\"Female\\",\\"size\\":\\"Medium\\",\\"shelterName\\":\\"Happy Paws Rescue\\",\\"shelterCity\\":\\"Brooklyn, NY\\"}"}
            """
        case "outfit":
            return """
            {\(base),"title":"Minimalist Summer Layers","body":"Light linen and earth tones for warm days.\\n**Trend:** Quiet luxury\\n**For you:** Based on your minimalist style preference",
            "display_hint":"outfit"}
            """
        case "movie":
            return """
            {\(base),"title":"The Brutalist","body":"Brady Corbet's sweeping epic about a Hungarian architect immigrating to America.",
            "display_hint":"movie",
            "external_url":"{\\"tmdbId\\":123456,\\"type\\":\\"movie\\",\\"title\\":\\"The Brutalist\\",\\"year\\":2025,\\"runtime\\":215,\\"genres\\":[\\"Drama\\",\\"Historical\\"],\\"director\\":\\"Brady Corbet\\",\\"cast\\":[\\"Adrien Brody\\",\\"Felicity Jones\\"],\\"tmdbRating\\":8.1,\\"status\\":\\"available\\",\\"streaming\\":[\\"A24\\"],\\"rentBuy\\":[],\\"inTheatres\\":false,\\"onTheAir\\":false}"}
            """
        case "show":
            return """
            {\(base),"title":"Severance","body":"Mark returns to Lumon Industries as the Innies fight for their freedom.",
            "display_hint":"show",
            "external_url":"{\\"tmdbId\\":654321,\\"type\\":\\"show\\",\\"title\\":\\"Severance\\",\\"year\\":2022,\\"genres\\":[\\"Sci-Fi\\",\\"Thriller\\"],\\"creator\\":\\"Dan Erickson\\",\\"network\\":\\"Apple TV+\\",\\"cast\\":[\\"Adam Scott\\",\\"Zach Cherry\\",\\"Britt Lower\\"],\\"tmdbRating\\":8.7,\\"seasons\\":2,\\"status\\":\\"available\\",\\"streaming\\":[\\"Apple TV+\\"],\\"rentBuy\\":[],\\"inTheatres\\":false,\\"onTheAir\\":true}"}
            """
        default: // "article" and any other
            return """
            {\(base),"title":"The Future of AI Agents","body":"How autonomous agents are reshaping software development and daily workflows.",
            "display_hint":"article"}
            """
        }
    }
}
