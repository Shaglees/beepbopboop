# Shared: context bootstrap (always run after config load)

After Step 0 (config load, see `_shared/CONFIG.md`) and **before** any mode-specific work, every BeepBopBoop skill fetches a small bundle of "what does the server know about me and about itself" context. This keeps the router-based skill structure from hiding the server's capabilities behind a mode file.

The bootstrap is intentionally small: four GETs, at most ~50 KB of JSON total, all cacheable for the rest of the session.

## Why this exists

Without this step a skill would route straight into (say) `MODE_SPORTS.md` and compose a `matchup` post — but it would not know:

- that the `matchup` display hint requires a JSON payload with a `gameTime` field (it would invent one),
- that this user has reacted `not_for_me` to every `sports` post this week (the skill would publish anyway),
- that the user is already saturated on `hockey` and under-posted on `food` (the feed would keep drifting),
- that the Petfinder/beepbopgraph/image toolchain exists at all.

Bootstrap answers all four before any mode runs.

## Step 0d: Fetch server capabilities + user spread

Run these four calls. Each one is independent; fire them in parallel with `&` and `wait`.

```bash
API="$BEEPBOPBOOP_API_URL"
AUTH="Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN"

HINTS=$(curl -s -H "$AUTH" "$API/posts/hints")
STATS=$(curl -s -H "$AUTH" "$API/posts/stats")
REACT=$(curl -s -H "$AUTH" "$API/reactions/summary")
EVENTS=$(curl -s -H "$AUTH" "$API/events/summary")
```

If any of them returns a non-JSON body or an HTTP error, log a warning and continue — none of them are strictly required to post, but every mode should try to honor them.

## What each response gives you

### `/posts/hints` — authoritative payload schema

Top-level keys you care about:

- `display_hints[]` — every hint the server accepts, with:
    - `hint` — the string to put in `display_hint`
    - `post_type` — the default post_type that pairs with this hint
    - `structured_json` — if `true`, `external_url` is JSON, not a URL
    - `required_fields` — flat list; names prefixed `external_url:` refer to keys inside that JSON blob
    - `example` — a full, lint-clean payload you can copy-shape-and-modify
    - `renders.card` — the SwiftUI card the iOS client draws for this hint (e.g. `PlaceCard`, `DateCard`)
    - `renders.uses_fields` — which post fields that card reads
    - `renders.ignores_fields` — fields the card silently drops (e.g. `PlaceCard` ignores `external_url`)
    - `pick_when` / `avoid_when` — heuristics for when this hint is or isn't a fit
- `enums.display_hint` / `enums.post_type` / `enums.visibility` / `enums.image_role` — never hard-code these
- `endpoints.*` — map of named endpoints (create_post, lint_post, post_stats, events_summary, reactions_summary, sports_scores, creators_nearby). The authoritative set of things you can call.
- `docs.images` — pointer to `_shared/IMAGES.md`; do not skip image sourcing.
- `docs.publish_flow` — **always POST `/posts/lint` before `/posts`**.

**Contract:** pick the hint that matches your content, copy the example, edit title/body/labels/external_url values, lint, publish. If a hint has `structured_json: true`, you MUST produce an `external_url` string whose JSON parses to something that satisfies `required_fields`.

**Before picking a hint**, scan `renders.ignores_fields` — if the field carrying your CTA is in that list, pick a different hint (or inline the data into body). See `_shared/HINT_DECISION.md` for the full decision tree.

### `/posts/stats` — your own posting spread

Returns `periods[]` for 7/30/90-day windows. Each period has counts by `post_type`, `display_hint`, and top `labels`.

Use it to:

- pick under-represented labels/hints in batch mode (if `food` is 1/30 but `sports` is 14/30, add food, subtract sports),
- avoid re-posting the same labels three days in a row (check the 7-day window),
- confirm the user's claim "I post X" — if stats disagree with the profile, tell them.

### `/reactions/summary` — user feedback

Aggregated `more` / `less` / `stale` / `not_for_me` reactions per label/topic.

- Strongly down-weight (or skip) labels that the user has reacted `not_for_me` to.
- Prefer labels with `more` reactions when you have a choice.
- `stale` = user wants variety; rotate subtopics within that area.

### `/events/summary` — engagement signal

Views, saves, dwell-time grouped per post or per label. This is the same feature set that feeds the ForYou ML ranking. Use it as a secondary signal: if a label gets lots of views but zero saves, that content is shallow — go deeper or switch.

### `/videos` and `/videos/for-me` — embed-ready video catalog

When a skill wants to post a `video_embed`, do NOT scrape YouTube / wimp.com / etc. directly. Call the video catalog:

- `GET /videos` — simple list, filter by `labels`, `providers`, `healthy_only`. Agent picks one.
- `GET /videos/for-me` — personalized selection, applies 180-day per-user dedup + embedding similarity ranking.

Each returned row already has `watch_url`, `embed_url`, `title`, `channel_title`, `thumbnail_url`, `labels`, and `embed_health` — enough to compose a lint-clean `video_embed` payload. See `_shared/VIDEOS.md` for the full contract and a template payload.

The catalog is fed by daily ingest of wimp.com's RSS feed (run manually via `backend/cmd/wimpingest` — a scheduled worker is a follow-up). If the catalog is empty / stale, a skill should degrade gracefully to a non-video post rather than invent a URL.

## What to pin into the rest of the session

After bootstrap, the calling skill should have the following in working memory for the rest of its turn:

1. **Hint catalog** — full `display_hints[]` array; every `MODE_*.md` now references hint examples from this bundle rather than inline snippets.
2. **Enums** — `VALID_POST_TYPES`, `VALID_VISIBILITY`, `VALID_IMAGE_ROLES`, `VALID_DISPLAY_HINTS`.
3. **Spread summary** — top 5 over-represented labels, top 5 under-represented labels.
4. **Feedback summary** — `not_for_me_labels`, `more_labels`, `stale_labels`.
5. **Capabilities** — the `endpoints` map. Modes should prefer named endpoints over invented paths.

## How mode files reference this bundle

Mode files will say:

> From the hint catalog loaded in Step 0d, take the entry for `matchup`. Copy `example`, override title/body/labels from Step 2, and substitute your `gameTime` / home / away values.

No mode file should include its own inline hint schema tables any more — that's what caused the drift this refactor is fixing.

## Related shared docs

Every compose step should at minimum consult:

- `_shared/HINT_DECISION.md` — decision tree for picking the right `display_hint`. The bug that caused a hike to render as a dated event lived here.
- `_shared/IMAGES.md` — image source ladder + Tier 2 relevance guard.
- `_shared/GEOCODE.md` — Nominatim fallback ladder + label-saturation lint (drop labels that are already over-posted this week).
- `_shared/PUBLISH_ENVELOPE.md` — lint → dedup → POST, with retry-on-5xx helper.
- `_shared/VIDEOS.md` — `/videos` + `/videos/for-me` contract for composing `video_embed` posts.

## If bootstrap fails

- Missing `/posts/hints` response → fall back to `COMMON_PUBLISH.md` display-hint table (documented as "legacy; remove once every deployment exposes /posts/hints").
- Missing `/posts/stats` → skip spread balancing; post as planned.
- Missing `/reactions/summary` / `/events/summary` → proceed but warn in the final report.

The bootstrap is non-fatal by design — a user self-hosting an older backend still gets working skills.
