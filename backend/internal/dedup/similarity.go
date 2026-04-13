package dedup

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/geo"
)

// genericLabels are low-specificity labels that get half weight in similarity scoring.
var genericLabels = map[string]bool{
	"place": true, "event": true, "article": true, "discovery": true, "video": true,
	"food": true, "drink": true, "entertainment": true, "trending": true,
	"local": true, "news": true, "weekly-digest": true,
}

const (
	wType    = 0.20
	wRecency = 0.15
)

// Check scores a single candidate against existing posts and returns a CheckResult.
func Check(existing []PostEntry, input CheckInput, ttlDays int) CheckResult {
	result := CheckResult{
		Title:   input.Title,
		Verdict: "OK",
	}

	inputLabels := normalizeLabels(input.Labels)

	for _, post := range existing {
		// Hard duplicate: exact URL match
		if input.URL != "" && post.ExternalURL != "" &&
			strings.EqualFold(strings.TrimSpace(input.URL), strings.TrimSpace(post.ExternalURL)) {
			result.Matches = append(result.Matches, Match{
				Title:      post.Title,
				DaysAgo:    DaysAgo(post.CreatedAt),
				Similarity: 1.0,
				SameType:   input.PostType == post.PostType,
				Reason:     "same URL — hard duplicate, skip",
			})
			result.Verdict = "DUPLICATE"
			continue
		}

		// Hard duplicate: exact title match
		if strings.EqualFold(strings.TrimSpace(input.Title), strings.TrimSpace(post.Title)) {
			result.Matches = append(result.Matches, Match{
				Title:      post.Title,
				DaysAgo:    DaysAgo(post.CreatedAt),
				Similarity: 1.0,
				SameType:   input.PostType == post.PostType,
				Reason:     "same title — hard duplicate, skip",
			})
			result.Verdict = "DUPLICATE"
			continue
		}

		// Composite similarity
		postLabels := normalizeLabels(post.Labels)
		lScore, overlap := labelScore(inputLabels, postLabels)
		tScore := typeScore(input.PostType, post.PostType)
		gScore, distKm := geoScore(input.Lat, input.Lon, post.Latitude, post.Longitude)
		rScore := recencyScore(post.CreatedAt, ttlDays)

		// Adjust weights based on geo availability
		var wLabel, wGeo float64
		if input.Lat != nil && post.Latitude != nil {
			wLabel = 0.50
			wGeo = 0.15
		} else {
			wLabel = 0.65
			wGeo = 0.00
		}

		sim := lScore*wLabel + tScore*wType + gScore*wGeo + rScore*wRecency
		sim = math.Round(sim*100) / 100

		if sim < 0.25 {
			continue
		}

		m := Match{
			Title:         post.Title,
			DaysAgo:       DaysAgo(post.CreatedAt),
			Similarity:    sim,
			OverlapLabels: overlap,
			SameType:      input.PostType == post.PostType,
			Reason:        generateReason(lScore, gScore, tScore, distKm),
		}
		if distKm != nil {
			m.DistanceKm = distKm
		}
		result.Matches = append(result.Matches, m)
	}

	// Sort by similarity descending, cap at 5
	sort.Slice(result.Matches, func(i, j int) bool {
		return result.Matches[i].Similarity > result.Matches[j].Similarity
	})
	if len(result.Matches) > 5 {
		result.Matches = result.Matches[:5]
	}

	// Set verdict from max score (hard dups already set DUPLICATE above)
	if result.Verdict != "DUPLICATE" && len(result.Matches) > 0 && result.Matches[0].Similarity >= 0.4 {
		result.Verdict = "SIMILAR"
	}

	return result
}

// CheckBatch scores multiple candidates and returns results for each.
func CheckBatch(existing []PostEntry, inputs []CheckInput, ttlDays int) []CheckResult {
	results := make([]CheckResult, len(inputs))
	for i, input := range inputs {
		results[i] = Check(existing, input, ttlDays)
	}
	return results
}

func labelScore(a, b map[string]float64) (float64, []string) {
	if len(a) == 0 && len(b) == 0 {
		return 0, nil
	}

	// Union of all labels
	all := make(map[string]bool)
	for k := range a {
		all[k] = true
	}
	for k := range b {
		all[k] = true
	}

	var interSum, unionSum float64
	var overlap []string
	for label := range all {
		w := labelWeight(label)
		unionSum += w
		if a[label] > 0 && b[label] > 0 {
			interSum += w
			overlap = append(overlap, label)
		}
	}

	if unionSum == 0 {
		return 0, nil
	}
	return interSum / unionSum, overlap
}

func labelWeight(label string) float64 {
	if genericLabels[label] {
		return 0.5
	}
	return 1.0
}

func typeScore(a, b string) float64 {
	if strings.EqualFold(a, b) {
		return 1.0
	}
	return 0.0
}

func geoScore(lat1, lon1, lat2, lon2 *float64) (float64, *float64) {
	if lat1 == nil || lon1 == nil || lat2 == nil || lon2 == nil {
		return 0, nil
	}
	d := geo.HaversineKm(*lat1, *lon1, *lat2, *lon2)
	d = math.Round(d*10) / 10

	var score float64
	switch {
	case d < 0.5:
		score = 1.0
	case d < 2:
		score = 0.7
	case d < 5:
		score = 0.4
	case d < 10:
		score = 0.2
	default:
		score = 0.0
	}
	return score, &d
}

func recencyScore(createdAt time.Time, ttlDays int) float64 {
	days := time.Since(createdAt).Hours() / 24
	score := 1.0 - days/float64(ttlDays)
	if score < 0 {
		return 0
	}
	return score
}

func generateReason(lScore, gScore, tScore float64, distKm *float64) string {
	var parts []string

	if lScore > 0.5 {
		parts = append(parts, "same topic")
	} else {
		parts = append(parts, "different topic")
	}

	if distKm != nil && *distKm < 2 {
		parts = append(parts, "same area")
	} else if distKm != nil && *distKm < 5 {
		parts = append(parts, "nearby")
	}

	if tScore > 0 {
		parts = append(parts, "same type")
	}

	base := strings.Join(parts, "+")

	// Add suggestion
	if lScore > 0.5 && tScore > 0 {
		if distKm != nil && *distKm < 2 {
			return fmt.Sprintf("%s — consider a different venue or angle", base)
		}
		return fmt.Sprintf("%s — OK if angle is fresh", base)
	}
	if lScore > 0.5 {
		return fmt.Sprintf("%s — same topic, try a different type or framing", base)
	}
	return fmt.Sprintf("%s — no conflict", base)
}

func normalizeLabels(labels []string) map[string]float64 {
	m := make(map[string]float64, len(labels))
	for _, l := range labels {
		l = strings.ToLower(strings.TrimSpace(l))
		if l != "" {
			m[l] = 1.0
		}
	}
	return m
}

func DaysAgo(t time.Time) int {
	d := int(time.Since(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}
