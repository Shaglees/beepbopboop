package embedding_test

// TDD tests for cold-start strategy: PrototypeStore and ColdStartUpdater.
// Uses 4-dimensional embeddings so expected values can be derived exactly.
// Seeding helpers insert directly via SQL using the system user/agents that
// database.Open creates on every fresh test database.

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sync/atomic"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// seqID generates collision-free IDs within a process.
var seqID int64

func nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, atomic.AddInt64(&seqID, 1))
}

// --- seeding helpers ---

// seedLabeledPost inserts a published post with one label and a 4-dim embedding.
// Uses the system agent so no user/agent creation is required.
func seedLabeledPost(t *testing.T, db *sql.DB, postEmbRepo *repository.PostEmbeddingRepo, id, label string, vec [4]float32) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, labels, status, display_hint)
		VALUES ($1, 'sports-bot', 'system', $2, 'body', $3, 'published', 'card')
		ON CONFLICT DO NOTHING`,
		id, "Post "+id, `["`+label+`"]`); err != nil {
		t.Fatalf("seedLabeledPost insert %s: %v", id, err)
	}
	if err := postEmbRepo.Upsert(id, vec[:], "test"); err != nil {
		t.Fatalf("seedLabeledPost upsert embedding %s: %v", id, err)
	}
}

// seedPopularPost inserts a recently-created published post with save_count > 0
// and a 4-dim embedding.
func seedPopularPost(t *testing.T, db *sql.DB, postEmbRepo *repository.PostEmbeddingRepo, id string, saveCount int, vec [4]float32) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, status, display_hint, save_count)
		VALUES ($1, 'sports-bot', 'system', $2, 'body', 'published', 'card', $3)
		ON CONFLICT DO NOTHING`,
		id, "Popular post "+id, saveCount); err != nil {
		t.Fatalf("seedPopularPost insert %s: %v", id, err)
	}
	if err := postEmbRepo.Upsert(id, vec[:], "test"); err != nil {
		t.Fatalf("seedPopularPost upsert embedding %s: %v", id, err)
	}
}

// seedOldPopularPost inserts a popular post with created_at set daysAgo days
// in the past (outside the 7-day fallback window).
func seedOldPopularPost(t *testing.T, db *sql.DB, postEmbRepo *repository.PostEmbeddingRepo, id string, daysAgo int, vec [4]float32) {
	t.Helper()
	q := fmt.Sprintf(`
		INSERT INTO posts (id, agent_id, user_id, title, body, status, display_hint, save_count, created_at)
		VALUES ($1, 'sports-bot', 'system', $2, 'body', 'published', 'card', 5,
		        NOW() - INTERVAL '%d days')
		ON CONFLICT DO NOTHING`, daysAgo)
	if _, err := db.Exec(q, id, "Old post "+id); err != nil {
		t.Fatalf("seedOldPopularPost insert %s: %v", id, err)
	}
	if err := postEmbRepo.Upsert(id, vec[:], "test"); err != nil {
		t.Fatalf("seedOldPopularPost upsert embedding %s: %v", id, err)
	}
}

// recordDwell creates a post with a 4-dim embedding and inserts a view event
// with the given dwell_ms for the user.
func recordDwell(t *testing.T, db *sql.DB, postEmbRepo *repository.PostEmbeddingRepo, userID string, dwellMs int) {
	t.Helper()
	id := nextID("dwell")
	if _, err := db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, status, display_hint)
		VALUES ($1, 'sports-bot', 'system', $2, 'body', 'published', 'card')
		ON CONFLICT DO NOTHING`,
		id, "Dwell post "+id); err != nil {
		t.Fatalf("recordDwell insert post: %v", err)
	}
	if err := postEmbRepo.Upsert(id, []float32{1, 0, 0, 0}, "test"); err != nil {
		t.Fatalf("recordDwell upsert embedding: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO post_events (post_id, user_id, event_type, dwell_ms)
		VALUES ($1, $2, 'view', $3)`,
		id, userID, dwellMs); err != nil {
		t.Fatalf("recordDwell insert event: %v", err)
	}
}

// embeddingUpdated reports whether a user_embeddings row exists for the user.
func embeddingUpdated(t *testing.T, db *sql.DB, userID string) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_embeddings WHERE user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("embeddingUpdated: %v", err)
	}
	return count > 0
}

// cosineSimilarity computes cosine similarity between two equal-length vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		magA += float64(a[i]) * float64(a[i])
		magB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(magA) * math.Sqrt(magB)
	if denom < 1e-10 {
		return 0
	}
	return dot / denom
}

// --- tests ---

