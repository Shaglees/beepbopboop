#!/usr/bin/env bash
set -euo pipefail

# ─── Multi-Model Skill Testing via Hermes Agent in Docker ───
# Config-driven pipeline: reads models.conf, generates docker-compose,
# captures JSON results, and runs quality scoring report.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$SCRIPT_DIR/llm-compare"
COMPOSE_FILE="$BUILD_DIR/docker-compose.yml"
MODELS_CONF="$BUILD_DIR/models.conf"
RESULTS_DIR="$BUILD_DIR/results"
EXPECTATIONS="$BUILD_DIR/expectations.json"

# ─── Parse models.conf ───
MODEL_KEYS=()
MODEL_IDS=()
MODEL_NAMES=()
MODEL_STATUS=()

parse_models_conf() {
  if [[ ! -f "$MODELS_CONF" ]]; then
    echo "ERROR: models.conf not found at $MODELS_CONF"
    exit 1
  fi
  while IFS='|' read -r key id name status; do
    [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
    MODEL_KEYS+=("$key")
    MODEL_IDS+=("$id")
    MODEL_NAMES+=("$name")
    MODEL_STATUS+=("$status")
  done < "$MODELS_CONF"
}

parse_models_conf

# Deterministic tokens — computed from key using a hash suffix so they don't
# look like redacted placeholders to the LLM agent (all-zero padding caused
# Claude to refuse to use the token, treating it as a censored value).
get_token() {
  local prefix="bbp_test_${1}_"
  local target_len=64
  local pad_len=$(( target_len - ${#prefix} ))
  if (( pad_len < 0 )); then pad_len=0; fi
  local suffix
  suffix=$(printf '%s' "bbp_seed_${1}" | shasum -a 256 | cut -c1-${pad_len})
  printf '%s%s' "$prefix" "$suffix"
}

# Lookup helpers (bash 3.2 compatible — no associative arrays)
get_model_index() {
  local key="$1"
  for i in "${!MODEL_KEYS[@]}"; do
    if [[ "${MODEL_KEYS[$i]}" == "$key" ]]; then echo "$i"; return; fi
  done
  echo "-1"
}

get_model_id() {
  local idx
  idx=$(get_model_index "$1")
  if [[ "$idx" == "-1" ]]; then echo ""; else echo "${MODEL_IDS[$idx]}"; fi
}

get_model_status() {
  local idx
  idx=$(get_model_index "$1")
  if [[ "$idx" == "-1" ]]; then echo ""; else echo "${MODEL_STATUS[$idx]}"; fi
}

# ─── CLI flags ───
RUN_MODEL=""
SKIP_SETUP=false
CLEANUP=false
PARALLEL=false
REPORT_ONLY=false
CAPTURE_ONLY=false
COMPARE_MODEL=""

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Options:
  --model <key>                  Run single model only
  --skip-setup                   Skip test user creation
  --cleanup                      Delete test posts and users, then exit
  --parallel                     Run all containers simultaneously
  --report-only                  Skip runs, generate report from existing results
  --capture-only                 Fetch posts for all models (no container runs)
  --compare-model <openrouter_id>  LLM judge for quality scoring (default: none)
  -h, --help                     Show this help

Models (from models.conf):
$(for i in "${!MODEL_KEYS[@]}"; do
    printf "  %-15s %-40s [%s]\n" "${MODEL_KEYS[$i]}" "${MODEL_IDS[$i]}" "${MODEL_STATUS[$i]}"
  done)
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --model)         RUN_MODEL="$2"; shift 2 ;;
    --skip-setup)    SKIP_SETUP=true; shift ;;
    --cleanup)       CLEANUP=true; shift ;;
    --parallel)      PARALLEL=true; shift ;;
    --report-only)   REPORT_ONLY=true; shift ;;
    --capture-only)  CAPTURE_ONLY=true; shift ;;
    --compare-model) COMPARE_MODEL="$2"; shift 2 ;;
    -h|--help)       usage ;;
    *)               echo "Unknown option: $1"; usage ;;
  esac
done

# ─── Section 1: Validation ───
echo "=== Validating environment ==="

# Parse config
CONFIG_FILE="${HOME}/.config/beepbopboop/config"
if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "ERROR: Config not found at $CONFIG_FILE"
  exit 1
fi

