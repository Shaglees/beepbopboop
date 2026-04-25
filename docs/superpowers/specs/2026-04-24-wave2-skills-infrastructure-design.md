# Wave 2: Skills Infrastructure Design

_Date: 2026-04-24_
_Issues: #201, #197, #198, #200, #199, #203, #202_

## Goal

Any skill can publish a post that iOS renders as a rich card with no decoding errors. Lint catches bad payloads before they hit the feed. All specialty skills are reachable from the main router.

## Source of Truth

`backend/internal/handler/hints.go` — the `GET /posts/hints` v2 response — is the single contract. Tests, skill docs, and lint validators all derive from it. No separate schema file.

---

## 1. Contract Testing (#201)

File: `backend/internal/handler/hints_test.go`

Four test categories:

### 1a. Round-trip test
For every hint in the catalog, take its `example` payload, POST it to `LintPost`, assert `valid: true` with zero errors.

### 1b. iOS decode test
For every structured hint, take the `example.external_url` JSON string, unmarshal into a Go struct mirroring the Swift Codable model. Assert all required fields are non-zero. This catches "lint passes but iOS can't decode" drift.

### 1c. Completeness test
Assert every entry in `ValidDisplayHints` has a corresponding entry in the hints catalog. No hint goes undocumented.

### 1d. Metadata test
Assert every catalog entry has non-empty `description`, `required_fields`, `example`, `renders.card`, `pick_when`, `avoid_when`.

---

## 2. Actionable Lint Warnings (#198, #197)

### 2a. Upgrade warning messages
Each per-hint validator in `post.go` (lines 398–1085) gets upgraded warning messages that tell the agent exactly what to add and show an example value.

Before:
```json
{"field": "external_url.rating", "code": "missing", "message": "missing field: rating"}
```

After:
```json
{"field": "external_url.rating", "code": "recommended", "message": "Add \"rating\": <number 1.0-5.0> to your external_url JSON for a richer RestaurantCard render. Example: \"rating\": 4.3"}
```

No changes to error/reject logic — these remain warnings. The agent reads warnings and patches its payload before calling `POST /posts`.

### 2b. Add missing field warnings
For any iOS-required field that currently has no lint check at all (e.g., FoodData.mustTry, TravelData.knownFor, ScienceData.tags), add a warning with patch instructions.

### 2c. external_url string-vs-object detection (#197)
Add a clear error when `external_url` for a structured hint is a raw JSON object instead of a string:

> "external_url must be a JSON string containing serialized JSON, not a raw object. Serialize your payload with JSON.stringify() before setting external_url."

---

## 3. Skill Documentation Fixes (#197) & Dispatch Table (#200)

### 3a. Canonical external_url pattern
Add to `COMMON_PUBLISH.md`:

```
## Structured external_url

For hints requiring structured JSON (scoreboard, restaurant, movie, etc.):
1. Build your data object
2. Serialize to a JSON STRING — external_url must be a string value, not a raw object
   e.g., "external_url": "{\"name\":\"Ramen House\",\"rating\":4.5}"
3. POST to /posts/lint, read warnings, patch if needed, then POST to /posts
```

Update each specialty skill SKILL.md to reference COMMON_PUBLISH.md instead of maintaining its own serialization guidance.

### 3b. Dispatch table in beepbopboop-post SKILL.md

| Topic Keywords | Skill | Display Hints |
|---|---|---|
| restaurant, food, dining, cuisine | beepbopboop-food | restaurant |
| movie, film, TV show, streaming | beepbopboop-movies | movie, show |
| album, artist, concert, music | beepbopboop-music | album, concert |
| pet, adoption, dog, cat, shelter | beepbopboop-pets | pet_spotlight |
| science, space, NASA, research | beepbopboop-science | science |
| travel, destination, flight, trip | beepbopboop-travel | destination |
| workout, fitness, exercise, gym | beepbopboop-fitness | fitness |
| celebrity, red carpet, awards | beepbopboop-celebrity | entertainment |
| basketball, NBA | beepbopboop-basketball | scoreboard, matchup, standings, player_spotlight, box_score |
| baseball, MLB | beepbopboop-baseball | scoreboard, matchup, standings, box_score |
| football, NFL | beepbopboop-football | scoreboard, matchup, standings, player_spotlight |
| soccer, Premier League, Champions League | beepbopboop-soccer | scoreboard, matchup, standings |
| fashion, outfit, style | beepbopboop-fashion | outfit |
| video game, gaming, release, review | beepbopboop-gaming | game_release, game_review |
| creator, artist spotlight, local maker | beepbopboop-creators | creator_spotlight |
| news, trending, current events | beepbopboop-news | article |

