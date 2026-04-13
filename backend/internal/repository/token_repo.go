package repository

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrTokenInvalid = errors.New("token invalid or revoked")

type TokenRepo struct {
	db *sql.DB
}

func NewTokenRepo(db *sql.DB) *TokenRepo {
	return &TokenRepo{db: db}
}

func (r *TokenRepo) Create(agentID string) (string, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	rawToken := "bbp_" + hex.EncodeToString(rawBytes)

	hash := hashToken(rawToken)
	id, err := generateID()
	if err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}

	_, err = r.db.Exec(
		"INSERT INTO agent_tokens (id, agent_id, token_hash) VALUES ($1, $2, $3)",
		id, agentID, hash,
	)
	if err != nil {
		return "", fmt.Errorf("insert token: %w", err)
	}

	return rawToken, nil
}

func (r *TokenRepo) ValidateToken(rawToken string) (string, error) {
	hash := hashToken(rawToken)

	var agentID string
	err := r.db.QueryRow(
		"SELECT agent_id FROM agent_tokens WHERE token_hash = $1 AND revoked = FALSE",
		hash,
	).Scan(&agentID)

	if err == sql.ErrNoRows {
		return "", ErrTokenInvalid
	}
	if err != nil {
		return "", fmt.Errorf("query token: %w", err)
	}
	return agentID, nil
}

func (r *TokenRepo) Revoke(agentID string) error {
	_, err := r.db.Exec(
		"UPDATE agent_tokens SET revoked = TRUE WHERE agent_id = $1",
		agentID,
	)
	if err != nil {
		return fmt.Errorf("revoke tokens: %w", err)
	}
	return nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
