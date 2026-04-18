# Soccer Enhanced Skill & iOS Card Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a dedicated `beepbopboop-soccer` skill for 6 international leagues and enhance the iOS `ScoreboardCard` / `MatchupCard` with league branding, goal scorers, and cards indicators.

**Architecture:** Extend the existing `GameData` struct with optional soccer fields (`goalScorers`, `leagueColor`, `matchday`, etc.); add `SoccerScoreboardExtras` and `SoccerMatchupHeader` sub-views inside `SportsCards.swift`; create the skill file with ESPN-based data fetching steps for PL/UCL/La Liga/Bundesliga/Serie A/MLS.

**Tech Stack:** SwiftUI (iOS card), Swift Codable (data model), Markdown skill file (Claude skill), ESPN public API (no key), optional API-Football (with key)

**Issues:** Implements #61, child of #50

---

## Task 1: Extend `GameData` with soccer fields

**Files:**
- Modify: `beepbopboop/beepbopboop/Models/SportsData.swift`

**Step 1: Add `GoalScorer` struct and extend `GameData`**

In `SportsData.swift`, after the `GameData` struct's closing brace (line 17), add the `GoalScorer` struct. Then add 7 new optional fields to `GameData`.

Replace the `GameData` struct (lines 6–17):

```swift
struct GameData: Codable {
    let sport: String?
    let league: String?
    let status: String
    let gameTime: String?
    let home: TeamInfo
    let away: TeamInfo
    let headline: String?
    let venue: String?
    let broadcast: String?
    let series: String?
    // Soccer-specific fields
    let leagueShortName: String?
    let leagueColor: String?
    let matchday: String?
    let competition: String?
    let goalScorers: [GoalScorer]?
    let yellowCards: Int?
    let redCards: Int?
}

struct GoalScorer: Codable {
    let player: String
    let team: String        // matches TeamInfo.abbr
    let minute: Int
    let assist: String?
}
```

**Step 2: Extend `isLive` to handle soccer half/HT statuses**

Find the `isLive` computed property (around line 94) and add soccer-specific status values:

```swift
var isLive: Bool {
    let s = status.lowercased()
    return s.hasPrefix("live") || s.contains("period") || s.contains("quarter")
        || s.contains("half") || s.contains("inning")
        || s == "ht" || s == "1h" || s == "2h" || s.hasPrefix("et")
}
```

**Step 3: Add `leagueAccentColor` computed property**

Add after the `statusColor` property (around line 113):

```swift
var leagueAccentColor: Color {
    guard let hex = leagueColor else { return .white.opacity(0.3) }
    let c = Color(hexString: hex)
    return c == .gray ? Color(red: 0.6, green: 0.6, blue: 0.9) : c
}
```

**Step 4: Build to verify no compile errors**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c/beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -20
```

Expected: `BUILD SUCCEEDED`

**Step 5: Commit**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c && git add beepbopboop/beepbopboop/Models/SportsData.swift && git commit -m "feat: extend GameData with soccer fields (goalScorers, leagueColor, matchday)"
```

---

## Task 2: Add `SoccerScoreboardExtras` view

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SportsCards.swift`

**Step 1: Add `SoccerScoreboardExtras` private struct**

Add a new `// MARK: - Soccer Extras` section just before `// MARK: - Shared Components` (before line 546). Insert:

