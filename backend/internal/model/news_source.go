package model

import "time"

type NewsSource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	FeedURL     string    `json:"feed_url,omitempty"`
	AreaLabel   string    `json:"area_label"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RadiusKm    float64   `json:"radius_km"`
	Topics      []string  `json:"topics"`
	TrustScore  int       `json:"trust_score"`
	FetchMethod string    `json:"fetch_method"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
