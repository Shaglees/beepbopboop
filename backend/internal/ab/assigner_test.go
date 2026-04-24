package ab_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/ab"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func ensureExperiment(db *sql.DB, name string) {
	db.Exec("INSERT INTO ab_experiments (name, treatment_pct, status) VALUES ($1, 50, 'running') ON CONFLICT DO NOTHING", name)
}

func TestVariant_IsDeterministic(t *testing.T) {
	db := database.OpenTestDB(t)
	a := ab.NewAssigner(db)
	ensureExperiment(db, "ml-feed-v1")

	first := a.Variant(context.Background(), "user-abc", "ml-feed-v1", 10)
	second := a.Variant(context.Background(), "user-abc", "ml-feed-v1", 10)

	if first != second {
		t.Errorf("variant changed between calls: %q then %q", first, second)
	}
	if first != "control" && first != "treatment" {
		t.Errorf("variant must be 'control' or 'treatment', got %q", first)
	}
}

func TestVariant_DistributionMatchesTarget(t *testing.T) {
	db := database.OpenTestDB(t)
	a := ab.NewAssigner(db)
	ensureExperiment(db, "dist-test")

	const n = 1000
	const targetPct = 10
	treatment := 0
	for i := 0; i < n; i++ {
		userID := fmt.Sprintf("dist-user-%d", i)
		if a.Variant(context.Background(), userID, "dist-test", targetPct) == "treatment" {
			treatment++
		}
	}
	actual := float64(treatment) / n * 100
	if actual < float64(targetPct)-2 || actual > float64(targetPct)+2 {
		t.Errorf("treatment rate %.1f%% outside [%d±2]%%", actual, targetPct)
	}
}

func TestVariant_PersistedToDatabase(t *testing.T) {
	db := database.OpenTestDB(t)
	a := ab.NewAssigner(db)

	userRepo := repository.NewUserRepo(db)
	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-ab-persist")

	ensureExperiment(db, "persist-test")
	variant := a.Variant(context.Background(), u.ID, "persist-test", 50)

	var stored string
	err := db.QueryRow(
		"SELECT variant FROM ab_assignments WHERE user_id=$1 AND experiment=$2",
		u.ID, "persist-test",
	).Scan(&stored)
	if err != nil {
		t.Fatalf("assignment not persisted: %v", err)
	}
	if stored != variant {
		t.Errorf("stored variant %q != returned variant %q", stored, variant)
	}
}

func TestVariant_DifferentExperimentsAreIndependent(t *testing.T) {
	db := database.OpenTestDB(t)
	a := ab.NewAssigner(db)

	userRepo := repository.NewUserRepo(db)
	u, _ := userRepo.FindOrCreateByFirebaseUID("firebase-ab-independent")

	ensureExperiment(db, "exp-alpha")
	ensureExperiment(db, "exp-beta")
	v1 := a.Variant(context.Background(), u.ID, "exp-alpha", 50)
	v2 := a.Variant(context.Background(), u.ID, "exp-beta", 50)

	var count int
	db.QueryRow(
		"SELECT COUNT(*) FROM ab_assignments WHERE user_id=$1", u.ID,
	).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 ab_assignments rows, got %d", count)
	}

	for _, v := range []string{v1, v2} {
		if v != "control" && v != "treatment" {
			t.Errorf("invalid variant %q", v)
		}
	}
}

func TestVariant_100PctTreatmentAssignsEveryone(t *testing.T) {
	db := database.OpenTestDB(t)
	a := ab.NewAssigner(db)
	ensureExperiment(db, "hundred-pct")

	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("hundred-user-%d", i)
		v := a.Variant(context.Background(), userID, "hundred-pct", 100)
		if v != "treatment" {
			t.Errorf("user %s: expected treatment with 100%% split, got %q", userID, v)
		}
	}
}

func TestVariant_0PctTreatmentAssignsNoOne(t *testing.T) {
	db := database.OpenTestDB(t)
	a := ab.NewAssigner(db)
	ensureExperiment(db, "zero-pct")

	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("zero-user-%d", i)
		v := a.Variant(context.Background(), userID, "zero-pct", 0)
		if v != "control" {
			t.Errorf("user %s: expected control with 0%% split, got %q", userID, v)
		}
	}
}
