# BeepBopBoop Go Backend Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Go REST API backend with SQLite persistence, agent token auth, post ingestion, and feed retrieval.

**Architecture:** Standard Go HTTP server using `net/http` with `chi` router, SQLite via `modernc.org/sqlite` (pure Go, no CGO), and structured JSON logging via `log/slog`. Simple layered structure: handlers → service → repository → SQLite.

**Tech Stack:** Go 1.22+, chi router, modernc.org/sqlite, slog, Firebase Admin SDK for Go, crypto/rand for token generation, embedded SQL migrations via `//go:embed`.

**Depends on:** Nothing (standalone, testable with curl)

**Ref docs:** `MVP_CHECKLIST.md` Sections 2, 3.2, 6.1 | `PRD.md` Section 10

---

## File Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                  # Entrypoint, wires dependencies, starts server
├── internal/
│   ├── config/
│   │   └── config.go                # Environment-based configuration
│   ├── database/
│   │   ├── database.go              # SQLite connection setup
│   │   └── migrations/
│   │       └── 001_initial.sql      # Schema: users, agents, agent_tokens, posts
│   ├── middleware/
│   │   ├── firebase_auth.go         # Firebase ID token verification middleware
│   │   └── agent_auth.go            # Agent token verification middleware
│   ├── model/
│   │   └── model.go                 # Domain structs: User, Agent, AgentToken, Post
│   ├── repository/
│   │   ├── user_repo.go             # User CRUD
│   │   ├── agent_repo.go            # Agent CRUD
│   │   ├── token_repo.go            # Agent token CRUD
│   │   └── post_repo.go             # Post CRUD
│   └── handler/
│       ├── health.go                # GET /health, GET /me
│       ├── agent.go                 # POST /agents, POST /agents/:id/tokens
│       ├── post.go                  # POST /posts (agent-auth)
│       └── feed.go                  # GET /feed (firebase-auth)
├── go.mod
├── go.sum
└── Makefile                         # dev commands: run, test, migrate
```

---

## Chunk 1: Project Bootstrap and Database Schema

### Task 1: Initialize Go module and dependencies

**Files:**
- Create: `backend/go.mod`
- Create: `backend/Makefile`

- [ ] **Step 1: Initialize Go module**

```bash
cd backend
go mod init github.com/shanegleeson/beepbopboop/backend
```

- [ ] **Step 2: Add dependencies**

```bash
cd backend
go get github.com/go-chi/chi/v5@latest
go get modernc.org/sqlite@latest
go get firebase.google.com/go/v4@latest
go get google.golang.org/api@latest
```

- [ ] **Step 3: Create Makefile**

Create `backend/Makefile`:

```makefile
.PHONY: run test migrate

run:
	go run ./cmd/server

test:
	go test ./... -v -count=1

migrate:
	go run ./cmd/server -migrate
