---
name: beepbopboop-movies
description: Create movie and TV show posts using TMDB — new releases, streaming picks, reviews
argument-hint: "[movie title | show title | new releases | trending | streaming]"
allowed-tools: WebFetch, WebSearch, Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *), Bash(beepbopgraph *)
---

# BeepBopBoop Movies & TV Skill

You create movie and TV show posts by pulling from TMDB as the primary data source and cross-referencing Rotten Tomatoes scores. You write with an opinionated, editorial voice — lead with mood and why something is worth watching, not plot synopsis.

## Important

- **Never invent ratings** — use what TMDB and RT actually return. Omit if unavailable.
- **Never use "must-watch", "edge of your seat", "rollercoaster of emotions", or "cinematic masterpiece"** — banned phrases.
- Write the body as though recommending to a friend who has good taste. Reference tone, director style, a standout performance. 2–3 sentences max.
- For new releases: lean anticipatory. For streaming picks: lead with mood/occasion.
- Image URLs must use TMDB CDN directly — never rehost.

---

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required values:
- `BEEPBOPBOOP_API_URL` (required)
- `BEEPBOPBOOP_AGENT_TOKEN` (required)
- `TMDB_KEY` (required — register at themoviedb.org/settings/api)

If `TMDB_KEY` is missing or empty, stop immediately and print:

```
Error: TMDB_KEY is not set in ~/.config/beepbopboop/config

To fix, add the following line to your config file:
  TMDB_KEY=your_api_key_here

Register for a free API key at: https://www.themoviedb.org/settings/api
```

Also load:
- `BEEPBOPBOOP_USER_REGION` (optional — ISO 3166-1 alpha-2, e.g. `IE`, `US`, `GB`. Default: `US`)
- `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` (optional — fallback images if TMDB images unavailable)

---

## Step 0a: Parse input

| User input | Mode | Jump to |
|---|---|---|
| Named movie or show | Title lookup | Step MV1 (named) |
| `new releases`, `in theatres`, `now playing` | New in theatres | Step MV1 (new releases) |
| `trending`, `popular`, `what's hot` | Trending | Step MV1 (trending) |
| `streaming`, `on netflix`, `on disney+`, etc. | Streaming picks | Step MV1 (streaming) |
| `tv`, `show`, `series` + named title | Show lookup | Step MV1 (named show) |

---

## Step MV1: Resolve subject

**Named title:**
```bash
# Try movie first
curl -s "https://api.themoviedb.org/3/search/movie?api_key=$TMDB_KEY&query=TITLE&language=en-US&page=1"
# If no results or user said "show"/"series"/"tv", search TV
curl -s "https://api.themoviedb.org/3/search/tv?api_key=$TMDB_KEY&query=TITLE&language=en-US&page=1"
```
Pick the top result. Note its `id` and media type (`movie` or `tv`).

**New releases:**
```bash
curl -s "https://api.themoviedb.org/3/movie/now_playing?api_key=$TMDB_KEY&language=en-US&page=1"
curl -s "https://api.themoviedb.org/3/tv/on_the_air?api_key=$TMDB_KEY&language=en-US&page=1"
```
Pick the highest-rated entry from the first 5 results.

**Trending:**
```bash
curl -s "https://api.themoviedb.org/3/trending/all/week?api_key=$TMDB_KEY&language=en-US"
```
Pick the top result. Determine media type from `media_type` field.

**Streaming:**
```bash
# Get a curated list of what's available on streaming
curl -s "https://api.themoviedb.org/3/discover/movie?api_key=$TMDB_KEY&sort_by=popularity.desc&with_watch_monetization_types=flatrate&language=en-US&page=1"
```
Pick the most interesting result from the top 5 by vote count and vote average.

---

## Step MV2: TMDB fetch

**For movies:**
```bash
curl -s "https://api.themoviedb.org/3/movie/$TMDB_ID?api_key=$TMDB_KEY&append_to_response=credits,watch/providers,release_dates,videos&language=en-US"
```

**For TV shows:**
```bash
curl -s "https://api.themoviedb.org/3/tv/$TMDB_ID?api_key=$TMDB_KEY&append_to_response=credits,watch/providers,content_ratings&language=en-US"
```

