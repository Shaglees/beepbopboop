package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// ── ComputeLabel unit tests (no DB) ─────────────────────────────────────────

func TestComputeLabel_SaveProducesPositive(t *testing.T) {
	if got := repository.ComputeLabel(true, false, 500, ""); got != 1.0 {
		t.Errorf("save: expected 1.0, got %f", got)
	}
}

func TestComputeLabel_MoreReactionProducesPositive(t *testing.T) {
	if got := repository.ComputeLabel(false, false, 200, "more"); got != 1.0 {
		t.Errorf("more reaction: expected 1.0, got %f", got)
	}
}

func TestComputeLabel_LessReactionIsHardNegative(t *testing.T) {
	// Hard negative even with high dwell
	if got := repository.ComputeLabel(false, false, 30000, "less"); got != 0.0 {
		t.Errorf("less reaction: expected 0.0, got %f", got)
	}
}

func TestComputeLabel_NotForMeIsHardNegative(t *testing.T) {
	// Hard negative even when saved
	if got := repository.ComputeLabel(true, false, 15000, "not_for_me"); got != 0.0 {
		t.Errorf("not_for_me reaction: expected 0.0, got %f", got)
	}
}

func TestComputeLabel_HighDwellProducesPositive(t *testing.T) {
	if got := repository.ComputeLabel(false, false, 10000, ""); got != 1.0 {
		t.Errorf("dwell=10000: expected 1.0, got %f", got)
	}
}

func TestComputeLabel_ClickProducesLabel(t *testing.T) {
	if got := repository.ComputeLabel(false, true, 500, ""); got != 0.8 {
		t.Errorf("click only: expected 0.8, got %f", got)
	}
}

func TestComputeLabel_MediumDwellProducesLabel(t *testing.T) {
	if got := repository.ComputeLabel(false, false, 5000, ""); got != 0.6 {
		t.Errorf("dwell=5000: expected 0.6, got %f", got)
	}
}

func TestComputeLabel_LowDwellProducesSmallLabel(t *testing.T) {
	if got := repository.ComputeLabel(false, false, 1500, ""); got != 0.3 {
		t.Errorf("dwell=1500: expected 0.3, got %f", got)
	}
}

func TestComputeLabel_VeryLowDwellProducesZero(t *testing.T) {
	if got := repository.ComputeLabel(false, false, 800, ""); got != 0.0 {
		t.Errorf("dwell=800: expected 0.0, got %f", got)
	}
}

func TestComputeLabel_SaveWinsOverClick(t *testing.T) {
	// When both save and click exist, save wins (1.0 not 0.8)
	if got := repository.ComputeLabel(true, true, 500, ""); got != 1.0 {
		t.Errorf("save+click: expected 1.0, got %f", got)
	}
}

// ── SplitByUser unit tests (no DB) ──────────────────────────────────────────

func TestSplitByUser_NoUserLeakage(t *testing.T) {
	// 10 users, 3 pairs each → 30 pairs
	var pairs []model.TrainingPair
	for i := 0; i < 10; i++ {
		uid := string(rune('A' + i))
		for j := 0; j < 3; j++ {
			pairs = append(pairs, model.TrainingPair{
				UserID: uid,
				PostID: uid + string(rune('0'+j)),
				Label:  float64(j) * 0.5,
			})
		}
	}

	train, val, test := repository.SplitByUser(pairs, 0.7, 0.15)

	trainUsers := userSet(train)
	valUsers := userSet(val)
	testUsers := userSet(test)

	for u := range trainUsers {
		if valUsers[u] {
			t.Errorf("user %s appears in both train and val", u)
		}
		if testUsers[u] {
			t.Errorf("user %s appears in both train and test", u)
		}
	}
	for u := range valUsers {
		if testUsers[u] {
			t.Errorf("user %s appears in both val and test", u)
		}
	}
}

func TestSplitByUser_AllPairsPreserved(t *testing.T) {
	pairs := []model.TrainingPair{
		{UserID: "u1", PostID: "p1", Label: 1.0},
		{UserID: "u1", PostID: "p2", Label: 0.0},
		{UserID: "u2", PostID: "p3", Label: 0.5},
	}

	train, val, test := repository.SplitByUser(pairs, 0.7, 0.15)

	total := len(train) + len(val) + len(test)
	if total != len(pairs) {
		t.Errorf("expected %d total pairs, got %d", len(pairs), total)
	}
}

// ── ValidateLabelDistribution unit tests ────────────────────────────────────

func TestValidateLabelDistribution_EmptyIsError(t *testing.T) {
	err := repository.ValidateLabelDistribution(nil, 0.2, 0.4)
	if err == nil {
		t.Error("expected error for empty pairs, got nil")
	}
}

func TestValidateLabelDistribution_TooFewPositivesIsError(t *testing.T) {
	// 5% positive (1 of 20) — below 20% threshold
	pairs := make([]model.TrainingPair, 20)
	pairs[0].Label = 1.0
	err := repository.ValidateLabelDistribution(pairs, 0.2, 0.4)
	if err == nil {
		t.Error("expected error for 5% positive rate, got nil")
	}
}

