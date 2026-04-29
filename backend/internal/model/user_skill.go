package model

import (
	"encoding/json"
	"time"
)

// UserSkill represents a niche skill authored from the iOS app, or an
// extension preferences file layered on top of a shipped skill. See
// docs/user-skills-protocol.md for the full contract.
//
// Cadence for a user-skill (how often it should produce posts) lives in the
// existing spread system — user_settings.spread_targets keyed by skill name.
// It is intentionally not duplicated on this row.
type UserSkill struct {
	ID        int64           `json:"-"`
	UserID    string          `json:"-"`
	Name      string          `json:"name"`
	Version   int             `json:"version"`
	Kind      string          `json:"kind"`
	Extends   *string         `json:"extends,omitempty"`
	Intent    string          `json:"intent,omitempty"`
	Hints     json.RawMessage `json:"hints,omitempty"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

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
// Weight is the cadence for a standalone skill — same wire shape as
// PUT /settings/spread targets values. The iOS skill-builder slider produces
// it directly ("every day" maps to a high weight, "every month" to a low
// weight; iOS owns the mapping). Backend writes it into the spread for
// standalone kinds and renormalizes other verticals so weights still sum to
// 1.0. Extensions ignore the field.
//
// Zero / missing means "leave the spread alone" — the user can manage it
// later via the existing PUT /settings/spread endpoint.
type CreateUserSkillRequest struct {
	Intent  string          `json:"intent"`
	Kind    string          `json:"kind,omitempty"`
	Extends string          `json:"extends,omitempty"`
	Weight  float64         `json:"weight,omitempty"`
	Hints   json.RawMessage `json:"hints,omitempty"`
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
