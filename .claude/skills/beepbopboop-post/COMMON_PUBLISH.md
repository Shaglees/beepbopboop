# Common publish / dedup / label contract

Every BeepBopBoop post mode ends by running Steps 4a → 4b → 4c → 4d → 5 → 5b → 6. This file is the single source of truth for those shared steps. Individual `MODE_*.md` files reference this document instead of repeating the logic.

> **Before reading the rest of this file**, confirm `HINTS`/`STATS`/`REACT`/`EVENTS` are loaded per `../_shared/CONTEXT_BOOTSTRAP.md`. Steps 4b–5 rely on them.

---

## Step 4a: Classify visibility

Evaluate visibility AFTER generating post content (since the body text determines the result):

| Content source / characteristic | Visibility | Why |
|--------------------------------|-----------|-----|
| Calendar mode (CL1–CL3) | `private` | Calendar events reveal personal schedule |
| Post body references family member names from `BEEPBOPBOOP_FAMILY` | `personal` | "Maja would love this" is personal |
| Post body contains "from your door", "from home", "X minutes from here" | `personal` | Reveals home location |
| Post body contains user's street/address | `personal` | Reveals home address |
| Comparison mode about a personal topic (e.g., "best coffee near me") | `personal` | Location-specific |
| Weather mode with family suggestions | `personal` | Combines location + family |
| All other posts | `public` | Safe for cross-user discovery |

---

## Hint diversity requirement (batch mode)

When publishing a batch of 8–10 posts, your `display_hint` selection **MUST** include ALL of the following hints. Each one is individually required — do not skip any:

| Required hint | Notes |
|---|---|
| `matchup` | Sports upcoming game — requires structured `external_url` JSON |
| `destination` | Travel/place post — requires structured `external_url` JSON |
| `entertainment` | Film, TV, music, podcast, or event — requires structured `external_url` JSON |
| `fitness` | Workout or fitness activity — requires structured `external_url` JSON |
| `comparison` | Ranked 3+ items — requires structured `external_url` JSON |
| `concert` | Live music event — requires structured `external_url` JSON |
| `restaurant` | Dining/food venue — requires structured `external_url` JSON |
| `album` | Music album review or recommendation — requires structured `external_url` JSON |

These 8 hints all require structured `external_url` JSON. See `../_shared/HINT_DECISION.md` for the exact key schemas and fill-in templates. **Use the templates exactly — wrong keys return a 422.**

With 8 required structured hints and a batch of 8–10 posts, most or all posts will have a structured hint. Fill remaining slots with `article`, `card`, `digest`, `event`, `place`, `video`, etc.

Before finalising your post plan, check this list one by one. If any of the 8 is missing, add a post that uses it.

---

## Step 4b: Find or generate post image

Every post should have an image. The iOS app loads images via `AsyncImage`, so `image_url` must be a direct, fast-loading URL to an image file — not a slow generation endpoint.

**Canonical reference: `../_shared/IMAGES.md`**, which documents the priority ladder and when to invoke the `../beepbopboop-images` subskill. The inline pipeline below is kept for quick lookups; if you add a new source or discover an edge case, update the subskill and the shared doc — not this file.

**Routing decision:**
- Post is **geographic** (`latitude` and `longitude` both set) → try priorities 1–4 in order, then 5, then 6.
- Post is **non-geographic** (no coordinates) → skip directly to **priority 5 (Unsplash)**. Do NOT jump to priority 6 (AI generator) without trying Unsplash first. Unsplash has images for every topic.

> **AI generators are last resort (priority 6) — not a default.** For non-geographic editorial content (articles, digests, music, entertainment), Unsplash always has coverage. Only reach priority 6 if priority 5 returned a null/error. See `../_shared/IMAGES.md` for the full tier breakdown.

Try the pipeline in order, using the first that succeeds.

### Priority 1 — Real poster/promo image (events only)

If Step 3 found a direct image URL (`.jpg`, `.png`, `.webp`) from a venue website or ticketing platform, use it. Real promotional images are always better than stock or AI-generated.

