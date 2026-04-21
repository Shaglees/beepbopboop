package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestForYouFeed_FallsBackToRuleBasedWhenRankerNil verifies ForYou returns 200 with posts
// when no two-tower checkpoint is configured (issue #44 TDD).
func TestForYouFeed_FallsBackToRuleBasedWhenRankerNil(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	postRepo := repository.NewPostRepo(db)
	// Explicitly no SetML — ranker nil, rule-only path.

	userSettingsRepo := repository.NewUserSettingsRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	eventRepo := repository.NewEventRepo(db)
	reactionRepo := repository.NewReactionRepo(db)
	followRepo := repository.NewFollowRepo(db)
	userEmbeddingRepo := repository.NewUserEmbeddingRepo(db)
	userEmbFront := repository.NewEmbeddingCacheFromLoader(userEmbeddingRepo, 100, 5*time.Minute)

	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-foryou-tdd")
	if err != nil {
		t.Fatal(err)
	}
	agentRepo := repository.NewAgentRepo(db)
	agent, err := agentRepo.Create(user.ID, "Agent")
	if err != nil {
		t.Fatal(err)
	}

	lat := 37.77
	lon := -122.42
	pLat := 37.78
	pLon := -122.43
	_, err = userSettingsRepo.Upsert(user.ID, "SF", &lat, &lon, 50.0, nil, true, 9, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID,
		UserID:  user.ID,
		Title:   "Nearby drop",
		Body:    "Something to read",
		Latitude: &pLat,
		Longitude: &pLon,
	})
	if err != nil {
		t.Fatal(err)
	}

	h := handler.NewMultiFeedHandler(userRepo, postRepo, userSettingsRepo, weightsRepo, eventRepo, reactionRepo, followRepo, userEmbFront)

	req := httptest.NewRequest("GET", "/feeds/foryou", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-foryou-tdd"))
	rec := httptest.NewRecorder()

	h.GetForYou(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp model.FeedResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Posts) < 1 {
		t.Fatalf("expected at least one post in ForYou response, got %d", len(resp.Posts))
	}
}