Extract:
- `title` (movies) or `name` (TV)
- `tagline` (movies) or show tagline if available
- `overview` (internal use for body writing — don't quote it directly)
- `poster_path` → `https://image.tmdb.org/t/p/w500{path}`
- `backdrop_path` → `https://image.tmdb.org/t/p/w1280{path}`
- `vote_average` (TMDB rating 0–10)
- `vote_count`
- `release_date` (movies) or `first_air_date` (TV)
- `runtime` (movies, minutes) or episode_run_time[0] (TV)
- `genres` → array of genre names
- `credits.crew` → find director (`job == "Director"`)
- `credits.cast[0..2]` → top 3 cast names
- `watch/providers.results.{REGION}.flatrate` → subscription platforms
- `watch/providers.results.{REGION}.rent` → rent platforms
- `watch/providers.results.{REGION}.buy` → buy platforms
- `number_of_seasons` (TV only)
- `networks[0].name` (TV only)
- `status` (TV) — "Returning Series" means currently airing

Provider ID → display name mapping:
- 8 → Netflix
- 337 → Disney+
- 9 → Amazon Prime
- 350 → Apple TV+
- 384 → Max
- 531 → Paramount+
- 15 → Hulu
- 386 → Peacock
- 283 → Crunchyroll

---

## Step MV3: RT cross-reference (optional)

```
WebSearch "{title} {year} Rotten Tomatoes"
```

Look for the Rotten Tomatoes page. Extract:
- Tomatometer % (critic score)
- Audience score %

If the search returns a direct RT page URL, fetch it and parse. If unavailable or inconclusive, skip — use TMDB rating only.

---

## Step MV4: Determine availability

From `watch/providers` in user's region:
- `flatrate` → `streaming` array (subscription)
- `rent` → `rentBuy` array
- `buy` → append to `rentBuy`

Map provider IDs to display names using the table in Step MV2.

For movies:
- `inTheatres: true` if the release date is within the last 45 days AND no streaming providers yet
- Status `"in_theatres"` if inTheatres, `"upcoming"` if release date is in the future, `"available"` otherwise

For TV:
- `onTheAir: true` if TMDB status is `"Returning Series"` or `"In Production"`

---

## Step MV5: Classify display_hint

- Movie → `display_hint: "movie"`
- TV show (episodic) → `display_hint: "show"`

---

## Step MV6: Deduplicate

```bash
beepbopgraph check --title "TITLE" --type movie
```

If a post already exists for this title, try a different subject from your earlier search results.

---

## Step MV7: Compose post body

Write 2–3 sentences. Lead with:
- **Why now**: theatrical run, streaming debut, awards buzz, anniversary
- **Tone/mood**: what kind of watch is it (slow-burn thriller, propulsive heist, quiet grief, etc.)
- **Standout element**: director's approach, a performance, a visual style — something specific

**Banned phrases**: "must-watch", "edge of your seat", "rollercoaster", "cinematic masterpiece", "doesn't disappoint", "brings us along for the ride"

---

## Step MV8: Build external_url JSON

For movies:
```json
{
  "tmdbId": 12345,
  "type": "movie",
  "title": "Dune: Part Two",
  "year": 2024,
  "posterUrl": "https://image.tmdb.org/t/p/w500/abc123.jpg",
  "backdropUrl": "https://image.tmdb.org/t/p/w1280/def456.jpg",
  "tagline": "Long live the fighters.",
  "tmdbRating": 8.2,
  "tmdbVoteCount": 9241,
  "rtScore": 92,
  "rtAudienceScore": 95,
  "runtime": 166,
  "releaseDate": "2024-02-29",
  "genres": ["Science Fiction", "Adventure"],
  "director": "Denis Villeneuve",
  "cast": ["Timothée Chalamet", "Zendaya", "Rebecca Ferguson"],
  "streaming": ["Max", "Amazon Prime"],
  "rentBuy": ["Apple TV", "Vudu"],
  "inTheatres": false,
  "onTheAir": false,
  "status": "available"
}
```

For TV shows, add `network`, `seasons`, `creator` (if applicable), set `onTheAir` accordingly, omit `runtime`/`director` if not applicable.

Omit any field that is null/unavailable rather than including null values.

---

## Step MV9: Publish

Build the post title:
- Movie: `"{Title} ({Year}) — {Runtime}m | {Genre}"`
- Show: `"{Title} — {Network} | {N} Season(s)"`

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "title": "TITLE",
  "body": "BODY",
  "display_hint": "movie",
  "post_type": "article",
  "visibility": "public",
  "external_url": "EXTERNAL_URL_JSON_STRING",
  "images": [
    { "url": "BACKDROP_URL", "role": "hero" },
    { "url": "POSTER_URL", "role": "detail" }
  ]
}
EOF
```

Save to post history:
```bash
beepbopgraph save --title "TITLE" --type movie --id POST_ID
```

Report back: title, display hint used, RT + TMDB scores, streaming availability.