get_config() { grep "^${1}=" "$CONFIG_FILE" | head -1 | cut -d= -f2-; }

API_URL="$(get_config BEEPBOPBOOP_API_URL)"
ADMIN_TOKEN="$(get_config BEEPBOPBOOP_AGENT_TOKEN)"

if [[ -z "$API_URL" ]]; then echo "ERROR: Missing BEEPBOPBOOP_API_URL in config"; exit 1; fi
if [[ -z "$ADMIN_TOKEN" ]]; then echo "ERROR: Missing BEEPBOPBOOP_AGENT_TOKEN in config"; exit 1; fi

# OpenRouter key
OR_KEY_FILE="${HOME}/.apikeys/openrouter"
if [[ -f "$OR_KEY_FILE" ]]; then
  OR_KEY="$(tr -d '[:space:]' < "$OR_KEY_FILE")"
elif [[ -n "${OPENROUTER_API_KEY:-}" ]]; then
  OR_KEY="$OPENROUTER_API_KEY"
else
  echo "ERROR: No OpenRouter key. Set OPENROUTER_API_KEY or create ~/.apikeys/openrouter"
  exit 1
fi
export OPENROUTER_API_KEY="$OR_KEY"

# Docker (skip for report-only)
if ! $REPORT_ONLY; then
  if ! docker info &>/dev/null; then
    echo "ERROR: Docker is not running"
    exit 1
  fi
fi

# Backend reachable
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/posts" \
  -H "Authorization: Bearer $ADMIN_TOKEN" 2>/dev/null || echo "000")
if [[ "$HTTP_CODE" != "200" ]]; then
  echo "ERROR: Backend not reachable at $API_URL (HTTP $HTTP_CODE)"
  exit 1
fi

echo "  Config:    $CONFIG_FILE"
echo "  API:       $API_URL"
echo "  Models:    ${#MODEL_KEYS[@]} (from models.conf)"
if ! $REPORT_ONLY; then echo "  Docker:    OK"; fi
echo "  Backend:   OK (HTTP $HTTP_CODE)"
echo "  OpenRouter: ...${OR_KEY: -8}"
if [[ -n "$COMPARE_MODEL" ]]; then
  echo "  Judge:     $COMPARE_MODEL"
fi
echo ""

# Create results dir
mkdir -p "$RESULTS_DIR"

# ─── Cleanup mode ───
if $CLEANUP; then
  echo "=== Cleaning up test data ==="
  DB_CONTAINER="backend-db-1"
  DB_CMD="docker exec $DB_CONTAINER psql -U beepbopboop -d beepbopboop -tAc"

  if ! docker exec "$DB_CONTAINER" pg_isready -U beepbopboop &>/dev/null; then
    echo "ERROR: Database container '$DB_CONTAINER' not running"
    exit 1
  fi

  for key in "${MODEL_KEYS[@]}"; do
    USER_ID="test-user-${key}"
    echo "  Deleting posts for $key (user=$USER_ID)..."
    COUNT=$($DB_CMD "DELETE FROM posts WHERE user_id = '${USER_ID}'; SELECT count(*) FROM posts WHERE user_id = '${USER_ID}';" 2>/dev/null || echo "?")
    echo "  Done ($key) — remaining: $COUNT"
  done
  echo "=== Cleanup complete ==="
  exit 0
fi

# ─── Capture-only mode ───
if $CAPTURE_ONLY; then
  echo "=== Capturing posts for all models ==="
  for key in "${MODEL_KEYS[@]}"; do
    TOKEN="$(get_token "$key")"
    echo "  Fetching $key..."
    curl -s "$API_URL/posts?limit=100" \
      -H "Authorization: Bearer $TOKEN" > "$RESULTS_DIR/${key}.json" 2>/dev/null
    COUNT=$(python3 -c "import json,sys; print(len(json.load(open(sys.argv[1]))))" "$RESULTS_DIR/${key}.json" 2>/dev/null || echo "0")
    echo "  Saved $COUNT posts to results/${key}.json"
  done
  echo ""
  # Run report if results exist
  echo "=== Generating report ==="
  REPORT_ARGS=("$RESULTS_DIR" "$MODELS_CONF" --expectations "$EXPECTATIONS")
  if [[ -n "$COMPARE_MODEL" ]]; then
    REPORT_ARGS+=(--compare-model "$COMPARE_MODEL" --openrouter-key "$OR_KEY")
  fi
  python3 "$BUILD_DIR/llm_compare_report.py" "${REPORT_ARGS[@]}"
  exit 0
