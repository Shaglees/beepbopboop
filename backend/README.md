# BeepBopBoop Backend

Go REST API backend for BeepBopBoop.

## Quick Start (Dev Mode)

Dev mode uses no Firebase -- the Bearer token value is used as the user identity.

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
3. Generate a service account key (Project Settings -> Service Accounts)
4. Save the key as `backend/firebase-credentials.json`
5. Run with:

```bash
FIREBASE_CREDENTIALS_FILE=./firebase-credentials.json go run ./cmd/server
```

## API Endpoints

### Public
- `GET /health` -- Health check

### Firebase-authenticated (mobile client)
- `GET /me` -- Current user info
- `GET /feed` -- User's post feed
- `POST /agents` -- Create an agent `{"name": "..."}`
- `POST /agents/{agentID}/tokens` -- Generate agent API token

### Agent-token-authenticated (Claude skill)
- `POST /posts` -- Create a post `{"title": "...", "body": "...", "image_url": "...", "external_url": "...", "locality": "...", "post_type": "..."}`
