# Sports Team/League Following — Design

_Issue #30 | 2026-04-17_

## Goal

Let users express team and league preferences. Followed teams' posts rank higher in the ForYou feed. Zero-config users see current behaviour.

## Data model

```sql
ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS followed_teams JSONB;
-- e.g. ["nba:lal", "nhl:van", "mlb:tor"]
```

Format: `"{league}:{team_abbr_lowercase}"` matching `external_url.sport` + `external_url.home.abbr`/`away.abbr` on sports posts.

## Backend changes

| File | Change |
|---|---|
| `database/database.go` | `ALTER TABLE … ADD COLUMN IF NOT EXISTS followed_teams JSONB` |
| `model/model.go` | `UserSettings.FollowedTeams []string` |
| `repository/user_settings_repo.go` | `Get` scans JSONB; `Upsert` writes it |
| `handler/settings.go` | `updateSettingsRequest.FollowedTeams []string` |
| `repository/post_repo.go` | `FeedWeights.FollowedTeams map[string]bool`; `scorePost` boosts +1.5 per matched team |
| `handler/multi_feed.go` | `GetForYou` populates `FeedWeights.FollowedTeams` from settings |

### Team extraction in scorePost

Parse `external_url` JSON for `sport`, `home.abbr`, `away.abbr`. Build keys `"{sport}:{abbr_lower}"`. If any key is in `w.FollowedTeams`, add +1.5 to score.

## iOS changes

| File | Change |
|---|---|
| `Models/UserSettings.swift` | `followedTeams: [String]?` + coding key |
| `Views/SportsSettingsView.swift` | New view: league sections → team toggles |
| `Views/SettingsView.swift` | "Sports & Teams" `NavigationLink` row |
| `ViewModels/SettingsViewModel` | Load/save `followedTeams` alongside existing fields |

Static team list bundled in `SportsSettingsView`. Four leagues (NHL, NBA, MLB, NFL) with ~10 popular teams each.

## Feed integration

`scorePost()` only runs when `w.FollowedTeams` is non-nil and non-empty. Users with no preferences see current recency-based ranking.

## Tests

`handler/settings_test.go`: GET returns `followed_teams`; PUT round-trips the array.
