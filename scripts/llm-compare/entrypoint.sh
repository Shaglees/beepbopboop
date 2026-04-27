#!/usr/bin/env bash
set -euo pipefail

# ─── Resolve Hermes binary ───
HERMES="/usr/local/bin/hermes"
if [[ ! -x "$HERMES" ]]; then
  echo "ERROR: hermes not found at $HERMES"
  exit 1
fi

# ─── Require env vars ───
: "${BBP_MODEL:?BBP_MODEL not set}"
: "${BBP_USER_TOKEN:?BBP_USER_TOKEN not set}"
: "${BBP_API_URL:?BBP_API_URL not set}"
: "${BBP_LABEL:?BBP_LABEL not set}"
: "${OPENROUTER_API_KEY:?OPENROUTER_API_KEY not set}"

# Optional overrides (defaults come from server profile)
BBP_BATCH_MIN="${BBP_BATCH_MIN:-8}"
BBP_BATCH_MAX="${BBP_BATCH_MAX:-10}"

# ─── Pre-write config files BEFORE Hermes starts ───
# Only API_URL and AGENT_TOKEN go in the config file.
# Location, interests, etc. should be fetched from the server
# via GET /user/profile (see _shared/CONFIG.md Step 0a).

# 1. Minimal BeepBopBoop skill config (connection only)
mkdir -p ~/.config/beepbopboop
cat > ~/.config/beepbopboop/config <<EOF
BEEPBOPBOOP_API_URL=${BBP_API_URL}
BEEPBOPBOOP_AGENT_TOKEN=${BBP_USER_TOKEN}
BEEPBOPBOOP_BATCH_MIN=${BBP_BATCH_MIN}
BEEPBOPBOOP_BATCH_MAX=${BBP_BATCH_MAX}
EOF

echo "[entrypoint] Wrote bbp config: API=${BBP_API_URL} TOKEN=...${BBP_USER_TOKEN: -8}"

# 2. Verify backend + profile are reachable
PROFILE=$(curl -sf "${BBP_API_URL}/user/profile" \
  -H "Authorization: Bearer ${BBP_USER_TOKEN}" 2>/dev/null || echo "{}")
LOCATION=$(echo "$PROFILE" | python3 -c "import sys,json; p=json.load(sys.stdin); print(p.get('identity',{}).get('home_location','(none)'))" 2>/dev/null || echo "(unknown)")
INTERESTS=$(echo "$PROFILE" | python3 -c "import sys,json; p=json.load(sys.stdin); print(','.join(i['topic'] for i in p.get('interests',[])))" 2>/dev/null || echo "(unknown)")
echo "[entrypoint] Server profile: location=${LOCATION}, interests=${INTERESTS}"

# 3. Pre-populate hints cache (avoids agent needing to fetch)
mkdir -p ~/.cache/beepbopboop
HINTS_RESPONSE=$(curl -sf "${BBP_API_URL}/posts/hints" \
  -H "Authorization: Bearer ${BBP_USER_TOKEN}" 2>/dev/null || echo "")
if [[ -n "$HINTS_RESPONSE" ]]; then
  echo "{\"fetched_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"hints\": ${HINTS_RESPONSE}}" \
    > ~/.cache/beepbopboop/hints.json
  echo "[entrypoint] Pre-cached hints"
else
  echo "[entrypoint] Warning: could not pre-fetch hints (continuing anyway)"
fi

# 4. OpenRouter key for Hermes
echo "OPENROUTER_API_KEY=${OPENROUTER_API_KEY}" > /root/.hermes/.env

# 5. Hermes config with model context length
cat > /root/.hermes/config.yaml <<YAML
terminal:
  backend: local
tool_output:
  max_bytes: 50000
delegation:
  max_concurrent_children: 1
model:
  context_length: 65000
auxiliary:
  compression:
    context_length: 65000
YAML

echo "[entrypoint] Config ready, launching Hermes with model=${BBP_MODEL}"

# ─── Build prompt ───
# The config file has API_URL and TOKEN only.
# The skill should fetch user profile from the server (Step 0a in CONFIG.md).
PROMPT="Use the beepbopboop-post skill in batch mode. \
Generate ${BBP_BATCH_MIN}-${BBP_BATCH_MAX} posts. \
Add labels llm-compare and ${BBP_LABEL} to every post. \
The config file is already set up at ~/.config/beepbopboop/config — do NOT write or modify it. \
Fetch user profile from the server API (GET /user/profile) to get location and interests."

exec "$HERMES" chat \
  -q "$PROMPT" \
  -m "$BBP_MODEL" \
  --provider openrouter \
  -s beepbopboop-post \
  -t "web,terminal" \
  --yolo \
  -Q
