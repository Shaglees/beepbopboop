---
name: beepbopboop-images
description: Find or generate a publish-ready image URL for a BeepBopBoop post. Invoke as a subtask from any publishing skill. Handles real-image sourcing (Wikimedia, Panoramax, Google Places, Unsplash) and AI generation (Pollinations/flux, Flex.1, Nanobanana) with imgur re-hosting.
argument-hint: <real|ai|poster|auto> [topic] [locality] [lat] [lon]
allowed-tools: Bash(curl *), Bash(jq *), Bash(rm *), Bash(cat *)
---

# BeepBopBoop Images Skill

Your job is to return a fast-loading, permanent, high-quality image URL (and optional `images[]` array) to a caller that is about to publish a post. The caller is usually `beepbopboop-post`, `beepbopboop-news`, `beepbopboop-fashion`, or a sport skill.

## Why this is its own skill

The image pipeline has real edge cases (`User-Agent` header for Wikimedia, `LON,LAT` order for Panoramax, re-upload for signed Google URLs, prompt rules for AI). Inlining those into every other skill meant they were easy to skip, and most drift bugs (dead images, slow-loading endpoints, missing posters) came from that. This skill is the discoverable, testable, single-owner home for image sourcing.

## Inputs from caller

A caller should describe the target post:

- `mode`: `real` | `ai` | `poster` | `auto` (default)
- `topic` (short string — "cherry blossoms", "new game release", "matchup LAL v BOS")
- `locality` / `latitude` / `longitude` (optional; enables geographic tiers)
- `keywords` (optional array of 2–4 Unsplash keywords)
- `aesthetic_hint` (optional, e.g. "editorial", "candid", "minimal")
- `poster_hint` (optional URL from caller's research — used by poster mode)
- `fallback_ok`: `true` | `false`. Default `true`. When `false`, only the first tier that succeeds is returned; AI fallback is skipped.

## Step 0: Load config

Read `../_shared/CONFIG.md`. The keys this skill cares about:

- `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY`
- `BEEPBOPBOOP_IMGUR_CLIENT_ID`
- `BEEPBOPBOOP_GOOGLE_PLACES_KEY`
- `BEEPBOPBOOP_POLLINATIONS_TOKEN` (optional)

If `BEEPBOPBOOP_IMGUR_CLIENT_ID` is missing, Google Places and Pollinations tiers are unavailable; skip them.

## Step 1: Route by mode

| Mode | Read |
|---|---|
| `auto` (default) | Pick `poster` if `poster_hint` provided, else `real` if `latitude`+`longitude`, else `ai`. |
| `real` | `MODE_REAL.md` |
| `ai` | `MODE_AI.md` |
| `poster` | `MODE_POSTER.md` |

## Step 2: Try the chosen tiers in order

Each mode file describes its tiers; stop at the first that returns a non-empty URL. If none succeed and `fallback_ok=true`, cascade into `MODE_AI.md`. If still nothing, return `{ image_url: "", reason: "no_image_available" }`.

## Step 3: Return

Return `{ image_url, source, images? }` where:

- `image_url` — direct CDN URL, `.jpg`/`.png`/`.webp`.
- `source` — which tier succeeded (`wikimedia`, `panoramax`, `google_places`, `unsplash`, `pollinations`, `flex1`, `nanobanana`, `poster`).
- `images` — optional array of `{url, role, caption}` for outfit mode.

Never return a short-lived URL (Google Places signed URLs, Pollinations raw endpoints). Always re-host to imgur first.
