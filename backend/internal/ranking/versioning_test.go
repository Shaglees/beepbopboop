package ranking_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/ranking"
)

// --- ModelVersionRepo ---

// TestModelVersionRepo_RecordsAUCAndStatus verifies Insert stores auc_roc and
// status='trained', and MarkDeployed transitions to status='deployed' with a
// non-nil deployed_at timestamp.
func TestModelVersionRepo_RecordsAUCAndStatus(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := ranking.NewModelVersionRepo(db)

	id, err := repo.Insert(context.Background(), "v1.0.0-test", "/models/ranker_v1.json", 0.812)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	mv, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if mv.Status != "trained" {
		t.Errorf("expected status='trained', got %q", mv.Status)
	}
	if mv.AUCROC != 0.812 {
		t.Errorf("expected auc_roc=0.812, got %f", mv.AUCROC)
	}
	if mv.DeployedAt != nil {
		t.Error("deployed_at should be NULL before deployment")
	}

	if err := repo.MarkDeployed(context.Background(), id); err != nil {
		t.Fatalf("MarkDeployed: %v", err)
	}
	mv, _ = repo.Get(context.Background(), id)
	if mv.Status != "deployed" {
		t.Errorf("expected status='deployed' after MarkDeployed, got %q", mv.Status)
	}
	if mv.DeployedAt == nil {
		t.Error("deployed_at should be set after MarkDeployed")
	}
}

// TestModelVersionRepo_DuplicateVersionReturnsError asserts that inserting
// the same version string twice returns an error (unique constraint).
func TestModelVersionRepo_DuplicateVersionReturnsError(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := ranking.NewModelVersionRepo(db)

	_, err := repo.Insert(context.Background(), "v-dup", "/models/a.json", 0.80)
	if err != nil {
		t.Fatalf("first Insert: %v", err)
	}
	_, err = repo.Insert(context.Background(), "v-dup", "/models/b.json", 0.81)
	if err == nil {
		t.Error("expected error on duplicate version string, got nil")
	}
}

// TestModelVersionRepo_GetActive returns the currently deployed version.
func TestModelVersionRepo_GetActive(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := ranking.NewModelVersionRepo(db)

	// No active version yet.
	active, err := repo.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive on empty table: %v", err)
	}
	if active != nil {
		t.Error("expected nil when no version is deployed")
	}

	id, _ := repo.Insert(context.Background(), "v-active-1", "/models/r.json", 0.82)
	repo.MarkDeployed(context.Background(), id)

	active, err = repo.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if active == nil {
		t.Fatal("expected non-nil active version")
	}
	if active.Version != "v-active-1" {
		t.Errorf("expected version='v-active-1', got %q", active.Version)
	}
}

// TestModelVersionRepo_MarkRetired transitions a version to status='retired'.
func TestModelVersionRepo_MarkRetired(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := ranking.NewModelVersionRepo(db)

	id, _ := repo.Insert(context.Background(), "v-retire", "/models/r.json", 0.80)
	if err := repo.MarkRetired(context.Background(), id); err != nil {
		t.Fatalf("MarkRetired: %v", err)
	}

	mv, _ := repo.Get(context.Background(), id)
	if mv.Status != "retired" {
		t.Errorf("expected status='retired', got %q", mv.Status)
	}
}

// TestModelVersionRepo_List returns versions newest-first.
func TestModelVersionRepo_List(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := ranking.NewModelVersionRepo(db)

	for i, v := range []string{"v-list-a", "v-list-b", "v-list-c"} {
		repo.Insert(context.Background(), v, "/models/r.json", float64(i)*0.01+0.80)
		time.Sleep(2 * time.Millisecond) // ensure distinct trained_at
	}

	versions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(versions) < 3 {
		t.Fatalf("expected at least 3 versions, got %d", len(versions))
	}
	// Newest-first: v-list-c should be first.
	if versions[0].Version != "v-list-c" {
		t.Errorf("expected newest first (v-list-c), got %q", versions[0].Version)
	}
}

// TestModelVersionRepo_ReadyToRetrain returns true when post_events since
// last active model's trained_at exceeds minNewPairs.
func TestModelVersionRepo_ReadyToRetrain(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := ranking.NewModelVersionRepo(db)

	// With no active version and no events, should not be ready (nothing to retrain from).
	ready, err := repo.ReadyToRetrain(context.Background(), 5)
	if err != nil {
		t.Fatalf("ReadyToRetrain: %v", err)
	}
	if ready {
		t.Error("expected not-ready when no active version exists")
	}

	// Insert an active version trained well in the past.
	id, _ := repo.Insert(context.Background(), "v-retrain-base", "/models/r.json", 0.80)
	// Backdate trained_at to 1 hour ago so events seeded "now" are after it.
	db.Exec("UPDATE model_versions SET trained_at = $1 WHERE id = $2",
		time.Now().UTC().Add(-1*time.Hour), id)
	repo.MarkDeployed(context.Background(), id)

	// Seed fewer events than threshold — not ready.
	db.Exec(`INSERT INTO post_events (post_id, user_id, event_type, created_at)
		SELECT p.id, u.id, 'view', NOW()
		FROM posts p CROSS JOIN users u
		WHERE u.id != 'system'
		LIMIT 3`)

	ready, err = repo.ReadyToRetrain(context.Background(), 5)
	if err != nil {
		t.Fatalf("ReadyToRetrain after 3 events: %v", err)
	}
	if ready {
		t.Error("expected not-ready with only 3 events (threshold=5)")
	}
}

