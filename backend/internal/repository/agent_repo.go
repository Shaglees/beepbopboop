package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type AgentRepo struct {
	db *sql.DB
}

func NewAgentRepo(db *sql.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

func (r *AgentRepo) Create(userID, name string) (*model.Agent, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	_, err = r.db.Exec(
		"INSERT INTO agents (id, user_id, name) VALUES ($1, $2, $3)",
		id, userID, name,
	)
	if err != nil {
		return nil, fmt.Errorf("insert agent: %w", err)
	}

	return r.GetByID(id)
}

func (r *AgentRepo) GetByID(id string) (*model.Agent, error) {
	var agent model.Agent
	err := r.db.QueryRow(
		"SELECT id, user_id, name, status, created_at FROM agents WHERE id = $1",
		id,
	).Scan(&agent.ID, &agent.UserID, &agent.Name, &agent.Status, &agent.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query agent: %w", err)
	}
	return &agent, nil
}

func (r *AgentRepo) ListByUserID(userID string) ([]model.Agent, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, name, status, created_at FROM agents WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	var agents []model.Agent
	for rows.Next() {
		var a model.Agent
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Status, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agents: %w", err)
	}
	return agents, nil
}
