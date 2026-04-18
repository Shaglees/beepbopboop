---
name: beepbopboop-basketball
description: Deep NBA/WNBA coverage — player spotlights, trade news, draft, stat leaders, beyond box scores
argument-hint: "[player name | trade news | draft | stat leaders | {team name}]"
allowed-tools: WebFetch, WebSearch, Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *)
---

# BeepBopBoop Basketball Skill

Generate player-centric NBA/WNBA posts. The `beepbopboop-news` skill covers game-level `scoreboard`/`matchup`/`standings` cards — this skill goes deeper into individual player performance, trades, and draft coverage.

## Important

- Every stat must come from ESPN API — never hallucinate numbers
- Always fetch the player's actual headshot URL from ESPN CDN
- Use `player_spotlight` display hint only for single-player performance posts
- Trade/draft content → `article` display hint (no structured data needed)
- Kill list: "balling out", "elite performance", "on another level", "playing like an MVP"

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required: `BEEPBOPBOOP_API_URL`, `BEEPBOPBOOP_AGENT_TOKEN`

## Step 0a: Parse command

| User input | Mode | Jump to |
|---|---|---|
| Player name (e.g. "Shai", "LeBron") | Player spotlight | Steps BB1–BB6 |
| "trade news", "trades" | Trade coverage | Steps BB1, BB3, BB5 (article) |
| "draft", "mock draft" | Draft coverage | Steps BB1, BB3, BB5 (article) |
| "stat leaders", "scoring leaders" | Stat leaders | Steps BB1, BB5 (article) |
| Team name (e.g. "Thunder", "Lakers") | Team overview | Steps BB1–BB6 |

---

## Steps BB1–BB6: Player Spotlight

### Step BB1 — Resolve subject

Determine what was requested:
- Player name → proceed to BB2 with player search
- Trade news → skip to BB3 (WebSearch only), post as `article`
- Draft → skip to BB3 (WebSearch only), post as `article`
- Stat leaders → fetch ESPN leaderboard, post as `article`
- Team name → find top performer from most recent game, proceed to BB2

```bash
# Check for duplicates before proceeding
beepbopgraph check "{player_name} basketball" 2>/dev/null
# If a recent post exists for the same player/game, skip to avoid duplicates
```

### Step BB2 — Fetch ESPN player data

> **WNBA players:** Replace `nba` with `wnba` in all ESPN API paths, use `league=wnba` in search URL, and use headshot URL: `https://a.espncdn.com/i/headshots/wnba/players/full/{athlete_id}.png`

**Search for player ID (extract athlete ID from uid field):**
```bash
curl -s "https://site.web.api.espn.com/apis/search/v2?query={player_name}&limit=5&sport=basketball&league=nba" | jq -r '.results[0].contents[0] | {id: (.uid | split("~a:")[1]), displayName}'
```

**Season statistics (use the year the season ends, e.g. 2026 for 2025-26 season):**
```bash
curl -s "https://sports.core.api.espn.com/v2/sports/basketball/leagues/nba/seasons/2026/types/2/athletes/{athlete_id}/statistics/0?lang=en&region=us" \
  -H "Accept: application/json" | jq '{
  ppg: (.splits.categories[] | select(.name=="offensive") | .stats[] | select(.name=="avgPoints") | .value),
  rpg: (.splits.categories[] | select(.name=="general") | .stats[] | select(.name=="avgRebounds") | .value),
  apg: (.splits.categories[] | select(.name=="offensive") | .stats[] | select(.name=="avgAssists") | .value)
}'
```

**Recent game stats — use WebSearch as most reliable source:**
```bash
# Search: "{player name} stats last game site:espn.com OR site:basketball-reference.com"
# Extract: points, rebounds, assists, steals, blocks, FG%, 3P%, +/-
# Fallback: check NBA.com box scores for the most recent game
```

