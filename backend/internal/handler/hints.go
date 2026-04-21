package handler

import (
	"encoding/json"
	"net/http"
)

// GET /posts/hints is the discoverability contract between the backend and any
// agent that wants to publish posts. The response tells a skill:
//
//  1. which display_hints the server accepts,
//  2. for each hint, whether external_url must carry structured JSON and what
//     its required shape is,
//  3. one canonical example per hint that is guaranteed to pass validatePost,
//  4. the canonical enumerations (post_type, visibility, image_role),
//  5. the set of agent-facing endpoints the skill can call.
//
// The examples embedded here are lint-tested in hints_test.go; if a validator
// is tightened and an example stops passing, the build breaks, which forces
// documentation and validator to stay in lockstep.
//
// Version is bumped manually when the response shape changes in a way skills
// need to notice (adding fields is non-breaking; removing/renaming fields is
// breaking). Skills should tolerate unknown keys.
const hintsResponseVersion = 1

// hintDescriptor is the per-display-hint shape returned by /posts/hints.
// It is purposefully hand-maintained (not derived from reflection) so the
// wording stays high-signal for skill authors rather than leaking Go field
// names.
type hintDescriptor struct {
	Hint           string          `json:"hint"`
	Description    string          `json:"description"`
	PostType       string          `json:"post_type"`
	StructuredJSON bool            `json:"structured_json"`
	RequiredFields []string        `json:"required_fields"`
	Example        json.RawMessage `json:"example"`
}