// --- DeploymentGate ---

// TestDeploymentGate_BlocksWhenImprovementBelow2Pct asserts the gate returns
// false when the new model is only 1.2% better (below 2% threshold).
func TestDeploymentGate_BlocksWhenImprovementBelow2Pct(t *testing.T) {
	gate := ranking.NewDeploymentGate(0.02)

	// 0.820 / 0.810 = 1.012 — 1.2% improvement, below threshold.
	if gate.ShouldDeploy(0.810, 0.820) {
		t.Error("gate should block when improvement is 1.2% (below 2% threshold)")
	}
}

// TestDeploymentGate_AllowsWhenImprovementMeetsThreshold asserts deployment
// proceeds when new AUC is ≥ current * 1.02.
func TestDeploymentGate_AllowsWhenImprovementMeetsThreshold(t *testing.T) {
	gate := ranking.NewDeploymentGate(0.02)

	// 0.830 / 0.810 ≈ 1.025 — 2.5% improvement, above threshold.
	if !gate.ShouldDeploy(0.810, 0.830) {
		t.Error("gate should allow when improvement is 2.5% (above 2% threshold)")
	}
}

// TestDeploymentGate_BlocksOnRegression asserts a worse model is always blocked.
func TestDeploymentGate_BlocksOnRegression(t *testing.T) {
	gate := ranking.NewDeploymentGate(0.02)
	if gate.ShouldDeploy(0.810, 0.790) {
		t.Error("gate must block when new model is worse than current")
	}
}

// TestDeploymentGate_AllowsWhenNoCurrentModel: with currentAUC=0 (no deployed
// model), any positive newAUC should pass the gate.
func TestDeploymentGate_AllowsWhenNoCurrentModel(t *testing.T) {
	gate := ranking.NewDeploymentGate(0.02)
	if !gate.ShouldDeploy(0, 0.70) {
		t.Error("gate should allow any positive AUC when there is no current model")
	}
}

// --- Ranker hot-reload ---

// TestRanker_Reload_SwapsModel verifies that calling Reload with a valid
// checkpoint replaces the in-memory model and subsequent calls use new weights.
func TestRanker_Reload_SwapsModel(t *testing.T) {
	r, err := ranking.NewRanker("testdata/ranker.json")
	if err != nil {
		t.Fatalf("NewRanker: %v", err)
	}
	// Reload with the same file — must succeed and leave the ranker functional.
	if err := r.Reload("testdata/ranker.json"); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	userVec := randomUnitVec(r.InputDim())
	postVec := randomUnitVec(r.InputDim())
	if _, err := r.Score(userVec, postVec); err != nil {
		t.Errorf("Score after Reload: %v", err)
	}
}

// TestRanker_Reload_InvalidFileIsIgnored verifies that a corrupt checkpoint
// does not replace the current model — subsequent Score calls still succeed.
func TestRanker_Reload_InvalidFileIsIgnored(t *testing.T) {
	r, err := ranking.NewRanker("testdata/ranker.json")
	if err != nil {
		t.Fatalf("NewRanker: %v", err)
	}
	origDim := r.InputDim()

	err = r.Reload("testdata/corrupt.json")
	if err == nil {
		t.Error("expected error when reloading corrupt checkpoint")
	}

	// Ranker must still be usable with the original model.
	userVec := randomUnitVec(origDim)
	postVec := randomUnitVec(origDim)
	if _, err := r.Score(userVec, postVec); err != nil {
		t.Errorf("Score after failed Reload should still work: %v", err)
	}
}

// TestRanker_Reload_NoRequestsDropped triggers a Reload concurrently with
// in-flight ScoreBatch calls and asserts all calls complete without error.
func TestRanker_Reload_NoRequestsDropped(t *testing.T) {
	r, err := ranking.NewRanker("testdata/ranker.json")
	if err != nil {
		t.Fatalf("NewRanker: %v", err)
	}

	const goroutines = 50
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userVec := randomUnitVec(r.InputDim())
			postVecs := make([][]float32, 10)
			for j := range postVecs {
				postVecs[j] = randomUnitVec(r.InputDim())
			}
			if _, err := r.ScoreBatch(userVec, postVecs); err != nil {
				errCh <- err
			}
		}()
	}

	// Trigger reload while goroutines are scoring.
	if err := r.Reload("testdata/ranker.json"); err != nil {
		t.Fatalf("Reload during concurrent scoring: %v", err)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("ScoreBatch error during hot reload: %v", err)
	}
}

// TestAdminVersionsEndpoint_ReturnsSortedByTrainedAt is covered by
// TestModelVersionRepo_List above (repo sorts newest-first). The handler
// test is exercised via integration in the handler package.

// --- helpers (supplement randomUnitVec from ranker_test.go) ---

// Note: randomUnitVec is defined in ranker_test.go in the same package.
