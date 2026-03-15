# BeepBopBoop Firebase Auth Setup Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Configure Firebase project for authentication and wire real Firebase token verification into the Go backend.

**Architecture:** Firebase project with Email/Password auth provider (simulator-compatible). Backend uses Firebase Admin SDK for Go to verify ID tokens. Dev mode (no Firebase client) remains available for local testing without Firebase.

**Tech Stack:** Firebase Console, Firebase Admin SDK for Go, service account JSON key

**Depends on:** Plan 1 (Go Backend) — needs the middleware and server entrypoint

**Ref docs:** `MVP_CHECKLIST.md` Section 3 | `PRD.md` Section 10

---

## File Structure

```
backend/
├── internal/
│   ├── config/
│   │   └── config.go                # Add FIREBASE_CREDENTIALS_FILE env var
│   └── middleware/
│       └── firebase_auth.go         # Already exists — wire real Firebase client
├── cmd/
│   └── server/
│       └── main.go                  # Wire Firebase client initialization
├── firebase-credentials.json        # .gitignored — service account key
```

---

## Chunk 1: Firebase Project and Backend Integration

### Task 1: Create Firebase project (manual)

- [ ] **Step 1: Create Firebase project**

Go to [Firebase Console](https://console.firebase.google.com/). Create a new project called `beepbopboop` (or similar). Disable Google Analytics if not needed.

- [ ] **Step 2: Enable Email/Password authentication**

In Firebase Console → Authentication → Sign-in method → Enable "Email/Password" provider. This is the simplest simulator-compatible auth method.

- [ ] **Step 3: Generate service account key**

In Firebase Console → Project Settings → Service Accounts → Generate new private key. Save the JSON file as `backend/firebase-credentials.json`.

- [ ] **Step 4: Note the Firebase project ID**

Record the project ID from Project Settings → General. You'll use this as `FIREBASE_PROJECT_ID` env var.

---

### Task 2: Add Firebase credentials config

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/.gitignore`

- [ ] **Step 1: Update .gitignore**

Add to `backend/.gitignore`:

```
firebase-credentials.json
```

- [ ] **Step 2: Write the failing test**

Add to `backend/internal/config/config_test.go`:

```go
func TestLoadFirebaseCredentials(t *testing.T) {
	os.Setenv("FIREBASE_CREDENTIALS_FILE", "/tmp/creds.json")
	defer os.Unsetenv("FIREBASE_CREDENTIALS_FILE")

	cfg := config.Load()
	if cfg.FirebaseCredentialsFile != "/tmp/creds.json" {
		t.Errorf("expected /tmp/creds.json, got %s", cfg.FirebaseCredentialsFile)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd backend && go test ./internal/config/... -v -run TestLoadFirebase
```

Expected: FAIL — `FirebaseCredentialsFile` field doesn't exist

- [ ] **Step 4: Add field to config**

In `backend/internal/config/config.go`, add to the `Config` struct:

```go
FirebaseCredentialsFile string
```

And in the `Load()` function:

```go
FirebaseCredentialsFile: os.Getenv("FIREBASE_CREDENTIALS_FILE"),
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd backend && go test ./internal/config/... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/config/ backend/.gitignore
git commit -m "feat(backend): add Firebase credentials file config"
```

---

### Task 3: Wire real Firebase auth client in server

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add Firebase client initialization to main.go**

Update the imports and add Firebase client initialization before the router setup in `backend/cmd/server/main.go`:

```go
import (
	// ... existing imports ...
	"context"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)
```

Add after config loading, before handler creation:

```go
	// Firebase auth client (nil = dev mode)
	var firebaseAuthClient *auth.Client
	if cfg.FirebaseCredentialsFile != "" {
		opt := option.WithCredentialsFile(cfg.FirebaseCredentialsFile)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			slog.Error("failed to initialize Firebase app", "error", err)
			os.Exit(1)
		}
		firebaseAuthClient, err = app.Auth(context.Background())
		if err != nil {
			slog.Error("failed to initialize Firebase auth client", "error", err)
			os.Exit(1)
		}
		slog.Info("Firebase auth enabled")
	} else {
		slog.Warn("Firebase auth disabled — running in dev mode")
	}
```

Update the middleware initialization:

```go
	firebaseAuth := middleware.FirebaseAuth(firebaseAuthClient)
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./cmd/server
```

Expected: builds without errors

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(backend): wire Firebase auth client with credentials file support"
```

---

### Task 4: Document Firebase setup for developers

**Files:**
- Create: `backend/README.md`

- [ ] **Step 1: Write backend README**

Create `backend/README.md`:

```markdown
# BeepBopBoop Backend

Go REST API backend for BeepBopBoop.

## Quick Start (Dev Mode)

Dev mode uses no Firebase — the Bearer token value is used as the user identity.

```bash
cd backend
go run ./cmd/server
```

Server starts on :8080. Test with:

```bash
# Health check
curl http://localhost:8080/health

# Create user (dev mode: bearer = firebase UID)
curl -H "Authorization: Bearer my-test-user" http://localhost:8080/me
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_PATH` | `beepbopboop.db` | SQLite database file path |
| `FIREBASE_PROJECT_ID` | (empty) | Firebase project ID |
| `FIREBASE_CREDENTIALS_FILE` | (empty) | Path to Firebase service account JSON. If empty, runs in dev mode. |

## Firebase Setup (Production Mode)

1. Create a Firebase project at https://console.firebase.google.com/
2. Enable Email/Password authentication
3. Generate a service account key (Project Settings → Service Accounts)
4. Save the key as `backend/firebase-credentials.json`
5. Run with:

```bash
FIREBASE_CREDENTIALS_FILE=./firebase-credentials.json go run ./cmd/server
```

## API Endpoints

### Public
- `GET /health` — Health check

### Firebase-authenticated (mobile client)
- `GET /me` — Current user info
- `GET /feed` — User's post feed
- `POST /agents` — Create an agent `{"name": "..."}`
- `POST /agents/{agentID}/tokens` — Generate agent API token

### Agent-token-authenticated (Claude skill)
- `POST /posts` — Create a post `{"title": "...", "body": "...", "image_url": "...", "external_url": "...", "locality": "...", "post_type": "..."}`
```

- [ ] **Step 2: Commit**

```bash
git add backend/README.md
git commit -m "docs(backend): add README with setup and API documentation"
```
