# Baseball Skill + BoxScoreCard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a baseball-specific Claude skill and iOS BoxScoreCard view with pitcher/batter stat lines for the `box_score` display_hint.

**Architecture:** Skill file drives MLB content generation via ESPN API → `BaseballData` struct stored in `externalURL` JSON → `BoxScoreCard` view decodes and renders pitcher/batter tables. Follows the exact same pattern as ScoreboardCard/StandingsCard in SportsCards.swift.

**Tech Stack:** SwiftUI (iOS card), Go (backend validation), Markdown (skill file)

---

### Task 1: Create baseball skill file

**Files:**
- Create: `.claude/skills/beepbopboop-baseball/SKILL.md`

**Step 1:** Create the skill file with steps BS1–BS5, ESPN MLB API calls, and box_score JSON schema (as specified in issue #59).

**Step 2:** Commit
```bash
git add .claude/skills/beepbopboop-baseball/SKILL.md
git commit -m "feat: add beepbopboop-baseball skill"
```

---

### Task 2: Add BaseballData Swift model

**Files:**
- Create: `beepbopboop/beepbopboop/Models/BaseballData.swift`

**Step 1:** Create structs: `BaseballData`, `PitcherLine`, `SavePitcher`, `BatterLine`. Reuse `TeamInfo` from SportsData.swift.

**Step 2:** Build app to verify it compiles.

**Step 3:** Commit
```bash
git add beepbopboop/beepbopboop/Models/BaseballData.swift
git commit -m "feat: add BaseballData model with pitcher/batter stat structs"
```

---

### Task 3: Update Post.swift

**Files:**
- Modify: `beepbopboop/beepbopboop/Models/Post.swift:181-203`

**Step 1:** Add `case boxScore` to `DisplayHintValue` enum.

**Step 2:** Add `case "box_score": return .boxScore` to `displayHintValue` switch.

**Step 3:** Add `var baseballData: BaseballData?` computed property (mirror `gameData` pattern, guard on `.boxScore`).

**Step 4:** Build to verify.

**Step 5:** Commit
```bash
git add beepbopboop/beepbopboop/Models/Post.swift
git commit -m "feat: add boxScore display hint to Post model"
```

---

### Task 4: Add BoxScoreCard view

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SportsCards.swift` (append)

**Step 1:** Add `BoxScoreCard` struct: dark `#0A1628` background, two-tone team gradient header, `figure.baseball` 120pt watermark, large score row with `F/9` status pill, divider, monospaced pitcher section (W/L/SV lines), conditional key batter section, italic headline strip, venue + reactions footer. Height: 280pt.

**Step 2:** Build to verify.

**Step 3:** Commit
```bash
git add beepbopboop/beepbopboop/Views/SportsCards.swift
git commit -m "feat: add BoxScoreCard view for box_score display hint"
```

---

### Task 5: Update FeedItemView routing

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift:8,29-67`

**Step 1:** Add `.boxScore` to the full-shadow hint list on line 8.

**Step 2:** Add routing case in `cardContent` switch:
```swift
case .boxScore:
    if let card = BoxScoreCard(post: post) {
        card
    } else {
        StandardCard(post: post)
    }
```

**Step 3:** Build to verify.

**Step 4:** Commit
```bash
git add beepbopboop/beepbopboop/Views/FeedItemView.swift
git commit -m "feat: route box_score hint to BoxScoreCard in FeedItemView"
```

---

### Task 6: Update backend validation

**Files:**
- Modify: `backend/internal/handler/post.go`

**Step 1:** Add `"box_score": true` to `ValidDisplayHints` map.

**Step 2:** Add `box_score` to `structuredHint` boolean (line ~171) so externalURL is not validated as a URL.

**Step 3:** Add `"box_score"` to the required-externalURL check (line ~240).

**Step 4:** Add `case "box_score": validateBoxScoreData(req.ExternalURL, &errs, &warns)` to the validation switch.

**Step 5:** Implement `validateBoxScoreData` function checking for `status`, `home`, `away`, `sport` fields (mirrors `validateGameData`).

**Step 6:** Build backend: `cd backend && go build ./...`

**Step 7:** Commit
```bash
git add backend/internal/handler/post.go
git commit -m "feat: add box_score display_hint validation to backend"
```
