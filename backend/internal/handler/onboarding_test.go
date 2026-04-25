package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestOnboardingHandler_StoresEmbeddingForKnownInterests verifies that posting
// known interests stores a non-zero embedding for the user.
func TestOnboardingHandler_StoresEmbeddingForKnownInterests(t *testing.T) {
	db := database.OpenTestDB(t)
	postEmbRepo := repository.NewPostEmbeddingRepo(db)

	// Seed a labeled post so Compute has something to work with.
	if _, err := db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, labels, status, display_hint)
		VALUES ('onb-post-1', 'sports-bot', 'system', 'T', 'B', '["sports"]', 'published', 'card')
		ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	if err := postEmbRepo.Upsert("onb-post-1", []float32{1, 0, 0, 0}, "test"); err != nil {
		t.Fatalf("seed embedding: %v", err)
	}

	store := embedding.NewPrototypeStore(db)
	if err := store.Compute(context.Background()); err != nil {
		t.Fatalf("Compute: %v", err)
	}

	userRepo := repository.NewUserRepo(db)
	userEmbRepo := repository.NewUserEmbeddingRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	h := handler.NewOnboardingHandler(userRepo, store, userEmbRepo, interestRepo)

	body := `{"interests":["Sports"]}`
	req := httptest.NewRequest(http.MethodPost, "/user/interests", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-onb-1"))
	rec := httptest.NewRecorder()
	h.SubmitInterests(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-onb-1")
	stored, err := userEmbRepo.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored == nil || embedding.IsZero(stored.Embedding) {
		t.Error("expected non-zero embedding stored after onboarding")
	}
}

// TestOnboardingHandler_UnknownInterests_Returns200NoEmbedding verifies that
// posting entirely unknown interests returns 200 but stores no embedding.
func TestOnboardingHandler_UnknownInterests_Returns200NoEmbedding(t *testing.T) {
	db := database.OpenTestDB(t)
	store := embedding.NewPrototypeStore(db)
	// No Compute call — store has no prototypes.

	userRepo := repository.NewUserRepo(db)
	userEmbRepo := repository.NewUserEmbeddingRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	h := handler.NewOnboardingHandler(userRepo, store, userEmbRepo, interestRepo)

	body := `{"interests":["Unicorns","Alchemy"]}`
	req := httptest.NewRequest(http.MethodPost, "/user/interests", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-onb-2"))
	rec := httptest.NewRecorder()
	h.SubmitInterests(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-onb-2")
	stored, _ := userEmbRepo.Get(context.Background(), u.ID)
	if stored != nil {
		t.Error("expected no embedding stored for unknown interests")
	}
}

// TestOnboardingHandler_InvalidJSON_Returns400 verifies that malformed request
// bodies are rejected with 400.
func TestOnboardingHandler_InvalidJSON_Returns400(t *testing.T) {
	db := database.OpenTestDB(t)
	store := embedding.NewPrototypeStore(db)
	userRepo := repository.NewUserRepo(db)
	userEmbRepo := repository.NewUserEmbeddingRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	h := handler.NewOnboardingHandler(userRepo, store, userEmbRepo, interestRepo)

	req := httptest.NewRequest(http.MethodPost, "/user/interests", bytes.NewBufferString("{bad json"))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-onb-3"))
	rec := httptest.NewRecorder()
	h.SubmitInterests(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