// TestOnboarding_ProducesNonZeroEmbedding verifies that completing onboarding
// with 3 interest selections immediately produces a non-zero vector that can be
// stored via UserEmbeddingRepo.
func TestOnboarding_ProducesNonZeroEmbedding(t *testing.T) {
	db := database.OpenTestDB(t)
	postEmbRepo := repository.NewPostEmbeddingRepo(db)
	userRepo := repository.NewUserRepo(db)

	seedLabeledPost(t, db, postEmbRepo, nextID("p"), "sports", [4]float32{1, 0, 0, 0})
	seedLabeledPost(t, db, postEmbRepo, nextID("p"), "fashion", [4]float32{0, 1, 0, 0})
	seedLabeledPost(t, db, postEmbRepo, nextID("p"), "weather", [4]float32{0, 0, 1, 0})

	store := embedding.NewPrototypeStore(db)
	if err := store.Compute(context.Background()); err != nil {
		t.Fatalf("Compute: %v", err)
	}

	u, err := userRepo.FindOrCreateByFirebaseUID("firebase-coldstart-test")
	if err != nil {
		t.Fatalf("FindOrCreateByFirebaseUID: %v", err)
	}

	combined := store.CombineFor([]string{"Sports", "Fashion", "Weather"})
	if embedding.IsZero(combined) {
		t.Error("CombineFor returned zero vector for 3 interest selections")
	}

	userEmbRepo := repository.NewUserEmbeddingRepo(db)
	if err := userEmbRepo.Upsert(context.Background(), u.ID, combined, 0); err != nil {
		t.Fatalf("Upsert embedding: %v", err)
	}
	stored, err := userEmbRepo.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("Get embedding: %v", err)
	}
	if stored == nil || embedding.IsZero(stored.Embedding) {
		t.Errorf("expected non-zero stored embedding, got %v", stored)
	}
}

// TestPrototypeVectors_SportsFashionCosineSimilarityLow asserts that sports and
// fashion prototype vectors are meaningfully distinct (cosine similarity < 0.5).
func TestPrototypeVectors_SportsFashionCosineSimilarityLow(t *testing.T) {
	db := database.OpenTestDB(t)
	postEmbRepo := repository.NewPostEmbeddingRepo(db)

	for i := 0; i < 3; i++ {
		seedLabeledPost(t, db, postEmbRepo, nextID("sports"), "sports", [4]float32{1, 0, 0, 0})
		seedLabeledPost(t, db, postEmbRepo, nextID("fashion"), "fashion", [4]float32{0, 1, 0, 0})
	}

	store := embedding.NewPrototypeStore(db)
	if err := store.Compute(context.Background()); err != nil {
		t.Fatalf("Compute: %v", err)
	}

	sportsVec, ok1 := store.VectorFor("sports")
	fashionVec, ok2 := store.VectorFor("fashion")
	if !ok1 || !ok2 {
		t.Fatal("missing prototype for sports or fashion")
	}
	sim := cosineSimilarity(sportsVec, fashionVec)
	if sim >= 0.5 {
		t.Errorf("sports/fashion cosine similarity %.3f >= 0.5 — prototypes not distinct", sim)
	}
}

// TestEarlySignalUpdate_FiresAfterThreeHighDwellPosts verifies MaybeRefresh
// stores an embedding after exactly 3 high-dwell posts, and not before.
func TestEarlySignalUpdate_FiresAfterThreeHighDwellPosts(t *testing.T) {
	db := database.OpenTestDB(t)
	postEmbRepo := repository.NewPostEmbeddingRepo(db)
	userRepo := repository.NewUserRepo(db)

	u, err := userRepo.FindOrCreateByFirebaseUID("firebase-earlysignal")
	if err != nil {
		t.Fatalf("FindOrCreateByFirebaseUID: %v", err)
	}
	cs := embedding.NewColdStartUpdater(db)

	// Two events: threshold not reached.
	recordDwell(t, db, postEmbRepo, u.ID, 6000)
	recordDwell(t, db, postEmbRepo, u.ID, 8000)
	if embeddingUpdated(t, db, u.ID) {
		t.Error("embedding must not exist before MaybeRefresh is called")
	}

	// Third event + explicit refresh: embedding must be written.
	recordDwell(t, db, postEmbRepo, u.ID, 7000)
	if err := cs.MaybeRefresh(context.Background(), u.ID); err != nil {
		t.Fatalf("MaybeRefresh: %v", err)
	}
	if !embeddingUpdated(t, db, u.ID) {
		t.Error("expected embedding in user_embeddings after MaybeRefresh with 3 high-dwell posts")
	}
}

