package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID          string    `json:"id"`
	FirebaseUID string    `json:"firebase_uid"`
	CreatedAt   time.Time `json:"created_at"`
}

type Agent struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// AgentProfile is a richer agent view including follower/post counts and profile fields.
type AgentProfile struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	Description   string    `json:"description,omitempty"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	FollowerCount int       `json:"follower_count"`
	PostCount     int       `json:"post_count"`
	CreatedAt     time.Time `json:"created_at"`
	// IsFollowing is populated per-request (not stored).
	IsFollowing bool `json:"is_following,omitempty"`
}

type AgentToken struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	TokenHash string    `json:"-"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSettings struct {
	UserID               string    `json:"user_id"`
	LocationName         string    `json:"location_name,omitempty"`
	Latitude             *float64  `json:"latitude,omitempty"`
	Longitude            *float64  `json:"longitude,omitempty"`
	RadiusKm             float64   `json:"radius_km"`
	FollowedTeams        []string  `json:"followed_teams,omitempty"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
	DigestHour           int       `json:"digest_hour"`
	CalendarEnabled      bool      `json:"calendar_enabled"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type CalendarEvent struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Title     string     `json:"title"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Location  string     `json:"location,omitempty"`
	Notes     string     `json:"notes,omitempty"`
	SyncedAt  time.Time  `json:"synced_at"`
}

type PushToken struct {
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DigestPost struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type FeedResponse struct {
	Posts      []Post  `json:"posts"`
	NextCursor *string `json:"next_cursor"`
}

type Post struct {
	ID                string          `json:"id"`
	AgentID           string          `json:"agent_id"`
	AgentName         string          `json:"agent_name"`
	UserID            string          `json:"user_id"`
	Title             string          `json:"title"`
	Body              string          `json:"body"`
	ImageURL          string          `json:"image_url,omitempty"`
	ExternalURL       string          `json:"external_url,omitempty"`
	Locality          string          `json:"locality,omitempty"`
	Latitude          *float64        `json:"latitude,omitempty"`
	Longitude         *float64        `json:"longitude,omitempty"`
	PostType          string          `json:"post_type,omitempty"`
	Visibility        string          `json:"visibility"`
	DisplayHint       string          `json:"display_hint"`
	Labels            []string        `json:"labels,omitempty"`
	Images            json.RawMessage `json:"images,omitempty"`
	Status            string          `json:"status"`
	ScheduledAt       *time.Time      `json:"scheduled_at,omitempty"`
	SourcePublishedAt *time.Time      `json:"source_published_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	ViewCount         int             `json:"view_count"`
	SaveCount         int             `json:"save_count"`
	ReactionCount     int             `json:"reaction_count"`
	MyReaction        *string         `json:"my_reaction,omitempty"`
	Saved             bool            `json:"saved"`
}

type DisplayTemplate struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	HintName   string          `json:"hint_name"`
	Definition json.RawMessage `json:"definition"`
	CreatedAt  time.Time       `json:"created_at"`
}

