package model

import (
	"encoding/json"
	"time"
)

// UserSkill represents a niche skill authored from the iOS app, or an
// extension preferences file layered on top of a shipped skill. See
// docs/user-skills-protocol.md for the full contract.
type UserSkill struct {
	ID                int64           `json:"-"`
	UserID            string          `json:"-"`
	Name              string          `json:"name"`
	Version           int             `json:"version"`
	Kind              string          `json:"kind"`
	Extends           *string         `json:"extends,omitempty"`
	Intent            string          `json:"intent,omitempty"`
	FrequencyPerMonth int             `json:"frequency_per_month,omitempty"`
	Hints             json.RawMessage `json:"hints,omitempty"`
	Status            string          `json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// FrequencyMin / FrequencyMax bound posts_per_month from the slider in the
// iOS skill-builder ("every day" -> 30, "every month" -> 1).
const (
	FrequencyMin     = 1
	FrequencyMax     = 30
	FrequencyDefault = 7 // weekly
)

// UserSkillKind values.
const (
	UserSkillKindStandalone = "standalone"
	UserSkillKindExtension  = "extension"
)

// UserSkillStatus values.
const (
	UserSkillStatusQueued   = "queued"
	UserSkillStatusBuilding = "building"
	UserSkillStatusReady    = "ready"
	UserSkillStatusFailed   = "failed"
)

// UserSkillFile is one file inside a user skill (e.g. SKILL.md, MODE_brief.md).
type UserSkillFile struct {
	Path      string    `json:"path"`
	SHA256    string    `json:"sha256"`
	Size      int       `json:"size"`
	Content   []byte    `json:"-"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserSkillRequest is the iOS-facing request body for POST /skills/user.
//
// FrequencyPerMonth is set by the iOS skill-builder slider (1 = "every month",
// 30 = "every day"). Missing / zero values default to FrequencyDefault. The
// backend uses it to allocate a slice of the user's spread on standalone
// skills; extensions ignore it.
type CreateUserSkillRequest struct {
	Intent            string          `json:"intent"`
	Kind              string          `json:"kind,omitempty"`
	Extends           string          `json:"extends,omitempty"`
	FrequencyPerMonth int             `json:"frequency_per_month,omitempty"`
	Hints             json.RawMessage `json:"hints,omitempty"`
}

// CreateUserSkillResponse is returned from POST /skills/user.
type CreateUserSkillResponse struct {
	SkillName   string    `json:"skill_name"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// UserSkillManifestEntry is one skill in the openclaw-facing manifest.
type UserSkillManifestEntry struct {
	Name    string                  `json:"name"`
	Version int                     `json:"version"`
	Kind    string                  `json:"kind"`
	Extends *string                 `json:"extends,omitempty"`
	Files   []UserSkillManifestFile `json:"files"`
}

// UserSkillManifestFile is one file's metadata in the manifest. Body is
// fetched separately via GET /skills/user/files/{name}/{path}.
type UserSkillManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

// UserSkillManifest is the response body for GET /skills/user/manifest.
type UserSkillManifest struct {
	UserID string                   `json:"user_id"`
	Skills []UserSkillManifestEntry `json:"skills"`
}
