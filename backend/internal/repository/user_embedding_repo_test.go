package repository_test

// Tests for UserEmbeddingRepo (Upsert, Get, GetAll round-trip).
// Computation tests live in backend/internal/embedding/user_embedder_test.go.

import (
	"context"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUpsertAndGetUserEmbedding_RoundTrip(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewUserEmbeddingRepo(db)
	userRepo := repository.NewUserRepo(db)

	u, err := userRepo.FindOrCreateByFirebaseUID("firebase-emb-roundtrip")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 4-dim test vector with mixed signs.
	vec := []float32{0.5, 0.25, -0.1, 0.8}
	if err := repo.Upsert(context.Background(), u.ID, vec, 3); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil UserEmbedding")
	}
	if got.PostCount != 3 {
		t.Errorf("expected post_count=3, got %d", got.PostCount)
	}
	if len(got.Embedding) != len(vec) {
		t.Fatalf("expected %d dims, got %d", len(vec), len(got.Embedding))
	}
	for i, v := range vec {
		if abs32(got.Embedding[i]-v) > 1e-5 {
			t.Errorf("dim %d: expected %f, got %f", i, v, got.Embedding[i])
		}
	}
}

func TestUpsertUserEmbedding_Idempotent(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewUserEmbeddingRepo(db)
	userRepo := repository.NewUserRepo(db)

	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-emb-idem")

	first := []float32{1.0, 0.0}
	second := []float32{0.0, 1.0}

	repo.Upsert(context.Background(), u.ID, first, 1)
	repo.Upsert(context.Background(), u.ID, second, 2)

	// Exactly one row must exist; second write must overwrite first.
	var count int
	db.QueryRow("SELECT COUNT(*) FROM user_embeddings WHERE user_id = $1", u.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row after two upserts, got %d", count)
	}

	got, _ := repo.Get(context.Background(), u.ID)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if abs32(got.Embedding[0]-0.0) > 1e-5 || abs32(got.Embedding[1]-1.0) > 1e-5 {
		t.Errorf("expected second vector [0,1], got %v", got.Embedding)
	}
	if got.PostCount != 2 {
		t.Errorf("expected post_count=2 after second upsert, got %d", got.PostCount)
	}
}

func TestUserEmbeddingRepo_Get_ReturnsNilForMissingUser(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewUserEmbeddingRepo(db)
	userRepo := repository.NewUserRepo(db)

	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-emb-norow")

	got, err := repo.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("Get on user with no embedding: %v", err)
	}
	if got != nil {
		t.Error("expected nil for user with no stored embedding")
	}
}

func TestUserEmbeddingRepo_GetAll_ReturnsMap(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewUserEmbeddingRepo(db)
	userRepo := repository.NewUserRepo(db)

	u1, _ := userRepo.FindOrCreateByFirebaseUID("firebase-getall-1")
	u2, _ := userRepo.FindOrCreateByFirebaseUID("firebase-getall-2")
	u3, _ := userRepo.FindOrCreateByFirebaseUID("firebase-getall-3")

	repo.Upsert(context.Background(), u1.ID, []float32{1.0, 0.0}, 1)
	repo.Upsert(context.Background(), u2.ID, []float32{0.0, 1.0}, 2)
	repo.Upsert(context.Background(), u3.ID, []float32{0.707, 0.707}, 3)

	all, err := repo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	for _, uid := range []string{u1.ID, u2.ID, u3.ID} {
		v, ok := all[uid]
		if !ok {
			t.Errorf("user %s missing from GetAll result", uid)
			continue
		}
		if len(v) != 2 {
			t.Errorf("expected 2-dim embedding for user %s, got %d dims", uid, len(v))
		}
	}
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
