package dedup

import "time"

type PostEntry struct {
	ID          int64
	Title       string
	ExternalURL string
	PostType    string
	Locality    string
	Latitude    *float64
	Longitude   *float64
	Labels      []string
	BodyHash    string
	Tag         string
	CreatedAt   time.Time
}

type CheckInput struct {
	Title    string   `json:"title"`
	Labels   []string `json:"labels"`
	PostType string   `json:"post_type"`
	Locality string   `json:"locality,omitempty"`
	Lat      *float64 `json:"lat,omitempty"`
	Lon      *float64 `json:"lon,omitempty"`
	URL      string   `json:"url,omitempty"`
	Body     string   `json:"body,omitempty"`
	Tag      string   `json:"tag,omitempty"`
}

type Match struct {
	Title         string   `json:"title"`
	DaysAgo       int      `json:"days_ago"`
	Similarity    float64  `json:"similarity"`
	OverlapLabels []string `json:"overlap_labels"`
	SameType      bool     `json:"same_type"`
	DistanceKm    *float64 `json:"distance_km,omitempty"`
	Reason        string   `json:"reason"`
}

type CheckResult struct {
	Title   string  `json:"title"`
	Verdict string  `json:"verdict"` // DUPLICATE, SIMILAR, OK
	Matches []Match `json:"matches"`
}
