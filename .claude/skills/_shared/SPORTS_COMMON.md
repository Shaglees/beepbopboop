# Sports Common Patterns

Shared conventions for all `beepbopboop-*` sport skills. Read this file once at Step 0 alongside CONFIG.md.

## Source Rules

- **Never hallucinate stats, scores, or records.** Every number must come from an API response or a cited web source.
- Use official league APIs, ESPN, or team sites as primary sources. Fan sites and social media are secondary.
- If a stat cannot be verified, omit it rather than guess.

## Display Hints

Sport skills produce posts with these structured display hints:

| Hint | When to use | Required external_url fields |
|------|-------------|------------------------------|
| `scoreboard` | Live or final game scores | `home_team`, `away_team`, `home_score`, `away_score`, `status`, `period` |
| `matchup` | Upcoming game previews | `home_team`, `away_team`, `date`, `venue`, `odds` (optional) |
| `standings` | League/division standings | `entries[]` with `team`, `wins`, `losses`, `pct`, `gb` |
| `box_score` | Detailed post-game stats | `home_team`, `away_team`, `home_score`, `away_score`, `leaders[]` |
| `player_spotlight` | Individual player features | `name`, `team`, `position`, `stats{}`, `headline` |

All `external_url` values must be JSON strings (not raw objects). Use the canonical pattern from `PUBLISH_ENVELOPE.md`:
```bash
EXTERNAL_URL=$(echo "$DATA_JSON" | jq -c . | jq -Rs .)
```

## Labels

Every sport post must include:
1. The sport label: `sports` (always first)
2. The league: `nba`, `nfl`, `mlb`, `premier-league`, `champions-league`, etc.
3. Team slugs if applicable: `nba:lal`, `nfl:sf`, `mlb:nyy`

## Team Data

If `BEEPBOPBOOP_SPORTS_TEAMS` is set in config, prioritize those teams. Format: comma-separated league:abbrev pairs (e.g., `nba:lal,nfl:sf,mlb:nyy`).

## Publishing

After building the post payload, follow `../_shared/PUBLISH_ENVELOPE.md` for lint + publish. Always lint before POST.