// TestEarlySignalUpdate_DoesNotFireBeforeThreshold verifies MaybeRefresh is a
// no-op when fewer than 3 high-dwell posts exist.
func TestEarlySignalUpdate_DoesNotFireBeforeThreshold(t *testing.T) {
	db := database.OpenTestDB(t)
	postEmbRepo := repository.NewPostEmbeddingRepo(db)
	userRepo := repository.NewUserRepo(db)

	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-earlysignal-noop")
	cs := embedding.NewColdStartUpdater(db)

	recordDwell(t, db, postEmbRepo, u.ID, 6000)
	recordDwell(t, db, postEmbRepo, u.ID, 8000)

	if err := cs.MaybeRefresh(context.Background(), u.ID); err != nil {
		t.Fatalf("MaybeRefresh: %v", err)
	}
	if embeddingUpdated(t, db, u.ID) {
		t.Error("embedding must not be stored when < 3 high-dwell events have occurred")
	}
}

// TestPopularityFallback_ReturnsNonZeroVector asserts PopularityFallback returns
// a non-zero vector when recent popular posts exist.
func TestPopularityFallback_ReturnsNonZeroVector(t *testing.T) {
	db := database.OpenTestDB(t)
	postEmbRepo := repository.NewPostEmbeddingRepo(db)

	for i := 0; i < 10; i++ {
		seedPopularPost(t, db, postEmbRepo, nextID("pop"), 5, [4]float32{1, 0, 0, 0})
	}

	store := embedding.NewPrototypeStore(db)
	fallback, err := store.PopularityFallback(context.Background())
	if err != nil {
		t.Fatalf("PopularityFallback: %v", err)
	}
	if embedding.IsZero(fallback) {
		t.Error("PopularityFallback returned zero vector despite seeded popular posts")
	}
}

// TestPopularityFallback_IgnoresOldPosts asserts posts older than 7 days are
// excluded from the fallback computation.
func TestPopularityFallback_IgnoresOldPosts(t *testing.T) {
	db := database.OpenTestDB(t)
	postEmbRepo := repository.NewPostEmbeddingRepo(db)

	for i := 0; i < 5; i++ {
		seedOldPopularPost(t, db, postEmbRepo, nextID("old"), 10, [4]float32{1, 0, 0, 0})
	}

	store := embedding.NewPrototypeStore(db)
	fallback, err := store.PopularityFallback(context.Background())
	if err != nil {
		t.Fatalf("PopularityFallback: %v", err)
	}
	if !embedding.IsZero(fallback) {
		t.Errorf("expected zero fallback when all posts older than 7 days, got %v", fallback)
	}
}

// --- edge cases ---

// TestCombineFor_UnknownInterestIsSkipped verifies unknown interest names are
// silently skipped, returning a zero vector without panicking.
func TestCombineFor_UnknownInterestIsSkipped(t *testing.T) {
	db := database.OpenTestDB(t)
	store := embedding.NewPrototypeStore(db)
	vec := store.CombineFor([]string{"UnknownCategory", "AlsoUnknown"})
	if !embedding.IsZero(vec) {
		t.Errorf("expected zero vector for unknown interests, got %v", vec)
	}
}

// TestCombineFor_EmptyInterests verifies an empty interest list returns a zero vector.
func TestCombineFor_EmptyInterests(t *testing.T) {
	db := database.OpenTestDB(t)
	store := embedding.NewPrototypeStore(db)
	vec := store.CombineFor([]string{})
	if !embedding.IsZero(vec) {
		t.Errorf("expected zero vector for empty interests, got %v", vec)
	}
}

// TestPrototypeStore_NoPostsForLabel verifies VectorFor returns (nil, false) for
// a label with no posts.
func TestPrototypeStore_NoPostsForLabel(t *testing.T) {
	db := database.OpenTestDB(t)
	store := embedding.NewPrototypeStore(db)
	if err := store.Compute(context.Background()); err != nil {
		t.Fatalf("Compute on empty DB: %v", err)
	}
	vec, ok := store.VectorFor("nonexistent")
	if ok {
		t.Errorf("expected ok=false for missing label, got vec=%v", vec)
	}
	if vec != nil {
		t.Errorf("expected nil for missing label, got %v", vec)
	}
}

// TestIsZero_EmptySlice ensures IsZero treats an empty slice as zero.
func TestIsZero_EmptySlice(t *testing.T) {
	if !embedding.IsZero([]float32{}) {
		t.Error("IsZero([]float32{}) should be true")
	}
}

// TestIsZero_NonZeroElement ensures IsZero returns false when any element is non-zero.
func TestIsZero_NonZeroElement(t *testing.T) {
	if embedding.IsZero([]float32{0, 0, 1, 0}) {
		t.Error("IsZero([0,0,1,0]) should be false")
	}
}