```swift
// MARK: - Soccer Extras

private struct SoccerScoreboardExtras: View {
    let game: GameData

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            // Matchday strip with league accent bar
            if let matchday = game.matchday {
                HStack(spacing: 6) {
                    RoundedRectangle(cornerRadius: 1)
                        .fill(game.leagueAccentColor)
                        .frame(width: 3, height: 10)
                    Text(matchday.uppercased())
                        .font(.system(size: 9, weight: .bold))
                        .tracking(1.2)
                        .foregroundStyle(.white.opacity(0.5))
                }
            }

            // Goal scorers (away left, home right)
            if let scorers = game.goalScorers, !scorers.isEmpty {
                HStack(alignment: .top, spacing: 8) {
                    scorerLine(scorers.filter { $0.team == game.away.abbr }, align: .leading)
                    Spacer(minLength: 0)
                    scorerLine(scorers.filter { $0.team == game.home.abbr }, align: .trailing)
                }
            }

            // Cards indicator
            let yellows = game.yellowCards ?? 0
            let reds = game.redCards ?? 0
            if yellows > 0 || reds > 0 {
                HStack(spacing: 6) {
                    if yellows > 0 { Text("🟨×\(yellows)").font(.system(size: 10)) }
                    if reds > 0   { Text("🟥×\(reds)").font(.system(size: 10)) }
                    Spacer()
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private func scorerLine(_ scorers: [GoalScorer], align: TextAlignment) -> some View {
        if !scorers.isEmpty {
            Text(
                scorers.map { s in
                    let lastName = s.player.components(separatedBy: " ").last ?? s.player
                    let assistStr = s.assist.map { " (\($0.components(separatedBy: " ").last ?? $0))" } ?? ""
                    return "\(s.minute)' \(lastName)\(assistStr)"
                }.joined(separator: " · ")
            )
            .font(.system(size: 9, weight: .medium))
            .foregroundStyle(.white.opacity(0.65))
            .multilineTextAlignment(align)
            .lineLimit(2)
        }
    }
}
```

**Step 2: Add `SoccerMatchupHeader` private struct**

Directly after `SoccerScoreboardExtras`, add:

```swift
private struct SoccerMatchupHeader: View {
    let game: GameData

    var body: some View {
        HStack(spacing: 6) {
            RoundedRectangle(cornerRadius: 1)
                .fill(game.leagueAccentColor)
                .frame(width: 3, height: 12)
            if let shortName = game.leagueShortName {
                Text(shortName)
                    .font(.system(size: 9, weight: .black))
                    .tracking(1.5)
                    .foregroundStyle(game.leagueAccentColor)
            }
            if let matchday = game.matchday {
                Text("·")
                    .foregroundStyle(.white.opacity(0.3))
                Text(matchday.uppercased())
                    .font(.system(size: 9, weight: .bold))
                    .tracking(1)
                    .foregroundStyle(.white.opacity(0.5))
            }
            Spacer()
        }
    }
}
```

**Step 3: Build to verify**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c/beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -20
```

Expected: `BUILD SUCCEEDED`

**Step 4: Commit**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c && git add beepbopboop/beepbopboop/Views/SportsCards.swift && git commit -m "feat: add SoccerScoreboardExtras and SoccerMatchupHeader sub-views"
```

---

## Task 3: Wire soccer extras into `ScoreboardCard`

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SportsCards.swift`

**Step 1: Insert `SoccerScoreboardExtras` between score and headline**

In `ScoreboardCard.body`, find the second `Spacer()` (between the score HStack and the headline VStack, around line 143). Replace:

```swift
                Spacer()

                // Headline stat line + venue
```

with:

```swift
                // Soccer: goal scorers, matchday, cards
                if game.sport?.lowercased() == "soccer" {
                    SoccerScoreboardExtras(game: game)
                        .padding(.horizontal, 4)
                } else {
                    Spacer()
                }

                // Headline stat line + venue
```

**Step 2: Make card height accommodate soccer extras**

Find `.frame(height: 220)` at line 179. Replace with:

```swift
        .frame(height: (game.sport?.lowercased() == "soccer" && game.goalScorers?.isEmpty == false) ? 250 : 220)
```

**Step 3: Build to verify**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c/beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -20
```

Expected: `BUILD SUCCEEDED`

**Step 4: Commit**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c && git add beepbopboop/beepbopboop/Views/SportsCards.swift && git commit -m "feat: wire SoccerScoreboardExtras into ScoreboardCard"
```

---

## Task 4: Wire `SoccerMatchupHeader` into `MatchupCard`

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SportsCards.swift`

**Step 1: Add matchday display to MatchupCard's series area**

