package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type WeightsRepo struct {
	db *sql.DB
}

func NewWeightsRepo(db *sql.DB) *WeightsRepo {
	return &WeightsRepo{db: db}
}

// Get returns the user's weights, or nil when none exist.
func (r *WeightsRepo) Get(userID string) (*model.UserWeights, error) {
	var w model.UserWeights
	var weightsRaw []byte

	err := r.db.QueryRow(`
		SELECT user_id, weights, updated_at
		FROM user_weights WHERE user_id = $1`, userID,
	).Scan(&w.UserID, &weightsRaw, &w.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user_weights: %w", err)
	}
	w.Weights = json.RawMessage(weightsRaw)
	return &w, nil
}

// ComputeFromEngagement derives feed weights from a user's engagement signals.
// Saves are the strongest signal, clicks are medium, views are weak.
// Weights are normalized to [0, 1.0] range. Returns nil if no engagement data.
func ComputeFromEngagement(summary *model.EventSummary, defaults *FeedWeights) *FeedWeights {
	if summary == nil || summary.TotalEvents == 0 {
		return nil
	}

	fw := &FeedWeights{
		FreshnessBias: defaults.FreshnessBias,
		GeoBias:       defaults.GeoBias,
		LabelWeights:  make(map[string]float64),
		TypeWeights:   make(map[string]float64),
	}

	// Compute label weights from engagement.
	// Score = saves*5 + clicks*2 + views*0.3 + dwell_bonus
	// Then normalize: top label = 0.8, scale rest relative to it.
	var maxLabelScore float64
	labelScores := make(map[string]float64)
	for _, le := range summary.LabelEngagement {
		dwellBonus := 0.0
		if le.AvgDwell > 5000 { // > 5 seconds average dwell
			dwellBonus = math.Min(le.AvgDwell/10000, 1.0) // caps at 1.0 for 10s+
		}
		score := float64(le.Saves)*5.0 + float64(le.Clicks)*2.0 + float64(le.Views)*0.3 + dwellBonus
		labelScores[le.Label] = score
		if score > maxLabelScore {
			maxLabelScore = score
		}
	}
	if maxLabelScore > 0 {
		for label, score := range labelScores {
			fw.LabelWeights[label] = (score / maxLabelScore) * 0.8
		}
	}

	// Compute type weights from engagement (same formula).
	var maxTypeScore float64
	typeScores := make(map[string]float64)
	for _, te := range summary.TypeEngagement {
		dwellBonus := 0.0
		if te.AvgDwell > 5000 {
			dwellBonus = math.Min(te.AvgDwell/10000, 1.0)
		}
		score := float64(te.Saves)*5.0 + float64(te.Clicks)*2.0 + float64(te.Views)*0.3 + dwellBonus
		typeScores[te.PostType] = score
		if score > maxTypeScore {
			maxTypeScore = score
		}
	}
	if maxTypeScore > 0 {
		for pt, score := range typeScores {
			fw.TypeWeights[pt] = (score / maxTypeScore) * 0.6
		}
	}

	// Merge: keep default weights for labels/types the user hasn't interacted with yet,
	// but at a lower base so engaged content wins.
	for label, dw := range defaults.LabelWeights {
		if _, exists := fw.LabelWeights[label]; !exists {
			fw.LabelWeights[label] = dw * 0.5 // half the default for unseen labels
		}
	}
	for pt, dw := range defaults.TypeWeights {
		if _, exists := fw.TypeWeights[pt]; !exists {
			fw.TypeWeights[pt] = dw * 0.5
		}
	}

	return fw
}

// GetOrCompute returns the user's weights if fresh (< 1 hour), otherwise
// recomputes from engagement data and persists the result.
func (r *WeightsRepo) GetOrCompute(userID string, eventRepo *EventRepo, defaults *FeedWeights) (*FeedWeights, error) {
	uw, err := r.Get(userID)
	if err != nil {
		return nil, err
	}

	// If weights exist and are less than 1 hour old, use them.
	if uw != nil && time.Since(uw.UpdatedAt) < time.Hour {
		var fw FeedWeights
		if err := json.Unmarshal(uw.Weights, &fw); err == nil {
			return &fw, nil
		}
		// Fall through if parse fails.
	}

	// Recompute from engagement data (last 14 days).
	summary, err := eventRepo.Summary(userID, 14)
	if err != nil {
		return nil, fmt.Errorf("compute weights: %w", err)
	}

	computed := ComputeFromEngagement(summary, defaults)
	if computed == nil {
		return defaults, nil // No engagement yet, use defaults.
	}

	// Persist for next time.
	raw, err := json.Marshal(computed)
	if err != nil {
		return computed, nil // Use computed even if persist fails.
	}
	r.Upsert(userID, raw)

	return computed, nil
}

// Upsert inserts or updates the user's weights.
func (r *WeightsRepo) Upsert(userID string, weights json.RawMessage) (*model.UserWeights, error) {
	_, err := r.db.Exec(`
		INSERT INTO user_weights (user_id, weights, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			weights = excluded.weights,
			updated_at = CURRENT_TIMESTAMP`,
		userID, weights,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user_weights: %w", err)
	}
	return r.Get(userID)
}