fi

# ─── Report-only mode ───
if $REPORT_ONLY; then
  echo "=== Generating report from existing results ==="
  REPORT_ARGS=("$RESULTS_DIR" "$MODELS_CONF" --expectations "$EXPECTATIONS")
  if [[ -n "$COMPARE_MODEL" ]]; then
    REPORT_ARGS+=(--compare-model "$COMPARE_MODEL" --openrouter-key "$OR_KEY")
  fi
  python3 "$BUILD_DIR/llm_compare_report.py" "${REPORT_ARGS[@]}"
  exit 0
fi

# ─── Section 1b: Pre-flight token health check ───
# Verifies that each model's deterministic token authenticates against the
# backend before wasting time building Docker images. Catches the "invalid or
# revoked token" failure that caused 5 of 6 models to produce zero posts in
# the first comparison run (#247).
preflight_check_tokens() {
  local models_to_check=("$@")
  local all_ok=true

  echo "=== Pre-flight: verifying tokens for ${#models_to_check[@]} model(s) ==="
  for key in "${models_to_check[@]}"; do
    local token
    token="$(get_token "$key")"
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
      "$API_URL/posts?limit=1" \
      -H "Authorization: Bearer $token" 2>/dev/null || echo "000")

    if [[ "$http_code" == "200" ]]; then
      echo "  ✓ $key — token OK (HTTP $http_code)"
    else
      echo "  ✗ $key — token FAILED (HTTP $http_code)"
      echo "    Token: ${token:0:20}..."
      echo "    Fix: re-run without --skip-setup to recreate the token in DB,"
      echo "         or check that the DB container is accessible."
      all_ok=false
    fi
  done

  if ! $all_ok; then
    echo ""
    echo "ERROR: One or more tokens failed pre-flight check. Aborting."
    echo "  Run without --skip-setup to recreate test users and tokens."
    exit 1
  fi
  echo "  All tokens verified."
  echo ""
}

# ─── Determine which models to run ───
RUN_MODELS=()
if [[ -n "$RUN_MODEL" ]]; then
  idx=$(get_model_index "$RUN_MODEL")
  if [[ "$idx" == "-1" ]]; then
    echo "ERROR: Unknown model key '$RUN_MODEL'. Valid: ${MODEL_KEYS[*]}"
    exit 1
  fi
  RUN_MODELS=("$RUN_MODEL")
else
  # Only run models with status=pending
  for i in "${!MODEL_KEYS[@]}"; do
    if [[ "${MODEL_STATUS[$i]}" == "pending" ]]; then
      RUN_MODELS+=("${MODEL_KEYS[$i]}")
    fi
  done
fi

