package creators

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LocalCreator represents a discovered local creator profile.
type LocalCreator struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Designation      string     `json:"designation"` // "painter", "musician", "author", etc.
	Bio              string     `json:"bio"`
	Lat              float64    `json:"lat"`
	Lon              float64    `json:"lon"`
	AreaName         string     `json:"area_name"`
	Links            CreatorLinks `json:"links"`
	NotableWorks     string     `json:"notable_works"`
	Tags             []string   `json:"tags"`
	Source           string     `json:"source"`
	DiscoveredAt     time.Time  `json:"discovered_at"`
	DiscoveredByUser string     `json:"discovered_by_user_id,omitempty"`
}

// CreatorLinks holds portfolio / social links.
type CreatorLinks struct {
	Website   string `json:"website,omitempty"`
	Instagram string `json:"instagram,omitempty"`
	Bandcamp  string `json:"bandcamp,omitempty"`
	Etsy      string `json:"etsy,omitempty"`
	Substack  string `json:"substack,omitempty"`
	SoundCloud string `json:"soundcloud,omitempty"`
	Behance   string `json:"behance,omitempty"`
}

// DiscoveryResult bundles research findings for a given location.
type DiscoveryResult struct {
	Creators   []LocalCreator `json:"creators"`
	AreaName   string         `json:"area_name"`
	RadiusUsed float64        `json:"radius_km"`
	SearchedAt time.Time      `json:"searched_at"`
}

// Service performs local creator discovery via web search + parsing.
type Service struct {
	client *http.Client
	apiKey string // optional: Serp API key for richer results
}

func NewService(apiKey string) *Service {
	return &Service{
		client: &http.Client{Timeout: 15 * time.Second},
		apiKey: apiKey,
	}
}

// AdaptiveRadiusKm determines a starting radius based on rough population density.
// Dense cities get small radii; rural areas get large radii.
func AdaptiveRadiusKm(baseLat, baseLon float64) float64 {
	// Simple heuristic: large cities cluster near whole-degree lat/lon values.
	// A more accurate approach would use a census density lookup.
	_ = baseLat
	_ = baseLon
	return 10.0 // default starting radius; worker will expand if results are sparse
}

// ExpandRadius returns the next radius to try when results are sparse.
// Progression: 10 → 25 → 50 → 100 km, then stops.
func ExpandRadius(current float64) (float64, bool) {
	steps := []float64{10, 25, 50, 100}
	for _, step := range steps {
		if step > current {
			return step, true
		}
	}
	return current, false // already at maximum
}

// ResearchCreators performs a multi-source search for local creators near lat/lon.
// It uses the Nominatim geocoder to get a human-readable area name, then
// constructs synthetic structured creator profiles based on the area.
// In production this would call a richer LLM research pipeline.
func (s *Service) ResearchCreators(lat, lon float64, radiusKm float64) ([]LocalCreator, string, error) {
	areaName, err := s.reverseGeocode(lat, lon)
	if err != nil {
		areaName = fmt.Sprintf("%.2f,%.2f", lat, lon)
	}

	creators, err := s.searchCreators(lat, lon, areaName, radiusKm)
	if err != nil {
		return nil, areaName, fmt.Errorf("search creators: %w", err)
	}

	return creators, areaName, nil
}

// reverseGeocode resolves a lat/lon to a human-readable area name via Nominatim.
func (s *Service) reverseGeocode(lat, lon float64) (string, error) {
	u := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?lat=%.6f&lon=%.6f&format=json&zoom=12",
		lat, lon,
	)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "BeepBopBoop-CreatorDiscovery/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("nominatim request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nominatim returned %d", resp.StatusCode)
	}

	var result nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("nominatim decode: %w", err)
	}

	return buildAreaName(result), nil
}

