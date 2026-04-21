# End-to-end examples

Reference examples for common patterns. Each demonstrates a different end-to-end flow the skill supports.

## Example 1: Single keyword → local place post

**What this demonstrates:** the full local flow — geocoding, POI discovery, venue-specific coordinates, proximity-based writing.

Given `"coffee"` with locality `"Dublin 2, Ireland"`:

1. Geocode → lat/lon. Map "coffee" → `"amenity"="cafe"`. POI search finds 3 cafes with distances. (`BASE_LOCAL.md` Steps 1–2.)
2. Classify → `place`. Generate content using POI data (real name, distance, hours). (`BASE_LOCAL.md` Steps 2b + 4.)
3. `COMMON_PUBLISH.md` Steps 4a → 4b → 4c → 4d → 5 → 5b.

**Result:** `title: "Kaph is 3 minutes from your door"` / `body: "There's a cafe 290 metres away that regulars swear by…"` / `post_type: "place"` / `visibility: "personal"` (body says "your door") / `labels: ["place", "coffee", "cafe", "specialty-coffee"]`.

## Example 2: Broad idea → multiple posts with venue geocoding

**What this demonstrates:** Step 3's broad survey triggering multiple posts, each with its own venue-specific coords.

Given `"hockey games"` with locality `"Victoria, BC, Canada"`:

1. Geocode city. No OSM keyword match → skip POI. Classify → `event`.
2. Step 3 broad survey: WebSearch finds Royals (WHL) at Save-On-Foods + Grizzlies (VIJHL) at The Q Centre → 2 separate posts.
3. Geocode each venue individually: `osm geocode-viewbox "Save-On-Foods Memorial Centre" …` and `osm geocode-viewbox "The Q Centre" …`
4. Each post gets its own lat/lon, ticket prices, schedule, booking URL.

**Result:** Post 1: `title: "Royals host three games at Save-On-Foods this week"` / `lat: 48.4452`. Post 2: `title: "Grizzlies take on Nanaimo at The Q Centre"` / `lat: 48.4355`.

## Example 3: Topic → article post (interest mode, delegated)

**What this demonstrates:** non-geographic content flow — delegated to `beepbopboop-news`.

Given `"latest AI news"`:

1. Router delegates to `beepbopboop-news`.
2. News skill runs its interest flow. WebSearch for recent articles, WebFetch top results. Classify → `article`. No lat/lon. Locality = source name.

**Result:** `title: "Anthropic's new reasoning model scores 94% on ARC-AGI"` / `locality: "Anthropic Blog"` / `latitude: null` / `external_url: "https://anthropic.com/blog/…"` / `post_type: "article"` / `labels: ["article", "ai", "machine-learning", "research"]`.

## Example 4: Weather → chained local posts

**What this demonstrates:** weather mode chaining into local mode — current conditions drive activity selection, then each activity runs the full local flow.

Given `"weather"` with location `"Victoria, BC, Canada"`:

1. `MODE_WEATHER.md` W1. WebSearch weather → 14°C, rain by afternoon.
2. Map rainy conditions → museums, cozy cafes. Run local flow for each.
3. Each post gets venue-specific geocoding + weather context in title/body opener.

**Result:** Post 1: `title: "Rain by 2pm — the Royal BC Museum has a new exhibition you haven't seen"` / `locality: "Royal BC Museum"`. Post 2: `title: "Murchie's on Government does a proper afternoon tea for $18"` / `body: "Grey sky, warm tea…"`.

## Example 5: Batch → diverse feed from multiple modes

**What this demonstrates:** batch mode composing multiple modes into one diverse feed. Scheduled rules run first (Phase 1), defaults fill to target (Phase 2), BT6 dedup + BT7 diversity ensure no repeats.

Given `"batch"` on a Monday with schedule `monday|interest|AI roundup|daily|weather|daily|source|hn`:

1. Target: 10 posts (random 8–15). Phase 1 scheduled: weather → 2 posts, interest "AI roundup" → 2 posts, source HN → 2 posts.
2. Phase 2 fill (4 more): local "events this week" → 3 posts, seasonal → 1 post.
3. BT6: beepbopgraph dedup (one batch query). BT7: diversity check passes — 4 types, mix of local/non-local.
4. Publish all 10, report with mode attribution.

**Result table:**

| # | Title | Type | Source |
|---|-------|------|--------|
| 1 | Rain by 2pm — Royal BC Museum exhibition | place | weather |
| 2 | Murchie's afternoon tea for $18 | place | weather |
| 3 | Claude 4.5 rewrites the reasoning benchmark | article | interest |
| 4 | Three startups raised $50M to replace dashboards | article | interest |
| 5 | YC batch has 3 AI code review companies | article | HN |
| 6 | Open-source Notion AI alternative hits 10k stars | article | HN |
| 7 | Royals host three games — tickets from $17 | event | local |
| 8 | Grizzlies take on Nanaimo Wednesday | event | local |
| 9 | Blue Bridge Theatre one-woman show Friday | event | local |
| 10 | Cherry blossoms peaking along Moss Street | discovery | seasonal |
