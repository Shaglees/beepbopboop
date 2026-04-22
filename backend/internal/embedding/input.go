package embedding

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// BuildEmbeddingInput composes a single embedding input string from a post's fields.
// Structured display hints (scoreboard, weather) are summarised into natural language
// so the embedding captures semantic meaning rather than raw JSON.
func BuildEmbeddingInput(p model.Post) string {
	parts := []string{}
	if p.Title != "" {
		parts = append(parts, p.Title)
	}
	if p.Body != "" {
		parts = append(parts, p.Body)
	}

	// Structured hints: summarise ExternalURL rather than embed raw JSON.
	if p.ExternalURL != "" && (p.DisplayHint == "scoreboard" || p.DisplayHint == "weather") {
		summary := SummariseForEmbedding(p.DisplayHint, p.ExternalURL)
		if summary != "" {
			parts = append(parts, summary)
		}
	}

	if len(p.Labels) > 0 {
		parts = append(parts, "Labels: "+strings.Join(p.Labels, ", "))
	}
	if p.PostType != "" {
		parts = append(parts, "Type: "+p.PostType)
	}
	if p.Locality != "" {
		parts = append(parts, "Location: "+p.Locality)
	}

	return strings.Join(parts, ". ")
}

// BuildEmbeddingPayload prepares a multimodal payload from a post.
// Providers that don't support multimodal inputs can still use Text.
func BuildEmbeddingPayload(p model.Post) EmbeddingInput {
	payload := EmbeddingInput{Text: BuildEmbeddingInput(p)}
	payload.ImageURLs = extractImageURLs(p)
	return payload
}

func extractImageURLs(p model.Post) []string {
	seen := map[string]struct{}{}
	urls := make([]string, 0, 4)
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || !strings.HasPrefix(u, "http") {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}

	add(p.ImageURL)

	if len(p.Images) > 0 {
		var imgs []map[string]any
		if err := json.Unmarshal(p.Images, &imgs); err == nil {
			for _, img := range imgs {
				if u, ok := img["url"].(string); ok {
					add(u)
				}
			}
		}
	}

	return urls
}

// SummariseForEmbedding converts a structured ExternalURL payload into a natural
// language phrase suitable for embedding. Returns the raw payload as-is for
// unknown hints (callers should not embed raw JSON for structured hints).
func SummariseForEmbedding(hint, externalURL string) string {
	switch hint {
	case "scoreboard":
		return summariseScoreboard(externalURL)
	case "weather":
		return summariseWeather(externalURL)
	default:
		return ""
	}
}

func summariseScoreboard(raw string) string {
	var g struct {
		Sport  string `json:"sport"`
		Status string `json:"status"`
		Home   struct {
			Name  string  `json:"name"`
			Score float64 `json:"score"`
		} `json:"home"`
		Away struct {
			Name  string  `json:"name"`
			Score float64 `json:"score"`
		} `json:"away"`
	}
	if err := json.Unmarshal([]byte(raw), &g); err != nil {
		return ""
	}
	if g.Home.Name == "" && g.Away.Name == "" {
		return ""
	}
	status := g.Status
	if status == "" {
		status = "game"
	}
	return fmt.Sprintf("%s %.0f, %s %.0f, %s",
		g.Home.Name, g.Home.Score, g.Away.Name, g.Away.Score, status)
}

func summariseWeather(raw string) string {
	var w struct {
		Current struct {
			TempC     *float64 `json:"temp_c"`
			TempF     *float64 `json:"temp_f"`
			Condition string   `json:"condition"`
		} `json:"current"`
	}
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		return ""
	}
	var temp float64
	unit := "C"
	switch {
	case w.Current.TempC != nil:
		temp = *w.Current.TempC
	case w.Current.TempF != nil:
		temp = *w.Current.TempF
		unit = "F"
	}
	condition := w.Current.Condition
	if condition == "" {
		condition = "weather"
	}
	return fmt.Sprintf("Weather: %.0f°%s, %s", temp, unit, condition)
}