### Priority 2 — Wikimedia Commons (geographic posts only)

No API key required — just a `User-Agent` header (403 without it).

**2a — Geosearch by coordinates** (geotagged images within 500m):

```bash
WC_IMG=$(curl -s -H "User-Agent: BeepBopBoop/1.0 (contact@beepbopboop.app)" \
  "https://commons.wikimedia.org/w/api.php?action=query&format=json&generator=geosearch&ggsprimary=all&ggsnamespace=6&ggsradius=500&ggscoord=LAT%7CLON&ggslimit=5&prop=imageinfo&iilimit=1&iiprop=url&iiurlwidth=1024" \
  | jq -r '[.query.pages[] | select(.imageinfo[0].thumburl)] | sort_by(.index) | .[0].imageinfo[0].thumburl // empty')
```

Replace `LAT` and `LON` with the post's coordinates; `%7C` is the URL-encoded pipe.

**2b — Text search by name** (fallback if geosearch returns nothing):

```bash
if [ -z "$WC_IMG" ]; then
  WC_IMG=$(curl -s -H "User-Agent: BeepBopBoop/1.0 (contact@beepbopboop.app)" \
    "https://commons.wikimedia.org/w/api.php?action=query&format=json&generator=search&gsrnamespace=6&gsrsearch=PLACE_NAME+CITY&gsrlimit=5&prop=imageinfo&iilimit=1&iiprop=url&iiurlwidth=1024" \
    | jq -r '[.query.pages[] | select(.imageinfo[0].thumburl)] | sort_by(.index) | .[0].imageinfo[0].thumburl // empty')
fi
```

Use `thumburl` at 1024px width (not full `url`). URLs are permanent Wikimedia CDN links. Strong coverage for landmarks, museums, parks.

### Priority 3 — Panoramax (geographic posts only)

Street-level exterior imagery by coordinates. No auth.

```bash
PX_IMG=$(curl -s "https://api.panoramax.xyz/api/search?place_position=LON,LAT&place_distance=0-100&limit=1" \
  | jq -r '.features[0].assets.sd.href // empty')
```

Coordinate order is **LON,LAT** (GeoJSON). Use the `sd` asset (2048px). Strong in France/EU, sparse in North America. Exterior perspective only.

### Priority 4 — Google Places Photos (geographic posts, requires keys)

Requires both `BEEPBOPBOOP_GOOGLE_PLACES_KEY` and `BEEPBOPBOOP_IMGUR_CLIENT_ID` (Google photo URLs are signed/temporary, must be re-uploaded for permanence).

**Step 1 — find place and get photo name:**

```bash
GP_PHOTO_NAME=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Goog-Api-Key: $BEEPBOPBOOP_GOOGLE_PLACES_KEY" \
  -H "X-Goog-FieldMask: places.photos" \
  -d "{\"textQuery\": \"PLACE_NAME CITY\"}" \
  "https://places.googleapis.com/v1/places:searchText" \
  | jq -r '.places[0].photos[0].name // empty')
```

**Step 2 — download and re-upload to imgur:**

```bash
if [ -n "$GP_PHOTO_NAME" ] && [ -n "$BEEPBOPBOOP_IMGUR_CLIENT_ID" ]; then
  curl -s -L -o /tmp/bbp_google_photo.jpg \
    "https://places.googleapis.com/v1/${GP_PHOTO_NAME}/media?key=$BEEPBOPBOOP_GOOGLE_PLACES_KEY&maxWidthPx=1024"
  GP_IMG=$(curl -s -X POST "https://api.imgur.com/3/image" \
    -H "Authorization: Client-ID $BEEPBOPBOOP_IMGUR_CLIENT_ID" \
    -F "image=@/tmp/bbp_google_photo.jpg" \
    -F "type=file" | jq -r '.data.link // empty')
  rm -f /tmp/bbp_google_photo.jpg
fi
```

