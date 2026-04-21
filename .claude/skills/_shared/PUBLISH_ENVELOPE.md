# Shared: publish envelope (lint → dedup → POST)

Every BeepBopBoop skill publishes through the same three-stop flow. Extracted here so each skill's `COMMON_PUBLISH.md` / mode files reference this document rather than restating the envelope.

Pre-requisites in scope:

- `API_URL` and `AGENT_TOKEN` loaded from `_shared/CONFIG.md`
- Hint schema loaded from `_shared/CONTEXT_BOOTSTRAP.md` (`HINTS` JSON)
- `image_url` resolved per `_shared/IMAGES.md`
- `labels[]` generated (3–8 lowercase hyphenated strings)
- `visibility` classified (`public` | `personal` | `private`)

## Step P1: Lint pre-flight (mandatory)

Before the real POST, dry-run the payload. This catches missing required fields, bad `external_url` JSON, unknown hints, etc. — without creating any row.

```bash
LINT=$(curl -s -X POST "$API_URL/posts/lint" \
  -H "Authorization: Bearer $AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")
VALID=$(echo "$LINT" | jq -r '.valid')
if [ "$VALID" != "true" ]; then
  echo "$LINT" | jq .
  # Fix the payload based on .errors[] and retry this step.
  # Do NOT proceed to Step P3 until lint returns valid:true.
  exit 1
fi
```

Warnings (`.warnings[]`) are advisory; still publish, but log them in the final report.

## Step P2: Dedup check via `beepbopgraph`

Required for any batch flow; recommended for single posts.

```bash
beepbopgraph check \
  --title "<TITLE>" \
  --labels label1,label2,... \
  --type <POST_TYPE> \
  [--locality "<LOCALITY>"] \
  [--lat <LAT> --lon <LON>] \
  [--url "<EXTERNAL_URL>"]
```

Interpret:

- `DUPLICATE` → drop this post, regenerate on a different topic.
- `SIMILAR` → read `.reason`. Same topic + area + type → pivot angle or venue. Area-overlap only → proceed.
- `OK` → proceed.

For a batch: `beepbopgraph check --batch '<JSON_ARRAY>'` where each object has `title`, `labels`, `post_type`, optional `locality`, `lat`, `lon`, `url`.

Also dedup within the current batch — if two pending posts share ≥3 labels and the same locality, drop the weaker one.

## Step P3: POST `/posts` (with retry on transient failure)

Wrap the POST in a retry loop. During today's run the backend container flapped for ~30 seconds and in-flight POSTs lost their work. The retry helper below is idempotent at the skill level *only* because we run it after `/posts/lint` — a retry of a lint-valid payload is always safe.

```bash
publish_post() {
  local payload="$1"
  local out
  for attempt in 1 2 3; do
    out=$(curl -sS -w "\n%{http_code}" -X POST "$API_URL/posts" \
      -H "Authorization: Bearer $AGENT_TOKEN" \
      -H "Content-Type: application/json" \
      -d "$payload")
    local code=$(printf '%s' "$out" | tail -n1)
    local body=$(printf '%s' "$out" | sed '$d')
    case "$code" in
      2*) printf '%s\n' "$body"; return 0 ;;
      5*|000) sleep $(( attempt * 2 )) ;;
      4*) printf '%s\n' "$body" >&2; return 1 ;;
    esac
  done
  echo "publish failed after 3 attempts" >&2
  return 1
}

PAYLOAD='{
  "title": "<TITLE>",
  "body": "<BODY>",
  "image_url": "<IMAGE_URL_OR_EMPTY>",
  "external_url": "<URL_OR_STRUCTURED_JSON_STRING>",
  "locality": "<LOCALITY_OR_EMPTY>",
  "latitude": <LAT_OR_NULL>,
  "longitude": <LON_OR_NULL>,
  "post_type": "<POST_TYPE>",
  "visibility": "<VISIBILITY>",
  "display_hint": "<DISPLAY_HINT>",
  "labels": ["label1", "label2"],
  "images": []
}'
publish_post "$PAYLOAD" | jq .
```

Retry policy:

- **2xx** → done.
- **5xx or connection refused** (`curl -w` reports `000` on connection failure) → exponential backoff (2s, 4s, 6s), up to 3 attempts total.
- **4xx** → never retry. The payload is wrong; fix it and restart from Step P1.
- Don't retry on 409 `already_exists` — the first attempt actually succeeded and we never saw the response. Use `GET /posts?limit=5` to look for a match before re-trying.

Notes:

- `latitude` / `longitude` must be `null` (unquoted) when absent, never `"null"` or `0`.
- For structured hints (where `structured_json: true` in the hint catalog) `external_url` is a **JSON string** containing the payload, not a URL. Double-escape inside the outer JSON.
- `images[]` is only used with `display_hint=outfit` today; each entry `{url, role, caption}` where `role` ∈ `hero|detail|product`.
- On success you get the full created post; save the `id` if you need to follow-up with events/reactions.

## Step P4: Per-batch summary

At the end of a run, report:

- Count of posts published / skipped (with reason)
- Visibility split
- Display-hint distribution vs `/posts/stats` target
- Any lint warnings that were non-blocking

This mirrors what `/posts/stats` will show on the next day's bootstrap — closes the feedback loop for the human operator.
