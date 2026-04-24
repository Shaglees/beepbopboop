package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type ExperimentRepo struct {
	db *sql.DB
}

func NewExperimentRepo(db *sql.DB) *ExperimentRepo {
	return &ExperimentRepo{db: db}
}

// Get returns the experiment definition or nil if not found.
func (r *ExperimentRepo) Get(ctx context.Context, name string) (*model.Experiment, error) {
	var exp model.Experiment
	err := r.db.QueryRowContext(ctx,
		"SELECT name, treatment_pct, status FROM ab_experiments WHERE name=$1", name,
	).Scan(&exp.Name, &exp.TreatmentPct, &exp.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
	}
	return &exp, nil
}

// Upsert creates or updates an experiment. On conflict, only treatment_pct is
// updated — status and paused_at are preserved.
func (r *ExperimentRepo) Upsert(ctx context.Context, name string, treatmentPct int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ab_experiments (name, treatment_pct, status)
		VALUES ($1, $2, 'running')
		ON CONFLICT (name) DO UPDATE SET treatment_pct = $2`,
		name, treatmentPct)
	if err != nil {
		return fmt.Errorf("upsert experiment: %w", err)
	}
	return nil
}

// VariantResult holds engagement metrics for one variant.
type VariantResult struct {
	Variant    string  `json:"variant"`
	Users      int     `json:"users"`
	Impressions int    `json:"impressions"`
	Saves      int     `json:"saves"`
	Clicks     int     `json:"clicks"`
	AvgDwellMs float64 `json:"avg_dwell_ms"`
	SaveRate   float64 `json:"save_rate"`
}

// Results returns per-variant engagement stats for the last 7 days.
func (r *ExperimentRepo) Results(ctx context.Context, experiment string) ([]VariantResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			pe.ab_variant,
			COUNT(DISTINCT pe.user_id)                                           AS users,
			COUNT(*) FILTER (WHERE pe.event_type = 'view')                      AS impressions,
			COUNT(*) FILTER (WHERE pe.event_type = 'save')                      AS saves,
			COUNT(*) FILTER (WHERE pe.event_type = 'click')                     AS clicks,
			COALESCE(AVG(pe.dwell_ms) FILTER (WHERE pe.event_type = 'view'), 0) AS avg_dwell_ms,
			COALESCE(
				COUNT(*) FILTER (WHERE pe.event_type = 'save')::float /
				NULLIF(COUNT(*) FILTER (WHERE pe.event_type = 'view'), 0),
				0
			) AS save_rate
		FROM post_events pe
		JOIN ab_assignments aa
			ON aa.user_id = pe.user_id
			AND aa.experiment = $1
			AND aa.variant = pe.ab_variant
		WHERE pe.created_at > NOW() - INTERVAL '7 days'
		GROUP BY pe.ab_variant
		ORDER BY pe.ab_variant`,
		experiment)
	if err != nil {
		return nil, fmt.Errorf("experiment results query: %w", err)
	}
	defer rows.Close()

	var results []VariantResult
	for rows.Next() {
		var vr VariantResult
		if err := rows.Scan(&vr.Variant, &vr.Users, &vr.Impressions, &vr.Saves, &vr.Clicks, &vr.AvgDwellMs, &vr.SaveRate); err != nil {
			return nil, fmt.Errorf("scan variant result: %w", err)
		}
		results = append(results, vr)
	}
	return results, rows.Err()
}