In `MatchupCard.body`, find the series pill section (around line 248):

```swift
                    if let series = game.series {
                        Text(series)
                            ...
                    }
```

Replace with:

```swift
                    if game.sport?.lowercased() == "soccer", game.matchday != nil || game.leagueShortName != nil {
                        SoccerMatchupHeader(game: game)
                    } else if let series = game.series {
                        Text(series)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(.white.opacity(0.15))
                            .cornerRadius(6)
                    }
```

**Step 2: Build to verify**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c/beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -20
```

Expected: `BUILD SUCCEEDED`

**Step 3: Commit**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c && git add beepbopboop/beepbopboop/Views/SportsCards.swift && git commit -m "feat: wire SoccerMatchupHeader into MatchupCard for league/matchday context"
```

---

## Task 5: Create the `beepbopboop-soccer` skill file

**Files:**
- Create: `.claude/skills/beepbopboop-soccer/SKILL.md`

**Step 1: Create skill directory and file**

Create `.claude/skills/beepbopboop-soccer/SKILL.md` with the full skill content:

```markdown
---
name: beepbopboop-soccer
description: International soccer coverage — Premier League, Champions League, La Liga, Bundesliga, Serie A, MLS — with goal scorers, league branding, and match context
argument-hint: "[premier league | champions league | la liga | bundesliga | serie a | mls | {team name}]"
allowed-tools: WebFetch, WebSearch, Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *)
---

# BeepBopBoop Soccer Skill

You generate high-quality soccer/football posts for BeepBopBoop: live scores, results, and upcoming fixtures for the 6 top competitions. Every fact comes from the ESPN API or a verified source — never hallucinate scorers, minutes, or standings.

## League Config

| Argument | Competition | ESPN endpoint slug | leagueColor | leagueShortName |
|---|---|---|---|---|
| `premier league`, `epl`, `pl` | Premier League | `soccer.eng.1` | `#3D195B` | `PL` |
| `champions league`, `ucl`, `cl` | Champions League | `soccer.uefa.champions` | `#003082` | `UCL` |
| `la liga`, `laliga` | La Liga | `soccer.esp.1` | `#EE2737` | `LL` |
| `bundesliga`, `buli` | Bundesliga | `soccer.ger.1` | `#D3010C` | `BL` |
| `serie a`, `seriea` | Serie A | `soccer.ita.1` | `#1A56DB` | `SA` |
| `mls` | MLS | `soccer.usa.1` | `#212121` | `MLS` |

---

## Step 0: Load config

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required: `BEEPBOPBOOP_API_URL`, `BEEPBOPBOOP_AGENT_TOKEN`
Optional: `APIFOOTBALL_KEY` (for richer goal/assist data)

---

## Step SC1: Resolve competition

Map the user's argument to one row in the League Config table above. If the argument is a team name (e.g., "Arsenal", "PSG"), infer the competition (Arsenal → Premier League, PSG → Champions League or Ligue 1).

Store: `LEAGUE_SLUG` (ESPN endpoint slug), `LEAGUE_NAME`, `LEAGUE_COLOR`, `LEAGUE_SHORT`.

---

## Step SC2: Fetch ESPN scoreboard

```bash
curl -s "https://site.api.espn.com/apis/site/v2/sports/soccer/${LEAGUE_SLUG}/scoreboard" | jq '{
  events: [.events[]? | {
    id: .id,
    name: .name,
    status: .status.type.shortDetail,
    homeTeam: .competitions[0].competitors[] | select(.homeAway=="home") | {name: .team.shortDisplayName, abbr: .team.abbreviation, score: .score, color: .team.color},
    awayTeam: .competitions[0].competitors[] | select(.homeAway=="away") | {name: .team.shortDisplayName, abbr: .team.abbreviation, score: .score, color: .team.color},
    venue: .competitions[0].venue.fullName,
    date: .date,
    note: .competitions[0].notes[0].headline
  }]
}'
```

