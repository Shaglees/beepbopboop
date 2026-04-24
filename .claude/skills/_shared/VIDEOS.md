# Shared: video catalog (embed-ready video posts)

The backend maintains a historical catalog of YouTube / Vimeo / Dailymotion / Twitch / Streamable videos, sourced from ingest of wimp.com's RSS feed and other curated sources. Two HTTP endpoints expose this catalog to skills composing `display_hint: video_embed` posts.

## Why this exists

Fresh video content is hard for an agent to find on its own. Scraping YouTube directly is quota-heavy, random, and rarely "interesting" — it lacks curation. Wimp.com is a long-running human-curated feed of ~5 interesting short videos per day; our ingest turns that curation into a local cache we can hand to skills.

Every cached row carries enough structured data for a well-formed post:

- `provider` + `provider_video_id` — stable catalog key
- `watch_url` + `embed_url` — both shapes, ready to drop into a `video_embed` payload
- `title` — upstream (YouTube/Vimeo) title when we have oEmbed enrichment, otherwise the scraped page title
- `channel_title` — creator's channel name when oEmbed succeeded
- `thumbnail_url` — 1280×720 og:image or hqdefault.jpg
- `published_at` — when the upstream video's wimp post was published
- `duration_sec` — set for Vimeo; nil for YouTube (oEmbed doesn't expose it)
- `labels[]` — editorial categories from the source feed (e.g. `["dogs", "funny", "technology"]`)
- `embed_health` — `"ok"` | `"dead"` | `"unknown"`; a background worker revalidates these every 6h

## Two endpoints, two use cases

### `GET /videos` — browse the catalog

Use this when the skill wants to pick a video *itself*, typically based on the user's current interests or the day's theme.

```bash
API="$BEEPBOPBOOP_API_URL"
AUTH="Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN"

# 5 newest healthy videos tagged "funny" or "dogs", YouTube only
curl -s -H "$AUTH" "$API/videos?limit=5&labels=funny,dogs&providers=youtube"
```

Query parameters (all optional):

| param           | default | description                                                        |
|-----------------|---------|--------------------------------------------------------------------|
| `limit`         | 20      | 1..100                                                             |
| `labels`        | —       | CSV. ANY-match include filter (e.g. `labels=dogs,funny`)           |
| `exclude_labels`| —       | CSV. NONE-match exclude filter                                      |
| `providers`     | —       | CSV. Whitelist providers (`youtube`, `vimeo`, `dailymotion`, etc.) |
| `healthy_only`  | `true`  | only return `embed_health='ok'`. `dead` rows are ALWAYS excluded.  |

Response shape:

```json
{
  "videos": [
    {
      "id": "vid_abc123",
      "provider": "youtube",
      "provider_video_id": "VFHQCX1pezY",
      "watch_url": "https://www.youtube.com/watch?v=VFHQCX1pezY",
      "embed_url": "https://www.youtube.com/embed/VFHQCX1pezY",
      "title": "Owner Tells Dog To Go Back Inside Via Spotlight Cam | RingTV",
      "channel_title": "Ring",
      "thumbnail_url": "https://i.ytimg.com/vi/VFHQCX1pezY/hqdefault.jpg",
      "labels": ["dogs", "funny", "technology"],
      "published_at": "2026-04-21T14:00:07Z",
      "embed_health": "ok"
    }
  ],
  "diagnostics": {
    "requested_limit": 5,
    "returned_count": 1,
    "include_labels": ["funny", "dogs"],
    "healthy_only": true,
    "personalized": false
  }
}
```

### `GET /videos/for-me` — personalized selection

Use this when the skill just wants "give me a good video for this user right now" — the server applies:

1. 180-day per-user dedup (won't return a video we already posted as a `video_embed` for this user)
2. User-embedding similarity ranking (if the user has an embedding; falls back to freshness)
3. Healthy-only (always)

```bash
curl -s -H "$AUTH" "$API/videos/for-me?limit=3"
```

Same response shape; `diagnostics.personalized` is `true`.

## Composing the post

Once you have a video, the `video_embed` payload is straightforward:

```json
{
  "display_hint": "video_embed",
  "title": "<video.title>",
  "body": "<short editorial framing — 1-2 sentences of WHY this is worth watching>",
  "labels": ["<video.labels joined or a curated subset>"],
  "external_url": {
    "url": "<video.watch_url>",
    "provider": "<video.provider>",
    "provider_video_id": "<video.provider_video_id>",
    "embed_url": "<video.embed_url>",
    "thumbnail_url": "<video.thumbnail_url>",
    "channel": "<video.channel_title>"
  }
}
```

**Always run `POST /posts/lint` first** (see `_shared/PUBLISH_ENVELOPE.md`). The server will emit a warning if the referenced video's `embed_health` is `unknown` — that's informational; proceed unless it's `dead`.

## Content quality rules

1. **Respect the user's reactions.** If `/reactions/summary` shows `less:video_embed`, skip this entirely that day and do something else.
2. **Don't overfill.** If `/posts/stats` shows `video_embed` is already saturated this week, post at most one.
3. **Add editorial context.** The `body` field is where you earn your keep — explain what's interesting, tie it to the user's interests, don't just re-state the title.
4. **Pick fresh.** Prefer `published_at` within the last 7 days. `GET /videos` already orders by this.

## Operational notes (for maintainers)

- Ingest runs via `backend/cmd/wimpingest` (one-shot CLI). Adding a scheduled worker is a follow-up.
- `backend/internal/ingest/wimp/orchestrator.go` is the single entry point. It reuses the same parser as the Wayback-backed historical backfill.
- oEmbed enrichment is best-effort: a failure doesn't fail the ingest. Rows persisted without enrichment still have the scraped wimp title and og:image.
- Provider coverage: YouTube, Vimeo, Dailymotion, Twitch clips, Streamable, and raw mp4 fallback. Facebook/Instagram/TikTok/X deliberately omitted — they don't expose stable anonymous oEmbed.
