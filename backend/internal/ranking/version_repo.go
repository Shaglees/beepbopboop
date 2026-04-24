package ranking

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// ModelVersionRepo persists trained model checkpoint metadata.
type ModelVersionRepo struct {
	db *sql.DB
}

func NewModelVersionRepo(db *sql.DB) *ModelVersionRepo {
	return &ModelVersionRepo{db: db}
}

// Insert records a newly trained checkpoint. Returns the auto-assigned ID.
func (r *ModelVersionRepo) Insert(ctx context.Context, version, modelPath string, aucROC float64) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO model_versions (version, model_path, auc_roc, status)
		VALUES ($1, $2, $3, 'trained')
		RETURNING id`,
		version, modelPath, aucROC,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert model version: %w", err)
	}
	return id, nil
}

// Get returns the model version with the given ID.
func (r *ModelVersionRepo) Get(ctx context.Context, id int64) (*model.ModelVersion, error) {
	var mv model.ModelVersion
	var deployedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, version, model_path, auc_roc, status, trained_at, deployed_at
		FROM model_versions WHERE id = $1`, id,
	).Scan(&mv.ID, &mv.Version, &mv.ModelPath, &mv.AUCROC, &mv.Status, &mv.TrainedAt, &deployedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model version: %w", err)
	}
	if deployedAt.Valid {
		mv.DeployedAt = &deployedAt.Time
	}
	return &mv, nil
}

// GetActive returns the currently deployed version, or nil if none exists.
func (r *ModelVersionRepo) GetActive(ctx context.Context) (*model.ModelVersion, error) {
	var mv model.ModelVersion
	var deployedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, version, model_path, auc_roc, status, trained_at, deployed_at
		FROM model_versions WHERE status = 'deployed'
		ORDER BY deployed_at DESC LIMIT 1`,
	).Scan(&mv.ID, &mv.Version, &mv.ModelPath, &mv.AUCROC, &mv.Status, &mv.TrainedAt, &deployedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active model version: %w", err)
	}
	if deployedAt.Valid {
		mv.DeployedAt = &deployedAt.Time
	}
	return &mv, nil
}

// MarkDeployed sets status='deployed' and deployed_at=NOW() for the given ID,
// and retires any previously deployed version.
// Returns an error if the ID does not exist (no rows updated).
func (r *ModelVersionRepo) MarkDeployed(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"UPDATE model_versions SET status = 'retired' WHERE status = 'deployed'",
	); err != nil {
		return fmt.Errorf("retire current model: %w", err)
	}

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx, `
		UPDATE model_versions SET status = 'deployed', deployed_at = $1 WHERE id = $2`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("mark deployed: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("mark deployed: model version %d not found", id)
	}
	return tx.Commit()
}

// MarkDeployedWithGate atomically checks the AUC improvement gate and deploys
// the candidate if it passes. The gate check and deployment happen inside a
// single transaction with FOR UPDATE locking on the deployed row to prevent
// concurrent deployments from racing on a stale baseline.
// Returns an error if the gate blocks or if the ID does not exist.
func (r *ModelVersionRepo) MarkDeployedWithGate(ctx context.Context, candidateID int64, minImprovement float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Lock the currently deployed row (if any) to serialise concurrent deploys.
	var currentAUC float64
	err = tx.QueryRowContext(ctx, `
		SELECT auc_roc FROM model_versions WHERE status = 'deployed'
		ORDER BY deployed_at DESC LIMIT 1
		FOR UPDATE`).Scan(&currentAUC)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("lock active version: %w", err)
	}
	// err == sql.ErrNoRows means no current model — any positive AUC passes.

	// Read candidate AUC.
	var candidateAUC float64
	if err2 := tx.QueryRowContext(ctx,
		"SELECT auc_roc FROM model_versions WHERE id = $1", candidateID,
	).Scan(&candidateAUC); err2 == sql.ErrNoRows {
		return fmt.Errorf("mark deployed with gate: model version %d not found", candidateID)
	} else if err2 != nil {
		return fmt.Errorf("read candidate auc: %w", err2)
	}

	gate := NewDeploymentGate(minImprovement)
	if !gate.ShouldDeploy(currentAUC, candidateAUC) {
		return fmt.Errorf("deployment blocked: AUC %.4f does not improve over current %.4f by %.0f%%",
			candidateAUC, currentAUC, minImprovement*100)
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE model_versions SET status = 'retired' WHERE status = 'deployed'",
	); err != nil {
		return fmt.Errorf("retire current model: %w", err)
	}

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx, `
		UPDATE model_versions SET status = 'deployed', deployed_at = $1 WHERE id = $2`,
		now, candidateID,
	)
	if err != nil {
		return fmt.Errorf("mark deployed: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("mark deployed: model version %d not found", candidateID)
	}
	return tx.Commit()
}

// MarkRetired sets status='retired' for the given ID.
func (r *ModelVersionRepo) MarkRetired(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE model_versions SET status = 'retired' WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("mark retired: %w", err)
	}
	return nil
}

// List returns all model versions ordered newest-first.
func (r *ModelVersionRepo) List(ctx context.Context) ([]model.ModelVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, version, model_path, auc_roc, status, trained_at, deployed_at
		FROM model_versions
		ORDER BY trained_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list model versions: %w", err)
	}
	defer rows.Close()

	var versions []model.ModelVersion
	for rows.Next() {
		var mv model.ModelVersion
		var deployedAt sql.NullTime
		if err := rows.Scan(&mv.ID, &mv.Version, &mv.ModelPath, &mv.AUCROC, &mv.Status, &mv.TrainedAt, &deployedAt); err != nil {
			return nil, fmt.Errorf("scan model version: %w", err)
		}
		if deployedAt.Valid {
			mv.DeployedAt = &deployedAt.Time
		}
		versions = append(versions, mv)
	}
	return versions, rows.Err()
}

// ReadyToRetrain returns true when the number of engagement post_events (save,
// click, share) recorded after the active model's trained_at exceeds minNewPairs.
// View and impression events are excluded — they are passive signals that don't
// contribute to training pair quality. Returns false (not an error) when no
// active model exists.
func (r *ModelVersionRepo) ReadyToRetrain(ctx context.Context, minNewPairs int) (bool, error) {
	active, err := r.GetActive(ctx)
	if err != nil {
		return false, err
	}
	if active == nil {
		return false, nil
	}

	var count int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM post_events
		WHERE created_at > $1
		  AND event_type IN ('save', 'click', 'share')`, active.TrainedAt,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("count new pairs: %w", err)
	}
	return count >= minNewPairs, nil
}
