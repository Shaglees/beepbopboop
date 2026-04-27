---
name: beepbopboop-baseball
description: Create MLB baseball posts — box scores, pitcher/batter highlights, standings, trade news
argument-hint: "[game result | {team name} | pitching matchup | standings | trades]"
allowed-tools: WebFetch, WebSearch, Bash
---

# BeepBopBoop Baseball Skill

Generate rich MLB baseball posts with pitcher/batter stat lines. Produces `box_score` display_hint posts with structured JSON for the BoxScoreCard iOS view.

## Step BS0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Parse and store: `BEEPBOPBOOP_API_URL`, `BEEPBOPBOOP_AGENT_TOKEN`, `BEEPBOPBOOP_SPORTS_TEAMS` (for preferred team filtering).

Read `../_shared/SPORTS_COMMON.md` for shared sport conventions (source rules, display hints, labels, team data, publishing).

**Do NOT proceed if `BEEPBOPBOOP_API_URL` or `BEEPBOPBOOP_AGENT_TOKEN` are missing.**

## Step BS1: Resolve subject

Parse the user's argument to determine fetch mode:

| Argument | Mode | Notes |
|----------|------|-------|
| `game result`, team name, no arg | Game result | Fetch yesterday's/today's completed games |
| `pitching matchup` | Upcoming starters | Today's/tonight's probable pitchers |
| `standings` | Division standings | AL/NL standings snapshot |
| `trades` | Trade news | Recent MLB trade tracker headlines |

If `BEEPBOPBOOP_SPORTS_TEAMS` contains an MLB team slug (e.g. `mlb:yankees`), prioritise that team's game.

## Step BS2: Fetch MLB data from ESPN

### Completed game (box score mode)

```bash
# Today's scoreboard
curl -s "https://site.api.espn.com/apis/site/v2/sports/baseball/mlb/scoreboard"

# Yesterday's scoreboard (if no games today)
curl -s "https://site.api.espn.com/apis/site/v2/sports/baseball/mlb/scoreboard?dates=$(date -v-1d +%Y%m%d 2>/dev/null || date -d yesterday +%Y%m%d)"
```

Find a completed game. Then fetch the box score summary:

```bash
curl -s "https://site.api.espn.com/apis/site/v2/sports/baseball/mlb/summary?event={game_id}"
```

Extract from the summary response:
- Final score, innings played, extra innings flag (check if innings > 9)
- Winning pitcher: name, record (W-L), ERA, innings pitched, strikeouts
- Losing pitcher: name, record (W-L), ERA, innings pitched, strikeouts
- Save pitcher (if present): name, saves total
- Key batter (HR/RBI leader for the game): name, team abbr, HR count, RBI, batting avg
- Home/away team: name, abbreviation, score, record, primary color (hex)
- Venue name

### Pitching matchup mode

```bash
curl -s "https://site.api.espn.com/apis/site/v2/sports/baseball/mlb/scoreboard"
```

Find scheduled games. Extract probable starters for tonight's games. Use `matchup` display_hint with pitcher names in the headline field. Skip to Step BS4 with `display_hint: "matchup"`.

For matchup posts, use this external_url template:
```json
{
  "sport": "baseball",
  "league": "MLB",
  "date": "YYYY-MM-DDThh:mm:ss-07:00",
  "home": { "name": "Home Team", "abbr": "HOM", "record": "W-L", "color": "#RRGGBB" },
  "away": { "name": "Away Team", "abbr": "AWY", "record": "W-L", "color": "#RRGGBB" },
  "venue": "Stadium Name",
  "broadcast": "Network",
  "headline": "Pitcher A vs Pitcher B"
}
```

### Standings mode

```bash
curl -s "https://site.api.espn.com/apis/site/v2/sports/baseball/mlb/standings"
```

Extract division standings rows. Use `standings` display_hint. Skip to Step BS4 with `display_hint: "standings"`.

### Trades mode

Use WebSearch: `site:espn.com mlb trades {current month} {year}` — summarise top 2-3 moves. Use standard card with `display_hint: "card"`.

## Step BS3: Build display_hint decision

| Condition | display_hint |
|-----------|-------------|
| Completed game with pitcher/batter data | `box_score` |
| Upcoming game with probable starters | `matchup` |
| Division standings snapshot | `standings` |
| Trade news / other narrative | `card` |

## Step BS4: Compose post text

**For box_score posts:**