**If the league slug returns an error:** Try alternative slug format `soccer/{LEAGUE_SLUG}` or use WebSearch for "ESPN soccer {LEAGUE_NAME} scoreboard API".

---

## Step SC3: Fetch goal scorers (ESPN summary)

For the chosen event ID from SC2:

```bash
EVENT_ID="<id>"
curl -s "https://site.api.espn.com/apis/site/v2/sports/soccer/${LEAGUE_SLUG}/summary?event=${EVENT_ID}" | jq '{
  scoringPlays: [.scoringPlays[]? | {
    player: .scoringPlay.athlete.displayName,
    team: .scoringPlay.team.abbreviation,
    minute: .scoringPlay.clock.displayValue,
    type: .scoringPlay.type.text
  }],
  yellowCards: ([.scoringPlays[]? | select(.scoringPlay.type.text | test("Yellow";"i"))] | length),
  redCards: ([.scoringPlays[]? | select(.scoringPlay.type.text | test("Red";"i"))] | length)
}'
```

**If ESPN summary is missing details:** Use API-Football (if `APIFOOTBALL_KEY` is set):

```bash
curl -H "x-rapidapi-key: ${APIFOOTBALL_KEY}" \
  "https://v3.football.api-sports.io/fixtures?id=${FIXTURE_ID}" | jq '{
  goals: [.response[0].goals[]? | {player: .player.name, team: .team.name, minute: .time.elapsed, assist: .assist.name}],
  yellowCards: [.response[0].statistics[]? | select(.statistics[]?.type=="Yellow Cards") | .statistics[] | select(.type=="Yellow Cards") | .value] | first,
  redCards: [.response[0].statistics[]? | select(.statistics[]?.type=="Red Cards") | .statistics[] | select(.type=="Red Cards") | .value] | first
}'
```

---

## Step SC4: Extract matchday / round

From the ESPN response `.competitions[0].notes[0].headline` or `.season.displayName`, extract the matchday string.
Format as: `"Matchday 32"`, `"Round of 16"`, `"Group Stage"`, `"Final"`, etc.

---

## Step SC5: Classify display hint

| Condition | display_hint | Card |
|---|---|---|
| Game finished (status contains "Final", "FT") | `scoreboard` | ScoreboardCard |
| Game live (status contains "1H", "2H", "HT", "ET") | `scoreboard` | ScoreboardCard (live) |
| Game upcoming (future date) | `matchup` | MatchupCard |
| Multiple games same day | `standings` | StandingsCard |

---

## Step SC6: Compose post copy

**For completed/live scoreboard posts:**
- Title: `"{Winner} {W}–{L} {Loser} | {LEAGUE_SHORT} · {Matchday}"`
- Body: Start with goal scorers and minutes (e.g., "Saka 23', 67' — Havertz 81'"). One sentence on match significance (title race, UCL qualification, relegation battle, upset). Keep under 100 words.
- Avoid: "clinical finish", "world-class", "tactical masterclass", "stunning", "incredible"

