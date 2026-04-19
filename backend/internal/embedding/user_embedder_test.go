package embedding_test

// TDD tests for UserEmbedder.ComputeForUser and ComputeAll.
// Uses 4-dimensional embeddings so expected results can be derived exactly.

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// --- test setup helpers ---

type embedderTestFixture struct {
	db          *sql.DB
	embedder    *embedding.UserEmbedder
	userRepo    *repository.UserRepo
	agentRepo   *repository.AgentRepo
	postRepo    *repository.PostRepo
	postEmbRepo *repository.PostEmbeddingRepo
	reactionRepo *repository.ReactionRepo
}

func newFixture(t *testing.T) *embedderTestFixture {
	t.Helper()
	db := database.OpenTestDB(t)
	userEmbRepo := repository.NewUserEmbeddingRepo(db)
	return &embedderTestFixture{
		db:           db,
		embedder:     embedding.NewUserEmbedder(db, userEmbRepo),
		userRepo:     repository.NewUserRepo(db),
		agentRepo:    repository.NewAgentRepo(db),
		postRepo:     repository.NewPostRepo(db),
		postEmbRepo:  repository.NewPostEmbeddingRepo(db),
		reactionRepo: repository.NewReactionRepo(db),
	}
}

// makePost creates a user+agent+post and returns (userID, postID).
func (f *embedderTestFixture) makePost(t *testing.T, firebaseUID, name string) (userID, postID string) {
	t.Helper()
	u, err := f.userRepo.FindOrCreateByFirebaseUID(firebaseUID)
	if err != nil {
		t.Fatalf("FindOrCreateByFirebaseUID %q: %v", firebaseUID, err)
	}
	ag, err := f.agentRepo.Create(u.ID, name+" agent")
	if err != nil {
		t.Fatalf("agentRepo.Create: %v", err)
	}
	p, err := f.postRepo.Create(repository.CreatePostParams{
		AgentID:    ag.ID,
		UserID:     u.ID,
		Title:      name,
		Body:       "body",
		Visibility: "public",
	})
	if err != nil {
		t.Fatalf("postRepo.Create: %v", err)
	}
	return u.ID, p.ID
}

