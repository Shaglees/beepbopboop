package model

import "time"

// UserInterest represents a user's declared or inferred interest.
type UserInterest struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Category    string     `json:"category"`
	Topic       string     `json:"topic"`
	Source      string     `json:"source"`
	Confidence  float64    `json:"confidence"`
	Dismissed   bool       `json:"-"`
	PausedUntil *time.Time `json:"paused_until"`
	LastAskedAt *time.Time `json:"-"`
	TimesAsked  int        `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// LifestyleTag represents a structured lifestyle attribute.
type LifestyleTag struct {
	ID       string `json:"id,omitempty"`
	Category string `json:"category"`
	Value    string `json:"value"`
}

// ContentPref represents content delivery preferences, global or per-category.
type ContentPref struct {
	ID        string  `json:"id,omitempty"`
	Category  *string `json:"category"`
	Depth     string  `json:"depth"`
	Tone      string  `json:"tone"`
	MaxPerDay *int    `json:"max_per_day"`
}

// UserProfileIdentity holds the identity portion of a user profile.
type UserProfileIdentity struct {
	DisplayName  string   `json:"display_name"`
	AvatarURL    string   `json:"avatar_url"`
	Timezone     string   `json:"timezone"`
	HomeLocation string   `json:"home_location"`
	HomeLat      *float64 `json:"home_lat"`
	HomeLon      *float64 `json:"home_lon"`
}

// UserProfile is the full profile response returned by GET /user/profile.
type UserProfile struct {
	Identity           UserProfileIdentity `json:"identity"`
	Interests          []UserInterest      `json:"interests"`
	Lifestyle          []LifestyleTag      `json:"lifestyle"`
	ContentPrefs       []ContentPref       `json:"content_prefs"`
	ProfileInitialized bool                `json:"profile_initialized"`
}
