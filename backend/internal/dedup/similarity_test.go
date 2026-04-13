package dedup

import (
	"testing"
	"time"
)

func ptr(f float64) *float64 { return &f }

func makePost(title string, labels []string, postType string, daysOld int, opts ...func(*PostEntry)) PostEntry {
	p := PostEntry{
		ID:        int64(daysOld + 1),
		Title:     title,
		PostType:  postType,
		Labels:    labels,
		CreatedAt: time.Now().AddDate(0, 0, -daysOld),
	}
	for _, o := range opts {
		o(&p)
	}
	return p
}

func withGeo(lat, lon float64) func(*PostEntry) {
	return func(p *PostEntry) {
		p.Latitude = &lat
		p.Longitude = &lon
	}
}

func withURL(url string) func(*PostEntry) {
	return func(p *PostEntry) { p.ExternalURL = url }
}

func TestExactURLDuplicate(t *testing.T) {
	existing := []PostEntry{
		makePost("Discovery Coffee", []string{"coffee", "cafe"}, "place", 3, withURL("https://discoverycoffee.com")),
	}
	input := CheckInput{
		Title:    "Best cortado in town",
		Labels:   []string{"coffee"},
		PostType: "place",
		URL:      "https://discoverycoffee.com",
	}
	result := Check(existing, input, 14)
	if result.Verdict != "DUPLICATE" {
		t.Errorf("expected DUPLICATE, got %s", result.Verdict)
	}
	if len(result.Matches) == 0 || result.Matches[0].Similarity != 1.0 {
		t.Error("expected exact match with similarity 1.0")
	}
}

func TestExactTitleDuplicate(t *testing.T) {
	existing := []PostEntry{
		makePost("Discovery Coffee cortado", []string{"coffee"}, "place", 2),
	}
	input := CheckInput{
		Title:    "Discovery Coffee cortado",
		Labels:   []string{"coffee", "latte"},
		PostType: "place",
	}
	result := Check(existing, input, 14)
	if result.Verdict != "DUPLICATE" {
		t.Errorf("expected DUPLICATE, got %s", result.Verdict)
	}
}

func TestHighLabelOverlapSameTypeSameArea(t *testing.T) {
	existing := []PostEntry{
		makePost("Discovery Coffee", []string{"coffee", "cafe", "victoria-bc"}, "place", 3,
			withGeo(48.4284, -123.3656)),
	}
	input := CheckInput{
		Title:    "Best espresso spot",
		Labels:   []string{"coffee", "cafe", "victoria-bc"},
		PostType: "place",
		Lat:      ptr(48.4284),
		Lon:      ptr(-123.3650),
	}
	result := Check(existing, input, 14)
	if result.Verdict != "SIMILAR" {
		t.Errorf("expected SIMILAR, got %s", result.Verdict)
	}
	if len(result.Matches) == 0 {
		t.Fatal("expected at least one match")
	}
	if result.Matches[0].Similarity < 0.5 {
		t.Errorf("expected high similarity, got %.2f", result.Matches[0].Similarity)
	}
}

func TestOldPostLowSimilarity(t *testing.T) {
	existing := []PostEntry{
		makePost("Discovery Coffee", []string{"coffee", "cafe", "victoria-bc"}, "place", 13),
	}
	input := CheckInput{
		Title:    "Best espresso spot",
		Labels:   []string{"coffee", "cafe", "victoria-bc"},
		PostType: "place",
	}
	result := Check(existing, input, 14)
	// Should still match but with lower recency contribution.
	// Labels (0.65) + type (0.20) dominate; recency (0.15 * ~0.07) is near zero.
	// A fresh post with same labels would score higher.
	if result.Verdict != "SIMILAR" {
		t.Errorf("expected SIMILAR for old post with high label overlap, got %s", result.Verdict)
	}
	// Compare against a fresh version to verify recency decay matters
	freshPost := makePost("Discovery Coffee", []string{"coffee", "cafe", "victoria-bc"}, "place", 1)
	freshResult := Check([]PostEntry{freshPost}, input, 14)
	if len(result.Matches) > 0 && len(freshResult.Matches) > 0 {
		if result.Matches[0].Similarity >= freshResult.Matches[0].Similarity {
			t.Errorf("old post (%.2f) should score lower than fresh post (%.2f)",
				result.Matches[0].Similarity, freshResult.Matches[0].Similarity)
		}
	}
}

func TestNoLabelOverlap(t *testing.T) {
	existing := []PostEntry{
		makePost("Discovery Coffee", []string{"coffee", "cafe"}, "place", 2),
	}
	input := CheckInput{
		Title:    "AI chip race",
		Labels:   []string{"ai", "semiconductors"},
		PostType: "article",
	}
	result := Check(existing, input, 14)
	if result.Verdict != "OK" {
		t.Errorf("expected OK, got %s", result.Verdict)
	}
}

func TestGenericOnlyOverlapLowScore(t *testing.T) {
	existing := []PostEntry{
		makePost("Some food post", []string{"place", "food"}, "place", 2),
	}
	input := CheckInput{
		Title:    "Different food thing",
		Labels:   []string{"place", "food"},
		PostType: "place",
	}
	result := Check(existing, input, 14)
	// Generic-only labels (weight 0.5 each) should produce lower scores
	// than specific labels would
	if result.Verdict == "DUPLICATE" {
		t.Error("generic-only overlap should not produce DUPLICATE")
	}
}

func TestSpecificOverlapHigherThanGeneric(t *testing.T) {
	existing := []PostEntry{
		makePost("Coffee spot A", []string{"coffee", "victoria-bc", "place"}, "place", 2),
	}

	// Specific labels overlap
	specific := CheckInput{
		Title:    "Coffee spot B",
		Labels:   []string{"coffee", "victoria-bc", "place"},
		PostType: "place",
	}
	rSpecific := Check(existing, specific, 14)

	// Generic-only overlap
	generic := CheckInput{
		Title:    "Random place",
		Labels:   []string{"place", "food", "drink"},
		PostType: "place",
	}
	rGeneric := Check(existing, generic, 14)

	var specScore, genScore float64
	if len(rSpecific.Matches) > 0 {
		specScore = rSpecific.Matches[0].Similarity
	}
	if len(rGeneric.Matches) > 0 {
		genScore = rGeneric.Matches[0].Similarity
	}

	if specScore <= genScore {
		t.Errorf("specific overlap (%.2f) should score higher than generic (%.2f)", specScore, genScore)
	}
}

func TestBatchCheck(t *testing.T) {
	existing := []PostEntry{
		makePost("Discovery Coffee", []string{"coffee"}, "place", 2),
	}
	inputs := []CheckInput{
		{Title: "Discovery Coffee", Labels: []string{"coffee"}, PostType: "place"},
		{Title: "AI news", Labels: []string{"ai", "tech"}, PostType: "article"},
	}
	results := CheckBatch(existing, inputs, 14)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Verdict != "DUPLICATE" {
		t.Errorf("first result: expected DUPLICATE, got %s", results[0].Verdict)
	}
	if results[1].Verdict != "OK" {
		t.Errorf("second result: expected OK, got %s", results[1].Verdict)
	}
}
