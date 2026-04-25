package calendar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"
)

// postContent holds the rendered content for a calendar-derived post.
type postContent struct {
	Title       string
	Body        string
	DisplayHint string
	ExternalURL string
}

// sportsPreviewTmpl renders a preview post for a sports game (T-24h to T-12h window).
var sportsPreviewTmpl = template.Must(template.New("sports_preview").Parse(
	`{{.Away}} @ {{.Home}} — {{.StartTimeFormatted}}`,
))

var sportsPreviewBodyTmpl = template.Must(template.New("sports_preview_body").Parse(
	`{{.Home}} ({{.HomeRecord}}) host {{.Away}} ({{.AwayRecord}}) at {{.Venue}}. {{.Broadcast}}.`,
))

// sportsImminentTmpl renders an imminent post for a sports game (T-2h to T-0 window).
var sportsImminentTmpl = template.Must(template.New("sports_imminent").Parse(
	`{{.Away}} @ {{.Home}} tips off in {{.TimeUntil}}`,
))

var sportsImminentBodyTmpl = template.Must(template.New("sports_imminent_body").Parse(
	`{{.Home}} ({{.HomeRecord}}) vs {{.Away}} ({{.AwayRecord}}).`,
))

// entertainmentPreviewTmpl renders a preview post for an entertainment release.
var entertainmentPreviewTmpl = template.Must(template.New("entertainment_preview").Parse(
	`{{.Title}} hits theaters {{.ReleaseDay}}`,
))

var entertainmentPreviewBodyTmpl = template.Must(template.New("entertainment_preview_body").Parse(
	`{{.Overview}}. Starring {{.Cast}}.`,
))

// entertainmentReleaseTmpl renders a release-day post.
var entertainmentReleaseTmpl = template.Must(template.New("entertainment_release").Parse(
	`{{.Title}} is out today`,
))

var entertainmentReleaseBodyTmpl = template.Must(template.New("entertainment_release_body").Parse(
	`{{.Overview}}. Now playing at theaters near you.`,
))

func renderTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderSportsPost renders a sports game post for the given window ("preview" or "imminent").
func renderSportsPost(payload json.RawMessage, startTime time.Time, window string) (*postContent, error) {
	var p map[string]interface{}
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("unmarshal sports payload: %w", err)
	}

	getString := func(key string) string {
		if v, ok := p[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	home := getString("home")
	away := getString("away")
	homeRecord := getString("home_record")
	awayRecord := getString("away_record")
	venue := getString("venue")
	broadcast := getString("broadcast")

	if home == "" {
		home = getString("home_team")
	}
	if away == "" {
		away = getString("away_team")
	}

	// external_url is the raw payload JSON for client-side parsing
	externalURL := string(payload)

	switch window {
	case "preview":
		titleData := struct {
			Away               string
			Home               string
			StartTimeFormatted string
		}{
			Away:               away,
			Home:               home,
			StartTimeFormatted: startTime.Format("Mon Jan 2 3:04 PM"),
		}
		title, err := renderTemplate(sportsPreviewTmpl, titleData)
		if err != nil {
			return nil, fmt.Errorf("render sports preview title: %w", err)
		}

		bodyData := struct {
			Home       string
			HomeRecord string
			Away       string
			AwayRecord string
			Venue      string
			Broadcast  string
		}{
			Home:       home,
			HomeRecord: homeRecord,
			Away:       away,
			AwayRecord: awayRecord,
			Venue:      venue,
			Broadcast:  broadcast,
		}
		body, err := renderTemplate(sportsPreviewBodyTmpl, bodyData)
		if err != nil {
			return nil, fmt.Errorf("render sports preview body: %w", err)
		}

		return &postContent{
			Title:       title,
			Body:        body,
			DisplayHint: "matchup",
			ExternalURL: externalURL,
		}, nil

	case "imminent":
		timeUntil := formatTimeUntil(time.Until(startTime))
		titleData := struct {
			Away      string
			Home      string
			TimeUntil string
		}{
			Away:      away,
			Home:      home,
			TimeUntil: timeUntil,
		}
		title, err := renderTemplate(sportsImminentTmpl, titleData)
		if err != nil {
			return nil, fmt.Errorf("render sports imminent title: %w", err)
		}

		bodyData := struct {
			Home       string
			HomeRecord string
			Away       string
			AwayRecord string
		}{
			Home:       home,
			HomeRecord: homeRecord,
			Away:       away,
			AwayRecord: awayRecord,
		}
		body, err := renderTemplate(sportsImminentBodyTmpl, bodyData)
		if err != nil {
			return nil, fmt.Errorf("render sports imminent body: %w", err)
		}

		return &postContent{
			Title:       title,
			Body:        body,
			DisplayHint: "matchup",
			ExternalURL: externalURL,
		}, nil

	default:
		return nil, fmt.Errorf("unknown sports window: %q", window)
	}
}

// renderEntertainmentPost renders an entertainment release post for the given window
// ("preview" or "release_day").
func renderEntertainmentPost(payload json.RawMessage, startTime time.Time, window string) (*postContent, error) {
	var p map[string]interface{}
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("unmarshal entertainment payload: %w", err)
	}

	getString := func(key string) string {
		if v, ok := p[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	title := getString("title")
	overview := getString("overview")
	cast := getString("cast")

	externalURL := string(payload)

	switch window {
	case "preview":
		titleData := struct {
			Title      string
			ReleaseDay string
		}{
			Title:      title,
			ReleaseDay: startTime.Format("January 2"),
		}
		renderedTitle, err := renderTemplate(entertainmentPreviewTmpl, titleData)
		if err != nil {
			return nil, fmt.Errorf("render entertainment preview title: %w", err)
		}

		bodyData := struct {
			Overview string
			Cast     string
		}{
			Overview: overview,
			Cast:     cast,
		}
		body, err := renderTemplate(entertainmentPreviewBodyTmpl, bodyData)
		if err != nil {
			return nil, fmt.Errorf("render entertainment preview body: %w", err)
		}

		return &postContent{
			Title:       renderedTitle,
			Body:        body,
			DisplayHint: "event",
			ExternalURL: externalURL,
		}, nil

	case "release_day":
		titleData := struct {
			Title string
		}{Title: title}
		renderedTitle, err := renderTemplate(entertainmentReleaseTmpl, titleData)
		if err != nil {
			return nil, fmt.Errorf("render entertainment release title: %w", err)
		}

		bodyData := struct {
			Overview string
		}{Overview: overview}
		body, err := renderTemplate(entertainmentReleaseBodyTmpl, bodyData)
		if err != nil {
			return nil, fmt.Errorf("render entertainment release body: %w", err)
		}

		return &postContent{
			Title:       renderedTitle,
			Body:        body,
			DisplayHint: "event",
			ExternalURL: externalURL,
		}, nil

	default:
		return nil, fmt.Errorf("unknown entertainment window: %q", window)
	}
}

// formatTimeUntil formats a duration into a human-readable string like "2h 30m".
func formatTimeUntil(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
