package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

const maxURLLength = 2048

var ValidPostTypes = map[string]bool{
	"event":     true,
	"place":     true,
	"discovery": true,
	"article":   true,
	"video":     true,
}

var ValidVisibility = map[string]bool{
	"public":   true,
	"personal": true,
	"private":  true,
}

var ValidDisplayHints = map[string]bool{
	"card":             true,
	"place":            true,
	"article":          true,
	"weather":          true,
	"calendar":         true,
	"deal":             true,
	"digest":           true,
	"brief":            true,
	"comparison":       true,
	"event":            true,
	"outfit":           true,
	"scoreboard":       true,
	"matchup":          true,
	"standings":        true,
	"movie":            true,
	"show":             true,
	"player_spotlight": true,
	"entertainment":    true,
	"album":            true,
	"concert":          true,
	"game_release":     true,
	"game_review":      true,
	"restaurant":       true,
	"science":          true,
}

var ValidImageRoles = map[string]bool{
	"hero":    true,
	"detail":  true,
	"product": true,
}

type PostHandler struct {
	agentRepo *repository.AgentRepo
	postRepo  *repository.PostRepo
}

func NewPostHandler(agentRepo *repository.AgentRepo, postRepo *repository.PostRepo) *PostHandler {
	return &PostHandler{
		agentRepo: agentRepo,
		postRepo:  postRepo,
	}
}

type createPostRequest struct {
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	ImageURL    string          `json:"image_url,omitempty"`
	ExternalURL string          `json:"external_url,omitempty"`
	Locality    string          `json:"locality,omitempty"`
	Latitude    *float64        `json:"latitude,omitempty"`
	Longitude   *float64        `json:"longitude,omitempty"`
	PostType    string          `json:"post_type,omitempty"`
	Visibility  string          `json:"visibility,omitempty"`
	DisplayHint string          `json:"display_hint,omitempty"`
	Labels      []string        `json:"labels,omitempty"`
	Images      json.RawMessage `json:"images,omitempty"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty"`
}

// --- Validation types ---

type validationIssue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type validationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []validationIssue `json:"errors"`
	Warnings []validationIssue `json:"warnings"`
}

