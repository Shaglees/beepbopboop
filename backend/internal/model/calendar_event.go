package model

import (
	"encoding/json"
	"time"
)

// InterestCalendarEvent represents a curated event in the interest_calendar_events table.
// This is distinct from CalendarEvent (iOS device calendar sync).
type InterestCalendarEvent struct {
	ID           string          `json:"id"`
	EventKey     string          `json:"event_key"`
	Domain       string          `json:"domain"`
	Title        string          `json:"title"`
	StartTime    time.Time       `json:"start_time"`
	EndTime      *time.Time      `json:"end_time,omitempty"`
	Timezone     string          `json:"timezone"`
	Status       string          `json:"status"`
	EntityType   string          `json:"entity_type"`
	EntityIDs    json.RawMessage `json:"entity_ids"`
	InterestTags []string        `json:"interest_tags"`
	Payload      json.RawMessage `json:"payload"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CalendarPostLog struct {
	EventKey  string    `json:"event_key"`
	UserID    string    `json:"user_id"`
	Window    string    `json:"window"`
	PostID    string    `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}