Cost: ~$0.04/place (free $200/month covers ~5000 lookups). Best global venue coverage of any source.

### Priority 5 — Unsplash search (if `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` set)

Fallback for all posts, geographic or not. Best option for non-geographic content (articles, abstract ideas).

```bash
curl -s "https://api.unsplash.com/search/photos?query=SEARCH_KEYWORDS&per_page=1&orientation=landscape" \
  -H "Authorization: Client-ID <UNSPLASH_ACCESS_KEY>" | jq -r '.results[0].urls.regular'
```

**Keyword rules:**
- 2–4 concrete, visual keywords from the post topic
- Include setting/locale when it improves relevance
- Prefer specific nouns over abstract concepts

| Post topic | Search keywords |
|------------|----------------|
| Coffee/cafe | `cafe coffee latte morning` |
| Cherry blossoms | `cherry blossom street spring pink` |
| Hockey game | `ice hockey arena crowd` |
| Museum visit | `museum exhibition gallery interior` |
| AI article | `artificial intelligence technology abstract` |
| Farmers market | `farmers market produce outdoor morning` |
| Theatre show | `theatre stage performance spotlight` |
| Park/hiking | `hiking trail nature forest` |
| Restaurant | `restaurant dining table food` |
| Beach/ocean | `beach ocean waves coast` |

Unsplash CDN URLs are fast and permanent. If API returns `null`, fall through to priority 6.

### Priority 6 — Pollinations AI → imgur (last resort; only if Unsplash also failed)

> ⚠️ **Only reach Priority 6 if Priority 5 (Unsplash) returned a null/error result.** For non-geographic editorial content, Unsplash has coverage for every topic — if you are reaching Priority 6 for an abstract tech article, a movie post, a digest, or a brief, you skipped Priority 5. Go back and try Unsplash.
>
> For geographic posts where geo-sources returned nothing and Unsplash also failed, Priority 6 is acceptable.

Generate a custom AI image and upload to imgur for reliable hosting.

**Step 1 — generate image:**

Craft a short, vivid scene description (15–30 words). No text/logos/UI elements. Style: editorial photography, natural light, candid.

```bash
curl -s -L -o /tmp/bbp_post_image.jpg "https://gen.pollinations.ai/image/URL_ENCODED_PROMPT?width=1024&height=768&model=flux&seed=-1&quality=medium&nologo=true"
```

**Step 2 — upload to imgur:**

```bash
curl -s -X POST "https://api.imgur.com/3/image" \
  -H "Authorization: Client-ID <IMGUR_CLIENT_ID>" \
  -F "image=@/tmp/bbp_post_image.jpg" \
  -F "type=file" | jq -r '.data.link'
```

Clean up: `rm -f /tmp/bbp_post_image.jpg`

**Example prompts:**
- Coffee → `"Warm morning light through cafe window, single origin pour over coffee, wooden counter, Pacific Northwest"`
- Market → `"Outdoor farmers market stalls with colorful produce, morning crowd, spring sunshine"`
- Event → `"Theatre marquee at dusk, warm glow from lobby windows, people arriving for evening show"`
- AI article → `"Abstract visualization of neural network connections, dark background, glowing nodes, futuristic"`
- YouTube video → `"Content creator workspace, multiple monitors, camera setup, warm desk lamp, modern studio"`

### Priority 7 — No image

Set `image_url` to empty string; iOS shows a gradient placeholder.

**When publishing multiple posts:** run all image fetches in parallel before publishing.

---

## Step 4c: Generate labels

Generate 3–8 labels per post. Labels exist for **cross-user interest matching** — think "would another person search for or follow this topic?"

**Source 1 — Post type label** (always):
- `event` → `["event"]`
- `place` → `["place"]`
- `discovery` → `["discovery"]`
- `article` → `["article"]`
- `video` → `["video"]`

**Source 2 — Category labels** (2–4):