// validateURL checks that a string is a well-formed http(s) URL within length limits.
// Returns an error message or "" if valid.
func validateURL(raw string) string {
	if len(raw) > maxURLLength {
		return fmt.Sprintf("URL exceeds maximum length of %d characters", maxURLLength)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Sprintf("malformed URL: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Sprintf("URL scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return "URL must have a host"
	}
	return ""
}

// validatePost checks a createPostRequest and returns structured errors/warnings.
// Pure function — no DB access.
func validatePost(req *createPostRequest) validationResult {
	var errs []validationIssue
	var warns []validationIssue

	// --- Basic fields ---
	if req.Title == "" {
		errs = append(errs, validationIssue{Field: "title", Code: "required", Message: "title is required"})
	}
	if req.Body == "" {
		errs = append(errs, validationIssue{Field: "body", Code: "required", Message: "body is required"})
	}

	// Apply defaults before validation
	if req.PostType == "" {
		req.PostType = "discovery"
	}
	if !ValidPostTypes[req.PostType] {
		errs = append(errs, validationIssue{
			Field:   "post_type",
			Code:    "invalid",
			Message: fmt.Sprintf("invalid post_type %q: must be event, place, discovery, article, or video", req.PostType),
		})
	}

	if req.Visibility == "" {
		req.Visibility = "public"
	}
	if !ValidVisibility[req.Visibility] {
		errs = append(errs, validationIssue{
			Field:   "visibility",
			Code:    "invalid",
			Message: fmt.Sprintf("invalid visibility %q: must be public, personal, or private", req.Visibility),
		})
	}

	if req.DisplayHint != "" && !ValidDisplayHints[req.DisplayHint] {
		errs = append(errs, validationIssue{
			Field:   "display_hint",
			Code:    "invalid",
			Message: fmt.Sprintf("unknown display_hint %q", req.DisplayHint),
		})
	}

	// --- URL validation ---
	if req.ImageURL != "" {
		if msg := validateURL(req.ImageURL); msg != "" {
			errs = append(errs, validationIssue{Field: "image_url", Code: "invalid_url", Message: msg})
		}
	}

	// external_url is validated as a URL only when the display_hint does NOT
	// expect structured JSON (weather, scoreboard, matchup, standings store
	// JSON payloads in this field, not actual URLs).
	structuredHint := req.DisplayHint == "weather" || req.DisplayHint == "scoreboard" ||
		req.DisplayHint == "matchup" || req.DisplayHint == "standings" || req.DisplayHint == "entertainment" ||
		req.DisplayHint == "album" || req.DisplayHint == "concert" ||
		req.DisplayHint == "game_release" || req.DisplayHint == "game_review" ||
		req.DisplayHint == "restaurant"
		req.DisplayHint == "matchup" || req.DisplayHint == "standings" || req.DisplayHint == "entertainment" ||
		req.DisplayHint == "destination"
	if req.ExternalURL != "" && !structuredHint {
		if msg := validateURL(req.ExternalURL); msg != "" {
			errs = append(errs, validationIssue{Field: "external_url", Code: "invalid_url", Message: msg})
		}
	}

	// Labels
	if len(req.Labels) > 20 {
		warns = append(warns, validationIssue{Field: "labels", Code: "truncated", Message: "more than 20 labels; will be truncated to 20"})
	}
	if len(req.Labels) == 0 {
		warns = append(warns, validationIssue{Field: "labels", Code: "missing", Message: "no labels; hurts feed ranking"})
	}

	// Images array validation
	if len(req.Images) > 0 {
		var images []map[string]interface{}
		if err := json.Unmarshal(req.Images, &images); err != nil {
			errs = append(errs, validationIssue{Field: "images", Code: "invalid_json", Message: "images must be a valid JSON array of objects"})
		} else {
			for i, img := range images {
				prefix := fmt.Sprintf("images[%d]", i)
				if urlVal, ok := img["url"]; !ok {
					errs = append(errs, validationIssue{
						Field:   prefix + ".url",
						Code:    "required",
						Message: fmt.Sprintf("image %d must have a url", i),
					})
				} else if urlStr, ok := urlVal.(string); ok {
					if msg := validateURL(urlStr); msg != "" {
						errs = append(errs, validationIssue{
							Field:   prefix + ".url",
							Code:    "invalid_url",
							Message: fmt.Sprintf("image %d url: %s", i, msg),
						})
					}
				}
				if role, ok := img["role"].(string); ok && !ValidImageRoles[role] {
					warns = append(warns, validationIssue{
						Field:   prefix + ".role",
						Code:    "unknown",
						Message: fmt.Sprintf("unknown image role %q; known: hero, detail, product", role),
					})
				}
			}
		}
	}

	// Public post without location
	if req.Visibility == "public" && req.Locality == "" && req.Latitude == nil {
		warns = append(warns, validationIssue{
			Field:   "locality",
			Code:    "missing",
			Message: "public post with no locality or coordinates",
		})
	}

	// --- external_url schema validation for structured hints ---
	if req.ExternalURL != "" {
		switch req.DisplayHint {
		case "weather":
			validateWeatherData(req.ExternalURL, &errs, &warns)
		case "scoreboard", "matchup":
			validateGameData(req.ExternalURL, req.DisplayHint, &errs, &warns)
		case "standings":
			validateStandingsData(req.ExternalURL, &errs, &warns)
		case "entertainment":
			validateEntertainmentData(req.ExternalURL, &errs, &warns)
		case "game_release", "game_review":
			validateVideoGameData(req.ExternalURL, req.DisplayHint, &errs, &warns)
		case "restaurant":
			validateFoodData(req.ExternalURL, &errs, &warns)
		}
	} else if req.DisplayHint == "weather" || req.DisplayHint == "scoreboard" || req.DisplayHint == "matchup" || req.DisplayHint == "standings" || req.DisplayHint == "entertainment" ||
		req.DisplayHint == "game_release" || req.DisplayHint == "game_review" || req.DisplayHint == "restaurant" {
	} else if req.DisplayHint == "weather" || req.DisplayHint == "scoreboard" || req.DisplayHint == "matchup" || req.DisplayHint == "standings" || req.DisplayHint == "entertainment" || req.DisplayHint == "destination" {
		errs = append(errs, validationIssue{
			Field:   "external_url",
			Code:    "required",
			Message: fmt.Sprintf("display_hint %q requires external_url with structured JSON data", req.DisplayHint),
		})
	}

	return validationResult{
		Valid:    len(errs) == 0,
		Errors:   errs,
		Warnings: warns,
	}
}

// --- Weather validation ---

type weatherDataValidation struct {
	Current *struct {
		TempC         *float64 `json:"temp_c"`
		FeelsLikeC    *float64 `json:"feels_like_c"`
		Humidity      *int     `json:"humidity"`
		WindSpeedKmh  *float64 `json:"wind_speed_kmh"`
		UVIndex       *float64 `json:"uv_index"`
		IsDay         *bool    `json:"is_day"`
		Condition     *string  `json:"condition"`
		ConditionCode *int     `json:"condition_code"`
	} `json:"current"`
	Hourly   *json.RawMessage `json:"hourly"`
	Daily    *json.RawMessage `json:"daily"`
	Location *struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Timezone  *string  `json:"timezone"`
	} `json:"location"`
}

