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
	PostType    string    `json:"post_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