// GetPostHints returns the discoverability catalog described above.
func (h *PostHandler) GetPostHints(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{
		"version":       hintsResponseVersion,
		"display_hints": buildHintCatalog(),
		"enums": map[string][]string{
			"post_type":   sortedKeys(ValidPostTypes),
			"visibility":  sortedKeys(ValidVisibility),
			"image_role":  sortedKeys(ValidImageRoles),
			"display_hint": sortedKeys(ValidDisplayHints),
		},
		"endpoints": map[string]map[string]string{
			"create_post": {
				"method":      "POST",
				"path":        "/posts",
				"description": "Create a post. Requires agent token. Validates with same rules as /posts/lint.",
			},
			"lint_post": {
				"method":      "POST",
				"path":        "/posts/lint",
				"description": "Dry-run validation. Returns {valid, errors, warnings}. Always call this before /posts when building a new payload shape.",
			},
			"list_posts": {
				"method":      "GET",
				"path":        "/posts",
				"description": "Recent posts for the authenticated agent's user. Supports ?status=scheduled and ?limit=N.",
			},
			"post_stats": {
				"method":      "GET",
				"path":        "/posts/stats",
				"description": "Rolling 7/30/90 day post counts grouped by post_type / display_hint / label. Use to pick under-represented topics.",
			},
			"events_summary": {
				"method":      "GET",
				"path":        "/events/summary",
				"description": "Engagement events aggregated per post (views, saves, dwell). Drives ForYou ranking; skills can use to learn what the user reads.",
			},
			"reactions_summary": {
				"method":      "GET",
				"path":        "/reactions/summary",
				"description": "Reaction tallies (more/less/stale/not_for_me) per post/label. Skills should weight away from not_for_me topics.",
			},
			"sports_scores": {
				"method":      "GET",
				"path":        "/sports/scores",
				"description": "Live scoreboard snapshots used to build scoreboard / matchup / box_score posts.",
			},
			"creators_nearby": {
				"method":      "GET",
				"path":        "/creators/nearby",
				"description": "Local creator directory; useful for creator_spotlight posts.",
			},
		},
		"docs": map[string]string{
			"publish_flow": "Always POST to /posts/lint first; only POST to /posts after you get {valid:true}.",
			"images":       "See .claude/skills/_shared/IMAGES.md for the full image pipeline (real > AI > provider posters). Every skill should consult it before publishing.",
			"dedup":        "Use beepbopgraph CLI for history lookups before composing; required for any batch flow.",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

// buildHintCatalog assembles the per-hint descriptors. Examples are raw JSON
// so the test can round-trip them through the same decoder /posts/lint uses.
func buildHintCatalog() []hintDescriptor {
	entries := []hintDescriptor{
		{
			Hint:           "card",
			Description:    "Default discovery card. Title + body + optional image_url. No external_url required.",
			PostType:       "discovery",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"Neighborhood gem","body":"A little bakery just opened on Main St.","post_type":"discovery","display_hint":"card","locality":"San Francisco"}`),
		},
		{
			Hint:           "place",
			Description:    "Local place/venue card. Prefer when latitude+longitude are known.",
			PostType:       "place",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body", "locality"},
			Example:        rawJSON(`{"title":"Cafe du Jour","body":"Tiny espresso bar with seasonal pastries.","post_type":"place","display_hint":"place","locality":"San Francisco","latitude":37.7749,"longitude":-122.4194}`),
		},
		{
			Hint:           "article",
			Description:    "External article/news card. external_url must be a real http(s) URL.",
			PostType:       "article",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body", "external_url"},
			Example:        rawJSON(`{"title":"Headline here","body":"Why this matters in 2-3 sentences.","post_type":"article","display_hint":"article","external_url":"https://example.com/article"}`),
		},
		{
			Hint:           "event",
			Description:    "Dated event card. Body should mention when/where; scheduled_at may be set if it is time-sensitive.",
			PostType:       "event",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"Summer jazz in the park","body":"Free outdoor concert this Saturday at 6pm.","post_type":"event","display_hint":"event","locality":"San Francisco"}`),
		},
		{
			Hint:           "calendar",
			Description:    "User-calendar surface entry (not feed). Emitted by skills that populate the calendar layer.",
			PostType:       "event",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"Soccer practice","body":"Tue + Thu 5pm at Franklin Park.","post_type":"event","display_hint":"calendar"}`),
		},
		{
			Hint:           "deal",
			Description:    "Local deal / promo card. Body should include price or % off.",
			PostType:       "discovery",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"20% off opening weekend","body":"New Thai spot on Valencia is doing 20% off all entrees this weekend.","post_type":"discovery","display_hint":"deal","locality":"San Francisco"}`),
		},
		{
			Hint:           "digest",
			Description:    "Daily/weekly digest. Body bullets, short intro.",
			PostType:       "article",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"Your Tuesday brief","body":"- Story one\n- Story two\n- Story three","post_type":"article","display_hint":"digest"}`),
		},
		{
			Hint:           "brief",
			Description:    "Short-form summary card. Shorter than digest, one topic.",
			PostType:       "article",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"Brief: new transit line","body":"The Central Subway extension opens Monday.","post_type":"article","display_hint":"brief"}`),
		},
		{
			Hint:           "comparison",
			Description:    "Two-option comparison card (X vs Y).",
			PostType:       "discovery",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"Italian vs Thai tonight?","body":"Italian: cozy, under 20 min wait. Thai: fast, vegan friendly.","post_type":"discovery","display_hint":"comparison","locality":"San Francisco"}`),
		},
		{
			Hint:           "outfit",
			Description:    "Fashion outfit card. Image(s) strongly recommended.",
			PostType:       "discovery",
			StructuredJSON: false,
			RequiredFields: []string{"title", "body"},
			Example:        rawJSON(`{"title":"Weeknight dinner look","body":"Oversized blazer, straight-leg denim, loafers.","post_type":"discovery","display_hint":"outfit"}`),
		},
		{
			Hint:           "creator_spotlight",
			Description:    "Spotlight on a local creator. external_url carries structured creator data.",
			PostType:       "discovery",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url"},
			Example:        rawJSON(`{"title":"Local creator: Maya's Ceramics","body":"Hand-thrown mugs from a Mission District studio.","post_type":"discovery","display_hint":"creator_spotlight","external_url":"https://example.com/creator/maya"}`),
		},
		{
			Hint:           "weather",
			Description:    "Hyperlocal weather card. external_url is JSON matching the weather schema.",
			PostType:       "discovery",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:current", "external_url:hourly", "external_url:daily", "external_url:location"},
			Example: rawJSON(`{"title":"Mild and sunny today","body":"20\u00b0C with clear skies; UV index 5.","post_type":"discovery","display_hint":"weather","locality":"Dublin","external_url":"{\"current\":{\"temp_c\":20,\"feels_like_c\":18,\"humidity\":60,\"wind_speed_kmh\":10,\"uv_index\":5,\"is_day\":true,\"condition\":\"Sunny\",\"condition_code\":1000},\"hourly\":[],\"daily\":[],\"location\":{\"latitude\":53.3,\"longitude\":-6.2,\"timezone\":\"Europe/Dublin\"}}"}`),
		},
		{
			Hint:           "scoreboard",
			Description:    "Final score card. external_url is JSON with status, home, away, sport.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:status", "external_url:home", "external_url:away"},
			Example: rawJSON(`{"title":"Lakers 110, Celtics 105","body":"LeBron 28/9/7 as LA closes out the series.","post_type":"article","display_hint":"scoreboard","external_url":"{\"status\":\"Final\",\"home\":{\"name\":\"Lakers\",\"abbr\":\"LAL\"},\"away\":{\"name\":\"Celtics\",\"abbr\":\"BOS\"},\"sport\":\"NBA\"}"}`),
		},
		{
			Hint:           "matchup",
			Description:    "Pre-game preview card. external_url requires gameTime in addition to scoreboard fields.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:status", "external_url:home", "external_url:away", "external_url:gameTime"},
			Example: rawJSON(`{"title":"Lakers host Celtics tonight","body":"First meeting since the Finals; tip-off 7pm PT.","post_type":"article","display_hint":"matchup","external_url":"{\"status\":\"Scheduled\",\"home\":{\"name\":\"Lakers\",\"abbr\":\"LAL\"},\"away\":{\"name\":\"Celtics\",\"abbr\":\"BOS\"},\"sport\":\"NBA\",\"gameTime\":\"2026-04-16T19:00:00Z\"}"}`),
		},
		{
			Hint:           "standings",
			Description:    "League standings snapshot.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:league", "external_url:date", "external_url:games"},
			Example:        rawJSON(`{"title":"NBA nightly recap","body":"Four games tonight.","post_type":"article","display_hint":"standings","external_url":"{\"league\":\"NBA\",\"date\":\"2026-04-16\",\"games\":[{\"home\":\"LAL\",\"away\":\"BOS\",\"homeScore\":110,\"awayScore\":105,\"status\":\"Final\"}]}"}`),
		},
		{
			Hint:           "box_score",
			Description:    "Per-game statistical breakdown. Same shape as scoreboard.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:status", "external_url:home", "external_url:away"},
			Example: rawJSON(`{"title":"Box score: LAL 110, BOS 105","body":"LeBron led all scorers with 28.","post_type":"article","display_hint":"box_score","external_url":"{\"status\":\"Final\",\"home\":{\"name\":\"Lakers\",\"abbr\":\"LAL\"},\"away\":{\"name\":\"Celtics\",\"abbr\":\"BOS\"},\"sport\":\"NBA\"}"}`),
		},
		{
			Hint:           "player_spotlight",
			Description:    "Deep dive on one player.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:playerName", "external_url:sport", "external_url:team"},
			Example:        rawJSON(`{"title":"LeBron's quiet efficiency","body":"28/9/7 on 61% shooting.","post_type":"article","display_hint":"player_spotlight","external_url":"{\"playerName\":\"LeBron James\",\"sport\":\"NBA\",\"team\":\"Lakers\"}"}`),
		},
		{
			Hint:           "entertainment",
			Description:    "Celebrity/entertainment news card. Sourced, not tabloid.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:subject", "external_url:headline"},
			Example:        rawJSON(`{"title":"Zendaya named TIME Entertainer of the Year","body":"Recognized for Dune: Part Two and Challengers.","post_type":"article","display_hint":"entertainment","external_url":"{\"subject\":\"Zendaya\",\"headline\":\"Zendaya Named TIME Entertainer of the Year\",\"source\":\"People\",\"category\":\"award\",\"tags\":[\"entertainment\"]}"}`),
		},
		{
			Hint:           "album",
			Description:    "Music album announcement/review.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:type", "external_url:artist", "external_url:title"},
			Example:        rawJSON(`{"title":"Taylor Swift drops TTPD","body":"31 tracks, double album.","post_type":"article","display_hint":"album","external_url":"{\"type\":\"album\",\"artist\":\"Taylor Swift\",\"title\":\"The Tortured Poets Department\"}"}`),
		},
		{
			Hint:           "concert",
			Description:    "Local concert / tour announcement.",
			PostType:       "event",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:type", "external_url:artist"},
			Example:        rawJSON(`{"title":"Coldplay tour hits SF","body":"Levi's Stadium, two nights in September.","post_type":"event","display_hint":"concert","external_url":"{\"type\":\"concert\",\"artist\":\"Coldplay\"}"}`),
		},
		{
			Hint:           "game_release",
			Description:    "Upcoming video-game release card.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:title", "external_url:status"},
			Example:        rawJSON(`{"title":"New game out next week","body":"Preorders open on Steam.","post_type":"article","display_hint":"game_release","external_url":"{\"title\":\"Test Game\",\"status\":\"upcoming\"}"}`),
		},
		{
			Hint:           "game_review",
			Description:    "Released-game review card.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:title", "external_url:status"},
			Example:        rawJSON(`{"title":"Test Game: first impressions","body":"Solid core loop, weak story.","post_type":"article","display_hint":"game_review","external_url":"{\"title\":\"Test Game\",\"status\":\"released\"}"}`),
		},
		{
			Hint:           "restaurant",
			Description:    "Specific restaurant card with coordinates.",
			PostType:       "place",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:name", "external_url:latitude", "external_url:longitude"},
			Example:        rawJSON(`{"title":"Test Cafe","body":"New neighborhood cafe with great espresso.","post_type":"place","display_hint":"restaurant","external_url":"{\"name\":\"Test Cafe\",\"latitude\":40.7,\"longitude\":-74.0}"}`),
		},
		{
			Hint:           "destination",
			Description:    "Travel destination card.",
			PostType:       "place",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:city", "external_url:country", "external_url:latitude", "external_url:longitude"},
			Example:        rawJSON(`{"title":"Weekend in Paris","body":"Museums, patisseries, the Seine at dusk.","post_type":"place","display_hint":"destination","external_url":"{\"city\":\"Paris\",\"country\":\"France\",\"latitude\":48.8566,\"longitude\":2.3522}"}`),
		},
		{
			Hint:           "movie",
			Description:    "Movie card, sourced from TMDB.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:tmdbId", "external_url:title"},
			Example:        rawJSON(`{"title":"Fight Club","body":"1999 cult classic still holds up.","post_type":"article","display_hint":"movie","external_url":"{\"tmdbId\":550,\"title\":\"Fight Club\"}"}`),
		},
		{
			Hint:           "show",
			Description:    "TV show card, sourced from TMDB.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:tmdbId", "external_url:title"},
			Example:        rawJSON(`{"title":"Game of Thrones","body":"Start of the fantasy prestige era.","post_type":"article","display_hint":"show","external_url":"{\"tmdbId\":1399,\"title\":\"Game of Thrones\"}"}`),
		},
		{
			Hint:           "pet_spotlight",
			Description:    "Adoptable pet card, sourced from Petfinder.",
			PostType:       "discovery",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:type", "external_url:name", "external_url:species", "external_url:petfinderUrl"},
			Example:        rawJSON(`{"title":"Meet Biscuit","body":"Young Lab mix, loves fetch.","post_type":"discovery","display_hint":"pet_spotlight","external_url":"{\"type\":\"adoption\",\"name\":\"Biscuit\",\"species\":\"dog\",\"breed\":\"Labrador Mix\",\"age\":\"Young\",\"gender\":\"Male\",\"shelterName\":\"SF SPCA\",\"shelterCity\":\"San Francisco\",\"petfinderUrl\":\"https://www.petfinder.com/dog/biscuit-12345678\"}"}`),
		},
		{
			Hint:           "fitness",
			Description:    "Fitness/workout card.",
			PostType:       "discovery",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:activity", "external_url:duration_min"},
			Example:        rawJSON(`{"title":"30 minute evening run","body":"Zone 2, flat loop.","post_type":"discovery","display_hint":"fitness","external_url":"{\"activity\":\"Running\",\"duration_min\":30}"}`),
		},
		{
			Hint:           "science",
			Description:    "Science/nature/space card.",
			PostType:       "article",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:category", "external_url:source", "external_url:headline"},
			Example:        rawJSON(`{"title":"NASA finds new exoplanet","body":"Earth-sized, in the habitable zone.","post_type":"article","display_hint":"science","external_url":"{\"category\":\"Space\",\"source\":\"NASA\",\"headline\":\"New Planet Discovered\"}"}`),
		},
		{
			Hint:           "feedback",
			Description:    "Interactive feedback post (poll/question). Drives /posts/{id}/response.",
			PostType:       "discovery",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:feedback_type", "external_url:question", "external_url:options"},
			Example:        rawJSON(`{"title":"Which do you prefer?","body":"Help us tune your feed.","post_type":"discovery","display_hint":"feedback","external_url":"{\"feedback_type\":\"poll\",\"question\":\"What do you think?\",\"options\":[{\"key\":\"a\",\"label\":\"Option A\"}]}"}`),
		},
		{
			Hint:           "video_embed",
			Description:    "YouTube/Vimeo embed. embed_url must be an /embed/ URL; watch_url and thumbnail_url recommended.",
			PostType:       "video",
			StructuredJSON: true,
			RequiredFields: []string{"title", "body", "external_url:provider", "external_url:embed_url"},
			Example:        rawJSON(`{"title":"Watch: short film","body":"A 4-minute doc from last month's festival.","post_type":"video","display_hint":"video_embed","external_url":"{\"provider\":\"youtube\",\"video_id\":\"dQw4w9WgXcQ\",\"embed_url\":\"https://www.youtube.com/embed/dQw4w9WgXcQ\",\"watch_url\":\"https://www.youtube.com/watch?v=dQw4w9WgXcQ\",\"thumbnail_url\":\"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg\",\"channel_title\":\"Rick Astley\"}"}`),
		},
	}
	return entries
}

func rawJSON(s string) json.RawMessage {
	return json.RawMessage(s)
}

// sortedKeys returns the keys of a bool map in deterministic order so the
// /posts/hints response is stable across requests (and easy to diff).
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

// sortStrings is a tiny insertion sort to avoid pulling in sort just for
// tiny slices; the largest map here has ~30 entries.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
