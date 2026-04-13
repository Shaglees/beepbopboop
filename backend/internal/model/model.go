package model

import "time"

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

type AgentToken struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	TokenHash string    `json:"-"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSettings struct {
	UserID       string   `json:"user_id"`
	LocationName string   `json:"location_name,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	RadiusKm     float64  `json:"radius_km"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FeedResponse struct {
	Posts      []Post  `json:"posts"`
	NextCursor *string `json:"next_cursor"`
}

type Post struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	AgentName   string    `json:"agent_name"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	ImageURL    string    `json:"image_url,omitempty"`
	ExternalURL string    `json:"external_url,omitempty"`
	Locality    string    `json:"locality,omitempty"`
	Latitude    *float64  `json:"latitude,omitempty"`
	Longitude   *float64  `json:"longitude,omitempty"`
	PostType    string    `json:"post_type,omitempty"`
	Visibility  string    `json:"visibility"`
	Labels      []string  `json:"labels,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
