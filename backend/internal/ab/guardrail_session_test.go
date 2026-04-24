package ab_test

import (
	"context"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ab"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestGuardrail_PausesOnSessionDropRegression verifies that CheckAndPause fires
// when treatment has significantly fewer impressions (sessions) than control,
// even when the save rate is identical between variants.
//
// Control:   100 views, 10 saves → 10% save rate
// Treatment:  50 views,  5 saves → 10% save rate (save rate unchanged)
// Session drop: (100-50)/100 * 100 = 50% > SessionDropPct=30 → should pause.
func TestGuardrail_PausesOnSessionDropRegression(t *testing.T) {
	db := database.OpenTestDB(t)
	// SaveRateDropPct=99 ensures save-rate branch cannot trigger the pause.
	g := ab.NewGuardrail(db, ab.GuardrailConfig{SaveRateDropPct: 99, SessionDropPct: 30})

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	expName := "session-drop-exp"

	controlUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-sd-control")
	agent, _ := agentRepo.Create(controlUser.ID, "sd-agent")
	controlPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: controlUser.ID, Title: "Control", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1,$2,'control')",
		controlUser.ID, expName)
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO post_events (post_id,user_id,event_type,ab_variant,created_at) VALUES ($1,$2,'view','control',$3)",
			controlPost.ID, controlUser.ID, time.Now().UTC())
	}
	for i := 0; i < 10; i++ {
		db.Exec("INSERT INTO post_events (post_id,user_id,event_type,ab_variant,created_at) VALUES ($1,$2,'save','control',$3)",
			controlPost.ID, controlUser.ID, time.Now().UTC())
	}

	treatmentUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-sd-treatment")
	treatmentPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: treatmentUser.ID, Title: "Treatment", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1,$2,'treatment')",
		treatmentUser.ID, expName)
	for i := 0; i < 50; i++ {
		db.Exec("INSERT INTO post_events (post_id,user_id,event_type,ab_variant,created_at) VALUES ($1,$2,'view','treatment',$3)",
			treatmentPost.ID, treatmentUser.ID, time.Now().UTC())
	}
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO post_events (post_id,user_id,event_type,ab_variant,created_at) VALUES ($1,$2,'save','treatment',$3)",
			treatmentPost.ID, treatmentUser.ID, time.Now().UTC())
	}

	paused, err := g.CheckAndPause(context.Background(), expName)
	if err != nil {
		t.Fatalf("CheckAndPause error: %v", err)
	}
	if !paused {
		t.Error("expected guardrail to pause when session drop (50%) exceeds SessionDropPct (30%)")
	}

	var status string
	db.QueryRow("SELECT status FROM ab_experiments WHERE name=$1", expName).Scan(&status)
	if status != "paused" {
		t.Errorf("expected ab_experiments.status='paused', got %q", status)
	}
}

// TestGuardrail_NoActionWhenSessionsAreEqual verifies that equal session counts
// do not trigger the session-drop guardrail.
func TestGuardrail_NoActionWhenSessionsAreEqual(t *testing.T) {
	db := database.OpenTestDB(t)
	g := ab.NewGuardrail(db, ab.GuardrailConfig{SaveRateDropPct: 99, SessionDropPct: 30})

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	expName := "session-equal-exp"

	controlUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-se-control")
	agent, _ := agentRepo.Create(controlUser.ID, "se-agent")
	controlPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: controlUser.ID, Title: "Control", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1,$2,'control')",
		controlUser.ID, expName)
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO post_events (post_id,user_id,event_type,ab_variant,created_at) VALUES ($1,$2,'view','control',$3)",
			controlPost.ID, controlUser.ID, time.Now().UTC())
	}

	treatmentUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-se-treatment")
	treatmentPost, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: treatmentUser.ID, Title: "Treatment", Body: "body",
	})
	db.Exec("INSERT INTO ab_assignments (user_id, experiment, variant) VALUES ($1,$2,'treatment')",
		treatmentUser.ID, expName)
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO post_events (post_id,user_id,event_type,ab_variant,created_at) VALUES ($1,$2,'view','treatment',$3)",
			treatmentPost.ID, treatmentUser.ID, time.Now().UTC())
	}

	paused, err := g.CheckAndPause(context.Background(), expName)
	if err != nil {
		t.Fatalf("CheckAndPause error: %v", err)
	}
	if paused {
		t.Error("expected no pause when session counts are equal")
	}
}
