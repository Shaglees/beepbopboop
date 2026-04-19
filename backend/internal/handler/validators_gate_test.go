package handler_test

// Tests A + B: ensure every structured hint has a validator and that valid payloads pass.
//
// structuredHintValidPayloads is the canonical source of truth for which hints
// require structured JSON in external_url. Entries here drive two tests:
//
//  TestAllStructuredHints_HaveValidators  — submits "not-valid-json" and asserts
//    lint rejects it.  If a validator is missing the post would silently pass.
//
//  TestAllStructuredHints_AcceptValidPayload — submits the valid payload and
//    asserts lint accepts it.  Catches over-strict validators.
//
// To add a new structured hint:
//  1. Add the hint to ValidDisplayHints in post.go
//  2. Add it to the structuredHint list in lintPostRequest (post.go)
//  3. Add a validate*Data function and wire it in the switch
//  4. Add a valid payload here — the tests will then auto-exercise it

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

var structuredHintValidPayloads = map[string]string{
	"weather":          `{"current":{"temp_c":20,"feels_like_c":18,"humidity":60,"wind_speed_kmh":10,"uv_index":5,"is_day":true,"condition":"Sunny","condition_code":1000},"hourly":[],"daily":[],"location":{"latitude":53.3,"longitude":-6.2,"timezone":"Europe/Dublin"}}`,
	"scoreboard":       `{"status":"Final","home":{"name":"Lakers","abbr":"LAL"},"away":{"name":"Celtics","abbr":"BOS"},"sport":"NBA"}`,
	"matchup":          `{"status":"Scheduled","home":{"name":"Lakers","abbr":"LAL"},"away":{"name":"Celtics","abbr":"BOS"},"sport":"NBA","gameTime":"2026-04-16T19:00:00Z"}`,
	"standings":        `{"league":"NBA","date":"2026-04-16","games":[{"home":"LAL","away":"BOS","homeScore":110,"awayScore":105,"status":"Final"}]}`,
	"entertainment":    `{"subject":"Zendaya","headline":"Zendaya Named TIME Entertainer of the Year","source":"People","category":"award","tags":["entertainment"]}`,
	"album":            `{"type":"album","artist":"Taylor Swift","title":"The Tortured Poets Department"}`,
	"concert":          `{"type":"concert","artist":"Coldplay"}`,
	"game_release":     `{"title":"Test Game","status":"upcoming"}`,
	"game_review":      `{"title":"Test Game","status":"released"}`,
	"restaurant":       `{"name":"Test Cafe","latitude":40.7,"longitude":-74.0}`,
	"fitness":          `{"activity":"Running","duration_min":30}`,
	"feedback":         `{"feedback_type":"poll","question":"What do you think?","options":[{"key":"a","label":"Option A"}]}`,
	"movie":            `{"tmdbId":550,"title":"Fight Club"}`,
	"show":             `{"tmdbId":1399,"title":"Game of Thrones"}`,
	"player_spotlight": `{"playerName":"LeBron James","sport":"NBA","team":"Lakers"}`,
	"box_score":        `{"status":"Final","home":{"name":"Lakers","abbr":"LAL"},"away":{"name":"Celtics","abbr":"BOS"},"sport":"NBA"}`,
	"pet_spotlight":    `{"type":"adoption","name":"Biscuit","species":"dog","breed":"Labrador Mix","age":"Young","gender":"Male","shelterName":"SF SPCA","shelterCity":"San Francisco","petfinderUrl":"https://www.petfinder.com/dog/biscuit-12345678"}`,
	"destination":      `{"city":"Paris","country":"France","latitude":48.8566,"longitude":2.3522}`,
	"science":          `{"category":"Space","source":"NASA","headline":"New Planet Discovered"}`,
}

// TestAllStructuredHints_HaveValidators is a hard gate: every hint in
// structuredHintValidPayloads must have a validator that rejects invalid JSON.
// If a new hint is added to the map but its validate*Data function is missing
// or not wired in the switch, this test will fail.
func TestAllStructuredHints_HaveValidators(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	for hint := range structuredHintValidPayloads {
		hint := hint
		t.Run(hint, func(t *testing.T) {
			body := `{"title":"t","body":"b","display_hint":"` + hint + `","external_url":"not-valid-json","labels":["test"]}`
			_, resp := lintCall(t, h, body)
			if resp["valid"] == true {
				t.Errorf("hint %q: lint accepted invalid JSON in external_url — missing validator or not wired in switch?", hint)
			}
		})
	}
}

// TestAllStructuredHints_AcceptValidPayload verifies each hint's canonical payload
// passes lint, catching regressions in over-strict validators.
func TestAllStructuredHints_AcceptValidPayload(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	for hint, payload := range structuredHintValidPayloads {
		hint, payload := hint, payload
		t.Run(hint, func(t *testing.T) {
			body := `{"title":"t","body":"b","display_hint":"` + hint + `","external_url":` + jsonString(payload) + `,"labels":["test"]}`
			_, resp := lintCall(t, h, body)
			if resp["valid"] != true {
				t.Errorf("hint %q: valid payload rejected: %v", hint, lintErrors(resp))
			}
		})
	}
}