| Topic area | Example labels |
|------------|---------------|
| Coffee/cafe | `coffee`, `cafe`, `specialty-coffee` |
| Restaurant/food | `restaurant`, `food`, cuisine type (`italian`, `sushi`) |
| Sports | `sports`, `live-events`, sport name (`hockey`) |
| Theatre/music | `theatre`, `performing-arts`, `live-music`, `concert` |
| AI/tech | `ai`, `machine-learning`, `tech`, `software` |
| Startup/business | `startup`, `business`, `investing` |
| Trending/viral | `trending`, `pop-culture`, `viral`, `world-news` |
| Weather/seasonal | `weather`, `rainy-day`, `seasonal`, season name |

For other topics, use lowercase hyphenated category terms that another user might follow.

**Source 3 — Specificity labels** (1–3):
- Content source / publication (e.g., `hacker-news`, `fireship`, `product-hunt`)
- Audience / context (`kid-friendly`, `date-night`, `budget`, `free`, `outdoor-seating`)
- Activity details (`indoor`, `outdoor`, `morning`, `evening`, `weekend`)
- Do NOT use venue-specific names as labels (venues are matched by GPS, not labels).

**Format:** lowercase, hyphenated, no duplicates, English only.

---

## Step 4d: Dedup check via `beepbopgraph`

**Single-post:**

```bash
beepbopgraph check --title "<TITLE>" --labels <LABEL1>,<LABEL2>,... --type <POST_TYPE> [--locality "<LOCALITY>"] [--lat <LAT> --lon <LON>] [--url "<EXTERNAL_URL>"]
```

**Batch:**

```bash
beepbopgraph check --batch '<JSON_ARRAY>'
```

Each object in the array: `title`, `labels` (array), `post_type`, optional `locality`, `lat`, `lon`, `url`.

**Interpret:**
- `DUPLICATE` → drop this post, generate a replacement on a different topic
- `SIMILAR` → read `reason`. Same topic+area+type → pivot angle/venue. Area overlap only → proceed.
- `OK` → proceed

Also dedup within the current batch — if two pending posts have high label overlap, drop the weaker one.

---

## Pre-publish image validation (run before Step 5)

Before publishing any post, answer these three questions. If any answer is NO, fix it first.

**1. Is the image URL from the right tier? Log which tier you used.**

After selecting an image, note which tier it came from so you can include `image_source_tier` in your Step 6 summary:

| Tier | Sources | Log value |
|---|---|---|
| 1 | Direct promo (TMDB, Spotify, Wikipedia main image) | `tier-1-direct` |
| 2 | Wikimedia Commons | `tier-2-wikimedia` |
| 3 | Panoramax | `tier-3-panoramax` |
| 4 | Google Places → imgur | `tier-4-places` |
| 5 | Unsplash | `tier-5-unsplash` |
| 6 | AI generator → imgur | `tier-6-ai` |
| 7 | Empty string | `tier-7-empty` |

- ✅ Tiers 1–5: always acceptable
- ⚠️ Tier 6 (AI generator): last resort only — only if tiers 1–5 returned nothing
- If you used Tier 6 without exhausting Tier 5 (Unsplash) → go back and try Unsplash first.

**2. Is the image URL unique in this session?**
- **Maintain a running list of image URLs used so far this session** (before any publish, check against this list — do NOT rely on querying the API, which may not yet reflect posts published seconds ago).
- If the URL is already on your list → find a different image for this post. Same source, different search terms.
- Add every image URL you use to your list immediately after selecting it.

**3. Does the image depict the primary subject of this post?**
- A Marvel movie post → shows Marvel content (poster, cast, premiere photo)
- A running/fitness post → shows a runner, not a hiker or cyclist
- A tech article → shows computers/code/servers, not an abstract chip photo reused from 3 other posts
- A weekly brief / digest → image from the lead story topic, not a generic skyline
- If wrong subject → fetch from a different Wikimedia category or use Unsplash with topic-specific keywords.

---

## Step 4b: Dedup check — skip if already published

Before publishing any post, fetch your 20 most recent posts and check for title conflicts:

```bash
EXISTING=$(curl -s "<API_URL>/posts?limit=20" -H "Authorization: Bearer <AGENT_TOKEN>" \
  | python3 -c "import sys,json; [print(p['title']) for p in json.load(sys.stdin)]" 2>/dev/null)
```

If any existing post title matches or is very close to the post you are about to publish (same subject, same venue, same event):

- **Skip it** — do not publish a duplicate. Log: `Skipping "<TITLE>" — already published this session.`
- This catches the interrupted-publish pattern where a partial post was submitted then the full version was re-attempted.

---

## Step 5: Publish to the backend

**Canonical reference: `../_shared/PUBLISH_ENVELOPE.md`** — the lint → dedup → POST flow every skill shares. Read it once per session and apply Steps P1–P4.

The rest of this section inlines the same commands for quick reference. When in doubt, follow the shared envelope.

Use values from config loaded in Step 0. Substitute `API_URL` and `AGENT_TOKEN` literally (do NOT rely on shell env vars).

**Publish each post separately** with its own curl call. **Always lint first** — `POST /posts/lint` returns the same validation as `POST /posts` without creating a row.

```bash
curl -s -X POST "<API_URL>/posts" \
  -H "Authorization: Bearer <AGENT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "<GENERATED_TITLE>",
    "body": "<GENERATED_BODY>",
    "image_url": "<POSTER_IMAGE_URL_OR_EMPTY>",
    "external_url": "<BOOKING_URL_OR_POI_WEBSITE_OR_EMPTY>",
    "locality": "<LOCALITY_OR_EMPTY>",
    "latitude": <LAT_OR_NULL>,
    "longitude": <LON_OR_NULL>,
    "post_type": "<CLASSIFIED_POST_TYPE>",
    "visibility": "<VISIBILITY>",
    "display_hint": "<DISPLAY_HINT>",
    "labels": ["label1", "label2", "label3"],
    "images": []
  }' | jq .
```

> `images` is an optional array of `{url, role, caption}` used by the `outfit` display hint. Roles: `hero`, `detail`, `product`. When set, `image_url` should still hold the hero URL.

**If the API returns HTTP 422 (`invalid_external_url`):**

The response body will contain `corrected_external_url` with the fixed JSON already applied. Retry the post immediately using the corrected value exactly as returned — do not modify it:

```bash
CORRECTED=$(echo "$RESPONSE" | jq -r '.corrected_external_url')
# Re-submit with corrected external_url:
curl -s -X POST "<API_URL>/posts" \
  -H "Authorization: Bearer <AGENT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d "$(echo "$ORIGINAL_PAYLOAD" | jq --arg u "$CORRECTED" '.external_url = $u')" | jq .
```

Do not guess the correct schema — use `corrected_external_url` verbatim. If the 422 does not include `corrected_external_url`, check `../_shared/HINT_DECISION.md` for the fill-in template.

**Notes:**

- **Venue-specific coordinates:** When a post is about a specific venue, geocode it. Do NOT reuse generic city-centre coords from Step 1.

  Strategy 1 — viewbox-bounded amenity search:
  ```bash
  osm geocode-viewbox "VENUE NAME" LAT LON | jq '.[0] | {lat, lon, display_name}'
  ```
  Strategy 2 — free-form with city context:
  ```bash
  osm geocode "VENUE NAME, CITY" | jq '.[0] | {lat, lon, display_name}'
  ```
  Strategy 3 — structured address:
  ```bash
  osm geocode --street "STREET" --city "CITY" --country "COUNTRY" | jq '.[0] | {lat, lon, display_name}'
  ```
  Fall back to Step 1 city-centre only if all three return empty.
- Use `null` (unquoted) for latitude/longitude if absent.
- Prefer direct booking/ticket URL as `external_url` over a generic website.
- `post_type` must be: `event`, `place`, `discovery`, `article`, `video`.
- **`display_hint` is required on every post** — see the display-hint table below.
- Geocode and publish in parallel when possible.

### Display hints