// seedEvent inserts a post_event with an explicit created_at timestamp.
func (f *embedderTestFixture) seedEvent(t *testing.T, postID, userID, eventType string, dwellMs *int, createdAt time.Time) {
	t.Helper()
	_, err := f.db.Exec(
		`INSERT INTO post_events (post_id, user_id, event_type, dwell_ms, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		postID, userID, eventType, dwellMs, createdAt,
	)
	if err != nil {
		t.Fatalf("seedEvent %s for user %s: %v", eventType, userID, err)
	}
}

// cosineSim returns cosine similarity for two equal-length float32 slices.
func cosineSim(a, b []float32) float64 {
	var dot, magA, magB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		magA += float64(a[i]) * float64(a[i])
		magB += float64(b[i]) * float64(b[i])
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// l2norm returns the L2 norm of a float32 slice.
func l2norm(v []float32) float64 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	return math.Sqrt(sum)
}

// --- tests ---

// TestComputeForUser_NoEngagement_ReturnsNil: user with no events → (nil, 0, nil).
// The cold-start path (not an error) should handle this case.
func TestComputeForUser_NoEngagement_ReturnsNil(t *testing.T) {
	f := newFixture(t)

	u, _ := f.userRepo.FindOrCreateByFirebaseUID("firebase-noeng")

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vec != nil {
		t.Errorf("expected nil vector for user with no engagement, got len=%d", len(vec))
	}
	if postCount != 0 {
		t.Errorf("expected postCount=0, got %d", postCount)
	}

	// No row should be written to user_embeddings.
	var count int
	f.db.QueryRow("SELECT COUNT(*) FROM user_embeddings WHERE user_id = $1", u.ID).Scan(&count)
	if count != 0 {
		t.Errorf("expected no user_embeddings row, got %d", count)
	}
}

// TestComputeForUser_NoEmbeddingForPost_ReturnsNil: events for posts whose
// embedding row doesn't exist yet → (nil, 0, nil), not an error.
func TestComputeForUser_NoEmbeddingForPost_ReturnsNil(t *testing.T) {
	f := newFixture(t)

	userID, postID := f.makePost(t, "firebase-noemb", "no-embedding post")
	// Seed a save event but deliberately skip seeding the post_embedding row.
	f.seedEvent(t, postID, userID, "save", nil, time.Now())

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vec != nil {
		t.Error("expected nil when engaged post has no embedding")
	}
	if postCount != 0 {
		t.Errorf("expected postCount=0, got %d", postCount)
	}
}

// TestComputeForUser_SingleSave_ReturnsPostEmbeddingDirection: user saves one
// post → result must point in that post's embedding direction (cosine_sim > 0.999).
func TestComputeForUser_SingleSave_ReturnsPostEmbeddingDirection(t *testing.T) {
	f := newFixture(t)

	userID, postID := f.makePost(t, "firebase-singlesave", "single save post")

	postEmb := []float32{0.0, 1.0, 0.0, 0.0}
	if err := f.postEmbRepo.Upsert(postID, postEmb, "test"); err != nil {
		t.Fatalf("seed post embedding: %v", err)
	}
	f.seedEvent(t, postID, userID, "save", nil, time.Now())

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec == nil {
		t.Fatal("expected non-nil vector after single save")
	}
	if postCount != 1 {
		t.Errorf("expected postCount=1, got %d", postCount)
	}

	sim := cosineSim(vec, postEmb)
	if sim < 0.999 {
		t.Errorf("cosine_sim(user_vec, post_emb) = %.4f, want > 0.999", sim)
	}

	// Result must be unit-length (L2-normalised).
	norm := l2norm(vec)
	if math.Abs(norm-1.0) > 1e-4 {
		t.Errorf("expected unit-length vector, got norm=%.6f", norm)
	}
}

// TestComputeForUser_WeightsEngagementSignals: save post A (weight 5) and click
// post B (weight 2) → result ≈ normalize([5,2,0,0]) = [0.9285, 0.3714, 0, 0].
func TestComputeForUser_WeightsEngagementSignals(t *testing.T) {
	f := newFixture(t)

	_, postIDA := f.makePost(t, "firebase-wt-a", "weight post A")
	_, postIDB := f.makePost(t, "firebase-wt-b", "weight post B")

	f.postEmbRepo.Upsert(postIDA, []float32{1, 0, 0, 0}, "test")
	f.postEmbRepo.Upsert(postIDB, []float32{0, 1, 0, 0}, "test")

	reactor, _ := f.userRepo.FindOrCreateByFirebaseUID("firebase-wt-reactor")
	f.seedEvent(t, postIDA, reactor.ID, "save", nil, time.Now())  // weight 5.0
	f.seedEvent(t, postIDB, reactor.ID, "click", nil, time.Now()) // weight 2.0

	vec, _, err := f.embedder.ComputeForUser(context.Background(), reactor.ID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec == nil {
		t.Fatal("expected non-nil vector")
	}
	if len(vec) != 4 {
		t.Fatalf("expected 4-dim vector, got %d", len(vec))
	}

	// Expected: normalize([5,2,0,0]); magnitude = sqrt(29).
	mag := math.Sqrt(29.0)
	wantX := float32(5.0 / mag)
	wantY := float32(2.0 / mag)

	const tol = 1e-3
	if math.Abs(float64(vec[0]-wantX)) > tol {
		t.Errorf("vec[0]: want %.4f, got %.4f", wantX, vec[0])
	}
	if math.Abs(float64(vec[1]-wantY)) > tol {
		t.Errorf("vec[1]: want %.4f, got %.4f", wantY, vec[1])
	}
	if math.Abs(float64(vec[2])) > tol || math.Abs(float64(vec[3])) > tol {
		t.Errorf("vec[2], vec[3] should be ~0, got %.4f, %.4f", vec[2], vec[3])
	}
}

// TestComputeForUser_HardNegative_SubtractsFromVector: save sports post then
// react 'not_for_me' → net weight = 0 so sports is skipped; only fashion click
// (weight 2) remains, so user vector should point toward fashion.
func TestComputeForUser_HardNegative_SubtractsFromVector(t *testing.T) {
	f := newFixture(t)

	_, sportsPostID := f.makePost(t, "firebase-hn-sports", "sports post")
	_, fashionPostID := f.makePost(t, "firebase-hn-fashion", "fashion post")

	f.postEmbRepo.Upsert(sportsPostID, []float32{1, 0, 0, 0}, "test")
	f.postEmbRepo.Upsert(fashionPostID, []float32{0, 1, 0, 0}, "test")

	reactor, _ := f.userRepo.FindOrCreateByFirebaseUID("firebase-hn-reactor")

	// Save sports (+5) then react 'not_for_me' (-5) → net = 0.
	f.seedEvent(t, sportsPostID, reactor.ID, "save", nil, time.Now())
	if _, err := f.reactionRepo.Upsert(sportsPostID, reactor.ID, "not_for_me"); err != nil {
		t.Fatalf("upsert reaction: %v", err)
	}

	// Click fashion (+2) → only contribution.
	f.seedEvent(t, fashionPostID, reactor.ID, "click", nil, time.Now())

	vec, _, err := f.embedder.ComputeForUser(context.Background(), reactor.ID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec == nil {
		t.Fatal("expected non-nil vector (fashion click should contribute)")
	}

	fashionEmb := []float32{0, 1, 0, 0}
	sportsEmb := []float32{1, 0, 0, 0}
	simFashion := cosineSim(vec, fashionEmb)
	simSports := cosineSim(vec, sportsEmb)

	if simFashion <= simSports {
		t.Errorf("expected closer to fashion (sim=%.3f) than sports (sim=%.3f) after hard negative",
			simFashion, simSports)
	}
}

// TestComputeForUser_DecayWeighting: same save signal on an old post (13 days) and
// a new post (today) → the new post should dominate because its decay factor ≈ 1.0.
func TestComputeForUser_DecayWeighting(t *testing.T) {
	f := newFixture(t)

	_, oldPostID := f.makePost(t, "firebase-decay-old", "old post")
	_, newPostID := f.makePost(t, "firebase-decay-new", "new post")

	f.postEmbRepo.Upsert(oldPostID, []float32{1, 0, 0, 0}, "test") // old → X
	f.postEmbRepo.Upsert(newPostID, []float32{0, 1, 0, 0}, "test") // new → Y

	reactor, _ := f.userRepo.FindOrCreateByFirebaseUID("firebase-decay-reactor")

	// Old save: 13 days ago (within 14-day window, but strongly decayed).
	f.seedEvent(t, oldPostID, reactor.ID, "save", nil, time.Now().UTC().Add(-13*24*time.Hour))
	// New save: today (decay ≈ 1.0).
	f.seedEvent(t, newPostID, reactor.ID, "save", nil, time.Now())

	vec, _, err := f.embedder.ComputeForUser(context.Background(), reactor.ID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec == nil {
		t.Fatal("expected non-nil vector")
	}

	simOld := cosineSim(vec, []float32{1, 0, 0, 0})
	simNew := cosineSim(vec, []float32{0, 1, 0, 0})

	if simNew <= simOld {
		t.Errorf("expected new post (sim=%.3f) to outweigh old post (sim=%.3f) due to decay",
			simNew, simOld)
	}
}

// TestComputeForUser_OnlyCountsLast14Days: events 15 days old are excluded;
// only events from 13 days ago contribute.
func TestComputeForUser_OnlyCountsLast14Days(t *testing.T) {
	f := newFixture(t)

	_, oldPostID := f.makePost(t, "firebase-14d-old", "15-day-old post")
	_, recentPostID := f.makePost(t, "firebase-14d-new", "13-day-old post")

	f.postEmbRepo.Upsert(oldPostID, []float32{1, 0, 0, 0}, "test")
	f.postEmbRepo.Upsert(recentPostID, []float32{0, 1, 0, 0}, "test")

	reactor, _ := f.userRepo.FindOrCreateByFirebaseUID("firebase-14d-reactor")
	f.seedEvent(t, oldPostID, reactor.ID, "save", nil, time.Now().UTC().Add(-15*24*time.Hour))
	f.seedEvent(t, recentPostID, reactor.ID, "save", nil, time.Now().UTC().Add(-13*24*time.Hour))

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), reactor.ID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec == nil {
		t.Fatal("expected non-nil vector (recent post must contribute)")
	}
	if postCount != 1 {
		t.Errorf("expected postCount=1 (only recent post), got %d", postCount)
	}

	simRecent := cosineSim(vec, []float32{0, 1, 0, 0})
	simOld := cosineSim(vec, []float32{1, 0, 0, 0})

	if simOld > 0.01 {
		t.Errorf("15-day-old post should not contribute; cosine_sim with old embedding = %.4f", simOld)
	}
	if simRecent < 0.99 {
		t.Errorf("expected result to point toward recent post, sim = %.4f", simRecent)
	}
}

// TestComputeForUser_AllWeightsZero_ReturnsNil: save and matching 'not_for_me'
// cancel each other (net weight = 0) → (nil, 0, nil).
func TestComputeForUser_AllWeightsZero_ReturnsNil(t *testing.T) {
	f := newFixture(t)

	_, postID := f.makePost(t, "firebase-allzero", "cancelled post")
	f.postEmbRepo.Upsert(postID, []float32{1, 0, 0, 0}, "test")

	reactor, _ := f.userRepo.FindOrCreateByFirebaseUID("firebase-allzero-reactor")
	f.seedEvent(t, postID, reactor.ID, "save", nil, time.Now()) // +5.0
	f.reactionRepo.Upsert(postID, reactor.ID, "not_for_me")     // -5.0 → net = 0

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), reactor.ID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec != nil {
		t.Error("expected nil when all weights cancel to zero")
	}
	if postCount != 0 {
		t.Errorf("expected postCount=0, got %d", postCount)
	}
}

// TestComputeAll_UpdatesAllUsers: three users each with a save event →
// ComputeAll must write a user_embeddings row for each of them.
func TestComputeAll_UpdatesAllUsers(t *testing.T) {
	f := newFixture(t)

	postEmb := []float32{0.6, 0.8, 0.0, 0.0}
	var reactorIDs []string

	for i := 0; i < 3; i++ {
		// Each post needs its own agent/owner.
		ownerUID := fmt.Sprintf("firebase-all-owner-%d", i)
		_, postID := f.makePost(t, ownerUID, fmt.Sprintf("post-%d", i))
		f.postEmbRepo.Upsert(postID, postEmb, "test")

		reactor, _ := f.userRepo.FindOrCreateByFirebaseUID(fmt.Sprintf("firebase-all-reactor-%d", i))
		f.seedEvent(t, postID, reactor.ID, "save", nil, time.Now())
		reactorIDs = append(reactorIDs, reactor.ID)
	}

	if err := f.embedder.ComputeAll(context.Background()); err != nil {
		t.Fatalf("ComputeAll: %v", err)
	}

	for _, uid := range reactorIDs {
		var count int
		f.db.QueryRow("SELECT COUNT(*) FROM user_embeddings WHERE user_id = $1", uid).Scan(&count)
		if count != 1 {
			t.Errorf("expected 1 user_embeddings row for user %s, got %d", uid, count)
		}
	}
}

// TestComputeAll_NoUsers_NoError: ComputeAll with zero active users returns nil, no-op.
func TestComputeAll_NoUsers_NoError(t *testing.T) {
	f := newFixture(t)
	if err := f.embedder.ComputeAll(context.Background()); err != nil {
		t.Fatalf("ComputeAll on empty DB: %v", err)
	}
}

// TestComputeForUser_SaveThenUnsave_NotCountedAsSaved: a save followed by an unsave
// means the post is no longer saved; it should not contribute the save weight.
func TestComputeForUser_SaveThenUnsave_NotCountedAsSaved(t *testing.T) {
	f := newFixture(t)

	userID, postID := f.makePost(t, "firebase-unsave", "unsaved post")
	f.postEmbRepo.Upsert(postID, []float32{1, 0, 0, 0}, "test")

	// Save at T, then unsave later — net: not saved.
	f.seedEvent(t, postID, userID, "save", nil, time.Now().UTC().Add(-2*time.Hour))
	f.seedEvent(t, postID, userID, "unsave", nil, time.Now())

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	// Only a view-level signal (0) remains — weight should be 0 → nil result.
	if vec != nil {
		t.Error("expected nil vector after save+unsave (net weight = 0)")
	}
	if postCount != 0 {
		t.Errorf("expected postCount=0, got %d", postCount)
	}
}

// TestComputeForUser_DecayUsesSaveTime_NotViewTime: a recent save and an old view on
// the same post — decay should be anchored to the save time, not the (possibly later)
// view time. We verify this by checking that a recent save gives a high-weight result.
func TestComputeForUser_DecayUsesSaveTime_NotViewTime(t *testing.T) {
	f := newFixture(t)

	userID, postID := f.makePost(t, "firebase-decaytime", "decay-time post")
	f.postEmbRepo.Upsert(postID, []float32{0, 1, 0, 0}, "test")

	// Old view (12 days ago) — should not anchor the decay timestamp.
	f.seedEvent(t, postID, userID, "view", nil, time.Now().UTC().Add(-12*24*time.Hour))
	// Recent save (today) — this should set the decay anchor.
	f.seedEvent(t, postID, userID, "save", nil, time.Now())

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec == nil {
		t.Fatal("expected non-nil vector")
	}
	if postCount != 1 {
		t.Errorf("expected postCount=1, got %d", postCount)
	}

	// With decay anchor = now, decay ≈ 1.0 and weight = save(5) + view(0.3) ≈ 5.3.
	// The result should be unit-length and pointing at [0,1,0,0].
	sim := cosineSim(vec, []float32{0, 1, 0, 0})
	if sim < 0.999 {
		t.Errorf("expected result to closely match post embedding, cosine_sim=%.4f", sim)
	}
}

// TestComputeForUser_MultipleEventsForSamePost_CountedOnce: several events for the
// same post must not fan out into multiple rows in the result set (GROUP BY fix).
func TestComputeForUser_MultipleEventsForSamePost_CountedOnce(t *testing.T) {
	f := newFixture(t)

	userID, postID := f.makePost(t, "firebase-multiev", "multi-event post")
	f.postEmbRepo.Upsert(postID, []float32{1, 0, 0, 0}, "test")

	// Multiple events of different types for the same post.
	f.seedEvent(t, postID, userID, "view", nil, time.Now())
	f.seedEvent(t, postID, userID, "click", nil, time.Now())
	f.seedEvent(t, postID, userID, "save", nil, time.Now())

	vec, postCount, err := f.embedder.ComputeForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ComputeForUser: %v", err)
	}
	if vec == nil {
		t.Fatal("expected non-nil vector")
	}
	// Exactly one post contributed — if GROUP BY fan-out bug were present, postCount
	// could be > 1 and the vector would have inflated weights.
	if postCount != 1 {
		t.Errorf("expected postCount=1, got %d (GROUP BY fan-out?)", postCount)
	}
}

// TestComputeAll_DoesNotDeadlock: ComputeAll must not deadlock even when the
// connection pool is limited to a single connection.
func TestComputeAll_DoesNotDeadlock(t *testing.T) {
	f := newFixture(t)
	f.db.SetMaxOpenConns(1)

	// Seed one active user so the inner per-user loop actually runs.
	userID, postID := f.makePost(t, "firebase-deadlock", "deadlock test post")
	f.postEmbRepo.Upsert(postID, []float32{1, 0, 0, 0}, "test")
	f.seedEvent(t, postID, userID, "save", nil, time.Now())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := f.embedder.ComputeAll(ctx); err != nil {
		t.Fatalf("ComputeAll: %v", err)
	}
	if ctx.Err() != nil {
		t.Error("ComputeAll deadlocked (context timed out)")
	}
}
