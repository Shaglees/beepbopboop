package repository

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// ErrUserSkillNotFound is returned when a (user, skill_name) pair has no row.
var ErrUserSkillNotFound = errors.New("user skill not found")

// UserSkillRepo persists user-authored skills and extension preferences.
// See docs/user-skills-protocol.md for the contract.
type UserSkillRepo struct {
	db *sql.DB
}

func NewUserSkillRepo(db *sql.DB) *UserSkillRepo {
	return &UserSkillRepo{db: db}
}

// FileInput is one file the caller wants to write as part of a skill.
// SHA256 + size are computed on insert if not provided.
type FileInput struct {
	Path    string
	Content []byte
}

// Upsert creates or replaces a user's skill atomically. On conflict by
// (user_id, skill_name), the skill row is updated, version is bumped, and
// all files are replaced.
func (r *UserSkillRepo) Upsert(
	userID, skillName, kind, extends, intent string,
	hints json.RawMessage,
	files []FileInput,
) (*model.UserSkill, error) {
	if userID == "" || skillName == "" {
		return nil, errors.New("user_id and skill_name are required")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var extendsArg any
	if extends != "" {
		extendsArg = extends
	}
	if hints == nil {
		hints = json.RawMessage("null")
	}

	var skillID int64
	var version int
	err = tx.QueryRow(`
		INSERT INTO user_skills (user_id, skill_name, version, kind, extends, intent, hints, status, updated_at)
		VALUES ($1, $2, 1, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, skill_name) DO UPDATE SET
			version    = user_skills.version + 1,
			kind       = excluded.kind,
			extends    = excluded.extends,
			intent     = excluded.intent,
			hints      = excluded.hints,
			status     = excluded.status,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, version`,
		userID, skillName, kind, extendsArg, intent, hints, model.UserSkillStatusReady,
	).Scan(&skillID, &version)
	if err != nil {
		return nil, fmt.Errorf("upsert user_skill: %w", err)
	}

	// Replace all files for this skill.
	if _, err := tx.Exec(`DELETE FROM user_skill_files WHERE skill_id = $1`, skillID); err != nil {
		return nil, fmt.Errorf("clear files: %w", err)
	}
	for _, f := range files {
		sum := sha256.Sum256(f.Content)
		if _, err := tx.Exec(`
			INSERT INTO user_skill_files (skill_id, path, sha256, size_bytes, content)
			VALUES ($1, $2, $3, $4, $5)`,
			skillID, f.Path, hex.EncodeToString(sum[:]), len(f.Content), f.Content,
		); err != nil {
			return nil, fmt.Errorf("insert file %q: %w", f.Path, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return r.GetByName(userID, skillName)
}

// GetByName returns the metadata row for one user skill, or
// ErrUserSkillNotFound when missing.
func (r *UserSkillRepo) GetByName(userID, skillName string) (*model.UserSkill, error) {
	var s model.UserSkill
	var extends sql.NullString
	var hints sql.NullString
	err := r.db.QueryRow(`
		SELECT id, user_id, skill_name, version, kind, extends, intent, hints, status, created_at, updated_at
		FROM user_skills
		WHERE user_id = $1 AND skill_name = $2`,
		userID, skillName,
	).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Version, &s.Kind,
		&extends, &s.Intent, &hints, &s.Status,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserSkillNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user_skill: %w", err)
	}
	if extends.Valid {
		v := extends.String
		s.Extends = &v
	}
	if hints.Valid && hints.String != "" {
		s.Hints = json.RawMessage(hints.String)
	}
	return &s, nil
}

// Manifest returns every ready skill owned by a user, with file-level metadata
// (no body). Skills with non-ready status are omitted, per spec.
func (r *UserSkillRepo) Manifest(userID string) ([]model.UserSkillManifestEntry, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.skill_name, s.version, s.kind, s.extends
		FROM user_skills s
		WHERE s.user_id = $1 AND s.status = $2
		ORDER BY s.skill_name`,
		userID, model.UserSkillStatusReady,
	)
	if err != nil {
		return nil, fmt.Errorf("query manifest: %w", err)
	}
	defer rows.Close()

	type entryRow struct {
		id      int64
		entry   model.UserSkillManifestEntry
		extends sql.NullString
	}
	var skills []entryRow
	for rows.Next() {
		var er entryRow
		if err := rows.Scan(&er.id, &er.entry.Name, &er.entry.Version, &er.entry.Kind, &er.extends); err != nil {
			return nil, fmt.Errorf("scan manifest row: %w", err)
		}
		if er.extends.Valid {
			v := er.extends.String
			er.entry.Extends = &v
		}
		skills = append(skills, er)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manifest: %w", err)
	}

	out := make([]model.UserSkillManifestEntry, 0, len(skills))
	for _, er := range skills {
		fileRows, err := r.db.Query(`
			SELECT path, sha256, size_bytes
			FROM user_skill_files
			WHERE skill_id = $1
			ORDER BY path`,
			er.id,
		)
		if err != nil {
			return nil, fmt.Errorf("query files for skill %d: %w", er.id, err)
		}
		var files []model.UserSkillManifestFile
		for fileRows.Next() {
			var f model.UserSkillManifestFile
			if err := fileRows.Scan(&f.Path, &f.SHA256, &f.Size); err != nil {
				fileRows.Close()
				return nil, fmt.Errorf("scan file row: %w", err)
			}
			files = append(files, f)
		}
		fileRows.Close()
		er.entry.Files = files
		out = append(out, er.entry)
	}
	return out, nil
}

// GetFile returns a single file by (user_id, skill_name, path), with content.
// Returns ErrUserSkillNotFound when the skill or file does not exist.
func (r *UserSkillRepo) GetFile(userID, skillName, path string) (*model.UserSkillFile, error) {
	var f model.UserSkillFile
	err := r.db.QueryRow(`
		SELECT f.path, f.sha256, f.size_bytes, f.content, f.updated_at
		FROM user_skill_files f
		JOIN user_skills s ON s.id = f.skill_id
		WHERE s.user_id = $1 AND s.skill_name = $2 AND f.path = $3`,
		userID, skillName, path,
	).Scan(&f.Path, &f.SHA256, &f.Size, &f.Content, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrUserSkillNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query file: %w", err)
	}
	return &f, nil
}

// CountByUser returns how many skills a user owns. Used for per-user caps.
func (r *UserSkillRepo) CountByUser(userID string) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM user_skills WHERE user_id = $1`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count user skills: %w", err)
	}
	return n, nil
}
