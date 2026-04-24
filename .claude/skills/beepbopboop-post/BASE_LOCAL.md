# Base local flow (Steps 1–4)

Shared by the default idea → local place/event path and by any other mode that falls back to "generate content from an idea + optional location." After Step 4, every mode proceeds to `COMMON_PUBLISH.md` (Steps 4a → 5b → 6).

---

## Step 1: Resolve location

Determine the location with this priority:

1. **Explicit locality argument** → geocode it (user is asking about a different place).
2. **No argument + `HOME_LAT`/`HOME_LON` set** → use those directly as lat/lon, set `display_name` to `BEEPBOPBOOP_DEFAULT_LOCATION`, **skip geocoding entirely**.
3. **No argument + no HOME coords** → geocode `BEEPBOPBOOP_DEFAULT_LOCATION`.
4. **None available** → proceed without coordinates.

Geocode via `osm`:

```bash
osm geocode "LOCATION_STRING" | jq '.[0] | {lat, lon, display_name}'
```

For addresses that fail free-form, use structured mode:

```bash
osm geocode --street "STREET" --city "CITY" --country "COUNTRY" | jq '.[0] | {lat, lon, display_name}'
```

Store the resolved `lat`, `lon`, and `display_name`. If geocoding fails, proceed without coordinates.

---

## Step 2: Discover nearby POIs

**Only run if lat/lon are available from Step 1.**

Map idea keyword → OSM tag:

| Keyword | OSM Query Filter |
|---------|-----------------|
| coffee, cafe, espresso | `"amenity"="cafe"` |
| restaurant, food, eat, dinner, lunch | `"amenity"="restaurant"` |
| bar, pub, drinks, beer | `"amenity"="bar"` |
| park, green, nature | `"leisure"="park"` |
| gym, fitness, workout | `"leisure"="fitness_centre"` |
| bakery, bread, pastry | `"shop"="bakery"` |
| cinema, movie, film | `"amenity"="cinema"` |
| museum, gallery, art | `"tourism"="museum"` |
| playground, kids | `"leisure"="playground"` |
| theatre, play, drama, acting, stage | `"amenity"="theatre"` |

Other keywords: use best judgment (`"shop"="books"` for bookshops, `"leisure"="pitch"["sport"="tennis"]` for tennis courts, etc.).

If the idea doesn't match, skip POI discovery.

```bash
osm pois '"amenity"="cafe"' LAT LON 1500 5
```

(1500m radius, 5 results.) Extract for each POI: `name`, amenity/leisure/shop type, `opening_hours`, `website`.

Compute approximate distance: `distance_km ≈ sqrt((lat2-lat1)² + (lon2-lon1)² × cos(lat1)²) × 111`. Express as meters if < 1 km.

If Overpass fails or returns nothing, proceed without POI data.

---

## Step 2b: Classify post type

If the user provided `$2`, use it directly (must be one of: `event`, `place`, `discovery`, `article`, `video`).

Otherwise auto-classify:

| Type | Trigger Keywords |
|------|-----------------|
| `event` | theatre, play, concert, gig, show, cinema, film screening, exhibition, festival, performance, recital, opera, ballet, comedy show, standup, launch, premiere, opening night — or any time-bound experience |
| `place` | cafe, restaurant, bar, park, gym, bakery, bookshop, museum, gallery, hotel, shop, playground, beach, market — or fundamentally about a venue. When post has lat/lon for a specific venue, set `display_hint: "place"` |
| `article` | Blog, news article, essay (interest mode) |
| `video` | YouTube video, video essay, podcast with video (interest mode) |
| `discovery` | Everything else — tips, observations, insights |

Apply in order:
1. Explicit `$2` → use as-is
2. Interest + video → `video`
3. Interest + written → `article`
4. Matches `event` keywords → `event`
5. Matches `place` keywords → `place`
6. Default → `discovery`

---

## Step 3: Research practical details + poster image

**Run when the idea involves events/venues/anything time-sensitive, or when POIs were found and deeper detail would make the post actionable.**

Answer the reader's real questions: what's on right now, how much, how to book, what time, is it available.

### Sports schedule lookup (FIRST for sports topics)

If the idea involves sports, games, matches, or a specific league/team, read `SPORTS_SOURCES.md` in this skill directory before any web search.

1. Match the sport/league to an entry in `SPORTS_SOURCES.md`.
2. Check the season window.
3. Check `BEEPBOPBOOP_SPORTS_TEAMS` for the preferred team.
4. Fetch schedule via ESPN API:
   ```bash
   curl -s "https://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard?dates={YYYYMMDD}" | jq '.events[] | {name, date, status: .status.type.description, venue: .competitions[0].venue.fullName}'
   ```
   Omit `?dates=` for today, or `?dates=YYYYMMDD-YYYYMMDD` for a range.