func validateWeatherData(externalURL string, errs *[]validationIssue, warns *[]validationIssue) {
	var w weatherDataValidation
	if err := json.Unmarshal([]byte(externalURL), &w); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "external_url must be valid JSON for weather hint"})
		return
	}

	if w.Current == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.current", Code: "required", Message: "current weather data is required"})
	} else {
		if w.Current.TempC == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.temp_c", Code: "required", Message: "current temp_c is required"})
		}
		if w.Current.FeelsLikeC == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.feels_like_c", Code: "required", Message: "current feels_like_c is required"})
		}
		if w.Current.Humidity == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.humidity", Code: "required", Message: "current humidity is required"})
		}
		if w.Current.WindSpeedKmh == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.wind_speed_kmh", Code: "required", Message: "current wind_speed_kmh is required"})
		}
		if w.Current.UVIndex == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.uv_index", Code: "required", Message: "current uv_index is required"})
		}
		if w.Current.IsDay == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.is_day", Code: "required", Message: "current is_day is required"})
		}
		if w.Current.Condition == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.condition", Code: "required", Message: "current condition is required"})
		}
		if w.Current.ConditionCode == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.current.condition_code", Code: "required", Message: "current condition_code is required"})
		}
	}

	if w.Hourly == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.hourly", Code: "required", Message: "hourly forecast array is required"})
	}
	if w.Daily == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.daily", Code: "required", Message: "daily forecast array is required"})
	}
	if w.Location == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.location", Code: "required", Message: "location data is required"})
	} else {
		if w.Location.Latitude == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.location.latitude", Code: "required", Message: "location latitude is required"})
		}
		if w.Location.Longitude == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.location.longitude", Code: "required", Message: "location longitude is required"})
		}
		if w.Location.Timezone == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.location.timezone", Code: "required", Message: "location timezone is required"})
		}
	}
}

// --- Game data validation (scoreboard/matchup) ---

type gameDataValidation struct {
	Sport    *string `json:"sport"`
	Status   *string `json:"status"`
	GameTime *string `json:"gameTime"`
	Home     *struct {
		Name *string `json:"name"`
		Abbr *string `json:"abbr"`
	} `json:"home"`
	Away *struct {
		Name *string `json:"name"`
		Abbr *string `json:"abbr"`
	} `json:"away"`
}