if [[ ${#RUN_MODELS[@]} -eq 0 ]]; then
  echo "No pending models to run. Use --model <key> to force a specific model,"
  echo "or change status in models.conf from 'done' to 'pending'."
  echo ""
  echo "Generating report from existing results..."
  REPORT_ARGS=("$RESULTS_DIR" "$MODELS_CONF" --expectations "$EXPECTATIONS")
  if [[ -n "$COMPARE_MODEL" ]]; then
    REPORT_ARGS+=(--compare-model "$COMPARE_MODEL" --openrouter-key "$OR_KEY")
  fi
  python3 "$BUILD_DIR/llm_compare_report.py" "${REPORT_ARGS[@]}"
  exit 0
fi

echo "Models to run: ${RUN_MODELS[*]}"
echo ""

# Pre-flight: verify tokens authenticate before building images.
# Only runs when --skip-setup is active — if we're running full setup, tokens
# are created in Section 4 and won't exist in the DB yet at this point.
if $SKIP_SETUP; then
  preflight_check_tokens "${RUN_MODELS[@]}"
else
  echo "  (full setup mode: tokens will be created in Section 4, skipping pre-flight)"
  echo ""
fi

# ─── Section 2: Generate docker-compose.yml ───
echo "=== Generating docker-compose.yml ==="

compose_content="# Auto-generated from models.conf — do not edit manually
services:"

for key in "${RUN_MODELS[@]}"; do
  model_id=$(get_model_id "$key")
  token=$(get_token "$key")
  compose_content+="
  bbp-${key}:
    build: .
    environment:
      - OPENROUTER_API_KEY
      - BBP_MODEL=${model_id}
      - BBP_USER_TOKEN=${token}
      - BBP_API_URL=http://host.docker.internal:8080
      - BBP_LABEL=test-${key}
      - BBP_BATCH_MIN=8
      - BBP_BATCH_MAX=10
    extra_hosts:
      - \"host.docker.internal:host-gateway\""
done

echo "$compose_content" > "$COMPOSE_FILE"
echo "  Generated with ${#RUN_MODELS[@]} services"
echo ""

# ─── Section 3: Build image ───
echo "=== Building Docker image ==="

# Pre-build beepbopgraph binary
echo "  Building beepbopgraph (linux/amd64)..."
(cd "$REPO_ROOT/backend" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/beepbopgraph" ./cmd/beepbopgraph)
echo "  beepbopgraph built"

# Copy skill files
echo "  Copying skills..."
rm -rf "$BUILD_DIR/skills"
mkdir -p "$BUILD_DIR/skills"
for skill_dir in beepbopboop-post beepbopboop-images _shared; do
  if [[ -d "$REPO_ROOT/.claude/skills/$skill_dir" ]]; then
    cp -r "$REPO_ROOT/.claude/skills/$skill_dir" "$BUILD_DIR/skills/"
  fi
done
echo "  Skills copied"

# Build Docker image
docker compose -f "$COMPOSE_FILE" build
echo "  Image built"
echo ""

# ─── Section 4: Create test users & agents in DB ───
DB_CONTAINER="backend-db-1"
DB_CMD="docker exec $DB_CONTAINER psql -U beepbopboop -d beepbopboop -tAc"

if ! $SKIP_SETUP; then
  echo "=== Creating test users & agents ==="

  if ! docker exec "$DB_CONTAINER" pg_isready -U beepbopboop &>/dev/null; then
    echo "ERROR: Database container '$DB_CONTAINER' not running"
    echo "  Start it with: cd backend && docker-compose up -d"
    exit 1
  fi

  # Test user profile — same for all models so comparison is fair
  TEST_LOCATION="Austin, TX"
  TEST_LAT="30.2672"
  TEST_LON="-97.7431"
  TEST_INTERESTS=("sports:Sports" "food:Food" "music:Music" "technology:Technology")

  for key in "${RUN_MODELS[@]}"; do
    TOKEN="$(get_token "$key")"
    TOKEN_HASH=$(echo -n "$TOKEN" | shasum -a 256 | cut -d' ' -f1)
    USER_ID="test-user-${key}"
    AGENT_ID="test-agent-${key}"
    TOKEN_ID="test-token-${key}"

    echo "  Creating $key (user=$USER_ID, agent=$AGENT_ID)..."

    # User with profile data
    $DB_CMD "
      INSERT INTO users (id, firebase_uid, display_name, home_location, home_lat, home_lon, created_at)
      VALUES ('${USER_ID}', 'test-firebase-${key}', 'Test ${key}', '${TEST_LOCATION}', ${TEST_LAT}, ${TEST_LON}, NOW())
      ON CONFLICT (id) DO UPDATE SET
        home_location = '${TEST_LOCATION}',
        home_lat = ${TEST_LAT},
        home_lon = ${TEST_LON};
    " >/dev/null

    # User settings (location + defaults)
    $DB_CMD "
      INSERT INTO user_settings (user_id, location_name, latitude, longitude)
      VALUES ('${USER_ID}', '${TEST_LOCATION}', ${TEST_LAT}, ${TEST_LON})
      ON CONFLICT (user_id) DO UPDATE SET
        location_name = '${TEST_LOCATION}',
        latitude = ${TEST_LAT},
        longitude = ${TEST_LON};
    " >/dev/null

    # Agent + token
    $DB_CMD "
      INSERT INTO agents (id, user_id, name, status, created_at)
      VALUES ('${AGENT_ID}', '${USER_ID}', 'Test ${key}', 'active', NOW())
      ON CONFLICT (id) DO NOTHING;
    " >/dev/null

    $DB_CMD "
      INSERT INTO agent_tokens (id, agent_id, token_hash, revoked, created_at)
      VALUES ('${TOKEN_ID}', '${AGENT_ID}', '${TOKEN_HASH}', false, NOW())
      ON CONFLICT (id) DO UPDATE SET token_hash = '${TOKEN_HASH}', revoked = false;
    " >/dev/null

    # User interests
    for interest_pair in "${TEST_INTERESTS[@]}"; do
      IFS=':' read -r topic category <<< "$interest_pair"
      INTEREST_ID="test-interest-${key}-${topic}"
      $DB_CMD "
        INSERT INTO user_interests (id, user_id, category, topic, source, confidence)
        VALUES ('${INTEREST_ID}', '${USER_ID}', '${category}', '${topic}', 'user', 1.0)
        ON CONFLICT (user_id, category, topic) DO NOTHING;
      " >/dev/null
    done

    echo "  Created ($key) — profile: ${TEST_LOCATION}, interests: sports,food,music,technology"
  done
  echo ""
fi

# ─── Section 5: Run containers ───
echo "=== Running model containers ==="

MODEL_TIMEOUT="${MODEL_TIMEOUT:-600}"  # 10 min default per model

run_model() {
  local key="$1"
  echo "--- Running bbp-${key} (timeout: ${MODEL_TIMEOUT}s) ---"
  local log="/tmp/bbp-run-${key}.log"

  # Run container in background, kill after timeout
  OPENROUTER_API_KEY="$OR_KEY" \
    docker compose -f "$COMPOSE_FILE" \
    run --rm "bbp-${key}" > "$log" 2>&1 &
  local pid=$!

  # Wait with timeout (portable — no gtimeout needed)
  local elapsed=0
  while kill -0 "$pid" 2>/dev/null; do
    sleep 10
    elapsed=$((elapsed + 10))
    # Check post count periodically
    local token
    token="$(get_token "$key")"
    local count
    count=$(curl -s "$API_URL/posts?limit=100" -H "Authorization: Bearer $token" \
      | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d) if isinstance(d,list) else 0)" 2>/dev/null || echo "0")
    echo "  [${elapsed}s] bbp-${key}: ${count} posts so far"

    if [[ "$elapsed" -ge "$MODEL_TIMEOUT" ]]; then
      echo "  TIMEOUT: bbp-${key} exceeded ${MODEL_TIMEOUT}s — killing"
      kill "$pid" 2>/dev/null || true
      # Also kill the docker container
      docker compose -f "$COMPOSE_FILE" kill "bbp-${key}" 2>/dev/null || true
      docker compose -f "$COMPOSE_FILE" rm -f "bbp-${key}" 2>/dev/null || true
      break
    fi
  done
  wait "$pid" 2>/dev/null || true
  echo "--- Finished bbp-${key} (log: $log) ---"

  # Capture posts to results JSON
  local token
  token="$(get_token "$key")"
  echo "  Capturing posts for $key..."
  curl -s "$API_URL/posts?limit=100" \
    -H "Authorization: Bearer $token" > "$RESULTS_DIR/${key}.json" 2>/dev/null
  local count
  count=$(python3 -c "import json,sys; print(len(json.load(open(sys.argv[1]))))" "$RESULTS_DIR/${key}.json" 2>/dev/null || echo "0")
  echo "  Saved $count posts to results/${key}.json"
  echo ""
}

if $PARALLEL; then
  echo "  Mode: parallel"
  for key in "${RUN_MODELS[@]}"; do
    run_model "$key" &
  done
  wait
else
  echo "  Mode: sequential"
  for key in "${RUN_MODELS[@]}"; do
    run_model "$key"
  done
fi

# ─── Section 6: Generate report ───
echo ""
echo "=== Generating quality report ==="
REPORT_ARGS=("$RESULTS_DIR" "$MODELS_CONF" --expectations "$EXPECTATIONS")
if [[ -n "$COMPARE_MODEL" ]]; then
  REPORT_ARGS+=(--compare-model "$COMPARE_MODEL" --openrouter-key "$OR_KEY")
fi
python3 "$BUILD_DIR/llm_compare_report.py" "${REPORT_ARGS[@]}"

echo ""
echo "Logs saved to /tmp/bbp-run-*.log"
echo "Results saved to $RESULTS_DIR/"
