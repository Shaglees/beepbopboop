package ab_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ab"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestGuardrail_PausesTreatmentOnSaveRateRegression(t *testing.T) {
	db := database.OpenTestDB(t)
	g := ab.NewGuardrail(db, ab.GuardrailConfig{SaveRateDropPct: 5, SessionDropPct: 10})

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	expName := "save-regression-exp"
	db.Exec("INSERT INTO ab_experiments (name, treatment_pct, status) VALUES ($1, 50, 'running') ON CONFLICT DO NOTHING", expName)

	controlUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-gr-control")
	agent, _ := agentRepo.Create(controlUser.ID, "guardrail-agent")
	controlPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: controlUser.ID, Title: "Control post", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1, $2, 'control')",
		controlUser.ID, expName)
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'view', 'control', $3)",
			controlPost.ID, controlUser.ID, time.Now().UTC())
	}
	for i := 0; i < 10; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'save', 'control', $3)",
			controlPost.ID, controlUser.ID, time.Now().UTC())
	}

	treatmentUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-gr-treatment")
	treatmentPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: treatmentUser.ID, Title: "Treatment post", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1, $2, 'treatment')",
		treatmentUser.ID, expName)
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'view', 'treatment', $3)",
			treatmentPost.ID, treatmentUser.ID, time.Now().UTC())
	}
	for i := 0; i < 4; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'save', 'treatment', $3)",
			treatmentPost.ID, treatmentUser.ID, time.Now().UTC())
	}

	paused, err := g.CheckAndPause(context.Background(), expName)
	if err != nil {
		t.Fatalf("guardrail check error: %v", err)
	}
	if !paused {
		t.Error("expected guardrail to pause treatment on >5% save rate drop")
	}

	var status string
	db.QueryRow("SELECT status FROM ab_experiments WHERE name=$1", expName).Scan(&status)
	if status != "paused" {
		t.Errorf("expected ab_experiments.status='paused', got %q", status)
	}
}

func TestGuardrail_NoActionWhenMetricsImprove(t *testing.T) {
	db := database.OpenTestDB(t)
	g := ab.NewGuardrail(db, ab.GuardrailConfig{SaveRateDropPct: 5, SessionDropPct: 10})

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	expName := fmt.Sprintf("save-improve-exp-%d", time.Now().UnixNano())
	db.Exec("INSERT INTO ab_experiments (name, treatment_pct, status) VALUES ($1, 50, 'running') ON CONFLICT DO NOTHING", expName)

	controlUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-gi-control")
	agent, _ := agentRepo.Create(controlUser.ID, "gi-agent")
	controlPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: controlUser.ID, Title: "C post", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1, $2, 'control')",
		controlUser.ID, expName)
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'view', 'control', $3)",
			controlPost.ID, controlUser.ID, time.Now().UTC())
	}
	for i := 0; i < 10; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'save', 'control', $3)",
			controlPost.ID, controlUser.ID, time.Now().UTC())
	}

	treatmentUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-gi-treatment")
	treatmentPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: treatmentUser.ID, Title: "T post", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1, $2, 'treatment')",
		treatmentUser.ID, expName)
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'view', 'treatment', $3)",
			treatmentPost.ID, treatmentUser.ID, time.Now().UTC())
	}
	for i := 0; i < 12; i++ {
		db.Exec("INSERT INTO post_events (post_id, user_id, event_type, ab_variant, created_at) VALUES ($1, $2, 'save', 'treatment', $3)",
			treatmentPost.ID, treatmentUser.ID, time.Now().UTC())
	}

	paused, err := g.CheckAndPause(context.Background(), expName)
	if err != nil {
		t.Fatalf("guardrail check error: %v", err)
	}
	if paused {
		t.Error("expected guardrail NOT to pause when treatment metrics improve")
	}
}

func TestGuardrail_InsufficientData(t *testing.T) {
	db := database.OpenTestDB(t)
	g := ab.NewGuardrail(db, ab.GuardrailConfig{SaveRateDropPct: 5, SessionDropPct: 10})

	paused, err := g.CheckAndPause(context.Background(), "nonexistent-experiment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paused {
		t.Error("expected no pause when there is no data")
	}
}
