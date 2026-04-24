package ab

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// GuardrailConfig holds thresholds for automatic experiment pause.
// Drop percentages are absolute (e.g. SaveRateDropPct=5 means pause if
// treatment save rate is more than 5 percentage points below control).
type GuardrailConfig struct {
	SaveRateDropPct float64
	SessionDropPct  float64
}

// Guardrail checks per-variant engagement metrics and pauses an experiment
// if treatment regresses beyond the configured thresholds.
type Guardrail struct {
	db  *sql.DB
	cfg GuardrailConfig
}

func NewGuardrail(db *sql.DB, cfg GuardrailConfig) *Guardrail {
	return &Guardrail{db: db, cfg: cfg}
}

type variantMetrics struct {
	impressions int
	saves       int
}

// CheckAndPause evaluates the last 7 days of events for the experiment.
// Returns (true, nil) if treatment was paused, (false, nil) if metrics are
// ok or there is insufficient data.
func (g *Guardrail) CheckAndPause(ctx context.Context, experiment string) (bool, error) {
	rows, err := g.db.QueryContext(ctx, `
		SELECT
			pe.ab_variant,
			COUNT(*) FILTER (WHERE pe.event_type = 'view') AS impressions,
			COUNT(*) FILTER (WHERE pe.event_type = 'save') AS saves
		FROM post_events pe
		WHERE pe.ab_variant IN ('control', 'treatment')
		  AND pe.created_at > NOW() - INTERVAL '7 days'
		  AND EXISTS (
			SELECT 1 FROM ab_assignments aa
			WHERE aa.user_id = pe.user_id
			  AND aa.experiment = $1
			  AND aa.variant = pe.ab_variant
		  )
		GROUP BY pe.ab_variant`,
		experiment)
	if err != nil {
		return false, fmt.Errorf("guardrail query: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]variantMetrics)
	for rows.Next() {
		var variant string
		var m variantMetrics
		if err := rows.Scan(&variant, &m.impressions, &m.saves); err != nil {
			return false, fmt.Errorf("guardrail scan: %w", err)
		}
		metrics[variant] = m
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("guardrail iterate: %w", err)
	}

	ctrl, hasCtrl := metrics["control"]
	tmt, hasTmt := metrics["treatment"]
	if !hasCtrl || !hasTmt || ctrl.impressions == 0 || tmt.impressions == 0 {
		return false, nil
	}

	controlSaveRate := float64(ctrl.saves) / float64(ctrl.impressions) * 100
	treatmentSaveRate := float64(tmt.saves) / float64(tmt.impressions) * 100
	drop := controlSaveRate - treatmentSaveRate

	slog.Info("guardrail check",
		"experiment", experiment,
		"control_save_rate", controlSaveRate,
		"treatment_save_rate", treatmentSaveRate,
		"drop_pp", drop,
		"threshold_pp", g.cfg.SaveRateDropPct,
	)

	if drop <= g.cfg.SaveRateDropPct {
		return false, nil
	}

	_, err = g.db.ExecContext(ctx, `
		INSERT INTO ab_experiments (name, status, paused_at)
		VALUES ($1, 'paused', NOW())
		ON CONFLICT (name) DO UPDATE SET status = 'paused', paused_at = NOW()`,
		experiment)
	if err != nil {
		return false, fmt.Errorf("pause experiment: %w", err)
	}

	slog.Warn("guardrail: paused experiment due to save rate regression",
		"experiment", experiment, "drop_pp", drop)
	return true, nil
}
