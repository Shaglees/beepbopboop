## Config file bootstrap (non-interactive)

If running in an environment without `AskUserQuestion` (OpenClaw, Codex, etc.), the config can be provided as a file:

**Config file location:** `~/.config/beepbopboop/config`

**Required keys (must be present or the skill will stop):**
```
BEEPBOPBOOP_API_URL=http://192.168.1.x:8080
BEEPBOPBOOP_AGENT_TOKEN=<agent-token>
```

**Optional keys:**
```
BEEPBOPBOOP_DEFAULT_LOCATION=Dublin, Ireland
BEEPBOPBOOP_HOME_LAT=53.3498
BEEPBOPBOOP_HOME_LON=-6.2603
BEEPBOPBOOP_INTERESTS=basketball,food,science,gaming
BEEPBOPBOOP_FAMILY=partner:Alex:na:cooking;child:Sam:8:legos,swimming
BEEPBOPBOOP_SOURCES=hn,substack:stratechery
BEEPBOPBOOP_CALENDAR_URL=https://calendar.google.com/...
BEEPBOPBOOP_SCHEDULE=Mon|batch|;Wed|weather|;Fri|batch|
BEEPBOPBOOP_BATCH_MIN=8
BEEPBOPBOOP_BATCH_MAX=15
BEEPBOPBOOP_UNSPLASH_ACCESS_KEY=...
BEEPBOPBOOP_IMGUR_CLIENT_ID=...
BEEPBOPBOOP_GOOGLE_PLACES_KEY=...
YELP_KEY=...
TMDB_KEY=...
RAWG_API_KEY=...
SPOTIFY_TOKEN=...
```

**If config file is missing AND `AskUserQuestion` is not available:**
Print the template above and stop with:
```
Config file not found at ~/.config/beepbopboop/config.
Create it with at minimum BEEPBOPBOOP_API_URL and BEEPBOPBOOP_AGENT_TOKEN, then re-run.
```

---

# Shared: load configuration

Every BeepBopBoop skill starts by loading the same config file. This document is the single source of truth so skills don't drift.

## Step 0: Load config

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

The file is shell-style `KEY=value` lines. Parse into your working set. Required:

- `BEEPBOPBOOP_API_URL`
- `BEEPBOPBOOP_AGENT_TOKEN`

Strongly recommended:

- `BEEPBOPBOOP_DEFAULT_LOCATION` — fallback locality when the user doesn't supply one
- `BEEPBOPBOOP_INTERESTS` — comma-separated list; drives discovery/trending mode selection
- `BEEPBOPBOOP_HOME_ADDRESS` / `BEEPBOPBOOP_HOME_LAT` / `BEEPBOPBOOP_HOME_LON` — precise home for local modes
- `BEEPBOPBOOP_FAMILY` — unlocks family-aware texture (see `beepbopboop-post/FAMILY_CONTEXT.md`)
- `BEEPBOPBOOP_USER_CONTEXT` — freeform personalization
- `BEEPBOPBOOP_CALENDAR_URL` — ICS feed for calendar mode

Image pipeline keys (see `_shared/IMAGES.md`):

- `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` — stock photo search
- `BEEPBOPBOOP_IMGUR_CLIENT_ID` — permanent image hosting (wraps Google Places + Pollinations AI)
- `BEEPBOPBOOP_GOOGLE_PLACES_KEY` — real venue photos
- `BEEPBOPBOOP_POLLINATIONS_TOKEN` — optional; higher rate limits for AI generation

Sport / news keys:

- `BEEPBOPBOOP_SPORTS_TEAMS` — semicolon-separated `league:team-slug`
- `BEEPBOPBOOP_SOURCES` — `hn`, `ph`, `rss:<URL>`, `substack:<URL>`, `reddit:<SUBREDDIT>`

Fashion keys (see `beepbopboop-fashion`):

- `BEEPBOPBOOP_FASHION_STYLES`, `BEEPBOPBOOP_FASHION_BUDGET`, `BEEPBOPBOOP_FASHION_BRANDS`

Batch tuning:

- `BEEPBOPBOOP_BATCH_MIN` (default 8), `BEEPBOPBOOP_BATCH_MAX` (default 15)
- `BEEPBOPBOOP_SCHEDULE` — pipe-separated triplets `DAY|MODE|ARGS` (days: `monday`–`sunday`, `daily`, `weekday`, `weekend`)

## Step 0a: Fetch user profile from server

After loading the config file, fetch the user's profile from the backend API. **Server profile is the primary source of truth** for identity and interests — config file values are fallbacks for fields the server doesn't have.

```bash
curl -s "$BEEPBOPBOOP_API_URL/user/profile" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN"
```

Expected response (JSON):
```json
{
  "identity": {
    "display_name": "Shane",
    "home_location": "Austin, TX",
    "home_lat": 30.2672,
    "home_lon": -97.7431,
    "timezone": "UTC-6"
  },
  "interests": [
    {"category": "Sports", "topic": "sports", "source": "user", "confidence": 1.0},
    {"category": "Food", "topic": "food", "source": "user", "confidence": 1.0}
  ],
  "profile_initialized": true
}
```

**Merge rules** (server wins where present, config fills gaps):

| Field | Server source | Config fallback |
|-------|--------------|-----------------|
| Location name | `identity.home_location` | `BEEPBOPBOOP_DEFAULT_LOCATION` |
| Latitude | `identity.home_lat` | `BEEPBOPBOOP_HOME_LAT` |
| Longitude | `identity.home_lon` | `BEEPBOPBOOP_HOME_LON` |
| Timezone | `identity.timezone` | (none) |
| Interests | `interests[].topic` (comma-joined) | `BEEPBOPBOOP_INTERESTS` |

**If the profile fetch fails** (network error, 401, 500): log a warning and continue with config-file values only. Do not stop the skill — the config file is a valid fallback.

**If `profile_initialized` is false**: the user hasn't completed onboarding yet. Use config-file values and log a note.

## Gate

**Do NOT proceed past Step 0/0a if `BEEPBOPBOOP_API_URL` or `BEEPBOPBOOP_AGENT_TOKEN` are missing.** Instead, invoke the init wizard of the calling skill (e.g. `beepbopboop-post/MODE_INIT.md`).
