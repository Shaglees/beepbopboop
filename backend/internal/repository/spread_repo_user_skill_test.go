package repository_test

import (
	"math"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func setupSpreadRepo(t *testing.T) (*repository.SpreadRepo, string) {
	t.Helper()
	db := database.OpenTestDB(t)
	user, err := repository.NewUserRepo(db).FindOrCreateByFirebaseUID("firebase-spread-test")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return repository.NewSpreadRepo(db), user.ID
}

func sumWeights(t *testing.T, r *repository.SpreadRepo, userID string) float64 {
	t.Helper()
	st, err := r.GetTargets(userID)
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	if st == nil {
		t.Fatal("expected targets after upsert")
	}
	sum := 0.0
	for _, v := range st.Verticals {
		sum += v.Weight
	}
	return sum
}

func TestSpreadRepo_UpsertVerticalForFrequency_Daily(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	if err := repo.UpsertVerticalForFrequency(userID, "local-hs-football", 30); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	st, _ := repo.GetTargets(userID)
	if st == nil || st.Verticals["local-hs-football"].Weight == 0 {
		t.Fatalf("new vertical not added: %+v", st)
	}
	// Daily -> 30/30 * 0.1 = 0.1
	if got := st.Verticals["local-hs-football"].Weight; math.Abs(got-0.1) > 1e-9 {
		t.Errorf("daily weight = %v, want 0.1", got)
	}
	if sum := sumWeights(t, repo, userID); math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("weights should sum to 1.0, got %v", sum)
	}
}

func TestSpreadRepo_UpsertVerticalForFrequency_Monthly(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	if err := repo.UpsertVerticalForFrequency(userID, "rare-skill", 1); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	st, _ := repo.GetTargets(userID)
	got := st.Verticals["rare-skill"].Weight
	want := 1.0 / 30.0 * 0.1
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("monthly weight = %v, want %v", got, want)
	}
	if sum := sumWeights(t, repo, userID); math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("weights should sum to 1.0, got %v", sum)
	}
}

func TestSpreadRepo_UpsertVerticalForFrequency_ReUpsertChangesWeight(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	if err := repo.UpsertVerticalForFrequency(userID, "skill-a", 7); err != nil {
		t.Fatalf("upsert weekly: %v", err)
	}
	if err := repo.UpsertVerticalForFrequency(userID, "skill-a", 30); err != nil {
		t.Fatalf("upsert daily: %v", err)
	}

	st, _ := repo.GetTargets(userID)
	if got := st.Verticals["skill-a"].Weight; math.Abs(got-0.1) > 1e-9 {
		t.Errorf("after re-upsert to daily, weight = %v, want 0.1", got)
	}
	if sum := sumWeights(t, repo, userID); math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("weights should still sum to 1.0 after re-upsert, got %v", sum)
	}
}

func TestSpreadRepo_UpsertVerticalForFrequency_PreservesPinned(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	// Seed with a pinned default; re-upsert and ensure pinned survives.
	defaults := repository.DefaultTargets()
	if err := repo.UpsertTargets(userID, defaults); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	// Manually pin "sports".
	st, _ := repo.GetTargets(userID)
	v := st.Verticals["sports"]
	v.Pinned = true
	st.Verticals["sports"] = v
	if err := repo.UpsertTargets(userID, st); err != nil {
		t.Fatalf("pin sports: %v", err)
	}

	if err := repo.UpsertVerticalForFrequency(userID, "sports", 30); err != nil {
		t.Fatalf("upsert sports daily: %v", err)
	}
	st, _ = repo.GetTargets(userID)
	if !st.Verticals["sports"].Pinned {
		t.Error("pinned status should survive re-upsert")
	}
}

func TestSpreadRepo_UpsertVerticalForFrequency_OutOfRange(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	// Below min -> clamped to 1.
	if err := repo.UpsertVerticalForFrequency(userID, "low", -5); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	st, _ := repo.GetTargets(userID)
	want := 1.0 / 30.0 * 0.1
	if math.Abs(st.Verticals["low"].Weight-want) > 1e-9 {
		t.Errorf("low clamp: got %v, want %v", st.Verticals["low"].Weight, want)
	}

	// Above max -> clamped to 30.
	if err := repo.UpsertVerticalForFrequency(userID, "high", 999); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	st, _ = repo.GetTargets(userID)
	if math.Abs(st.Verticals["high"].Weight-0.1) > 1e-9 {
		t.Errorf("high clamp: got %v, want 0.1", st.Verticals["high"].Weight)
	}
}
