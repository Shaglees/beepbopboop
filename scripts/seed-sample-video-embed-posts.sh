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

# YouTube — "Me at the zoo" (jNQXAC9IVRw): first YouTube upload; historically embed-friendly.
# Many music videos / meme reuploads block embedding in third-party apps — avoid those.
post "$(jq -n \
  --argjson embed '{
    "provider":"youtube",
    "video_id":"jNQXAC9IVRw",
    "embed_url":"https://www.youtube.com/embed/jNQXAC9IVRw",
    "watch_url":"https://www.youtube.com/watch?v=jNQXAC9IVRw",
    "thumbnail_url":"https://i.ytimg.com/vi/jNQXAC9IVRw/hqdefault.jpg",
    "channel_title":"jawed",
    "supports_preview_cap": true
  }' \
  '{
    title: "Sample: Me at the zoo (YouTube — embed-friendly)",
    body: "video_embed test — thumbnail + sheet player. Share uses watch_url. Classic viral clip.",
    post_type: "video",
    display_hint: "video_embed",
    locality: "YouTube",
    labels: ["sample","video_embed","youtube","meme"],
    image_url: $embed.thumbnail_url,
    external_url: ($embed | tojson)
  }')"

# YouTube — privacy-enhanced host (same video id; different embed host)
post "$(jq -n \
  --argjson embed '{
    "provider":"youtube",
    "video_id":"jNQXAC9IVRw",
    "embed_url":"https://www.youtube-nocookie.com/embed/jNQXAC9IVRw",
    "watch_url":"https://www.youtube.com/watch?v=jNQXAC9IVRw",
    "thumbnail_url":"https://i.ytimg.com/vi/jNQXAC9IVRw/hqdefault.jpg",
    "channel_title":"jawed",
    "supports_preview_cap": true
  }' \
  '{
    title: "Sample: Me at the zoo (youtube-nocookie)",
    body: "Same clip as the first sample — validates youtube-nocookie.com embed host.",
    post_type: "video",
    display_hint: "video_embed",
    locality: "YouTube",
    labels: ["sample","video_embed","youtube"],
    image_url: $embed.thumbnail_url,
    external_url: ($embed | tojson)
  }')"

# Vimeo — Blender’s Big Buck Bunny (ID 1084537). Older sample used 148751763 which 404s on Vimeo now.
post "$(jq -n \
  --argjson embed '{
    "provider":"vimeo",
    "video_id":"1084537",
    "embed_url":"https://player.vimeo.com/video/1084537",
    "watch_url":"https://vimeo.com/1084537",
    "thumbnail_url":"https://i.vimeocdn.com/video/20963649-f02817456fc48e7c317ef4c07ba259cd4b40a3649bd8eb50a4418b59ec3f5af5-d_640",
    "channel_title":"Blender",
    "supports_preview_cap": true
  }' \
  '{
    title: "Sample: Big Buck Bunny (Vimeo)",
    body: "Vimeo embed test — verify ID with: curl -s \"https://vimeo.com/api/oembed.json?url=https://vimeo.com/1084537\" | jq .title",
    post_type: "video",
    display_hint: "video_embed",
    locality: "Vimeo",
    labels: ["sample","video_embed","vimeo"],
    image_url: $embed.thumbnail_url,
    external_url: ($embed | tojson)
  }')"

echo "Done. Pull the feed in the app or: curl -s -H \"Authorization: Bearer <firebase>\" \"${API_URL}/posts?limit=10\" | jq ."
