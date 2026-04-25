package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestUpsertSportsPost_InvalidHintRejected verifies that UpsertSportsPost returns an
// error when given a display_hint that is not in the valid sports hints set.
// This guards against the sports worker bypassing the ValidDisplayHints whitelist.
func TestUpsertSportsPost_InvalidHintRejected(t *testing.T) {
	db := database.OpenTestDB(t)
	postRepo := repository.NewPostRepo(db)

	err := postRepo.UpsertSportsPost(
		"test-game-1", "LAL 110 · BOS 105", "Final · NBA",
		"NBA", `{"sport":"basketball","league":"NBA","status":"Final","home":{"name":"Celtics","abbr":"BOS"},"away":{"name":"Lakers","abbr":"LAL"}}`,
		"not-a-valid-hint",
	)
	if err == nil {
		t.Error("expected error for invalid display hint, got nil")
	}
}

// TestUpsertSportsPost_ValidHintsAccepted verifies that the two valid sports hints
// ("matchup" and "scoreboard") are both accepted without error.
func TestUpsertSportsPost_ValidHintsAccepted(t *testing.T) {
	db := database.OpenTestDB(t)
	postRepo := repository.NewPostRepo(db)

	gameJSON := `{"sport":"basketball","league":"NBA","status":"7:30 PM ET","home":{"name":"Celtics","abbr":"BOS"},"away":{"name":"Lakers","abbr":"LAL"}}`

	for _, hint := range []string{"matchup", "scoreboard"} {
		err := postRepo.UpsertSportsPost(
			"test-game-valid-"+hint, "LAL @ BOS", "7:30 PM ET",
			"NBA", gameJSON, hint,
		)
		if err != nil {
			t.Errorf("UpsertSportsPost with hint %q returned unexpected error: %v", hint, err)
		}
	}
}
