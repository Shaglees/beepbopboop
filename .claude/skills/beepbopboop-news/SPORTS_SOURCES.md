# Sports Schedule Sources

Official sources for league schedules. Always fetch from these before falling back to web search.

## How to use

1. Check if the sport/league matches an entry below
2. Check the season window — if out of season, skip or note it to the user
3. Fetch the schedule via ESPN API (preferred) or official website:
   ```bash
   curl -s "https://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard?dates={YYYYMMDD}" | jq '.events[] | {name, date, status: .status.type.description, venue: .competitions[0].venue.fullName, broadcast: .competitions[0].broadcasts[0].names[0]}'
   ```
   - Omit `?dates=` to get today's games
   - Use `?dates=YYYYMMDD` for a specific date
   - Use `?dates=YYYYMMDD-YYYYMMDD` for a date range (max 7 days)
4. Filter results by the user's preferred team (from `BEEPBOPBOOP_SPORTS_TEAMS` config)
5. Use the official data for dates, times, opponents, and venues
6. Only use WebSearch for enrichment (ticket links, venue atmosphere, travel info) — **never for the schedule itself**

## Preferred Teams

Read from `BEEPBOPBOOP_SPORTS_TEAMS` in `~/.config/beepbopboop/config`.

Format: `league:team-slug` pairs separated by semicolons.

Example: `nhl:canucks;mlb:blue-jays;mls:whitecaps-fc;epl:arsenal`

When filtering ESPN API results, match against `.competitions[0].competitors[].team.displayName` (e.g., "Vancouver Canucks") or `.team.abbreviation` (e.g., "VAN"). The `.team.slug` field is often null.

Common team name mappings from config slugs:
- `canucks` → "Vancouver Canucks" / "VAN"
- `blue-jays` → "Toronto Blue Jays" / "TOR"

---

## Leagues

### NHL (National Hockey League)
- Schedule: https://www.nhl.com/schedule
- Team schedule: https://www.nhl.com/{team}/schedule
- ESPN API: `sports/hockey/nhl/scoreboard`
- Season: October–April (playoffs April–June)

### MLB (Major League Baseball)
- Schedule: https://www.mlb.com/schedule
- Team schedule: https://www.mlb.com/{team}/schedule
- ESPN API: `sports/baseball/mlb/scoreboard`
- Season: April–October (postseason Oct–Nov)

### NBA (National Basketball Association)
- Schedule: https://www.nba.com/games
- ESPN API: `sports/basketball/nba/scoreboard`
- Season: October–April (playoffs April–June)

### UFC (Ultimate Fighting Championship)
- Schedule: https://www.ufc.com/events
- ESPN API: `sports/mma/ufc/scoreboard`
- Season: Year-round (numbered events most Saturdays)

### EPL (English Premier League)
- Schedule: https://www.premierleague.com/fixtures
- ESPN API: `sports/soccer/eng.1/scoreboard`
- Season: August–May

### Bundesliga
- Schedule: https://www.bundesliga.com/en/bundesliga/matchday
- ESPN API: `sports/soccer/ger.1/scoreboard`
- Season: August–May

### MLS (Major League Soccer)
- Schedule: https://www.mlssoccer.com/schedule
- ESPN API: `sports/soccer/usa.1/scoreboard`
- Season: February–October (playoffs Oct–Dec)

### Serie A
- Schedule: https://www.legaseriea.it/en/serie-a/fixture-and-results
- ESPN API: `sports/soccer/ita.1/scoreboard`
- Season: August–May

### Ligue 1
- Schedule: https://www.ligue1.com/fixture-results
- ESPN API: `sports/soccer/fra.1/scoreboard`
- Season: August–May

### PGA Tour
- Schedule: https://www.pgatour.com/schedule
- ESPN API: `sports/golf/pga/scoreboard`
- Season: January–August (FedExCup Sept)
- Note: Events span multiple days (Thu–Sun). Check `.competitions[0].status` for round info.

### LPGA Tour
- Schedule: https://www.lpga.com/tournaments
- ESPN API: `sports/golf/lpga/scoreboard`
- Season: January–November
- Note: Same multi-day format as PGA.

### AHL (American Hockey League)
- Schedule: https://theahl.com/stats/schedule
- No ESPN API — use WebFetch on the official schedule page
- Season: October–April (playoffs April–June)

### OHL (Ontario Hockey League)
- Schedule: https://ontariohockeyleague.com/schedule
- No ESPN API — use WebFetch on the official schedule page
- Season: September–March (playoffs March–June)

---

## ESPN API Notes

- Base URL: `https://site.api.espn.com/apis/site/v2/`
- No auth required (public API)
- Returns JSON with `.events[]` array
- Each event has `.competitions[0].competitors[]` with home/away teams
- Team data: `.team.displayName`, `.team.abbreviation`, `.team.slug`
- Venue: `.competitions[0].venue.fullName`, `.venue.address.city`
- Status: `.status.type.description` — "Scheduled", "In Progress", "Final", "Postponed"
- Broadcasts: `.competitions[0].broadcasts[0].names[]`
- Odds (when available): `.competitions[0].odds[0]`
