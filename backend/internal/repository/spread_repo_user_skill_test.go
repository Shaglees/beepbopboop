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

func TestSpreadRepo_UpsertVertical_AddsAndNormalizes(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	if err := repo.UpsertVertical(userID, "local-hs-football", 0.1); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	st, _ := repo.GetTargets(userID)
	if st == nil {
		t.Fatal("targets should exist after upsert")
	}
	if got := st.Verticals["local-hs-football"].Weight; math.Abs(got-0.1) > 1e-9 {
		t.Errorf("weight = %v, want 0.1", got)
	}
	if sum := sumWeights(t, repo, userID); math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("weights should sum to 1.0, got %v", sum)
	}
}

func TestSpreadRepo_UpsertVertical_ReUpsertChangesWeight(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	if err := repo.UpsertVertical(userID, "skill-a", 0.05); err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	if err := repo.UpsertVertical(userID, "skill-a", 0.1); err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	st, _ := repo.GetTargets(userID)
	if got := st.Verticals["skill-a"].Weight; math.Abs(got-0.1) > 1e-9 {
		t.Errorf("after re-upsert, weight = %v, want 0.1", got)
	}
	if sum := sumWeights(t, repo, userID); math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("weights should still sum to 1.0 after re-upsert, got %v", sum)
	}
}

func TestSpreadRepo_UpsertVertical_PreservesPinned(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	defaults := repository.DefaultTargets()
	if err := repo.UpsertTargets(userID, defaults); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	st, _ := repo.GetTargets(userID)
	v := st.Verticals["sports"]
	v.Pinned = true
	st.Verticals["sports"] = v
	if err := repo.UpsertTargets(userID, st); err != nil {
		t.Fatalf("pin sports: %v", err)
	}

	if err := repo.UpsertVertical(userID, "sports", 0.2); err != nil {
		t.Fatalf("upsert sports: %v", err)
	}
	st, _ = repo.GetTargets(userID)
	if !st.Verticals["sports"].Pinned {
		t.Error("pinned status should survive re-upsert")
	}
}

func TestSpreadRepo_UpsertVertical_ClampsOutOfRange(t *testing.T) {
	repo, userID := setupSpreadRepo(t)

	if err := repo.UpsertVertical(userID, "neg", -1); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	st, _ := repo.GetTargets(userID)
	if got := st.Verticals["neg"].Weight; got != 0 {
		t.Errorf("negative weight should clamp to 0, got %v", got)
	}

	if err := repo.UpsertVertical(userID, "huge", 5); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	st, _ = repo.GetTargets(userID)
	if got := st.Verticals["huge"].Weight; math.Abs(got-1) > 1e-9 {
		t.Errorf("over-1 weight should clamp to 1, got %v", got)
	}
}
