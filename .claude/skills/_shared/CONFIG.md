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

## Gate

**Do NOT proceed past Step 0 if `BEEPBOPBOOP_API_URL` or `BEEPBOPBOOP_AGENT_TOKEN` are missing.** Instead, invoke the init wizard of the calling skill (e.g. `beepbopboop-post/INIT_WIZARD.md`).
