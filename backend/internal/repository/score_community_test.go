package repository_test

import (
	"math"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func postAt(hoursAgo float64) model.Post {
	lat, lon := 53.35, -6.26
	return model.Post{
		CreatedAt: time.Now().Add(-time.Duration(hoursAgo * float64(time.Hour))),
		Latitude:  &lat,
		Longitude: &lon,
	}
}

func TestScoreCommunityPost_RecencyDecay(t *testing.T) {
	userLat, userLon, radius := 53.35, -6.26, 10.0

	score1h := repository.ScoreCommunityPost(postAt(1), userLat, userLon, radius)
	score4h := repository.ScoreCommunityPost(postAt(4), userLat, userLon, radius)
	score12h := repository.ScoreCommunityPost(postAt(12), userLat, userLon, radius)

	if score1h <= score4h {
		t.Errorf("1h post (%.3f) should score higher than 4h post (%.3f)", score1h, score4h)
	}
	if score4h <= score12h {
		t.Errorf("4h post (%.3f) should score higher than 12h post (%.3f)", score4h, score12h)
	}

	// 4h half-life: exp(-ln(2)/4 * 4) ≈ 0.5; allow generous range
	expectedAt4h := math.Exp(-0.173 * 4)
	if score4h < expectedAt4h*0.4 || score4h > expectedAt4h*3.0 {
		t.Errorf("4h score %.3f out of expected range around recency component %.3f", score4h, expectedAt4h)
	}
}

func TestScoreCommunityPost_GeoProximity(t *testing.T) {
	userLat, userLon, radius := 53.35, -6.26, 10.0

	// Post exactly at user location
	centerLat, centerLon := 53.35, -6.26
	centerPost := model.Post{
		CreatedAt: time.Now().Add(-1 * time.Hour),
		Latitude:  &centerLat,
		Longitude: &centerLon,
	}

	// Post near the edge (~8.9km away)
	edgeLat, edgeLon := 53.43, -6.26
	edgePost := model.Post{
		CreatedAt: time.Now().Add(-1 * time.Hour),
		Latitude:  &edgeLat,
		Longitude: &edgeLon,
	}

	centerScore := repository.ScoreCommunityPost(centerPost, userLat, userLon, radius)
	edgeScore := repository.ScoreCommunityPost(edgePost, userLat, userLon, radius)

	if centerScore <= edgeScore {
		t.Errorf("center post (%.3f) should score higher than edge post (%.3f)", centerScore, edgeScore)
	}
}

func TestScoreCommunityPost_EngagementBoosts(t *testing.T) {
	userLat, userLon, radius := 53.35, -6.26, 10.0
	lat, lon := 53.35, -6.26
	age := time.Now().Add(-2 * time.Hour)

	noEngagement := model.Post{CreatedAt: age, Latitude: &lat, Longitude: &lon}
	withEngagement := model.Post{
		CreatedAt: age, Latitude: &lat, Longitude: &lon,
		ReactionCount: 5, SaveCount: 3,
	}

	s1 := repository.ScoreCommunityPost(noEngagement, userLat, userLon, radius)
	s2 := repository.ScoreCommunityPost(withEngagement, userLat, userLon, radius)

	if s2 <= s1 {
		t.Errorf("post with engagement (%.3f) should score higher than no engagement (%.3f)", s2, s1)
	}
}

func TestScoreCommunityPost_RecencyDominatesEngagement(t *testing.T) {
	// A fresh nearby post with no reactions should beat a day-old post with many reactions.
	userLat, userLon, radius := 53.35, -6.26, 10.0
	lat, lon := 53.35, -6.26

	freshPost := model.Post{
		CreatedAt:     time.Now().Add(-30 * time.Minute),
		Latitude:      &lat,
		Longitude:     &lon,
		ReactionCount: 0,
		SaveCount:     0,
	}
	viralOldPost := model.Post{
		CreatedAt:     time.Now().Add(-24 * time.Hour),
		Latitude:      &lat,
		Longitude:     &lon,
		ReactionCount: 50,
		SaveCount:     20,
	}

	freshScore := repository.ScoreCommunityPost(freshPost, userLat, userLon, radius)
	viralScore := repository.ScoreCommunityPost(viralOldPost, userLat, userLon, radius)

	if freshScore <= viralScore {
		t.Errorf("fresh post (%.3f) should beat 24h viral post (%.3f)", freshScore, viralScore)
	}
}

func TestScoreCommunityPost_EventTimingBoost(t *testing.T) {
	userLat, userLon, radius := 53.35, -6.26, 10.0
	lat, lon := 53.35, -6.26

	// Event starting in 1 hour
	gameTimeStr := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	externalURL := `{"sport":"soccer","status":"pre","gameTime":"` + gameTimeStr + `"}`

	withEvent := model.Post{
		CreatedAt:   time.Now().Add(-30 * time.Minute),
		Latitude:    &lat,
		Longitude:   &lon,
		ExternalURL: externalURL,
	}
	withoutEvent := model.Post{
		CreatedAt: time.Now().Add(-30 * time.Minute),
		Latitude:  &lat,
		Longitude: &lon,
	}

	withScore := repository.ScoreCommunityPost(withEvent, userLat, userLon, radius)
	withoutScore := repository.ScoreCommunityPost(withoutEvent, userLat, userLon, radius)

	if withScore <= withoutScore {
		t.Errorf("upcoming event post (%.3f) should score higher than identical post without event (%.3f)", withScore, withoutScore)
	}
}

func TestScoreCommunityPost_NoLatLonHandled(t *testing.T) {
	// Posts without geo coords should not panic — geo score is 0.
	p := model.Post{CreatedAt: time.Now().Add(-1 * time.Hour)}
	score := repository.ScoreCommunityPost(p, 53.35, -6.26, 10.0)
	if score < 0 {
		t.Errorf("score should be non-negative, got %.3f", score)
	}
}