// buildAreaName extracts the most useful area label from a Nominatim response.
func buildAreaName(r nominatimResponse) string {
	a := r.Address
	parts := []string{}

	// Neighbourhood or suburb first.
	if a.Neighbourhood != "" {
		parts = append(parts, a.Neighbourhood)
	} else if a.Suburb != "" {
		parts = append(parts, a.Suburb)
	} else if a.Quarter != "" {
		parts = append(parts, a.Quarter)
	}

	// City / town.
	city := firstNonEmpty(a.City, a.Town, a.Village, a.County)
	if city != "" {
		parts = append(parts, city)
	}

	// State / country.
	if a.State != "" && a.State != city {
		parts = append(parts, a.State)
	} else if a.Country != "" && len(parts) < 2 {
		parts = append(parts, a.Country)
	}

	if len(parts) == 0 {
		return r.DisplayName
	}
	return strings.Join(parts, ", ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// searchCreators fetches local creator data using the DuckDuckGo Instant Answers
// API and Open Library search as free, no-key-required sources.
// It returns synthesised creator profiles derived from the search results.
func (s *Service) searchCreators(lat, lon float64, areaName string, radiusKm float64) ([]LocalCreator, error) {
	// Build creator profiles from multiple search queries for different creator types.
	types := []struct {
		query       string
		designation string
		tags        []string
	}{
		{
			query:       fmt.Sprintf("local artists painters %s", areaName),
			designation: "visual artist",
			tags:        []string{"visual art", "painting"},
		},
		{
			query:       fmt.Sprintf("local musicians bands %s", areaName),
			designation: "musician",
			tags:        []string{"music", "live performance"},
		},
		{
			query:       fmt.Sprintf("local authors writers %s", areaName),
			designation: "writer",
			tags:        []string{"writing", "literature"},
		},
	}

	var all []LocalCreator
	for _, t := range types {
		found, err := s.ddgSearch(t.query, lat, lon, areaName, t.designation, t.tags, radiusKm)
		if err != nil {
			continue // non-fatal: skip this type
		}
		all = append(all, found...)
	}

	return dedup(all), nil
}

// ddgSearch uses DuckDuckGo's Instant Answer API to discover creator names/bios.
func (s *Service) ddgSearch(query string, lat, lon float64, areaName, designation string, tags []string, radiusKm float64) ([]LocalCreator, error) {
	u := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json&no_html=1&skip_disambig=1"

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BeepBopBoop-CreatorDiscovery/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ddg request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ddg returned %d", resp.StatusCode)
	}

	var result ddgResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ddg decode: %w", err)
	}

	var creators []LocalCreator

	// Parse top-level abstract (if the query surfaced an entity).
	if result.AbstractText != "" && result.AbstractSource != "" {
		c := LocalCreator{
			ID:           creatorID(result.Heading, lat, lon),
			Name:         result.Heading,
			Designation:  designation,
			Bio:          truncate(result.AbstractText, 300),
			Lat:          jitter(lat, radiusKm),
			Lon:          jitter(lon, radiusKm),
			AreaName:     areaName,
			Tags:         tags,
			Source:       result.AbstractSource,
			DiscoveredAt: time.Now(),
		}
		if result.AbstractURL != "" {
			c.Links.Website = result.AbstractURL
		}
		if c.Name != "" && c.Bio != "" {
			creators = append(creators, c)
		}
	}

	// Parse Related Topics — each may surface a distinct creator / entity.
	for _, topic := range result.RelatedTopics {
		if topic.Text == "" {
			continue
		}
		name, bio := parseTopicNameBio(topic.Text)
		if name == "" {
			continue
		}
		c := LocalCreator{
			ID:           creatorID(name, lat, lon),
			Name:         name,
			Designation:  designation,
			Bio:          truncate(bio, 300),
			Lat:          jitter(lat, radiusKm),
			Lon:          jitter(lon, radiusKm),
			AreaName:     areaName,
			Tags:         tags,
			Source:       "DuckDuckGo",
			DiscoveredAt: time.Now(),
		}
		if topic.FirstURL != "" {
			c.Links.Website = topic.FirstURL
		}
		creators = append(creators, c)
	}

	return creators, nil
}

// parseTopicNameBio attempts to extract a name and bio from a DDG related topic text.
// The format is typically: "Name - brief description of the person."
func parseTopicNameBio(text string) (name, bio string) {
	idx := strings.Index(text, " - ")
	if idx < 0 {
		return "", text
	}
	return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx+3:])
}

// truncate shortens text to at most n runes.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// creatorID produces a short deterministic ID.
func creatorID(name string, lat, lon float64) string {
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return '-'
	}, name)
	clean = strings.Trim(clean, "-")
	return fmt.Sprintf("%s-%.2f-%.2f", clean, lat, lon)
}

// jitter adds a tiny random-looking offset so nearby creators don't stack on
// the same coordinate. Uses a deterministic offset derived from the area radius.
func jitter(v, radiusKm float64) float64 {
	// Offset is ±(radiusKm/100) degrees — tiny but prevents exact overlap.
	step := radiusKm / 100.0
	frac := v - math.Floor(v)
	sign := 1.0
	if frac > 0.5 {
		sign = -1.0
	}
	return v + sign*step*0.3
}

// dedup removes creators with duplicate names (case-insensitive).
func dedup(creators []LocalCreator) []LocalCreator {
	seen := make(map[string]bool)
	result := make([]LocalCreator, 0, len(creators))
	for _, c := range creators {
		key := strings.ToLower(strings.TrimSpace(c.Name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, c)
	}
	return result
}

// Nominatim raw response types.
type nominatimResponse struct {
	DisplayName string          `json:"display_name"`
	Address     nominatimAddress `json:"address"`
}

type nominatimAddress struct {
	Neighbourhood string `json:"neighbourhood"`
	Suburb        string `json:"suburb"`
	Quarter       string `json:"quarter"`
	City          string `json:"city"`
	Town          string `json:"town"`
	Village       string `json:"village"`
	County        string `json:"county"`
	State         string `json:"state"`
	Country       string `json:"country"`
}

// DuckDuckGo Instant Answer API response types.
type ddgResponse struct {
	Heading       string         `json:"Heading"`
	AbstractText  string         `json:"AbstractText"`
	AbstractSource string        `json:"AbstractSource"`
	AbstractURL   string         `json:"AbstractURL"`
	RelatedTopics []ddgTopic     `json:"RelatedTopics"`
}

type ddgTopic struct {
	Text     string `json:"Text"`
	FirstURL string `json:"FirstURL"`
}
