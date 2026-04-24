package repository_test

import (
	"context"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestExperimentRepo_UpsertDoesNotRevivePausedExperiment verifies that calling
// Upsert on an already-paused experiment does not reset it to 'running'.
func TestExperimentRepo_UpsertDoesNotRevivePausedExperiment(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewExperimentRepo(db)
	ctx := context.Background()

	// Create experiment and immediately pause it.
	if err := repo.Upsert(ctx, "guardrail-paused", 10); err != nil {
		t.Fatalf("initial upsert failed: %v", err)
	}
	db.Exec(`UPDATE ab_experiments SET status='paused', paused_at=NOW() WHERE name='guardrail-paused'`)

	// Simulates an agent updating treatment_pct after guardrail has fired.
	if err := repo.Upsert(ctx, "guardrail-paused", 20); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}

	exp, err := repo.Get(ctx, "guardrail-paused")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if exp.Status != "paused" {
		t.Errorf("Upsert revived a paused experiment: expected status='paused', got %q", exp.Status)
	}
	// treatment_pct update should still take effect.
	if exp.TreatmentPct != 20 {
		t.Errorf("expected treatment_pct=20 after upsert, got %d", exp.TreatmentPct)
	}
}

// TestExperimentRepo_UpsertCreatesRunning confirms that a brand-new experiment
// starts in the 'running' state.
func TestExperimentRepo_UpsertCreatesRunning(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewExperimentRepo(db)
	ctx := context.Background()

	if err := repo.Upsert(ctx, "new-exp", 10); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	exp, err := repo.Get(ctx, "new-exp")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if exp.Status != "running" {
		t.Errorf("new experiment should be 'running', got %q", exp.Status)
	}
}
