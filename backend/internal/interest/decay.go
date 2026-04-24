package interest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

const (
	disengagementDays = 30
	backoffDays       = 90
	maxAsks           = 3
)

type DecayChecker struct {
	db           *sql.DB
	interestRepo *repository.UserInterestRepo
	postRepo     *repository.PostRepo
	agentID      string // system agent that posts feedback panels
}

func NewDecayChecker(db *sql.DB, interestRepo *repository.UserInterestRepo, postRepo *repository.PostRepo, agentID string) *DecayChecker {
	return &DecayChecker{db: db, interestRepo: interestRepo, postRepo: postRepo, agentID: agentID}
}

type decayCandidate struct {
	InterestID string
	UserID     string
	Category   string
	Topic      string
	TimesAsked int
	LastAsked  *time.Time
}

func (d *DecayChecker) Run(ctx context.Context, interval time.Duration) {
	slog.Info("interest decay checker started", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.RunOnce(ctx); err != nil {
				slog.Warn("interest decay check failed", "error", err)
			}
		}
	}
}

func (d *DecayChecker) RunOnce(ctx context.Context) error {
	if d.agentID == "" {
		slog.Warn("decay checker skipped: FEEDBACK_AGENT_ID not set")
		return nil
	}
	rows, err := d.db.QueryContext(ctx, `
		SELECT ui.id, ui.user_id, ui.category, ui.topic, ui.times_asked, ui.last_asked_at
		FROM user_interests ui
		WHERE ui.source = 'user'
		  AND ui.dismissed = FALSE
		  AND (ui.paused_until IS NULL OR ui.paused_until < NOW())
		  AND ui.created_at < NOW() - INTERVAL '30 days'
		  AND ui.times_asked < $1
		  AND (ui.last_asked_at IS NULL OR ui.last_asked_at < NOW() - INTERVAL '90 days')
		  AND NOT EXISTS (
			SELECT 1 FROM post_events pe
			JOIN posts p ON p.id = pe.post_id
			WHERE pe.user_id = ui.user_id
			  AND pe.event_type IN ('save', 'dwell')
			  AND pe.created_at > NOW() - INTERVAL '30 days'
			  AND p.labels::jsonb ? ui.category
		  )`, maxAsks)
	if err != nil {
		return fmt.Errorf("decay query: %w", err)
	}
	defer rows.Close()

	var candidates []decayCandidate
	for rows.Next() {
		var c decayCandidate
		if err := rows.Scan(&c.InterestID, &c.UserID, &c.Category, &c.Topic, &c.TimesAsked, &c.LastAsked); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("decay iterate: %w", err)
	}

	for _, c := range candidates {
		if err := d.generateFeedbackPost(ctx, c); err != nil {
			slog.Warn("decay: failed to generate feedback post",
				"user_id", c.UserID, "interest", c.Category, "error", err)
			continue
		}
		if err := d.interestRepo.MarkAsked(c.InterestID); err != nil {
			slog.Warn("decay: failed to mark asked", "interest_id", c.InterestID, "error", err)
		}
	}

	slog.Info("interest decay check complete", "candidates_checked", len(candidates))
	return nil
}

func (d *DecayChecker) generateFeedbackPost(ctx context.Context, c decayCandidate) error {
	feedbackData := map[string]interface{}{
		"feedback_type": "interest_check",
		"interest_id":   c.InterestID,
		"question":      fmt.Sprintf("You haven't been engaging with %s posts recently. What would you like to do?", c.Topic),
		"options": []map[string]string{
			{"key": "still_interested", "label": "Still interested"},
			{"key": "pause", "label": "Pause for a while"},
			{"key": "less", "label": "Less of this"},
			{"key": "remove", "label": "Remove it"},
		},
	}

	externalURL, _ := json.Marshal(feedbackData)

	_, err := d.postRepo.Create(repository.CreatePostParams{
		AgentID:     d.agentID,
		UserID:      c.UserID,
		Title:       fmt.Sprintf("Still interested in %s?", c.Topic),
		Body:        fmt.Sprintf("We noticed you haven't been engaging with %s content lately. Let us know how you'd like to adjust.", c.Topic),
		ExternalURL: string(externalURL),
		PostType:    "discovery",
		Visibility:  "personal",
		DisplayHint: "feedback",
		Labels:      []string{c.Category, "feedback"},
	})
	return err
}