func TestValidateLabelDistribution_TooManyPositivesIsError(t *testing.T) {
	// 90% positive (18 of 20) — above 40% threshold
	pairs := make([]model.TrainingPair, 20)
	for i := 0; i < 18; i++ {
		pairs[i].Label = 1.0
	}
	err := repository.ValidateLabelDistribution(pairs, 0.2, 0.4)
	if err == nil {
		t.Error("expected error for 90% positive rate, got nil")
	}
}

func TestValidateLabelDistribution_BalancedPassesValidation(t *testing.T) {
	// 30% positive (6 of 20)
	pairs := make([]model.TrainingPair, 20)
	for i := 0; i < 6; i++ {
		pairs[i].Label = 1.0
	}
	err := repository.ValidateLabelDistribution(pairs, 0.2, 0.4)
	if err != nil {
		t.Errorf("expected nil error for 30%% positive rate, got %v", err)
	}
}

// ── ExportPairs integration tests (real Postgres via testcontainers) ─────────

func TestTrainingRepo_ExportPairs_DeduplicatesUserPostPairs(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	trainingRepo := repository.NewTrainingRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-dedup")
	agent, _ := agentRepo.Create(user.ID, "Test Agent")
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Deduplicate me", Body: "body",
	})

	// Two save events for the same (user, post) pair
	eventRepo.Create(post.ID, user.ID, "save", nil)
	eventRepo.Create(post.ID, user.ID, "save", nil)

	pairs, err := trainingRepo.ExportPairs(90)
	if err != nil {
		t.Fatalf("ExportPairs failed: %v", err)
	}

	count := 0
	for _, p := range pairs {
		if p.UserID == user.ID && p.PostID == post.ID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 deduplicated pair, got %d", count)
	}
}

func TestTrainingRepo_ExportPairs_SavedPairHasLabelOne(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	trainingRepo := repository.NewTrainingRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-label-save")
	agent, _ := agentRepo.Create(user.ID, "Test Agent")
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Save me", Body: "body",
	})
	eventRepo.Create(post.ID, user.ID, "save", nil)

	pairs, err := trainingRepo.ExportPairs(90)
	if err != nil {
		t.Fatalf("ExportPairs failed: %v", err)
	}

	for _, p := range pairs {
		if p.UserID == user.ID && p.PostID == post.ID {
			if p.Label != 1.0 {
				t.Errorf("saved pair: expected label 1.0, got %f", p.Label)
			}
			return
		}
	}
	t.Error("saved pair not found in export")
}

func TestTrainingRepo_ExportPairs_LessReactionIsHardNegative(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	reactionRepo := repository.NewReactionRepo(db)
	trainingRepo := repository.NewTrainingRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-label-less")
	agent, _ := agentRepo.Create(user.ID, "Test Agent")
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "I don't like this", Body: "body",
	})

	// High dwell but explicit negative reaction — hard negative wins
	dwell := 30000
	eventRepo.Create(post.ID, user.ID, "view", &dwell)
	reactionRepo.Upsert(post.ID, user.ID, "less")

	pairs, err := trainingRepo.ExportPairs(90)
	if err != nil {
		t.Fatalf("ExportPairs failed: %v", err)
	}

	for _, p := range pairs {
		if p.UserID == user.ID && p.PostID == post.ID {
			if p.Label != 0.0 {
				t.Errorf("less reaction: expected label 0.0, got %f", p.Label)
			}
			return
		}
	}
	t.Error("pair not found in export")
}

func TestTrainingRepo_ExportPairs_LabelDistributionWithinBounds(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	trainingRepo := repository.NewTrainingRepo(db)

	// 3 users × 10 posts = 30 pairs: ~30% will be positive (saves)
	for u := 0; u < 3; u++ {
		uid := []string{"firebase-dist-a", "firebase-dist-b", "firebase-dist-c"}[u]
		user, _ := userRepo.FindOrCreateByFirebaseUID(uid)
		agent, _ := agentRepo.Create(user.ID, "Agent")
		for p := 0; p < 10; p++ {
			dwell := 500 // below all thresholds → 0.0
			post, _ := postRepo.Create(repository.CreatePostParams{
				AgentID: agent.ID, UserID: user.ID, Title: "Post", Body: "body",
			})
			eventRepo.Create(post.ID, user.ID, "view", &dwell)
			if p < 3 { // 3 out of 10 are saves = 30%
				eventRepo.Create(post.ID, user.ID, "save", nil)
			}
		}
	}

	pairs, err := trainingRepo.ExportPairs(90)
	if err != nil {
		t.Fatalf("ExportPairs failed: %v", err)
	}

	// Fresh DB contains only the test data created above, so no filtering needed.
	if err := repository.ValidateLabelDistribution(pairs, 0.2, 0.4); err != nil {
		t.Errorf("label distribution out of bounds: %v", err)
	}
}

// userSet builds a set of user IDs from a slice of training pairs.
func userSet(pairs []model.TrainingPair) map[string]bool {
	m := make(map[string]bool)
	for _, p := range pairs {
		m[p.UserID] = true
	}
	return m
}