```
title: "{Winner} {W-score}–{L-score} {Loser} | {Pitcher} wins, {Batter} goes deep"
body: Lead with how the game was decided. Name the starting pitchers and their outcomes.
      One standout offensive performance. Final record context ("extends lead in AL East").
      2 sentences max.
```

**For matchup posts:**

```
title: "{Away} @ {Home} | {Away starter} vs {Home starter}"
body: Preview tonight's pitching matchup. Include ERA, recent form, series context if applicable.
      2 sentences max.
```

## Step BS5: Build external_url JSON (box_score only)

Construct the JSON payload for the `external_url` field:

```json
{
  "sport": "baseball",
  "league": "MLB",
  "status": "Final",
  "innings": 9,
  "extraInnings": false,
  "home": {
    "name": "Yankees",
    "abbr": "NYY",
    "score": 5,
    "record": "18-12",
    "color": "#003087"
  },
  "away": {
    "name": "Red Sox",
    "abbr": "BOS",
    "score": 3,
    "record": "15-14",
    "color": "#BD3039"
  },
  "winningPitcher": {
    "name": "Gerrit Cole",
    "record": "4-1",
    "era": "2.34",
    "inningsPitched": 7.0,
    "strikeouts": 9
  },
  "losingPitcher": {
    "name": "Nick Pivetta",
    "record": "2-3",
    "era": "4.56",
    "inningsPitched": 5.1,
    "strikeouts": 4
  },
  "savePitcher": {
    "name": "Clay Holmes",
    "saves": 8
  },
  "keyBatter": {
    "name": "Aaron Judge",
    "team": "NYY",
    "hr": 1,
    "rbi": 3,
    "avg": ".297"
  },
  "headline": "Cole deals 7 strong, Judge crushes 2-run shot in 7th",
  "venue": "Yankee Stadium"
}
```

Rules:
- `status`: `"Final"` for completed games, `"F/10"` for extra innings (use `"F/" + innings` when `extraInnings: true`)
- `innings`: actual innings played (9 for regulation, 10+ for extras)
- `extraInnings`: `true` when innings > 9
- `savePitcher`: omit entirely if no save was recorded
- `keyBatter`: omit entirely if no standout batter data available
- `inningsPitched`: use decimal format (5.1 = 5 and ⅓ innings, 5.2 = 5 and ⅔ innings)
- All pitcher/batter fields are optional — omit rather than guess

## Step BS6: Publish the post

```bash
PAYLOAD=$(jq -n \
  --arg title "..." \
  --arg body "..." \
  --argjson external_url "$(echo "$BOX_SCORE_JSON" | jq -c . | jq -Rs .)" \
  --arg locality "{city where game was played}" \
  '{
    title: $title, body: $body, display_hint: "box_score",
    external_url: $external_url, locality: $locality,
    labels: ["baseball", "mlb", "sports", "{home_team_slug}", "{away_team_slug}"]
  }')

# Lint pre-flight
LINT=$(curl -s -X POST "${BEEPBOPBOOP_API_URL}/posts/lint" \
  -H "Authorization: Bearer ${BEEPBOPBOOP_AGENT_TOKEN}" \
  -H "Content-Type: application/json" -d "$PAYLOAD")
if [ "$(echo "$LINT" | jq -r '.valid')" != "true" ]; then
  echo "$LINT" | jq .; exit 1
fi

# Publish with 422 retry
RESP=$(curl -s -o /tmp/bbp_resp.json -w "%{http_code}" -X POST "${BEEPBOPBOOP_API_URL}/posts" \
  -H "Authorization: Bearer ${BEEPBOPBOOP_AGENT_TOKEN}" \
  -H "Content-Type: application/json" -d "$PAYLOAD")
if [ "$RESP" = "422" ]; then
  CORRECTED=$(cat /tmp/bbp_resp.json | jq -r '.corrected_external_url')
  PAYLOAD=$(echo "$PAYLOAD" | jq --arg u "$CORRECTED" '.external_url = $u')
  curl -s -X POST "${BEEPBOPBOOP_API_URL}/posts" \
    -H "Authorization: Bearer ${BEEPBOPBOOP_AGENT_TOKEN}" \
    -H "Content-Type: application/json" -d "$PAYLOAD" | jq .
else
  cat /tmp/bbp_resp.json | jq .
fi
```

Check the response for `"valid": true`. If there are validation errors, fix and retry.

Report to the user:
- Post title and body
- Teams and final score
- Pitcher/batter highlights used
- Post ID from the API response
