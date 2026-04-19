package handler_test

// Test E: the /user/digest endpoint must return 200 with a valid JSON array.
// Verifies the push-token handler is wired up correctly and the digest query
// does not crash on a user with no posts.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestDigestScheduling_IsWiredUp(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)

	_, _ = userRepo.FindOrCreateByFirebaseUID("firebase-digest-test")

	h := handler.NewPushTokenHandler(userRepo, pushTokenRepo)

	req := httptest.NewRequest("GET", "/user/digest", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-digest-test"))
	rec := httptest.NewRecorder()

	h.GetDigestPosts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from digest endpoint, got %d: %s", rec.Code, rec.Body.String())
	}

	// Response must decode as a JSON array (empty or not).
	var posts []any
	if err := json.NewDecoder(rec.Body).Decode(&posts); err != nil {
		t.Errorf("digest response is not a JSON array: %v", err)
	}
}
