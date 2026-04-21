#!/usr/bin/env bash
# Create sample video_embed posts for manual / iOS testing.
#
# Requires: curl, jq
# Env (optional if repo-root .env exists):
#   BEEPBOPBOOP_API_URL  (default http://localhost:8080)
#   BEEPBOPBOOP_AGENT_TOKEN — required (copy .env.example to .env; see .claude/connection-details.md)
#
# Usage:
#   ./scripts/seed-sample-video-embed-posts.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/load-repo-env.sh"

API_URL="${BEEPBOPBOOP_API_URL:-http://localhost:8080}"
TOKEN="${BEEPBOPBOOP_AGENT_TOKEN:-}"

if [[ -z "$TOKEN" ]]; then
  echo "error: set BEEPBOPBOOP_AGENT_TOKEN (Bearer agent token for POST /posts)" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required" >&2
  exit 1
fi

if ! curl -sS -f -o /dev/null --max-time 3 "${API_URL}/health"; then
  echo "error: ${API_URL}/health did not respond — start the backend (e.g. cd backend && go run ./cmd/server)" >&2
  exit 1
fi

post() {
  local body="$1"
  local tmp="/tmp/beepbop_post_body.$$"
  echo "POST $API_URL/posts ..."
  local http
  http="$(curl -sS -o "$tmp" -w "%{http_code}" -X POST "${API_URL}/posts" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "${body}")"
  if [[ "$http" != "201" ]]; then
    echo "error: HTTP $http — body:" >&2
    cat "$tmp" >&2
    rm -f "$tmp"
    exit 1
  fi
  jq . <"$tmp"
  rm -f "$tmp"
  echo ""
}

# YouTube — standard embed
post "$(jq -n \
  --argjson embed '{
    "provider":"youtube",
    "video_id":"dQw4w9WgXcQ",
    "embed_url":"https://www.youtube.com/embed/dQw4w9WgXcQ",
    "watch_url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    "thumbnail_url":"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
    "channel_title":"Rick Astley"
  }' \
  '{
    title: "Sample: Never Gonna Give You Up (YouTube)",
    body: "video_embed test — thumbnail + sheet player. Share should use watch_url.",
    post_type: "video",
    display_hint: "video_embed",
    locality: "YouTube",
    labels: ["sample","video_embed","youtube"],
    image_url: $embed.thumbnail_url,
    external_url: ($embed | tojson)
  }')"

# YouTube — privacy-enhanced host (same video id as first; different embed host)
post "$(jq -n \
  --argjson embed '{
    "provider":"youtube",
    "video_id":"dQw4w9WgXcQ",
    "embed_url":"https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
    "watch_url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    "thumbnail_url":"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
    "channel_title":"Rick Astley"
  }' \
  '{
    title: "Sample: youtube-nocookie embed",
    body: "Second card — validates youtube-nocookie.com host (same clip as the first sample).",
    post_type: "video",
    display_hint: "video_embed",
    locality: "YouTube",
    labels: ["sample","video_embed","youtube"],
    image_url: $embed.thumbnail_url,
    external_url: ($embed | tojson)
  }')"

# Vimeo — public embed (no thumbnail_url in JSON; hero uses image_url)
post "$(jq -n \
  --argjson embed '{
    "provider":"vimeo",
    "video_id":"148751763",
    "embed_url":"https://player.vimeo.com/video/148751763",
    "watch_url":"https://vimeo.com/148751763",
    "channel_title":"Blender Foundation"
  }' \
  '{
    title: "Sample: Big Buck Bunny (Vimeo)",
    body: "Vimeo player embed test.",
    post_type: "video",
    display_hint: "video_embed",
    locality: "Vimeo",
    labels: ["sample","video_embed","vimeo"],
    image_url: "https://picsum.photos/seed/bbbvimeo/1280/720",
    external_url: ($embed | tojson)
  }')"

echo "Done. Pull the feed in the app or: curl -s -H \"Authorization: Bearer <firebase>\" \"${API_URL}/posts?limit=10\" | jq ."
