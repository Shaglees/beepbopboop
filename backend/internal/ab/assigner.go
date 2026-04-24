package ab

import (
	"context"
	"database/sql"
	"hash/fnv"
	"log/slog"
)

// Assigner deterministically assigns users to experiment variants using a
// hash of (userID + experiment name), then persists the assignment for analytics.
type Assigner struct {
	db *sql.DB
}

func NewAssigner(db *sql.DB) *Assigner {
	return &Assigner{db: db}
}

// hashUserExperiment returns a stable uint32 in [0, 100) from user ID and
// experiment name. FNV-1a is fast and has good distribution.
func hashUserExperiment(userID, experiment string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(userID))
	h.Write([]byte{':'})
	h.Write([]byte(experiment))
	return h.Sum32() % 100
}

// Variant returns the stable variant ("control" or "treatment") for the given
// user+experiment pair and persists the assignment on first call.
// treatmentPct controls what fraction [0,100] lands in treatment.
func (a *Assigner) Variant(ctx context.Context, userID, experiment string, treatmentPct int) string {
	bucket := hashUserExperiment(userID, experiment)
	variant := "control"
	if int(bucket) < treatmentPct {
		variant = "treatment"
	}

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO ab_assignments (user_id, experiment, variant)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, experiment) DO NOTHING`,
		userID, experiment, variant)
	if err != nil {
		slog.Warn("ab: failed to persist assignment", "user_id", userID, "experiment", experiment, "error", err)
	}

	return variant
}
