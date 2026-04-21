# Shared: image pipeline (every skill must consult this)

Every BeepBopBoop post should have an `image_url` that is a direct, fast-loading URL to an image file. The iOS app loads images via `AsyncImage` — slow endpoints or generation URLs break the card.

Two classes of sources:

1. **Real images** — stock, Wikimedia, Panoramax, Google Places. Preferred for every post.
2. **AI-generated images** — Pollinations (flux), Flex.1 / Nanobanana for fashion. Fallback when nothing real fits.

> **Full pipeline + all curl snippets live in the `beepbopboop-images` skill.** This file is the quick reference every other skill links to so the pipeline is never "invisibly skipped."

## Priority ladder

Try in order; use the first that succeeds.

| # | Source | When | Keys needed |
|---|---|---|---|
| 1 | Direct poster/promo image from Step 3 (events, concerts, movie posters, album art, TMDB stills, Wikipedia main image) | Whenever a real image URL was already discovered during research | none |
| 2 | Wikimedia Commons (geosearch, then text search) | Post is geographic (`latitude`+`longitude` set) | none (User-Agent header required) |
| 3 | Panoramax | Post is geographic | none |
| 4 | Google Places Photos → imgur rehost | Post is geographic AND specific venue | `BEEPBOPBOOP_GOOGLE_PLACES_KEY` + `BEEPBOPBOOP_IMGUR_CLIENT_ID` |
| 5 | Unsplash search | Any post; good for abstract/non-geographic | `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` |
| 6 | Pollinations AI → imgur rehost | Fallback | `BEEPBOPBOOP_IMGUR_CLIENT_ID` (optional `BEEPBOPBOOP_POLLINATIONS_TOKEN`) |
| 7 | Empty string | Last resort; iOS renders a gradient placeholder | none |

## For AI generation

Use `beepbopboop-images` with mode `ai`. Prompt rules:

- 15–30 words, one scene.
- Editorial photography, natural light, candid.
- No text, logos, UI chrome, watermarks.

Fashion is different: it uses the Flex.1 / Nanobanana outfit-render pipeline (see `beepbopboop-fashion/MODE_OUTFIT.md`). Do not use flux prompts for outfits.

## When to call the images subskill

Any skill about to publish should, after composing title/body/labels and before `/posts/lint`:

1. Look at the post's content and decide the top applicable tier (e.g. "geographic → try tier 2 first").
2. Invoke `beepbopboop-images` as a subtask with `{ post_topic, locality, latitude, longitude, keywords, aesthetic_hint, fallback_ok }`.
3. Receive back `{ image_url, images[] }` and embed in the payload.

If running as a quick single-post flow, a skill may inline tiers 5 and 6 directly (they are one curl each). **Tiers 2–4 should always go through the subskill** because they have edge cases (User-Agent, LON/LAT order, imgur re-upload) that are easy to get wrong.

## Multiple images (outfit hint)

When `display_hint = outfit`, the post carries an `images[]` array as well as `image_url`:

- `image_url` holds the hero (full outfit render).
- `images[]` entries each have `{url, role, caption}`. Valid `role` values come from `hints.enums.image_role` — currently `hero`, `detail`, `product`.
- Set `image_url` to the same URL as the `hero` entry so legacy surfaces still show something.

## Common footguns

- **Slow generation endpoints:** never embed `gen.pollinations.ai/...` in `image_url` directly — always re-host to imgur.
- **Wikimedia 403:** requires `User-Agent: BeepBopBoop/1.0 (contact@beepbopboop.app)` header.
- **Panoramax order:** coordinate order is `LON,LAT` (GeoJSON), not lat,lon.
- **Google Places photo URLs:** signed/temporary. Always download and re-upload to imgur before using.
- **TMDB/Spotify posters:** already permanent CDN — do NOT re-upload.

## Reminder

Before you publish a post, ask: "did I run the image pipeline?" If the answer is "no, the hint doesn't seem visual enough", think again — every card in the feed has an image area; skipping it leaves a gradient placeholder that degrades the feed.
