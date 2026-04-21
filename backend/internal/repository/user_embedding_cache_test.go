package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func randomVec128() []float32 {
	v := make([]float32, 128)
	for i := range v {
		v[i] = float32(i%10) / 10.0
	}
	return v
}

func TestEmbeddingCache_SecondLookupDoesNotHitDB(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-cache-1")

	cache := repository.NewEmbeddingCache(db, 1000, 5*time.Minute)

	if err := repository.NewUserEmbeddingRepo(db).Upsert(context.Background(), u.ID, randomVec128(), 1); err != nil {
		t.Fatal(err)
	}

	_, _ = cache.Get(context.Background(), u.ID)
	_, _ = cache.Get(context.Background(), u.ID)

	if cache.DBHits() != 1 {
		t.Errorf("expected 1 DB hit across 2 lookups, got %d", cache.DBHits())
	}
}

func TestEmbeddingCache_ExpiredEntryRefetchesFromDB(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-cache-ttl")

	cache := repository.NewEmbeddingCache(db, 1000, 2*time.Millisecond)

	if err := repository.NewUserEmbeddingRepo(db).Upsert(context.Background(), u.ID, randomVec128(), 1); err != nil {
		t.Fatal(err)
	}

	_, _ = cache.Get(context.Background(), u.ID)
	time.Sleep(15 * time.Millisecond)
	_, _ = cache.Get(context.Background(), u.ID)

	if cache.DBHits() != 2 {
		t.Errorf("expected 2 DB hits after TTL expiry, got %d", cache.DBHits())
	}
}

func TestEmbeddingCache_EvictsLeastRecentlyUsedWhenFull(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	u1, _ := userRepo.FindOrCreateByFirebaseUID("firebase-lru-1")
	u2, _ := userRepo.FindOrCreateByFirebaseUID("firebase-lru-2")
	u3, _ := userRepo.FindOrCreateByFirebaseUID("firebase-lru-3")

	repo := repository.NewUserEmbeddingRepo(db)
	vec64 := make([]float32, 64)
	for i := range vec64 {
		vec64[i] = 0.1
	}
	if err := repo.Upsert(context.Background(), u1.ID, vec64, 1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(context.Background(), u2.ID, vec64, 1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(context.Background(), u3.ID, vec64, 1); err != nil {
		t.Fatal(err)
	}

	cache := repository.NewEmbeddingCache(db, 2, 5*time.Minute)

	_, _ = cache.Get(context.Background(), u1.ID)
	_, _ = cache.Get(context.Background(), u2.ID)
	_, _ = cache.Get(context.Background(), u1.ID)
	_, _ = cache.Get(context.Background(), u3.ID)

	if cache.DBHits() != 3 {
		t.Fatalf("expected 3 loads (u1,u2,u3), got %d", cache.DBHits())
	}

	_, _ = cache.Get(context.Background(), u2.ID)
	if cache.DBHits() != 4 {
		t.Fatalf("u2 was evicted; expected 4th DB hit, got %d", cache.DBHits())
	}
}
