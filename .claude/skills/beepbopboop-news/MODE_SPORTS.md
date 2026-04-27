# Sports mode (SP1–SP3)

**Trigger:** `sports`, `games`, `scores`, or any league/team name from `BEEPBOPBOOP_SPORTS_TEAMS`.

## SP1: Load sports sources

Read `SPORTS_SOURCES.md` from the sibling skill directory:

```bash
cat ~/.claude/skills/beepbopboop-post/SPORTS_SOURCES.md 2>/dev/null
```

Parse `BEEPBOPBOOP_SPORTS_TEAMS` from config. Format: `league:team-slug` pairs separated by semicolons.

## SP2: Fetch schedules for preferred teams

For each preferred team, fetch upcoming games via ESPN API:

```bash
# Today's and next 7 days of games
curl -s "https://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard?dates=$(date +%Y%m%d)-$(date -v+7d +%Y%m%d 2>/dev/null || date -d '+7 days' +%Y%m%d)" | jq '.events[] | {name, date, status: .status.type.description, venue: .competitions[0].venue.fullName, broadcast: .competitions[0].broadcasts[0].names[0]}'
```

League-to-API mappings (full list in `SPORTS_SOURCES.md`):

- `nhl` → `sports/hockey/nhl/scoreboard`
- `mlb` → `sports/baseball/mlb/scoreboard`
- `nba` → `sports/basketball/nba/scoreboard`
- `mls` → `sports/soccer/usa.1/scoreboard`
- `epl` → `sports/soccer/eng.1/scoreboard`
- `bundesliga` → `sports/soccer/ger.1/scoreboard`
- `seriea` → `sports/soccer/ita.1/scoreboard`
- `ligue1` → `sports/soccer/fra.1/scoreboard`
- `ufc` → `sports/mma/ufc/scoreboard`
- `pga` → `sports/golf/pga/scoreboard`
- `lpga` → `sports/golf/lpga/scoreboard`

For AHL and OHL (no ESPN API), use `WebFetch` on their official schedule page.

Filter results to the user's preferred team(s).

## SP3: Generate sports posts

### Upcoming game (`status: "Scheduled"`)

- `title`: `"[Team] vs [Opponent] — [Day of week]"` or `"[Team] at [Opponent] — [Day]"`
- `body`: date/time (user's timezone), venue, broadcast info, any relevant storyline from a quick `WebSearch "[team] [opponent] preview"`
- `post_type`: `event`
- `display_hint`: `matchup`
- `external_url`: **JSON object** (NOT a ticket link) with structured game data for the iOS matchup card:

  ```json
  {
    "sport": "hockey",
    "league": "NHL",
    "status": "Scheduled",
    "date": "2026-04-17T18:00:00-07:00",
    "home": { "name": "Oilers", "abbr": "EDM", "record": "45-25-4", "color": "#041E42" },
    "away": { "name": "Canucks", "abbr": "VAN", "record": "42-28-6", "color": "#00205B" },
    "venue": "Rogers Place",
    "broadcast": "ESPN+",
    "series": "Game 3 · Series tied 1-1"
  }
  ```

  Include `series` only during playoffs. Team colors = team's primary brand hex. Use ESPN API for records/venue/broadcast. `date` is ISO-8601 with timezone offset.
- `labels`: `["sports", "<league>", "<team-slug>", "event"]`

### Recent result (`status: "Final"`)

- `title`: `"[Team] [W/L] [Score] — [Headline moment]"`
- `body`: final score, key moments, standout performers. Quick `WebSearch "[team] game recap"` for color.
- `post_type`: `article`
- `display_hint`: `scoreboard`
- `external_url`: **JSON object** (NOT a recap link):

  ```json
  {
    "sport": "hockey",
    "league": "NHL",
    "status": "Final",
    "home": { "name": "Canucks", "abbr": "VAN", "score": 5, "record": "42-28-6", "color": "#00205B" },
    "away": { "name": "Ducks", "abbr": "ANA", "score": 2, "record": "28-38-8", "color": "#F47A38" },
    "headline": "Miller 2G 1A · Demko 31 saves",
    "venue": "Rogers Arena",
    "broadcast": "Sportsnet"
  }
  ```

  `headline` = key stat line from the game (top performers, notable achievements).
- `labels`: `["sports", "<league>", "<team-slug>", "recap"]`

### Daily roundup (3+ games from same league on same day)

Instead of individual scoreboard posts, create a single standings/digest:

- `title`: `"[League] Scores — [Date]"`
- `body`: brief summary of the day's action
- `post_type`: `article`
- `display_hint`: `standings`
- `external_url`: **JSON object** with multi-game data:

  ```json
  {
    "league": "NHL",
    "leagueColor": "#000000",
    "date": "2026-04-16",
    "games": [
      { "home": "VAN", "away": "ANA", "homeScore": 5, "awayScore": 2, "status": "Final", "homeColor": "#00205B", "awayColor": "#F47A38" },
      { "home": "EDM", "away": "CGY", "homeScore": 3, "awayScore": 1, "status": "Final", "homeColor": "#041E42", "awayColor": "#D2001C" }
    ],
    "headline": "Canucks clinch playoff spot"
  }
  ```

  Include ALL games from that league on that day, not just the user's preferred team.
- `labels`: `["sports", "<league>", "digest"]`

### Team news (always check, with date guardrail)

1. Gather 5–10 candidate links:
   - `WebSearch "<team-name> latest news"` and `WebSearch "<team-name> injury update"`
   - For each, extract `title`, `url`, `published_at` (ISO-8601 if available)
2. Validate publication date against the user's local date before writing:

   ```bash
   cat /tmp/sports_news_candidates.json | \
     python3 ./scripts/filter_sports_news_by_date.py \
       --timezone America/Vancouver \
       --max-age-days 10 > /tmp/sports_news_filtered.json
   ```

3. Only use items from `fresh[]` (last 10 days in local TZ). Never use `stale[]` or `invalid[]`.
4. If `fresh[]` is empty, **skip** team-news generation for that team.
5. If newsworthy fresh items exist, generate **article** posts:
   - `post_type`: `article`
   - `display_hint`: `article`
   - `labels`: `["sports", "<league>", "<team-slug>", "news"]`
   - Mention publication date in body when context is time-sensitive (injury, lineup, playoff status).

After generating posts, proceed to `COMMON_PUBLISH.md`.