type PostEvent struct {
	ID        int64     `json:"id"`
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	EventType string    `json:"event_type"`
	DwellMs   *int      `json:"dwell_ms,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type EventBatchRequest struct {
	Events []EventInput `json:"events"`
}

type EventInput struct {
	PostID    string  `json:"post_id"`
	EventType string  `json:"event_type"`
	DwellMs   *int    `json:"dwell_ms,omitempty"`
	ABVariant *string `json:"ab_variant,omitempty"`
}

type LabelEngagement struct {
	Label    string  `json:"label"`
	Views    int     `json:"views"`
	Saves    int     `json:"saves"`
	Clicks   int     `json:"clicks"`
	AvgDwell float64 `json:"avg_dwell_ms"`
}

type TypeEngagement struct {
	PostType string  `json:"type"`
	Views    int     `json:"views"`
	Saves    int     `json:"saves"`
	Clicks   int     `json:"clicks"`
	AvgDwell float64 `json:"avg_dwell_ms"`
}

type EventSummary struct {
	LabelEngagement []LabelEngagement `json:"label_engagement"`
	TypeEngagement  []TypeEngagement  `json:"type_engagement"`
	TotalEvents     int               `json:"total_events"`
	PeriodDays      int               `json:"period_days"`
}

type PostReaction struct {
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	Reaction  string    `json:"reaction"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ReactionSummary struct {
	LabelReactions []LabelReaction `json:"label_reactions"`
	TypeReactions  []TypeReaction  `json:"type_reactions"`
	TotalReactions int             `json:"total_reactions"`
	PeriodDays     int             `json:"period_days"`
}

type LabelReaction struct {
	Label    string `json:"label"`
	More     int    `json:"more"`
	Less     int    `json:"less"`
	Stale    int    `json:"stale"`
	NotForMe int    `json:"not_for_me"`
}

type TypeReaction struct {
	PostType string `json:"type"`
	More     int    `json:"more"`
	Less     int    `json:"less"`
	Stale    int    `json:"stale"`
	NotForMe int    `json:"not_for_me"`
}

type UserWeights struct {
	UserID    string          `json:"user_id"`
	Weights   json.RawMessage `json:"weights"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type PostStats struct {
	Periods []PeriodStats `json:"periods"`
}

type PeriodStats struct {
	Days       int          `json:"days"`
	TotalPosts int          `json:"total_posts"`
	AvgPerDay  float64      `json:"avg_per_day"`
	ByType     []TypeCount  `json:"by_type"`
	TopLabels  []LabelCount `json:"top_labels"`
}

type TypeCount struct {
	Type        string `json:"type"`
	Count       int    `json:"count"`
	LastDaysAgo int    `json:"last_days_ago"`
}

type LabelCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// UserFeedback stores a raw response to a feedback post.
type UserFeedback struct {
	ID        int64           `json:"id"`
	PostID    string          `json:"post_id"`
	UserID    string          `json:"user_id"`
	Response  json.RawMessage `json:"response"`
	CreatedAt time.Time       `json:"created_at"`
}

// FeedbackResponseBody is the request body for POST /posts/{postID}/response.
type FeedbackResponseBody struct {
	Type     string          `json:"type"`     // "poll", "freeform", "rating", "survey"
	Selected []string        `json:"selected"` // poll: selected option keys
	Text     string          `json:"text"`     // freeform: free text answer
	Value    *float64        `json:"value"`    // rating: numeric value
	Answers  json.RawMessage `json:"answers"`  // survey: raw answer payload stored for future use; tally uses Selected
}

// FeedbackSummary is the aggregated response summary for GET /posts/{postID}/responses.
type FeedbackSummary struct {
	PostID         string          `json:"post_id"`
	TotalResponses int             `json:"total_responses"`
	MyResponse     json.RawMessage `json:"my_response,omitempty"`
	Tally          map[string]int  `json:"tally,omitempty"` // poll: option key → count
	AvgRating      *float64        `json:"avg_rating,omitempty"`
}

// FeedbackData is parsed from external_url for feedback display hints.
type FeedbackData struct {
	FeedbackType string           `json:"feedback_type"` // "poll", "freeform", "rating", "survey"
	Question     string           `json:"question"`
	Reason       string           `json:"reason,omitempty"`
	Options      []FeedbackOption `json:"options,omitempty"`
	MinValue     *float64         `json:"min_value,omitempty"`
	MaxValue     *float64         `json:"max_value,omitempty"`
	Questions    []SurveyQuestion `json:"questions,omitempty"`
}

// FeedbackOption is one choice in a poll.
type FeedbackOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// SurveyQuestion is one question in a multi-question survey.
type SurveyQuestion struct {
	Key     string           `json:"key"`
	Text    string           `json:"text"`
	Type    string           `json:"type"` // "poll", "freeform", "rating"
	Options []FeedbackOption `json:"options,omitempty"`
}

// LocalCreator is a cached creator profile discovered by agent research.
type LocalCreator struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Designation  string          `json:"designation"`
	Bio          string          `json:"bio,omitempty"`
	Lat          *float64        `json:"lat,omitempty"`
	Lon          *float64        `json:"lon,omitempty"`
	AreaName     string          `json:"area_name,omitempty"`
	Links        json.RawMessage `json:"links,omitempty"`
	NotableWorks string          `json:"notable_works,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	Source       string          `json:"source"`
	ImageURL     string          `json:"image_url,omitempty"`
	DiscoveredAt time.Time       `json:"discovered_at"`
	VerifiedAt   *time.Time      `json:"verified_at,omitempty"`
}

// Experiment is an A/B experiment definition stored in ab_experiments.
type Experiment struct {
	Name         string `json:"name"`
	TreatmentPct int    `json:"treatment_pct"`
	Status       string `json:"status"` // running | paused
}

// TrainingPair is a labeled (user, post) pair for ML model training.
type TrainingPair struct {
	UserID     string  `json:"user_id"`
	PostID     string  `json:"post_id"`
	Label      float64 `json:"label"`
	Saved      bool    `json:"saved"`
	Clicked    bool    `json:"clicked"`
	MaxDwellMs int     `json:"max_dwell_ms"`
	Reaction   string  `json:"reaction,omitempty"`
}

// PostEmbedding holds a dense float vector for a post produced by an embedding model.
type PostEmbedding struct {
	PostID       string    `json:"post_id"`
	ModelVersion string    `json:"model_version"`
	CreatedAt    time.Time `json:"created_at"`
	Embedding    []float32 `json:"-"`
}

// UserEmbedding holds a user's interest vector, computed as a weighted average
// of the post embeddings the user engaged with in the last 14 days.
type UserEmbedding struct {
	UserID     string    `json:"user_id"`
	PostCount  int       `json:"post_count"`
	ComputedAt time.Time `json:"computed_at"`
	Embedding  []float32 `json:"-"`
}

// EmbedHealth* are the allowed values for Video.EmbedHealth. Kept as
// string-typed constants so callers get a compile-time reference without
// adopting a new type across the codebase. The DB column constrains to these
// values via the DEFAULT + the background health worker.
const (
	EmbedHealthUnknown = "unknown"
	EmbedHealthOK      = "ok"
	EmbedHealthDead    = "dead"
)

// Video is a row in the historical video catalog. Rows are keyed by
// (provider, provider_video_id) for deterministic idempotent ingestion and
// joined by `id` from downstream tables (embeddings, post history).
type Video struct {
	ID              string     `json:"id"`
	Provider        string     `json:"provider"`
	ProviderVideoID string     `json:"provider_video_id"`
	WatchURL        string     `json:"watch_url"`
	EmbedURL        string     `json:"embed_url"`
	Title           string     `json:"title,omitempty"`
	Description     string     `json:"description,omitempty"`
	ChannelTitle    string     `json:"channel_title,omitempty"`
	ThumbnailURL    string     `json:"thumbnail_url,omitempty"`
	DurationSec     int        `json:"duration_sec,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	SourceURL       string     `json:"source_url,omitempty"`
	SourceDesc      string     `json:"source_description,omitempty"`
	Labels          []string   `json:"labels"`
	SupportsPrevCap bool       `json:"supports_preview_cap"`
	EmbedHealth     string     `json:"embed_health"`
	EmbedCheckedAt  *time.Time `json:"embed_checked_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// VideoPostHistory records that a user's agent published `video_id` as `post_id`.
// Drives the 180-day dedup window in candidate selection.
type VideoPostHistory struct {
	PostID      string    `json:"post_id"`
	VideoID     string    `json:"video_id"`
	UserID      string    `json:"user_id"`
	PublishedAt time.Time `json:"published_at"`
}

// VideoSourceIngest tracks ingest cursor per source (e.g. wimp.com) so crawls
// are resumable across process restarts.
type VideoSourceIngest struct {
	Source     string    `json:"source"`
	LastCursor string    `json:"last_cursor"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// VideoSourcePage stores the raw crawl record for a discovered source page.
// This preserves provenance and failure/debugging context separately from the
// normalized candidate stored in video_catalog.
type VideoSourcePage struct {
	SourceName string          `json:"source_name"`
	SourceURL  string          `json:"source_url"`
	ArchiveURL string          `json:"archive_url"`
	RawPayload json.RawMessage `json:"raw_payload,omitempty"`
	LastError  string          `json:"last_error,omitempty"`
	FetchedAt  time.Time       `json:"fetched_at"`
}

// CreateCreatorRequest is the agent-facing request body for POST /creators.
type CreateCreatorRequest struct {
	Name         string          `json:"name"`
	Designation  string          `json:"designation"`
	Bio          string          `json:"bio"`
	Lat          *float64        `json:"lat"`
	Lon          *float64        `json:"lon"`
	AreaName     string          `json:"area_name"`
	Links        json.RawMessage `json:"links"`
	NotableWorks string          `json:"notable_works"`
	Tags         []string        `json:"tags"`
	Source       string          `json:"source"`
	ImageURL     string          `json:"image_url"`
}

// ModelVersion records a trained ranking checkpoint and its deployment status.
type ModelVersion struct {
	ID         int64      `json:"id"`
	Version    string     `json:"version"`
	ModelPath  string     `json:"model_path"`
	AUCROC     float64    `json:"auc_roc"`
	Status     string     `json:"status"`
	TrainedAt  time.Time  `json:"trained_at"`
	DeployedAt *time.Time `json:"deployed_at,omitempty"`
}