**For upcoming fixtures (matchup posts):**
- Title: `"{Away} vs {Home} — {LEAGUE_SHORT} {Matchday}"`
- Body: Context on the fixture (table positions, recent form, what's at stake). Under 60 words.

---

## Step SC7: Build GameData JSON

Construct the `external_url` JSON string. All soccer fields are optional — omit any you don't have data for.

```json
{
  "sport": "soccer",
  "league": "Premier League",
  "leagueShortName": "PL",
  "leagueColor": "#3D195B",
  "status": "Final",
  "gameTime": null,
  "matchday": "Matchday 32",
  "competition": "Premier League",
  "home": {
    "name": "Arsenal",
    "abbr": "ARS",
    "score": 3,
    "record": "22-4-5",
    "color": "#EF0107"
  },
  "away": {
    "name": "Tottenham",
    "abbr": "TOT",
    "score": 1,
    "record": "15-9-7",
    "color": "#132257"
  },
  "venue": "Emirates Stadium",
  "broadcast": null,
  "series": null,
  "headline": "Saka 23', 67' — Havertz 81'",
  "goalScorers": [
    { "player": "Bukayo Saka", "team": "ARS", "minute": 23, "assist": "Martinelli" },
    { "player": "Bukayo Saka", "team": "ARS", "minute": 67, "assist": null },
    { "player": "Kai Havertz", "team": "ARS", "minute": 81, "assist": "Ødegaard" },
    { "player": "Heung-min Son", "team": "TOT", "minute": 55, "assist": null }
  ],
  "yellowCards": 2,
  "redCards": 0
}
```

**Important:** The team colors in `home.color` and `away.color` drive the card gradient. Use the official team hex color. For well-known clubs:
- Arsenal: `#EF0107` | Tottenham: `#132257` | Man City: `#6CABDD` | Liverpool: `#C8102E`
- Real Madrid: `#FEBE10` | Barcelona: `#A50044` | PSG: `#004170` | Bayern: `#DC052D`
- Inter Milan: `#010E80` | AC Milan: `#FB090B` | Juventus: `#000000`
- NYCFC: `#003DA5` | LA Galaxy: `#00245D` | Portland Timbers: `#00482B`

---

## Step SC8: Publish post

```bash
GAME_DATA_JSON='<JSON from SC7>'
LABELS='["sports","soccer","premier-league"]'  # adjust per league

curl -s -X POST "${BEEPBOPBOOP_API_URL}/posts" \
  -H "Authorization: Bearer ${BEEPBOPBOOP_AGENT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"<title from SC6>\",
    \"body\": \"<body from SC6>\",
    \"display_hint\": \"<scoreboard|matchup|standings>\",
    \"post_type\": \"event\",
    \"external_url\": $(echo $GAME_DATA_JSON | jq -c . | jq -Rs .),
    \"labels\": ${LABELS},
    \"visibility\": \"public\"
  }"
```

**Labels by league:**
- Premier League: `["sports","soccer","premier-league","epl"]`
- Champions League: `["sports","soccer","champions-league","ucl"]`
- La Liga: `["sports","soccer","la-liga"]`
- Bundesliga: `["sports","soccer","bundesliga"]`
- Serie A: `["sports","soccer","serie-a"]`
- MLS: `["sports","soccer","mls"]`

Verify `"id"` is present in the response. If you get a 4xx, check the JSON escaping of `external_url`.

---

## Step SC9: Multiple leagues / standings

If generating a multi-match roundup for the same competition (e.g., all PL results from a matchday), use `display_hint: "standings"` with `StandingsData` format:

```json
{
  "league": "Premier League",
  "leagueColor": "#3D195B",
  "date": "2026-04-17",
  "games": [
    { "home": "ARS", "away": "TOT", "homeScore": 3, "awayScore": 1, "status": "FT", "homeColor": "#EF0107", "awayColor": "#132257" },
    { "home": "LIV", "away": "CHE", "homeScore": 2, "awayScore": 2, "status": "FT", "homeColor": "#C8102E", "awayColor": "#034694" }
  ],
  "headline": "Arsenal close gap on City — Matchday 32 results"
}
```
```

**Step 2: Build to verify (no iOS changes here, just ensure skill file is valid Markdown)**

```bash
head -10 /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c/.claude/skills/beepbopboop-soccer/SKILL.md
```

Expected: frontmatter with `name: beepbopboop-soccer`

**Step 3: Commit**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c && git add .claude/skills/beepbopboop-soccer/SKILL.md && git commit -m "feat: add beepbopboop-soccer skill for PL/UCL/La Liga/Bundesliga/Serie A/MLS"
```

---

## Task 6: Final iOS build verification

**Step 1: Full clean build**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c/beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' clean build 2>&1 | grep -E "(error:|warning:|BUILD)"
```

Expected: `BUILD SUCCEEDED` with no errors.

**Step 2: If there are errors**, fix them and rebuild before moving to Task 7.

---

## Task 7: Update issue and create PR

**Step 1: Comment on issue #61 with progress**

```bash
gh issue comment 61 --body "## Implementation complete

### Changes
- **\`.claude/skills/beepbopboop-soccer/SKILL.md\`** — new skill covering PL, UCL, La Liga, Bundesliga, Serie A, MLS with ESPN-based data fetching, goal scorer extraction, and full GameData JSON construction
- **\`SportsData.swift\`** — extended \`GameData\` with \`goalScorers\`, \`leagueColor\`, \`leagueShortName\`, \`matchday\`, \`competition\`, \`yellowCards\`, \`redCards\`; added \`GoalScorer\` struct; enhanced \`isLive\` for soccer halves (HT/1H/2H/ET)
- **\`SportsCards.swift\`** — added \`SoccerScoreboardExtras\` (matchday strip + goal scorers + cards indicator) and \`SoccerMatchupHeader\` (league accent bar + matchday); wired into \`ScoreboardCard\` and \`MatchupCard\`

App builds clean. PR incoming."
```

**Step 2: Create PR**

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/exciting-franklin-9bcb9c && gh pr create \
  --title "feat: soccer skill + enhanced ScoreboardCard with goal scorers and league branding (#61)" \
  --body "$(cat <<'EOF'
## Summary

Implements #61 — dedicated soccer skill and enhanced iOS card for international soccer coverage.

### Changes

**New skill:** `.claude/skills/beepbopboop-soccer/SKILL.md`
- 6 competitions: Premier League, Champions League, La Liga, Bundesliga, Serie A, MLS
- ESPN scoreboard + summary endpoints for live/final data
- Optional API-Football integration for richer goal/assist data
- Full `GameData` JSON construction with league branding fields
- `scoreboard` / `matchup` / `standings` display hint classification

**`SportsData.swift`**
- Added `GoalScorer` struct (`player`, `team`, `minute`, `assist?`)
- Extended `GameData` with: `goalScorers?`, `leagueColor?`, `leagueShortName?`, `matchday?`, `competition?`, `yellowCards?`, `redCards?`
- Added `leagueAccentColor` computed property
- Enhanced `isLive` to handle soccer statuses: `HT`, `1H`, `2H`, `ET`

**`SportsCards.swift`**
- `SoccerScoreboardExtras` sub-view: matchday strip with league color accent bar, goal scorers by team (minute + last name + assist), cards indicator (🟨/🟥)
- `SoccerMatchupHeader` sub-view: league shortname + matchday in accent color
- `ScoreboardCard`: wires in soccer extras; height 220→250 for games with scorers
- `MatchupCard`: shows soccer matchday/league header instead of series pill

## Test plan

- [ ] Publish a Premier League result post using `/beepbopboop-soccer premier league` — verify ScoreboardCard renders with PL purple accent, goal scorers, and cards
- [ ] Publish a UCL upcoming fixture — verify MatchupCard shows "UCL · Round of 16" header
- [ ] Publish an MLS multi-game standings post — verify StandingsCard header is dark with MLS branding
- [ ] Verify app builds clean with `xcodebuild`
- [ ] Verify old non-soccer sports posts still render correctly (no regressions)

Closes #61

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

**Step 3: Close the loop on the issue**

After PR is created, note the PR URL and add it to the issue comment if needed.

---

## Success Criteria Checklist

- [ ] `.claude/skills/beepbopboop-soccer/SKILL.md` covers 6 leagues with correct ESPN endpoint slugs
- [ ] `GameData` extended with `goalScorers`, `leagueColor`, `leagueShortName`, `matchday`
- [ ] `SoccerScoreboardExtras` renders: matchday strip with accent bar, goal scorer minutes, cards indicator
- [ ] `SoccerMatchupHeader` renders: league shortname + matchday in header
- [ ] `isLive` handles HT/1H/2H/ET for soccer
- [ ] App builds clean (`BUILD SUCCEEDED`)
- [ ] PR created referencing #61
- [ ] Issue #61 updated with progress comment