```

- [ ] **Step 4: Commit**

```bash
git add backend/
git commit -m "feat(backend): initialize Go module with dependencies"
```

---

### Task 2: Environment-based configuration

**Files:**
- Create: `backend/internal/config/config.go`
- Test: `backend/internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/config/config_test.go`:

```go
package config_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("FIREBASE_PROJECT_ID", "")
	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "beepbopboop.db" {
		t.Errorf("expected default db path beepbopboop.db, got %s", cfg.DatabasePath)
	}
	if cfg.FirebaseProjectID != "" {
		t.Errorf("expected empty firebase project id, got %s", cfg.FirebaseProjectID)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_PATH", "/tmp/test.db")
	t.Setenv("FIREBASE_PROJECT_ID", "my-project")

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("expected db path /tmp/test.db, got %s", cfg.DatabasePath)
	}
	if cfg.FirebaseProjectID != "my-project" {
		t.Errorf("expected firebase project id my-project, got %s", cfg.FirebaseProjectID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/config/... -v
```

Expected: FAIL — package not found

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/config/config.go`:

```go
package config

import "os"

type Config struct {
	Port              string
	DatabasePath      string
	FirebaseProjectID string
}

func Load() Config {
	return Config{
		Port:              envOr("PORT", "8080"),
		DatabasePath:      envOr("DATABASE_PATH", "beepbopboop.db"),
		FirebaseProjectID: os.Getenv("FIREBASE_PROJECT_ID"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/config/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/config/
git commit -m "feat(backend): add environment-based configuration"
```

---

### Task 3: Domain models

**Files:**
- Create: `backend/internal/model/model.go`

- [ ] **Step 1: Create domain model structs**

Create `backend/internal/model/model.go`:

```go
package model

import "time"

type User struct {
	ID          string    `json:"id"`
	FirebaseUID string    `json:"firebase_uid"`
	CreatedAt   time.Time `json:"created_at"`
}

type Agent struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentToken struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	TokenHash string    `json:"-"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

type Post struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	AgentName   string    `json:"agent_name"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	ImageURL    string    `json:"image_url,omitempty"`
	ExternalURL string    `json:"external_url,omitempty"`
	Locality    string    `json:"locality,omitempty"`
	PostType    string    `json:"post_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/model/
git commit -m "feat(backend): add domain model structs"
```

---

### Task 4: Database setup and migrations

**Files:**
- Create: `backend/internal/database/database.go`
- Create: `backend/internal/database/migrations/001_initial.sql`
- Test: `backend/internal/database/database_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/database/database_test.go`:

```go
package database_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
)

func TestOpenAndMigrate(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Verify tables exist by querying them
	tables := []string{"users", "agents", "agent_tokens", "posts"}
	for _, table := range tables {
		_, err := db.Exec("SELECT count(*) FROM " + table)
		if err != nil {
			t.Errorf("table %s does not exist: %v", table, err)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/database/... -v
```

Expected: FAIL

- [ ] **Step 3: Create migration SQL**

Create `backend/internal/database/migrations/001_initial.sql`:

```sql
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    firebase_uid TEXT UNIQUE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_tokens (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    token_hash TEXT NOT NULL,
    revoked INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    image_url TEXT,
    external_url TEXT,
    locality TEXT,
    post_type TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_posts_user_id_created ON posts(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agents_user_id ON agents(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_tokens_agent_id ON agent_tokens(agent_id);
```

- [ ] **Step 4: Write database setup implementation**

Create `backend/internal/database/database.go`:

```go
package database

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial.sql
var migrationSQL string

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// Run migrations
	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd backend && go test ./internal/database/... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/database/
git commit -m "feat(backend): add SQLite database setup with schema migration"
```

---

## Chunk 2: Repository Layer

### Task 5: User repository

**Files:**
- Create: `backend/internal/repository/user_repo.go`
- Test: `backend/internal/repository/user_repo_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/user_repo_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserRepo_FindOrCreateByFirebaseUID(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewUserRepo(db)

	// First call creates user
	user1, err := repo.FindOrCreateByFirebaseUID("firebase-abc")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if user1.FirebaseUID != "firebase-abc" {
		t.Errorf("expected firebase_uid firebase-abc, got %s", user1.FirebaseUID)
	}
	if user1.ID == "" {
		t.Error("expected non-empty user ID")
	}

	// Second call returns same user
	user2, err := repo.FindOrCreateByFirebaseUID("firebase-abc")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if user1.ID != user2.ID {
		t.Errorf("expected same user ID, got %s and %s", user1.ID, user2.ID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/repository/... -v
```

Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/repository/user_repo.go`:

```go
package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindOrCreateByFirebaseUID(firebaseUID string) (*model.User, error) {
	var user model.User
	err := r.db.QueryRow(
		"SELECT id, firebase_uid, created_at FROM users WHERE firebase_uid = ?",
		firebaseUID,
	).Scan(&user.ID, &user.FirebaseUID, &user.CreatedAt)

	if err == sql.ErrNoRows {
		id, err := generateID()
		if err != nil {
			return nil, fmt.Errorf("generate id: %w", err)
		}
		_, err = r.db.Exec(
			"INSERT INTO users (id, firebase_uid) VALUES (?, ?)",
			id, firebaseUID,
		)
		if err != nil {
			return nil, fmt.Errorf("insert user: %w", err)
		}
		return r.FindOrCreateByFirebaseUID(firebaseUID)
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &user, nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/repository/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/
git commit -m "feat(backend): add user repository with find-or-create"
```

---

### Task 6: Agent repository

**Files:**
- Create: `backend/internal/repository/agent_repo.go`
- Test: `backend/internal/repository/agent_repo_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/agent_repo_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestAgentRepo_Create(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	if err != nil {
		t.Fatal(err)
	}

	agentRepo := repository.NewAgentRepo(db)
	agent, err := agentRepo.Create(user.ID, "My Agent")
	if err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	if agent.Name != "My Agent" {
		t.Errorf("expected name My Agent, got %s", agent.Name)
	}
	if agent.UserID != user.ID {
		t.Errorf("expected user_id %s, got %s", user.ID, agent.UserID)
	}
	if agent.Status != "active" {
		t.Errorf("expected status active, got %s", agent.Status)
	}
}

func TestAgentRepo_GetByID(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	created, _ := agentRepo.Create(user.ID, "My Agent")

	found, err := agentRepo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get agent failed: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, found.ID)
	}
}

func TestAgentRepo_ListByUserID(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agentRepo.Create(user.ID, "Agent 1")
	agentRepo.Create(user.ID, "Agent 2")

	agents, err := agentRepo.ListByUserID(user.ID)
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/repository/... -v -run TestAgent
```

Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/repository/agent_repo.go`:

```go
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
		"INSERT INTO agents (id, user_id, name) VALUES (?, ?, ?)",
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
		"SELECT id, user_id, name, status, created_at FROM agents WHERE id = ?",
		id,
	).Scan(&agent.ID, &agent.UserID, &agent.Name, &agent.Status, &agent.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query agent: %w", err)
	}
	return &agent, nil
}

func (r *AgentRepo) ListByUserID(userID string) ([]model.Agent, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, name, status, created_at FROM agents WHERE user_id = ?",
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/repository/... -v -run TestAgent
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/agent_repo.go backend/internal/repository/agent_repo_test.go
git commit -m "feat(backend): add agent repository"
```

---

### Task 7: Agent token repository

**Files:**
- Create: `backend/internal/repository/token_repo.go`
- Test: `backend/internal/repository/token_repo_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/token_repo_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestTokenRepo_CreateAndValidate(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	tokenRepo := repository.NewTokenRepo(db)

	// Create token - returns raw token only once
	rawToken, err := tokenRepo.Create(agent.ID)
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}
	if rawToken == "" {
		t.Error("expected non-empty raw token")
	}

	// Validate token returns agent ID
	agentID, err := tokenRepo.ValidateToken(rawToken)
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}
	if agentID != agent.ID {
		t.Errorf("expected agent_id %s, got %s", agent.ID, agentID)
	}
}

func TestTokenRepo_RevokedTokenFails(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	tokenRepo := repository.NewTokenRepo(db)
	rawToken, _ := tokenRepo.Create(agent.ID)

	// Revoke
	err = tokenRepo.Revoke(agent.ID)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	// Validate should fail
	_, err = tokenRepo.ValidateToken(rawToken)
	if err == nil {
		t.Error("expected error for revoked token, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/repository/... -v -run TestToken
```

Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/repository/token_repo.go`:

```go
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

// Create generates a new token for the agent. Returns the raw token (shown only once).
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
		"INSERT INTO agent_tokens (id, agent_id, token_hash) VALUES (?, ?, ?)",
		id, agentID, hash,
	)
	if err != nil {
		return "", fmt.Errorf("insert token: %w", err)
	}

	return rawToken, nil
}

// ValidateToken checks if a raw token is valid and not revoked. Returns the agent ID.
func (r *TokenRepo) ValidateToken(rawToken string) (string, error) {
	hash := hashToken(rawToken)

	var agentID string
	err := r.db.QueryRow(
		"SELECT agent_id FROM agent_tokens WHERE token_hash = ? AND revoked = 0",
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

// Revoke marks all tokens for an agent as revoked.
func (r *TokenRepo) Revoke(agentID string) error {
	_, err := r.db.Exec(
		"UPDATE agent_tokens SET revoked = 1 WHERE agent_id = ?",
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/repository/... -v -run TestToken
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/token_repo.go backend/internal/repository/token_repo_test.go
git commit -m "feat(backend): add agent token repository with hash-based validation"
```

---

### Task 8: Post repository

**Files:**
- Create: `backend/internal/repository/post_repo.go`
- Test: `backend/internal/repository/post_repo_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/post_repo_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestPostRepo_CreateAndListByUser(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	postRepo := repository.NewPostRepo(db)

	// Create a post
	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID,
		UserID:  user.ID,
		Title:   "Tennis courts 6 minutes away",
		Body:    "A park near your home has tennis courts.",
	})
	if err != nil {
		t.Fatalf("create post failed: %v", err)
	}
	if post.Title != "Tennis courts 6 minutes away" {
		t.Errorf("expected title, got %s", post.Title)
	}

	// List posts for user (newest first)
	posts, err := postRepo.ListByUserID(user.ID, 20)
	if err != nil {
		t.Fatalf("list posts failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].AgentName != "My Agent" {
		t.Errorf("expected agent name My Agent, got %s", posts[0].AgentName)
	}
}

func TestPostRepo_ListByUserID_NewestFirst(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	postRepo := repository.NewPostRepo(db)

	postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "First", Body: "body",
	})
	postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Second", Body: "body",
	})

	posts, _ := postRepo.ListByUserID(user.ID, 20)
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if posts[0].Title != "Second" {
		t.Errorf("expected newest first, got %s", posts[0].Title)
	}
}

func TestPostRepo_EmptyFeed(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	postRepo := repository.NewPostRepo(db)
	posts, err := postRepo.ListByUserID("nonexistent", 20)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if posts == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}

func TestPostRepo_OptionalFields(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	postRepo := repository.NewPostRepo(db)

	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID:     agent.ID,
		UserID:      user.ID,
		Title:       "Test",
		Body:        "Body",
		ImageURL:    "https://i.imgur.com/example.jpg",
		ExternalURL: "https://example.com",
		Locality:    "Dublin 2",
		PostType:    "discovery",
	})
	if err != nil {
		t.Fatal(err)
	}
	if post.ImageURL != "https://i.imgur.com/example.jpg" {
		t.Errorf("expected image url, got %s", post.ImageURL)
	}
	if post.Locality != "Dublin 2" {
		t.Errorf("expected locality Dublin 2, got %s", post.Locality)
	}

	// Verify via list too
	posts, _ := postRepo.ListByUserID(user.ID, 20)
	_ = model.Post{} // ensure model import
	if posts[0].ImageURL != "https://i.imgur.com/example.jpg" {
		t.Errorf("expected image url in feed, got %s", posts[0].ImageURL)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/repository/... -v -run TestPost
```

Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/repository/post_repo.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type CreatePostParams struct {
	AgentID     string
	UserID      string
	Title       string
	Body        string
	ImageURL    string
	ExternalURL string
	Locality    string
	PostType    string
}

type PostRepo struct {
	db *sql.DB
}

func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

func (r *PostRepo) Create(p CreatePostParams) (*model.Post, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, image_url, external_url, locality, post_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.AgentID, p.UserID, p.Title, p.Body,
		nullString(p.ImageURL), nullString(p.ExternalURL),
		nullString(p.Locality), nullString(p.PostType),
	)
	if err != nil {
		return nil, fmt.Errorf("insert post: %w", err)
	}

	return r.GetByID(id)
}

func (r *PostRepo) GetByID(id string) (*model.Post, error) {
	var post model.Post
	var imageURL, externalURL, locality, postType sql.NullString
	err := r.db.QueryRow(`
		SELECT p.id, p.agent_id, a.name, p.user_id, p.title, p.body,
		       p.image_url, p.external_url, p.locality, p.post_type, p.created_at
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.id = ?`, id,
	).Scan(&post.ID, &post.AgentID, &post.AgentName, &post.UserID,
		&post.Title, &post.Body,
		&imageURL, &externalURL, &locality, &postType, &post.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query post: %w", err)
	}
	post.ImageURL = imageURL.String
	post.ExternalURL = externalURL.String
	post.Locality = locality.String
	post.PostType = postType.String
	return &post, nil
}

// ListByUserID returns posts for a user, newest first, up to limit.
func (r *PostRepo) ListByUserID(userID string, limit int) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.agent_id, a.name, p.user_id, p.title, p.body,
		       p.image_url, p.external_url, p.locality, p.post_type, p.created_at
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.user_id = ?
		ORDER BY p.created_at DESC, p.rowid DESC
		LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	for rows.Next() {
		var p model.Post
		var imageURL, externalURL, locality, postType sql.NullString
		if err := rows.Scan(&p.ID, &p.AgentID, &p.AgentName, &p.UserID,
			&p.Title, &p.Body,
			&imageURL, &externalURL, &locality, &postType, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		p.ImageURL = imageURL.String
		p.ExternalURL = externalURL.String
		p.Locality = locality.String
		p.PostType = postType.String
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	return posts, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/repository/... -v -run TestPost
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/post_repo.go backend/internal/repository/post_repo_test.go
git commit -m "feat(backend): add post repository with feed query"
```

---

## Chunk 3: HTTP Handlers and Middleware

### Task 9: Agent token auth middleware

**Files:**
- Create: `backend/internal/middleware/agent_auth.go`
- Test: `backend/internal/middleware/agent_auth_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/middleware/agent_auth_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestAgentAuth_ValidToken(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("fb-123")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Test Agent")

	tokenRepo := repository.NewTokenRepo(db)
	rawToken, _ := tokenRepo.Create(agent.ID)

	handler := middleware.AgentAuth(tokenRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID := middleware.AgentIDFromContext(r.Context())
		if agentID != agent.ID {
			t.Errorf("expected agent ID %s, got %s", agent.ID, agentID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/posts", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAgentAuth_MissingToken(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	tokenRepo := repository.NewTokenRepo(db)

	handler := middleware.AgentAuth(tokenRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/posts", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAgentAuth_InvalidToken(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	tokenRepo := repository.NewTokenRepo(db)

	handler := middleware.AgentAuth(tokenRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/posts", nil)
	req.Header.Set("Authorization", "Bearer bbp_invalidtoken")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/middleware/... -v
```

Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/middleware/agent_auth.go`:

```go
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type contextKey string

const agentIDKey contextKey = "agent_id"

func AgentIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(agentIDKey).(string)
	return v
}

func AgentAuth(tokenRepo *repository.TokenRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
				return
			}

			rawToken := strings.TrimPrefix(auth, "Bearer ")
			agentID, err := tokenRepo.ValidateToken(rawToken)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or revoked token"})
				return
			}

			ctx := context.WithValue(r.Context(), agentIDKey, agentID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/middleware/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/middleware/
git commit -m "feat(backend): add agent token auth middleware"
```

---

### Task 10: Firebase auth middleware (stub for local dev)

**Files:**
- Create: `backend/internal/middleware/firebase_auth.go`
- Test: `backend/internal/middleware/firebase_auth_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/middleware/firebase_auth_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
)

func TestFirebaseAuth_DevMode_ValidHeader(t *testing.T) {
	handler := middleware.FirebaseAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := middleware.FirebaseUIDFromContext(r.Context())
		if uid != "test-user-123" {
			t.Errorf("expected uid test-user-123, got %s", uid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/feed", nil)
	req.Header.Set("Authorization", "Bearer test-user-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestFirebaseAuth_DevMode_MissingHeader(t *testing.T) {
	handler := middleware.FirebaseAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/feed", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/middleware/... -v -run TestFirebase
```

Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/middleware/firebase_auth.go`:

```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

const firebaseUIDKey contextKey = "firebase_uid"

func FirebaseUIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(firebaseUIDKey).(string)
	return v
}

// FirebaseAuth verifies Firebase ID tokens. If authClient is nil, runs in dev mode
// where the Bearer token value is used directly as the Firebase UID.
func FirebaseAuth(authClient *auth.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			var uid string
			if authClient == nil {
				// Dev mode: treat bearer value as Firebase UID directly
				uid = token
			} else {
				// Production: verify with Firebase
				decoded, err := authClient.VerifyIDToken(r.Context(), token)
				if err != nil {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid firebase token"})
					return
				}
				uid = decoded.UID
			}

			ctx := context.WithValue(r.Context(), firebaseUIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/middleware/... -v -run TestFirebase
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/middleware/firebase_auth.go backend/internal/middleware/firebase_auth_test.go
git commit -m "feat(backend): add Firebase auth middleware with dev mode"
```

---

### Task 11: Health handler

**Files:**
- Create: `backend/internal/handler/health.go`
- Test: `backend/internal/handler/health_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/health_test.go`:

```go
package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
)

func TestHealthHandler(t *testing.T) {
	h := handler.NewHealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/handler/... -v -run TestHealth
```

Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/handler/health.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/handler/... -v -run TestHealth
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/
git commit -m "feat(backend): add health endpoint handler"
```

---

### Task 12: Agent handler (create agent + generate token)

**Files:**
- Create: `backend/internal/handler/agent.go`
- Test: `backend/internal/handler/agent_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/agent_test.go`:

```go
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func setupAgentTest(t *testing.T) (*handler.AgentHandler, *repository.UserRepo, *repository.AgentRepo, *repository.TokenRepo) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	tokenRepo := repository.NewTokenRepo(db)
	h := handler.NewAgentHandler(userRepo, agentRepo, tokenRepo)
	return h, userRepo, agentRepo, tokenRepo
}

func TestAgentHandler_CreateAgent(t *testing.T) {
	h, _, _, _ := setupAgentTest(t)

	body := `{"name": "My Agent"}`
	req := httptest.NewRequest("POST", "/agents", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.CreateAgent(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["name"] != "My Agent" {
		t.Errorf("expected name My Agent, got %v", resp["name"])
	}
}

func TestAgentHandler_CreateToken(t *testing.T) {
	h, userRepo, agentRepo, _ := setupAgentTest(t)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	req := httptest.NewRequest("POST", "/agents/"+agent.ID+"/tokens", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agentID", agent.ID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(middleware.WithFirebaseUID(ctx, "firebase-abc"))
	rec := httptest.NewRecorder()

	h.CreateToken(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	token, ok := resp["token"].(string)
	if !ok || token == "" {
		t.Error("expected non-empty token in response")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/handler/... -v -run TestAgent
```

Expected: FAIL

- [ ] **Step 3: Add WithFirebaseUID helper to middleware package**

Add to `backend/internal/middleware/firebase_auth.go` (append):

```go
// WithFirebaseUID sets the Firebase UID in context. Used for testing.
func WithFirebaseUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, firebaseUIDKey, uid)
}
```

- [ ] **Step 4: Write agent handler implementation**

Create `backend/internal/handler/agent.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type AgentHandler struct {
	userRepo  *repository.UserRepo
	agentRepo *repository.AgentRepo
	tokenRepo *repository.TokenRepo
}

func NewAgentHandler(userRepo *repository.UserRepo, agentRepo *repository.AgentRepo, tokenRepo *repository.TokenRepo) *AgentHandler {
	return &AgentHandler{
		userRepo:  userRepo,
		agentRepo: agentRepo,
		tokenRepo: tokenRepo,
	}
}

func (h *AgentHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	agent, err := h.agentRepo.Create(user.ID, req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create agent"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

func (h *AgentHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	agentID := chi.URLParam(r, "agentID")

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	// Verify agent belongs to user
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
		return
	}
	if agent.UserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "agent does not belong to user"})
		return
	}

	rawToken, err := h.tokenRepo.Create(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"token": rawToken})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd backend && go test ./internal/handler/... -v -run TestAgent
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/agent.go backend/internal/handler/agent_test.go backend/internal/middleware/firebase_auth.go
git commit -m "feat(backend): add agent and token creation handlers"
```

---

### Task 13: Post handler (agent-authenticated post creation)

**Files:**
- Create: `backend/internal/handler/post.go`
- Test: `backend/internal/handler/post_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/post_test.go`:

```go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestPostHandler_CreatePost(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Tennis courts nearby", "body": "A park near you has tennis courts."}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["title"] != "Tennis courts nearby" {
		t.Errorf("expected title, got %v", resp["title"])
	}
	if resp["agent_name"] != "My Agent" {
		t.Errorf("expected agent_name My Agent, got %v", resp["agent_name"])
	}
}

func TestPostHandler_MissingTitle(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"body": "no title"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/handler/... -v -run TestPost
```

Expected: FAIL

- [ ] **Step 3: Add WithAgentID helper to middleware package**

Add to `backend/internal/middleware/agent_auth.go` (append):

```go
// WithAgentID sets the agent ID in context. Used for testing.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDKey, agentID)
}
```

- [ ] **Step 4: Write post handler implementation**

Create `backend/internal/handler/post.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type PostHandler struct {
	agentRepo *repository.AgentRepo
	postRepo  *repository.PostRepo
}

func NewPostHandler(agentRepo *repository.AgentRepo, postRepo *repository.PostRepo) *PostHandler {
	return &PostHandler{
		agentRepo: agentRepo,
		postRepo:  postRepo,
	}
}

type createPostRequest struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	ImageURL    string `json:"image_url,omitempty"`
	ExternalURL string `json:"external_url,omitempty"`
	Locality    string `json:"locality,omitempty"`
	PostType    string `json:"post_type,omitempty"`
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())

	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Title == "" || req.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and body are required"})
		return
	}

	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	post, err := h.postRepo.Create(repository.CreatePostParams{
		AgentID:     agentID,
		UserID:      agent.UserID,
		Title:       req.Title,
		Body:        req.Body,
		ImageURL:    req.ImageURL,
		ExternalURL: req.ExternalURL,
		Locality:    req.Locality,
		PostType:    req.PostType,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd backend && go test ./internal/handler/... -v -run TestPost
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/post.go backend/internal/handler/post_test.go backend/internal/middleware/agent_auth.go
git commit -m "feat(backend): add post creation handler with agent auth"
```

---

### Task 14: Feed handler

**Files:**
- Create: `backend/internal/handler/feed.go`
- Test: `backend/internal/handler/feed_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/feed_test.go`:

```go
package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestFeedHandler_EmptyFeed(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewFeedHandler(userRepo, postRepo)

	req := httptest.NewRequest("GET", "/feed", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.GetFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var posts []model.Post
	json.NewDecoder(rec.Body).Decode(&posts)
	if len(posts) != 0 {
		t.Errorf("expected empty feed, got %d posts", len(posts))
	}
}

func TestFeedHandler_WithPosts(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")
	postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID,
		Title: "Test Post", Body: "Test body",
	})

	h := handler.NewFeedHandler(userRepo, postRepo)

	req := httptest.NewRequest("GET", "/feed", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.GetFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var posts []model.Post
	json.NewDecoder(rec.Body).Decode(&posts)
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Title != "Test Post" {
		t.Errorf("expected title Test Post, got %s", posts[0].Title)
	}
	if posts[0].AgentName != "My Agent" {
		t.Errorf("expected agent name My Agent, got %s", posts[0].AgentName)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/handler/... -v -run TestFeed
```

Expected: FAIL

- [ ] **Step 3: Write feed handler implementation**

Create `backend/internal/handler/feed.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type FeedHandler struct {
	userRepo *repository.UserRepo
	postRepo *repository.PostRepo
}

func NewFeedHandler(userRepo *repository.UserRepo, postRepo *repository.PostRepo) *FeedHandler {
	return &FeedHandler{
		userRepo: userRepo,
		postRepo: postRepo,
	}
}

func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	posts, err := h.postRepo.ListByUserID(user.ID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load feed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/handler/... -v -run TestFeed
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/feed.go backend/internal/handler/feed_test.go
git commit -m "feat(backend): add feed retrieval handler"
```

---

## Chunk 4: Server Entrypoint and Integration

### Task 15: Me (who-am-I) handler

**Files:**
- Modify: `backend/internal/handler/health.go`
- Modify: `backend/internal/handler/health_test.go`

- [ ] **Step 1: Write the failing test**

Update `backend/internal/handler/health_test.go` imports to:

```go
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)
```

Then add the test function:

```go
func TestMeHandler(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	h := handler.NewMeHandler(userRepo)

	req := httptest.NewRequest("GET", "/me", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["firebase_uid"] != "firebase-abc" {
		t.Errorf("expected firebase_uid firebase-abc, got %v", resp["firebase_uid"])
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty user id")
	}
}
```

Add necessary imports to the test file: `database`, `middleware`, `repository`.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/handler/... -v -run TestMe
```

Expected: FAIL

- [ ] **Step 3: Add Me handler**

Update `backend/internal/handler/health.go` imports to:

```go
import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)
```

Then add the MeHandler after the existing HealthHandler code:

```go
type MeHandler struct {
	userRepo *repository.UserRepo
}

func NewMeHandler(userRepo *repository.UserRepo) *MeHandler {
	return &MeHandler{userRepo: userRepo}
}

func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/handler/... -v -run TestMe
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/
git commit -m "feat(backend): add /me debug endpoint"
```

---

### Task 16: Server entrypoint with router wiring

**Files:**
- Create: `backend/cmd/server/main.go`

- [ ] **Step 1: Create the server entrypoint**

Create `backend/cmd/server/main.go`:

```go
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/config"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("starting server", "port", cfg.Port, "db", cfg.DatabasePath)

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Repositories
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	tokenRepo := repository.NewTokenRepo(db)
	postRepo := repository.NewPostRepo(db)

	// Handlers
	healthH := handler.NewHealthHandler()
	meH := handler.NewMeHandler(userRepo)
	agentH := handler.NewAgentHandler(userRepo, agentRepo, tokenRepo)
	postH := handler.NewPostHandler(agentRepo, postRepo)
	feedH := handler.NewFeedHandler(userRepo, postRepo)

	// Middleware
	firebaseAuth := middleware.FirebaseAuth(nil) // dev mode: no Firebase client
	agentAuth := middleware.AgentAuth(tokenRepo)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Public
	r.Get("/health", healthH.Health)

	// Firebase-authenticated routes (mobile client)
	r.Group(func(r chi.Router) {
		r.Use(firebaseAuth)
		r.Get("/me", meH.Me)
		r.Get("/feed", feedH.GetFeed)
		r.Post("/agents", agentH.CreateAgent)
		r.Post("/agents/{agentID}/tokens", agentH.CreateToken)
	})

	// Agent-token-authenticated routes (Claude skill / agent client)
	r.Group(func(r chi.Router) {
		r.Use(agentAuth)
		r.Post("/posts", postH.CreatePost)
	})

	slog.Info("listening", "addr", ":"+cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./cmd/server
```

Expected: builds without errors

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(backend): add server entrypoint with full router wiring"
```

---

### Task 17: Manual integration smoke test

- [ ] **Step 1: Start the server**

```bash
cd backend && go run ./cmd/server &
```

- [ ] **Step 2: Test health endpoint**

```bash
curl -s http://localhost:8080/health | jq .
```

Expected: `{"status": "ok"}`

- [ ] **Step 3: Test user creation via /me**

```bash
curl -s -H "Authorization: Bearer test-user-1" http://localhost:8080/me | jq .
```

Expected: JSON with `id` and `firebase_uid: "test-user-1"`

- [ ] **Step 4: Create an agent**

```bash
curl -s -X POST -H "Authorization: Bearer test-user-1" \
  -H "Content-Type: application/json" \
  -d '{"name": "My First Agent"}' \
  http://localhost:8080/agents | jq .
```

Expected: JSON with agent details. Save the `id` value.

- [ ] **Step 5: Generate an agent token**

```bash
curl -s -X POST -H "Authorization: Bearer test-user-1" \
  http://localhost:8080/agents/{AGENT_ID}/tokens | jq .
```

Expected: JSON with `token` field starting with `bbp_`. Save this token.

- [ ] **Step 6: Create a post using agent token**

```bash
curl -s -X POST -H "Authorization: Bearer {AGENT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"title": "Tennis courts 6 minutes away", "body": "A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer."}' \
  http://localhost:8080/posts | jq .
```

Expected: JSON with created post including `agent_name`

- [ ] **Step 7: Fetch the feed**

```bash
curl -s -H "Authorization: Bearer test-user-1" http://localhost:8080/feed | jq .
```

Expected: JSON array with 1 post

- [ ] **Step 8: Stop the server and commit**

```bash
kill %1
```

No code changes — this is verification only.

---

### Task 18: Add .gitignore

**Files:**
- Create: `backend/.gitignore`

- [ ] **Step 1: Create .gitignore**

Create `backend/.gitignore`:

```
*.db
*.db-wal
*.db-shm
/server
```

- [ ] **Step 2: Commit**

```bash
git add backend/.gitignore
git commit -m "chore(backend): add .gitignore for SQLite files and binary"
```