func validateGameData(externalURL string, hint string, errs *[]validationIssue, warns *[]validationIssue) {
	var g gameDataValidation
	if err := json.Unmarshal([]byte(externalURL), &g); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: fmt.Sprintf("external_url must be valid JSON for %s hint", hint)})
		return
	}

	if g.Status == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.status", Code: "required", Message: "game status is required"})
	}
	if g.Home == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.home", Code: "required", Message: "home team is required"})
	} else {
		if g.Home.Name == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.home.name", Code: "required", Message: "home team name is required"})
		}
		if g.Home.Abbr == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.home.abbr", Code: "required", Message: "home team abbr is required"})
		}
	}
	if g.Away == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.away", Code: "required", Message: "away team is required"})
	} else {
		if g.Away.Name == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.away.name", Code: "required", Message: "away team name is required"})
		}
		if g.Away.Abbr == nil {
			*errs = append(*errs, validationIssue{Field: "external_url.away.abbr", Code: "required", Message: "away team abbr is required"})
		}
	}

	// Warnings
	if hint == "matchup" && g.GameTime == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.gameTime", Code: "missing", Message: "matchup without gameTime"})
	}
	if g.Sport == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.sport", Code: "missing", Message: "missing sport field on game data"})
	}
}

// --- Standings data validation ---

type standingsDataValidation struct {
	League *string          `json:"league"`
	Date   *string          `json:"date"`
	Games  *json.RawMessage `json:"games"`
}

type standingsGameValidation struct {
	Home      *string `json:"home"`
	Away      *string `json:"away"`
	HomeScore *int    `json:"homeScore"`
	AwayScore *int    `json:"awayScore"`
	Status    *string `json:"status"`
}

func validateStandingsData(externalURL string, errs *[]validationIssue, warns *[]validationIssue) {
	var s standingsDataValidation
	if err := json.Unmarshal([]byte(externalURL), &s); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "external_url must be valid JSON for standings hint"})
		return
	}

	if s.League == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.league", Code: "required", Message: "league is required"})
	}
	if s.Date == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.date", Code: "required", Message: "date is required"})
	}
	if s.Games == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.games", Code: "required", Message: "games array is required"})
	} else {
		var games []standingsGameValidation
		if err := json.Unmarshal(*s.Games, &games); err != nil {
			*errs = append(*errs, validationIssue{Field: "external_url.games", Code: "invalid_json", Message: "games must be a valid JSON array"})
		} else {
			for i, g := range games {
				prefix := fmt.Sprintf("external_url.games[%d]", i)
				if g.Home == nil {
					*errs = append(*errs, validationIssue{Field: prefix + ".home", Code: "required", Message: fmt.Sprintf("game %d home team is required", i)})
				}
				if g.Away == nil {
					*errs = append(*errs, validationIssue{Field: prefix + ".away", Code: "required", Message: fmt.Sprintf("game %d away team is required", i)})
				}
				if g.HomeScore == nil {
					*errs = append(*errs, validationIssue{Field: prefix + ".homeScore", Code: "required", Message: fmt.Sprintf("game %d homeScore is required", i)})
				}
				if g.AwayScore == nil {
					*errs = append(*errs, validationIssue{Field: prefix + ".awayScore", Code: "required", Message: fmt.Sprintf("game %d awayScore is required", i)})
				}
				if g.Status == nil {
					*errs = append(*errs, validationIssue{Field: prefix + ".status", Code: "required", Message: fmt.Sprintf("game %d status is required", i)})
				}
			}
		}
	}
}

// --- Entertainment data validation ---

type entertainmentDataValidation struct {
	Subject  *string `json:"subject"`
	Headline *string `json:"headline"`
	Source   *string `json:"source"`
}

