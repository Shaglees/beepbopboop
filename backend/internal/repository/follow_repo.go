package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type FollowRepo struct {
	db *sql.DB
}

func NewFollowRepo(db *sql.DB) *FollowRepo {
	return &FollowRepo{db: db}
}

// Follow adds a follow relationship between a user and an agent.
// Idempotent: double-follow is a no-op.
// Returns the updated follower count.
func (r *FollowRepo) Follow(userID, agentID string) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO agent_follows (user_id, agent_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, agent_id) DO NOTHING`,
		userID, agentID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert follow: %w", err)
	}

	count, err := syncFollowerCount(tx, agentID)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

// Unfollow removes a follow relationship between a user and an agent.
// Idempotent: double-unfollow is a no-op.
// Returns the updated follower count.
func (r *FollowRepo) Unfollow(userID, agentID string) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM agent_follows WHERE user_id = $1 AND agent_id = $2`,
		userID, agentID,
	)
	if err != nil {
		return 0, fmt.Errorf("delete follow: %w", err)
	}

	count, err := syncFollowerCount(tx, agentID)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

// IsFollowing returns true if userID follows agentID.
func (r *FollowRepo) IsFollowing(userID, agentID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM agent_follows WHERE user_id = $1 AND agent_id = $2)`,
		userID, agentID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check follow: %w", err)
	}
	return exists, nil
}

// ListFollowedAgentIDs returns the set of agent IDs the user follows.
func (r *FollowRepo) ListFollowedAgentIDs(userID string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT agent_id FROM agent_follows WHERE user_id = $1 ORDER BY followed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query followed agents: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan agent id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListFollowedAgents returns full agent profiles for agents the user follows.
func (r *FollowRepo) ListFollowedAgents(userID string) ([]model.AgentProfile, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.user_id, a.name, a.status, a.follower_count,
		       COALESCE(a.description, ''), COALESCE(a.avatar_url, ''),
		       a.created_at,
		       COUNT(p.id) AS post_count
		FROM agent_follows af
		JOIN agents a ON a.id = af.agent_id
		LEFT JOIN posts p ON p.agent_id = a.id AND p.status = 'published'
		WHERE af.user_id = $1
		GROUP BY a.id
		ORDER BY af.followed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query followed agents: %w", err)
	}
	defer rows.Close()

	var agents []model.AgentProfile
	for rows.Next() {
		var ap model.AgentProfile
		if err := rows.Scan(&ap.ID, &ap.UserID, &ap.Name, &ap.Status,
			&ap.FollowerCount, &ap.Description, &ap.AvatarURL,
			&ap.CreatedAt, &ap.PostCount); err != nil {
			return nil, fmt.Errorf("scan agent profile: %w", err)
		}
		agents = append(agents, ap)
	}
	return agents, rows.Err()
}

// GetAgentProfile returns a full profile for a single agent.
func (r *FollowRepo) GetAgentProfile(agentID string) (*model.AgentProfile, error) {
	var ap model.AgentProfile
	err := r.db.QueryRow(`
		SELECT a.id, a.user_id, a.name, a.status, a.follower_count,
		       COALESCE(a.description, ''), COALESCE(a.avatar_url, ''),
		       a.created_at,
		       COUNT(p.id) AS post_count
		FROM agents a
		LEFT JOIN posts p ON p.agent_id = a.id AND p.status = 'published'
		WHERE a.id = $1
		GROUP BY a.id`,
		agentID,
	).Scan(&ap.ID, &ap.UserID, &ap.Name, &ap.Status,
		&ap.FollowerCount, &ap.Description, &ap.AvatarURL,
		&ap.CreatedAt, &ap.PostCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent profile: %w", err)
	}
	return &ap, nil
}

// FollowedAgentIDSet returns a set of agent IDs the user follows (for O(1) lookup).
func (r *FollowRepo) FollowedAgentIDSet(userID string) (map[string]bool, error) {
	ids, err := r.ListFollowedAgentIDs(userID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// syncFollowerCount reconciles the denormalized follower_count on the agents table.
func syncFollowerCount(tx *sql.Tx, agentID string) (int, error) {
	var count int
	err := tx.QueryRow(`
		UPDATE agents
		SET follower_count = (SELECT COUNT(*) FROM agent_follows WHERE agent_id = $1)
		WHERE id = $1
		RETURNING follower_count`,
		agentID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("sync follower_count: %w", err)
	}
	return count, nil
}

// AgentFollow represents a single follow relationship (for list-followers response).
type AgentFollow struct {
	UserID     string    `json:"user_id"`
	AgentID    string    `json:"agent_id"`
	FollowedAt time.Time `json:"followed_at"`
}