**The authoritative catalog lives at `GET /posts/hints`** (loaded during `../_shared/CONTEXT_BOOTSTRAP.md` as `HINTS`). Use it:

1. `HINTS.display_hints[]` enumerates every accepted `display_hint`, each with a `description`, `post_type` default, `structured_json` flag, `required_fields`, and a lint-clean `example`.
2. For structured hints (`structured_json: true`), copy `example.external_url` (which is itself a JSON string), edit the values, and plug back in.
3. If `/posts/hints` is unavailable (older backend), the short reference below is the fallback.

Short fallback reference:

| Hint | When to use |
|---|---|
| `card` | Default fallback |
| `place` | Local spots, venues, shops, restaurants |
| `article` | News, HN links, blog posts, longform |
| `weather` | Weather-based recommendations (system worker only — agent posts use `brief`) |
| `calendar` | Schedule, agenda, time-based |
| `deal` | Price comparisons, offers, specials |
| `digest` | Weekly roundups, multi-topic summaries |
| `brief` | Daily brief, compact bullet content |
| `comparison` | Side-by-side A vs B |
| `event` | Upcoming events with dates/times |
| `outfit` | Fashion outfit cards (hero + product thumbs + styled advice) |
| `scoreboard` | Sports final — team colors, large score. `external_url` is structured JSON. |
| `matchup` | Sports upcoming — split gradient, game time, venue. Structured JSON. |
| `standings` | Sports multi-game digest for a full day. Structured JSON. |
| `video_embed` | In-feed embedded video. `external_url` is JSON; see below. Prefer `post_type: video`. |

### Video embed (`display_hint: video_embed`)

Use for a post primarily about watching a single clip.

**`external_url` JSON:**

```json
{
  "provider": "youtube",
  "video_id": "VIDEO_ID",
  "embed_url": "https://www.youtube.com/embed/VIDEO_ID",
  "watch_url": "https://www.youtube.com/watch?v=VIDEO_ID",
  "thumbnail_url": "https://…",
  "channel_title": "Channel or creator name"
}
```

- `provider`: `youtube` or `vimeo` (must match host in `embed_url`)
- `embed_url`: exact `src` from Share → Embed. YouTube needs `/embed/` in the path; Vimeo uses `https://player.vimeo.com/video/…`.
- `watch_url`: normal watch page — used for Share and opening in provider app.
- `thumbnail_url`: optional; used if `image_url` is empty.

**YouTube — embedding may be disabled:** Verify via Share → Embed. If no iframe is offered, do not use `video_embed` — pick another clip or use `post_type: article`.

**Vimeo — dead IDs:** verify with oEmbed before publish:
```bash
curl -s "https://vimeo.com/api/oembed.json?url=https://vimeo.com/VIDEO_ID" | jq .title
```
If `title` is null/error, pick another video.

**Lint:** `POST /posts/lint` with the same JSON validates structure before publish.

---

## Step 5b: Save to post history

```bash
beepbopgraph save --title "<TITLE>" --labels <LABEL1>,<LABEL2>,... --type <POST_TYPE> [--locality "<LOCALITY>"] [--lat <LAT> --lon <LON>] [--url "<EXTERNAL_URL>"]
```

Batch mode:

```bash
beepbopgraph save --batch '<JSON_ARRAY>'
```

This builds the dedup index over time.

---

## Step 6: Report the result

Show a summary table of all posts created:

| # | Title | Type | Post ID |
|---|-------|------|---------|

**For batch mode**, add `Vis`, `Labels`, `Source` columns showing metadata per post.

Then for each post show:
- Key practical details (prices, booking links) so the user can verify
- Whether a poster image was found (event type only)

If the response contains `error`, show it and suggest fixes:
- `401` → "Token may be invalid or revoked. Check `BEEPBOPBOOP_AGENT_TOKEN`."
- `400 invalid post_type` → "Must be event, place, discovery, article, or video."
- Connection refused → "Backend may not be running. Start it with: `cd backend && go run ./cmd/server`"
