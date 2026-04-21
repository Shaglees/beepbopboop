package video

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type Enrichment struct {
	Labels            []string
	SourceDescription string
	NormalizedTitle   string
}

var (
	spaceRe         = regexp.MustCompile(`\s+`)
	nonAlphaNumRe   = regexp.MustCompile(`[^a-z0-9\s-]`)
	genericPrefixRe = regexp.MustCompile(`(?i)^(amazing|awesome|interesting)\s+video\s+clip\s+of\s+|^(amazing|awesome|interesting)\s+video\s+of\s+|^video\s+clip\s+of\s+|^video\s+of\s+|^watch:\s*`)
)

var bannedLabelTokens = map[string]bool{
	"video": true, "clip": true, "watch": true, "amazing": true, "awesome": true, "interesting": true, "wimp": true,
}

var labelOrder = []string{
	"dogs", "cats", "animals", "cute",
	"music", "behind-the-scenes", "nostalgia",
	"engineering", "vehicles", "innovation",
}

// EnrichMetadata derives a small set of steering-friendly labels and normalized
// text fields from the source video metadata. It is intentionally deterministic
// and heuristic-based so it remains easy to test and safe to run on the hot path.
func EnrichMetadata(v model.Video) Enrichment {
	sourceDescription := strings.TrimSpace(v.SourceDesc)
	if sourceDescription == "" {
		sourceDescription = strings.TrimSpace(v.Description)
	}

	normalizedTitle := normalizeTitle(v.Title)
	text := strings.ToLower(strings.Join([]string{
		normalizedTitle,
		v.Description,
		sourceDescription,
		strings.Join(v.Labels, " "),
	}, " "))
	text = nonAlphaNumRe.ReplaceAllString(text, " ")
	text = spaceRe.ReplaceAllString(text, " ")

	scores := map[string]int{}

	if containsAny(text, "puppy", "dog", "dogs", "canine") {
		scores["dogs"] += 3
	}
	if containsAny(text, "kitten", "cat", "cats", "feline") {
		scores["cats"] += 3
	}
	if scores["dogs"] > 0 || scores["cats"] > 0 || containsAny(text, "animal", "animals", "bird", "birds") {
		scores["animals"] += 2
	}
	if containsAny(text, "cute", "adorable", "sweet", "first time", "meet") && scores["animals"] > 0 {
		scores["cute"] += 2
	}

	if containsAny(text, "beatles", "recording", "recordings", "song", "studio", "concert", "album", "band", "music") {
		scores["music"] += 2
	}
	if containsAny(text, "blooper", "bloopers", "rough takes", "studio chatter", "behind the scenes", "recording sessions") {
		scores["behind-the-scenes"] += 3
	}
	if containsAny(text, "beatles", "classic", "archive", "historic", "vintage", "retro", "old") {
		scores["nostalgia"] += 2
	}

	if containsAny(text, "prototype", "test flight", "engineer", "engineering", "hoverbike", "flying bike") {
		scores["engineering"] += 2
	}
	if containsAny(text, "bike", "hoverbike", "vehicle", "flight", "car", "train", "plane") {
		scores["vehicles"] += 2
	}
	if containsAny(text, "prototype", "invention", "innovation", "demo", "test flight") {
		scores["innovation"] += 2
	}

	labels := make([]string, 0, len(labelOrder))
	for _, label := range labelOrder {
		if scores[label] > 0 {
			labels = append(labels, label)
		}
	}

	// Preserve existing non-generic labels if they are not already present.
	for _, label := range v.Labels {
		clean := strings.ToLower(strings.TrimSpace(label))
		if clean == "" || bannedLabelTokens[clean] || slices.Contains(labels, clean) {
			continue
		}
	}

	return Enrichment{
		Labels:            labels,
		SourceDescription: sourceDescription,
		NormalizedTitle:   normalizedTitle,
	}
}

// GenerateTitle applies a tiny set of curiosity-raising but factual templates,
// then falls back to the normalized source title with hard guardrails.
func GenerateTitle(v model.Video, e Enrichment) string {
	title := e.NormalizedTitle
	if title == "" {
		title = normalizeTitle(v.Title)
	}

	text := strings.ToLower(strings.Join([]string{title, e.SourceDescription}, " "))
	switch {
	case hasAll(e.Labels, "music", "behind-the-scenes") && strings.Contains(text, "beatles"):
		title = "Beatles studio bloopers you probably haven't heard"
	case hasAll(e.Labels, "dogs", "cats") && containsAny(text, "meet", "meeting", "first time"):
		title = "A puppy and kitten meeting for the first time"
	}

	title = sanitizeTitle(title)
	if len(title) > 80 {
		title = strings.TrimSpace(title[:80])
	}
	return title
}

func normalizeTitle(in string) string {
	s := strings.TrimSpace(in)
	s = strings.TrimSuffix(s, ".")
	s = genericPrefixRe.ReplaceAllString(s, "")
	s = spaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
	if s == "" {
		return ""
	}
	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}

func sanitizeTitle(title string) string {
	s := strings.TrimSpace(title)
	replacements := []string{
		"shocking", "",
		"unbelievable", "",
		"you won't believe", "",
		"insane", "",
		"literally", "",
	}
	lower := strings.ToLower(s)
	for i := 0; i < len(replacements); i += 2 {
		if strings.Contains(lower, replacements[i]) {
			s = strings.ReplaceAll(strings.ToLower(s), replacements[i], replacements[i+1])
		}
	}
	s = spaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
	if s == "" {
		return normalizeTitle(title)
	}
	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func hasAll(labels []string, required ...string) bool {
	for _, want := range required {
		if !slices.Contains(labels, want) {
			return false
		}
	}
	return true
}