5. Filter by preferred team if set.
6. Use this official data for dates/times/opponents/venues — **do not use WebSearch for schedule data**.
7. Use WebSearch only for enrichment (ticket links, venue atmosphere, travel info, watch party locations).

For AHL and OHL (no ESPN API), WebFetch official schedule pages listed in `SPORTS_SOURCES.md`.

**Skip to Phase 2** (deep dive) after sports lookup — Phase 1 broad survey is unnecessary when you have official data.

### How to research (non-sports)

**Phase 1 — Broad survey.** Cast a wide net with 2–3 parallel WebSearch queries:
- General: `<TOPIC> <LOCALITY> <MONTH> <YEAR>`
- Specific leagues/orgs/genres/categories
- Aggregator: `<TOPIC> <LOCALITY> schedule this week`

Build a list of all distinct options (teams, venues, events, organizations). Don't stop at the first hit.

**Phase 2 — Deep dive.** Fetch venue/org websites for the top 2–3 options. Look for: event name, dates, showtimes, prices, booking URL, sold-out status. Fill gaps with targeted WebSearch.

**Phase 3 — Decide single vs. multiple posts.**

- Different venues/teams/orgs → separate posts (a Royals game and a Grizzlies game = 2 posts).
- Same venue, same event series → single post.
- Same venue, different events → separate posts.

### Poster image search (event type only)

1. WebSearch: `"<EVENT_NAME>" "<VENUE_NAME>" poster image` or `"<SHOW_NAME>" <YEAR> poster`.
2. WebFetch on the most promising results (venue website, ticketing page, event listing).
3. Find direct image URL (`.jpg`, `.png`, `.webp`). Prefer venue's own domain, official ticketing, high-res promotional.
4. Must be a direct image link, not an HTML page.
5. If nothing suitable, use empty string — the iOS app shows a theater-mask gradient placeholder.

### What to extract

For each researched venue: event/show name, dates & showtimes, price/range, booking URL, availability, anything notable, poster image URL.

If research fails, proceed without it — the post should still work with POI data.

---

## Step 4: Generate post content

**If Step 3 identified multiple distinct posts**, generate each separately. Otherwise generate a single post.

Each post needs:
- **title**: compelling, specific headline, max 80 chars, not clickbait.
- **body**: 2–3 sentences. Personal, actionable, or thought-provoking.

### Writing Quality Standards (required)

**Headline rules:**
- Be specific, not generic. Numbers, names, distances create curiosity.
- Formulas that work: proximity + specificity, urgency + detail, counterintuitive, insider knowledge.
- Max 80 chars.

**Body rules:**
- First sentence rule: must add NEW information not in the title.
- 2–3 sentences. Each does a different job: (1) specifics, (2) context, (3) actionable close.
- End with something actionable whenever possible.

**Kill list (banned phrases):** "Check out", "hidden gem", "whether you're", "looking for", "if you're in the area", "don't miss", "perfect for", "nestled in", "boasts", "a must-visit", "vibrant", "bustling", "tucked away". Never start with "This [noun] is...". Never write a sentence that could describe any city on earth.

**Tone test:** Read aloud. Friend who just discovered something, not a tourism brochure.

**When POI data and research are available:**
- Reference actual place names
- Include real distances from the user's location
- Mention opening hours if relevant
- Include prices, showtimes, booking info
- If sold out / nearly sold out, say so
- Use booking URL as `external_url` (preferred over generic homepage)
- Each post stands alone

**When POI data is NOT available:** write the post from the idea alone.

Locality context: `display_name` from geocoding if available, else the raw locality arg (`$1`). Post type (optional): `$2`.

### Anti-example

BAD: `"Check Out This Hidden Gem Cafe in Dublin"` / `"Whether you're looking for a great cup of coffee or a cozy spot to work, this vibrant cafe is a must-visit…"` — 5 kill-list phrases, could describe any cafe anywhere.

FIXED: `"Kaph is 3 minutes from your door"` / `"There's a cafe 290 metres away that regulars swear by. Kaph on Drury Street does single-origin pourovers in a space small enough to guarantee you'll overhear something interesting. Open until 6pm."` — names the place, gives a distance, tells you what it's known for, gives a reason to go now.

---

After Step 4, proceed to **`COMMON_PUBLISH.md`** (Steps 4a → 4b → 4c → 4d → 5 → 5b → 6).