func validateEntertainmentData(externalURL string, errs *[]validationIssue, warns *[]validationIssue) {
	var e entertainmentDataValidation
	if err := json.Unmarshal([]byte(externalURL), &e); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "external_url must be valid JSON for entertainment hint"})
		return
	}

	if e.Subject == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.subject", Code: "required", Message: "subject is required"})
	}
	if e.Headline == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.headline", Code: "required", Message: "headline is required"})
	}
	if e.Source == nil {
		*errs = append(*errs, validationIssue{Field: "external_url.source", Code: "required", Message: "source is required"})
	}
}

// --- Video game data validation (game_release / game_review) ---

type videoGameDataValidation struct {
	Title       *string          `json:"title"`
	Status      *string          `json:"status"`
	Platforms   *json.RawMessage `json:"platforms"`
	Genres      *json.RawMessage `json:"genres"`
	ReleaseDate *string          `json:"releaseDate"`
}

func validateVideoGameData(externalURL string, hint string, errs *[]validationIssue, warns *[]validationIssue) {
	var g videoGameDataValidation
	if err := json.Unmarshal([]byte(externalURL), &g); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "external_url must be valid JSON for " + hint + " hint"})
		return
	}

	if g.Title == nil || *g.Title == "" {
		*errs = append(*errs, validationIssue{Field: "external_url.title", Code: "required", Message: "game title is required"})
	}
	if g.Status == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.status", Code: "missing", Message: "missing status field (upcoming | released | early_access)"})
	}
	if hint == "game_release" && g.ReleaseDate == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.releaseDate", Code: "missing", Message: "game_release without releaseDate"})
	}
}

// --- Food/restaurant data validation ---

type foodDataValidation struct {
	Name        *string  `json:"name"`
	Rating      *float64 `json:"rating"`
	ReviewCount *int     `json:"reviewCount"`
	Cuisine     []string `json:"cuisine"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

func validateFoodData(externalURL string, errs *[]validationIssue, warns *[]validationIssue) {
	var fd foodDataValidation
	if err := json.Unmarshal([]byte(externalURL), &fd); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "restaurant external_url must be valid JSON"})
		return
	}
	if fd.Name == nil || *fd.Name == "" {
		*errs = append(*errs, validationIssue{Field: "external_url.name", Code: "required", Message: "restaurant data missing name"})
	}
	if fd.Rating == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.rating", Code: "missing", Message: "restaurant data missing rating"})
	}
	if fd.Latitude == nil || fd.Longitude == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.latitude", Code: "missing", Message: "restaurant data missing coordinates"})
	}
}

// --- Handlers ---

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())

	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result := validatePost(&req)
	if !result.Valid {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}

	// Truncate labels if needed (warning was already issued)
	if len(req.Labels) > 20 {
		req.Labels = req.Labels[:20]
	}

	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	post, err := h.postRepo.Create(repository.CreatePostParams{
		AgentID:     agentID,
		UserID:      agent.UserID,
		Title:       req.Title,
		Body:        req.Body,
		ImageURL:    req.ImageURL,
		ExternalURL: req.ExternalURL,
		Locality:    req.Locality,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		PostType:    req.PostType,
		Visibility:  req.Visibility,
		DisplayHint: req.DisplayHint,
		Labels:      req.Labels,
		Images:      req.Images,
		ScheduledAt: req.ScheduledAt,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

func (h *PostHandler) LintPost(w http.ResponseWriter, r *http.Request) {
	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result := validatePost(&req)
	writeJSON(w, http.StatusOK, result)
}

func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())

	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	limit := 50
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "scheduled" {
		posts, err := h.postRepo.ListByUserIDWithStatus(agent.UserID, "scheduled", limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load posts"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
		return
	}

	posts, err := h.postRepo.ListByUserID(agent.UserID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load posts"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func (h *PostHandler) GetPostStats(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())

	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	periods := []int{7, 30, 90}
	var stats model.PostStats

	for _, days := range periods {
		ps, err := h.postRepo.Stats(agent.UserID, days)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to compute stats"})
			return
		}
		stats.Periods = append(stats.Periods, *ps)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