Extract from API response:
- `playerName`, `playerId`, `team`, `teamAbbr`, `teamColor`, `position`
- Headshot URL pattern: `https://a.espncdn.com/combiner/i?img=/i/headshots/nba/players/full/{athlete_id}.png`
- Last game: `points`, `rebounds`, `assists`, `steals`, `blocks`, `fieldGoalPct`, `threePointPct`, `plusMinus`
- Season averages: PPG, RPG, APG
- Game result: W/L, opponent, score
- `teamColor`: Use WebSearch `"{team name} primary color hex"` if not returned by the API.

### Step BB3 — Contextual enrichment

WebSearch: `"{player name} NBA latest news"` — extract 1-2 recent storylines (injury update, trade rumour, milestone). Keep it to one sentence for `storyline` field.

### Step BB4 — Classify display_hint

- Single player performance → `player_spotlight`
- Trade/transaction news → `article`
- Draft content → `article`
- Stat leaders digest → `article`

### Step BB5 — Compose post title and body

**For player_spotlight:**
```
title: "{Player Name} — {key stat highlight} | {Team} {optional context}"
  e.g.: "Shai Gilgeous-Alexander — 36pts, 7ast in OT | Thunder lead 3-1"
  e.g.: "Nikola Jokić — Triple-double night as Nuggets survive Dallas scare"

body: 2-3 sentences. Frame stats in series/season context. Name the opponent.
      One sentence on what makes this performance notable historically or in the standings.
      If storyline exists, weave it in naturally.
```

**For article (trade/draft):**
```
title: Clear, factual headline
body:  Who, what, when. Player names, teams, terms if known. What it means for both sides.
```

```bash
# Publish article (trade/draft/stat-leaders)
curl -s -X POST "{API_URL}/posts" \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"...\",
    \"body\": \"...\",
    \"post_type\": \"article\",
    \"display_hint\": \"article\",
    \"labels\": [\"basketball\", \"nba\"]
  }"
```

### Step BB6 — Build external_url JSON and publish

For `player_spotlight` posts only, build this JSON object for `external_url`:

```json
{
  "type": "player_spotlight",
  "sport": "basketball",
  "league": "NBA",
  "playerId": "{athlete_id}",
  "playerName": "{full name}",
  "playerHeadshotUrl": "https://a.espncdn.com/combiner/i?img=/i/headshots/nba/players/full/{athlete_id}.png",
  "team": "{full team name}",
  "teamAbbr": "{3-letter abbr}",
  "teamColor": "{hex color}",
  "position": "{Guard|Forward|Center}",
  "gameDate": "{YYYY-MM-DD}",
  "opponent": "{opponent team name}",
  "gameResult": "{W|L} {score}",
  "lastGameStats": {
    "points": 0,
    "rebounds": 0,
    "assists": 0,
    "steals": 0,
    "blocks": 0,
    "fieldGoalPct": 0.000,
    "threePointPct": 0.000,
    "plusMinus": 0
  },
  "seasonAverages": {
    "points": 0.0,
    "rebounds": 0.0,
    "assists": 0.0
  },
  "seriesContext": "{e.g. OKC lead series 3-1 — omit if not playoffs}",
  "storyline": "{one sentence context — omit if nothing notable}"
}
```

**Note:** `external_url` must be passed as a serialised JSON string, not a raw object. Serialise the object to a string before placing it in the curl body.

**Publish:**
```bash
# Parse API URL and token from config (already loaded in Step 0):
#   API_URL = value of BEEPBOPBOOP_API_URL from config
#   TOKEN   = value of BEEPBOPBOOP_AGENT_TOKEN from config
# Substitute literal values below — do NOT rely on shell env vars

curl -s -X POST "{API_URL}/posts" \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"...\",
    \"body\": \"...\",
    \"post_type\": \"discovery\",
    \"display_hint\": \"player_spotlight\",
    \"labels\": [\"basketball\", \"nba\", \"{team_abbr_lowercase}\"],
    \"external_url\": \"{serialised JSON string — must be a string, not a JSON object}\"
  }"
```

Verify response contains `"id"` field confirming creation.

```bash
# Save to graph to prevent duplicates
beepbopgraph save "{player_name} {gameDate} basketball" 2>/dev/null
```
