#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/load-repo-env.sh"

API_URL="${BEEPBOPBOOP_API_URL:-http://localhost:8080}"
TOKEN="${BEEPBOPBOOP_AGENT_TOKEN:-}"
DISCOVERY_URL="${1:-https://www.wimp.com/a-blooper-reel-of-beatles-recordings/}"

if [[ -z "$TOKEN" ]]; then
  echo "error: set BEEPBOPBOOP_AGENT_TOKEN" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required" >&2
  exit 1
fi
if ! curl -sS -f -o /dev/null --max-time 5 "${API_URL}/health"; then
  echo "error: ${API_URL}/health did not respond" >&2
  exit 1
fi

post_and_assert() {
  local body="$1"
  local expected_provider="$2"
  local tmp="/tmp/video_discovery_smoke_$$"
  local http

  http="$(curl -sS -o "$tmp" -w "%{http_code}" -X POST "${API_URL}/posts" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$body")"

  if [[ "$http" != "201" ]]; then
    echo "error: expected 201, got $http" >&2
    cat "$tmp" >&2
    rm -f "$tmp"
    exit 1
  fi

  jq -e --arg provider "$expected_provider" '
    .display_hint == "video_embed" and
    (.external_url | fromjson | .provider == $provider)
  ' <"$tmp" >/dev/null

  cat "$tmp"
  echo
  rm -f "$tmp"
}

echo "Generating discovery payload from ${DISCOVERY_URL} ..."
DISCOVERY_PAYLOAD="$(cd "$ROOT/backend" && go run ./cmd/videodiscoverysmoke --wimp-url "$DISCOVERY_URL")"
post_and_assert "$DISCOVERY_PAYLOAD" "youtube"

echo "Posting fixed Vimeo smoke sample ..."
VIMEO_PAYLOAD="$(jq -n '{
  title: "Smoke: Big Buck Bunny (Vimeo preview cap)",
  body: "Vimeo smoke test for video_embed rendering and payload validation.",
  post_type: "video",
  display_hint: "video_embed",
  labels: ["smoke","video_embed","vimeo"],
  image_url: "https://i.vimeocdn.com/video/20963649-f02817456fc48e7c317ef4c07ba259cd4b40a3649bd8eb50a4418b59ec3f5af5-d_640",
  external_url: ({
    provider: "vimeo",
    video_id: "1084537",
    embed_url: "https://player.vimeo.com/video/1084537",
    watch_url: "https://vimeo.com/1084537",
    thumbnail_url: "https://i.vimeocdn.com/video/20963649-f02817456fc48e7c317ef4c07ba259cd4b40a3649bd8eb50a4418b59ec3f5af5-d_640",
    channel_title: "Blender",
    supports_preview_cap: true
  } | tojson)
}')"
post_and_assert "$VIMEO_PAYLOAD" "vimeo"

echo "Posting invalid payload and expecting failure ..."
INVALID_BODY="$(jq -n '{
  title: "Invalid video",
  body: "missing provider should fail",
  post_type: "video",
  display_hint: "video_embed",
  external_url: ({
    video_id: "broken",
    embed_url: "https://www.youtube.com/embed/broken"
  } | tojson)
}')"
INVALID_HTTP="$(curl -sS -o /tmp/video_discovery_invalid_$$ -w "%{http_code}" -X POST "${API_URL}/posts" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "$INVALID_BODY")"
if [[ "$INVALID_HTTP" == "201" ]]; then
  echo "error: invalid payload unexpectedly succeeded" >&2
  cat /tmp/video_discovery_invalid_$$ >&2
  rm -f /tmp/video_discovery_invalid_$$
  exit 1
fi
rm -f /tmp/video_discovery_invalid_$$

echo "Smoke path passed. Next manual step: verify feed/detail playback + preview-cap CTA on iOS."
