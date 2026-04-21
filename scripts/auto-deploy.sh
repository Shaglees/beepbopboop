#!/bin/bash
# Auto-deploy: polls git for new commits, rebuilds and restarts backend if changed.
# Usage: nohup ./scripts/auto-deploy.sh &

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND_DIR="$REPO_DIR/backend"
POLL_INTERVAL=900  # 15 minutes
LOG_FILE="$REPO_DIR/scripts/auto-deploy.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_FILE"
}

log "auto-deploy started (pid $$), polling every ${POLL_INTERVAL}s"

while true; do
    cd "$REPO_DIR" || exit 1

    # Fetch latest from remote
    git fetch origin main --quiet 2>/dev/null

    LOCAL=$(git rev-parse HEAD)
    REMOTE=$(git rev-parse origin/main)

    if [ "$LOCAL" != "$REMOTE" ]; then
        log "new commits detected: $LOCAL -> $REMOTE"

        # Pull latest
        git pull --ff-only origin main >> "$LOG_FILE" 2>&1

        if [ $? -ne 0 ]; then
            log "ERROR: git pull failed, skipping deploy"
            sleep "$POLL_INTERVAL"
            continue
        fi

        # Rebuild and restart backend
        cd "$BACKEND_DIR" || exit 1
        log "rebuilding backend..."
        docker compose down >> "$LOG_FILE" 2>&1
        docker compose up -d --build >> "$LOG_FILE" 2>&1

        if [ $? -eq 0 ]; then
            log "deploy successful: $(git rev-parse --short HEAD)"
        else
            log "ERROR: docker compose build/up failed"
        fi
    fi

    sleep "$POLL_INTERVAL"
done