In batch mode, classify each post idea against this table. If a match is found, delegate to the specialty skill. Otherwise handle with generic internal modes.

---

## 4. New Skills (#199)

### 4a. beepbopboop-gaming
Generates `game_release` and `game_review` posts.

**Data sources:** RAWG API (free tier, 20k req/month), Steam store API (free, no key), IGDB (Twitch auth, free).

**Env vars:** `RAWG_API_KEY` (optional — falls back to Steam). No hard requirements.

**Flow:** Fetch upcoming/recent releases → match to user interests → build VideoGameData JSON → lint → publish.

**VideoGameData schema (from iOS):**
- `title` (required)
- `status`: upcoming | released | early_access (required)
- `releaseDate` (optional)
- `platforms[]` (optional)
- `genres[]` (optional)
- `metacriticScore` (optional)
- `description` (optional)
- `coverURL` (optional)
- `screenshotURLs[]` (optional)

### 4b. beepbopboop-creators
Generates `creator_spotlight` posts.

**Data sources:** Web search for local artists/makers, Instagram public profiles, Bandcamp, Etsy, Substack.

**Env vars:** None required (web search based).

**Flow:** Search for local creators matching user interests/location → build CreatorData JSON → lint → publish.

**CreatorData schema (from iOS):**
- `designation` (required)
- `links`: website, instagram, bandcamp, etsy, substack, soundcloud, behance (all optional)
- `notable_works` (optional)
- `tags[]` (optional)
- `source` (optional)
- `area_name` (optional)

### 4c. feedback — system-only
Mark in hints.go metadata as `"generator": "system"` so skills know not to generate feedback posts.

---

## 5. Preflight (#203)

Add Step 0 to beepbopboop-post SKILL.md. Before generating any posts:

1. **Backend reachable:** `curl -sf $BEEPBOPBOOP_URL/posts/hints` — fail if backend is down
2. **Auth valid:** `curl -sf -H "Authorization: Bearer $TOKEN" $BEEPBOPBOOP_URL/posts?limit=1` — verify token
3. **Required env vars:** `BEEPBOPBOOP_URL` and `BEEPBOPBOOP_TOKEN` must exist
4. **Capability matrix:** For each specialty skill, check env vars/CLIs. Build usable skills list:
   ```
   Skill availability:
     ✓ beepbopboop-news (no external deps)
     ✓ beepbopboop-food (YELP_KEY found)
     ✗ beepbopboop-music (SPOTIFY_TOKEN missing — album/concert unavailable)
     ✓ beepbopboop-movies (TMDB_KEY found)
   ```
5. **CLI checks:** Verify `jq` and `curl` are available
6. In batch mode, only route to skills that passed preflight

---

## 6. Portable Onboarding (#202)

Replace `AskUserQuestion` calls in `INIT_WIZARD.md` with a config-file pattern:

1. If `~/.beepbopboop/config.json` exists, read it. Schema:
   ```json
   {
     "url": "http://192.168.1.x:8080",
     "token": "agent-token",
     "location": {"city": "Dublin", "lat": 53.35, "lon": -6.26},
     "interests": ["basketball", "food", "science"],
     "api_keys": {
       "TMDB_KEY": "...",
       "YELP_KEY": "...",
       "RAWG_API_KEY": "..."
     }
   }
   ```
2. If config is missing, print the required template and stop: "Create ~/.beepbopboop/config.json with this structure and re-run."
3. For Claude Code, the existing `AskUserQuestion` wizard remains as a convenience — it writes to the same config file.

---

## Implementation Order

```
1. Contract tests (hints_test.go)          — #201
2. Actionable lint warnings + string fix   — #198, #197
3. Skill doc fixes + dispatch table        — #197, #200
4. New skills (gaming, creators)           — #199
5. Preflight step 0                        — #203
6. Portable onboarding                     — #202
```

## Exit Criteria

- All contract tests pass (round-trip, iOS decode, completeness, metadata)
- Lint returns actionable patch instructions for missing fields
- All specialty skills reachable from main router dispatch table
- beepbopboop-gaming and beepbopboop-creators skills exist and produce valid posts
- Preflight blocks batch mode from routing to unavailable skills
- Onboarding works via config file without AskUserQuestion
