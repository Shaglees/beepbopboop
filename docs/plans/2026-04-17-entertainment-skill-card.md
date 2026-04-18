# Entertainment Skill & Card Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the `beepbopboop-celebrity` skill (quality-gated entertainment news) and the `EntertainmentCard` iOS card for the `entertainment` display hint.

**Architecture:** `entertainment` is a structured-JSON hint (like `scoreboard`/`matchup`) — `external_url` holds `EntertainmentData` JSON. Backend validates required fields; iOS decodes and renders a photo-forward editorial card with category badge, source badge, and optional quote strip.

**Tech Stack:** Go (backend validation), Swift/SwiftUI (iOS model + card), Markdown skill file.

**GitHub issue:** #62

---

### Task 1: Backend — register `entertainment` hint and validate JSON

**Files:**
- Modify: `backend/internal/handler/post.go` (lines 32–47, 230–246)
- Modify: `backend/internal/handler/post_test.go` (lines 964–982, 1011)

#### Step 1: Write failing tests first

Add three new test functions to `backend/internal/handler/post_test.go` **before** the `TestValidDisplayHints_AllSyncWithLint` function:

```go
func TestLintPost_EntertainmentMissingExternalURL(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","display_hint":"entertainment"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false when entertainment has no external_url")
	}
	if !hasFieldError(lintErrors(resp), "external_url") {
		t.Error("expected error for missing external_url on entertainment")
	}
}

func TestLintPost_EntertainmentBadJSON(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","display_hint":"entertainment","external_url":"{}"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false for empty entertainment data")
	}
	errs := lintErrors(resp)
	if !hasFieldError(errs, "external_url.subject") {
		t.Error("expected error for missing subject")
	}
	if !hasFieldError(errs, "external_url.headline") {
		t.Error("expected error for missing headline")
	}
	if !hasFieldError(errs, "external_url.source") {
		t.Error("expected error for missing source")
	}
}

func TestLintPost_EntertainmentValid(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	entJSON := `{"subject":"Zendaya","headline":"Zendaya Named TIME Entertainer of the Year","source":"People","category":"award","tags":["awards","zendaya"]}`
	body := `{"title":"t","body":"b","display_hint":"entertainment","external_url":` + jsonString(entJSON) + `,"labels":["entertainment"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != true {
		t.Errorf("expected valid=true for good entertainment data, errors: %v", lintErrors(resp))
	}
}
```

#### Step 2: Run tests to confirm they fail

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/backend
go test ./internal/handler/ -run "TestLintPost_Entertainment" -v
```

Expected: compilation error or `entertainment` not recognised.

#### Step 3: Add `entertainment` to `ValidDisplayHints` in `post.go`

In `ValidDisplayHints` map (around line 46), add after `"standings"`:

```go
"entertainment": true,
```

#### Step 4: Add structured-hint cases in `post.go`

In the `switch req.DisplayHint` block (around line 232), add:

```go
case "entertainment":
    validateEntertainmentData(req.ExternalURL, &errs, &warns)
```

In the `else if` required-external_url check (around line 240), append `|| req.DisplayHint == "entertainment"`:

```go
} else if req.DisplayHint == "weather" || req.DisplayHint == "scoreboard" || req.DisplayHint == "matchup" || req.DisplayHint == "standings" || req.DisplayHint == "entertainment" {
```

#### Step 5: Add `validateEntertainmentData` function in `post.go`

Add after the standings validation section (after line ~430):

```go
// --- Entertainment data validation ---

type entertainmentDataValidation struct {
    Subject  *string `json:"subject"`
    Headline *string `json:"headline"`
    Source   *string `json:"source"`
}

func validateEntertainmentData(externalURL string, errs *[]validationIssue, warns *[]validationIssue) {
    var e entertainmentDataValidation
    if err := json.Unmarshal([]byte(externalURL), &e); err != nil {
        *errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "external_url must be valid JSON for entertainment hint"})
        return
    }

    if e.Subject == nil {
        *errs = append(*errs, validationIssue{Field: "external_url.subject", Code: "required", Message: "subject is required"})
    }
    if e.Headline == nil {
        *errs = append(*errs, validationIssue{Field: "external_url.headline", Code: "required", Message: "headline is required"})
    }
    if e.Source == nil {
        *errs = append(*errs, validationIssue{Field: "external_url.source", Code: "required", Message: "source is required"})
    }
}
```

#### Step 6: Update `TestValidDisplayHints_AllSyncWithLint` in `post_test.go`

In the `structuredHints` map (line ~964), add:

```go
"entertainment": `{"subject":"Zendaya","headline":"Zendaya Named TIME Entertainer of the Year","source":"People","category":"award","tags":["entertainment"]}`,
```

#### Step 7: Update `TestValidationMaps_Sorted` expected hints list

On line ~1011, add `"entertainment"` to the sorted `expectedHints` slice (keep alphabetical order):

```go
expectedHints := []string{"article", "brief", "calendar", "card", "comparison", "deal", "digest", "entertainment", "event", "matchup", "outfit", "place", "scoreboard", "standings", "weather"}
```

#### Step 8: Run all tests to confirm passing

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/backend
go test ./internal/handler/ -v 2>&1 | tail -20
```

Expected: all PASS, no FAIL.

#### Step 9: Commit

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6
git add backend/internal/handler/post.go backend/internal/handler/post_test.go
git commit -m "feat: register entertainment as structured display_hint with JSON validation"
```

---

### Task 2: iOS model — `EntertainmentData.swift`

**Files:**
- Create: `beepbopboop/beepbopboop/Models/EntertainmentData.swift`

#### Step 1: Create the model file

```swift
import Foundation
import SwiftUI

struct EntertainmentData: Codable {
    let subject: String
    let subjectImageUrl: String?
    let headline: String
    let source: String
    let sourceUrl: String?
    let publishedAt: String?
    let category: String?       // "award" | "appearance" | "project" | "social" | "news"
    let quote: String?
    let relatedProject: String?
    let tags: [String]?

    enum CodingKeys: String, CodingKey {
        case subject
        case subjectImageUrl = "subjectImageUrl"
        case headline
        case source
        case sourceUrl
        case publishedAt
        case category
        case quote
        case relatedProject
        case tags
    }

    var categoryBadgeColor: Color {
        switch category {
        case "award":      return Color(hex: "#F59E0B")
        case "project":    return Color(hex: "#8B5CF6")
        case "appearance": return Color(hex: "#EC4899")
        default:           return Color(hex: "#6B7280")
        }
    }

    var categoryLabel: String {
        switch category {
        case "award":      return "🏆 AWARD"
        case "project":    return "🎬 PROJECT"
        case "appearance": return "👠 APPEARANCE"
        case "social":     return "📱 SOCIAL"
        default:           return "📰 NEWS"
        }
    }
}
```

> **Note on `Color(hex:)`:** The codebase already has a `Color(hexString:)` extension used in `SportsData.swift`. Check if `Color(hex:)` exists — if not, use `Color(hexString:)` instead.

#### Step 2: Verify the hex color initialiser name

```bash
grep -r "Color(hex" /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/beepbopboop --include="*.swift" | head -5
```

Use whichever form exists in the project.

#### Step 3: No tests needed for pure data model — proceed to commit

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6
git add beepbopboop/beepbopboop/Models/EntertainmentData.swift
git commit -m "feat: add EntertainmentData model for entertainment display_hint"
```

---

### Task 3: iOS `Post.swift` — add `.entertainment` enum case + computed property

**Files:**
- Modify: `beepbopboop/beepbopboop/Models/Post.swift`

All line numbers are approximate — confirm with the actual file.

#### Step 1: Add `.entertainment` to `DisplayHintValue` enum (line ~182)

Change:
```swift
case scoreboard, matchup, standings
```
To:
```swift
case scoreboard, matchup, standings, entertainment
```

#### Step 2: Add routing in `displayHintValue` computed property (after line ~200)

After `case "standings": return .standings`, add:
```swift
case "entertainment": return .entertainment
```

#### Step 3: Add `hintColor` case (after `.standings: return .secondary`)

```swift
case .entertainment: return Color(red: 0.984, green: 0.502, blue: 0.180)  // warm amber
```

#### Step 4: Add `hintIcon` case (after `.standings: return "list.number"`)

```swift
case .entertainment: return "star.fill"
```

#### Step 5: Add `hintLabel` case (after `.standings: return "Scores"`)

```swift
case .entertainment: return "Entertainment"
```

#### Step 6: Add `entertainmentData` computed property (after `standingsData`, around line ~406)

```swift
/// Parsed entertainment data from externalURL (for entertainment display_hint posts).
var entertainmentData: EntertainmentData? {
    guard displayHintValue == .entertainment,
          let json = externalURL,
          let data = json.data(using: .utf8) else { return nil }
    return try? JSONDecoder().decode(EntertainmentData.self, from: data)
}
```

#### Step 7: Build to confirm no Swift errors

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/beepbopboop
xcodebuild -scheme beepbopboop -destination "platform=iOS Simulator,name=iPhone 16" build 2>&1 | grep -E "error:|warning:|BUILD"
```

Expected: `BUILD SUCCEEDED` (or only pre-existing warnings).

#### Step 8: Commit

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6
git add beepbopboop/beepbopboop/Models/Post.swift
git commit -m "feat: add entertainment DisplayHintValue case and entertainmentData accessor"
```

---

### Task 4: iOS `EntertainmentCards.swift` — create the card view

**Files:**
- Create: `beepbopboop/beepbopboop/Views/EntertainmentCards.swift`

Design spec (from issue #62):
- Full-width hero image (200pt height), warm gradient overlay at bottom
- Category badge top-left, source badge top-right (both floating over image)
- Subject name overlay at bottom of image (white 20pt semibold)
- Headline below image (16pt, 2-line max)
- Quote strip if present: 4pt left accent bar in category color + italic text
- Related project chip: `re: Dune: Part Two`
- Timestamp: `2 hours ago · People`
- Tags: up to 3 horizontal chips
- Footer: reactions + `Read full story →` link
- White card background — editorial magazine feel
- Falls back gracefully when `entertainmentData` is nil (uses `StandardCard`)

#### Step 1: Create the file

```swift
import SwiftUI

// MARK: - EntertainmentCard

struct EntertainmentCard: View {
    let post: Post
    private var data: EntertainmentData? { post.entertainmentData }

    var body: some View {
        if let data = data {
            EntertainmentCardContent(post: post, data: data)
        } else {
            StandardCard(post: post)
        }
    }
}

private struct EntertainmentCardContent: View {
    let post: Post
    let data: EntertainmentData

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            heroSection
            contentSection
        }
    }

    // MARK: Hero image with overlay badges and subject name

    private var heroSection: some View {
        ZStack(alignment: .bottom) {
            heroImage
            LinearGradient(
                colors: [.clear, .black.opacity(0.6)],
                startPoint: .center,
                endPoint: .bottom
            )
            .frame(height: 200)

            // Subject name at bottom
            HStack {
                Text(data.subject)
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundColor(.white)
                    .lineLimit(1)
                Spacer()
            }
            .padding(.horizontal, 12)
            .padding(.bottom, 10)

            // Floating badges over image
            VStack {
                HStack {
                    categoryBadge
                    Spacer()
                    sourceBadge
                }
                .padding(.horizontal, 10)
                .padding(.top, 10)
                Spacer()
            }
        }
        .frame(height: 200)
        .clipped()
    }

    @ViewBuilder
    private var heroImage: some View {
        if let urlString = data.subjectImageUrl, let url = URL(string: urlString) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().scaledToFill()
                case .failure, .empty:
                    fallbackHeroBackground
                @unknown default:
                    fallbackHeroBackground
                }
            }
            .frame(height: 200)
            .clipped()
        } else if let urlString = post.imageURL, let url = URL(string: urlString) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().scaledToFill()
                default:
                    fallbackHeroBackground
                }
            }
            .frame(height: 200)
            .clipped()
        } else {
            fallbackHeroBackground
                .frame(height: 200)
        }
    }

    private var fallbackHeroBackground: some View {
        LinearGradient(
            colors: [Color(red: 0.15, green: 0.12, blue: 0.20), Color(red: 0.25, green: 0.18, blue: 0.30)],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }

    private var categoryBadge: some View {
        Text(data.categoryLabel)
            .font(.system(size: 10, weight: .bold))
            .foregroundColor(.white)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(data.categoryBadgeColor)
            .clipShape(Capsule())
    }

    private var sourceBadge: some View {
        Text(data.source)
            .font(.system(size: 10, weight: .semibold))
            .foregroundColor(Color(.label))
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(Color(.systemBackground).opacity(0.92))
            .clipShape(Capsule())
    }

    // MARK: Content below image

    private var contentSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Headline
            Text(data.headline)
                .font(.system(size: 16, weight: .semibold))
                .foregroundColor(Color(.label))
                .lineLimit(2)

            // Quote strip
            if let quote = data.quote, !quote.isEmpty {
                HStack(spacing: 8) {
                    Rectangle()
                        .fill(data.categoryBadgeColor)
                        .frame(width: 4)
                        .cornerRadius(2)
                    Text(quote)
                        .font(.system(size: 13, weight: .regular))
                        .italic()
                        .foregroundColor(Color(.secondaryLabel))
                        .lineLimit(3)
                }
            }

            // Related project chip
            if let project = data.relatedProject, !project.isEmpty {
                Text("re: \(project)")
                    .font(.system(size: 12))
                    .foregroundColor(Color(.secondaryLabel))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color(.tertiarySystemFill))
                    .clipShape(Capsule())
            }

            // Timestamp + source
            Text("\(post.relativeTime) · \(data.source)")
                .font(.caption)
                .foregroundColor(Color(.tertiaryLabel))

            // Tags (max 3)
            if let tags = data.tags, !tags.isEmpty {
                HStack(spacing: 6) {
                    ForEach(tags.prefix(3), id: \.self) { tag in
                        Text("#\(tag)")
                            .font(.system(size: 11))
                            .foregroundColor(Color(.secondaryLabel))
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(Color(.systemFill))
                            .clipShape(Capsule())
                    }
                }
            }

            // Footer
            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(Color(.systemBackground))
    }
}
```

> **Note:** `CardFooter` and `StandardCard` are `private` in `FeedItemView.swift` — if they can't be accessed, replicate the footer pattern inline. Check with `grep -n "private struct CardFooter\|struct CardFooter"`.

#### Step 2: Check if CardFooter is accessible

```bash
grep -n "private struct CardFooter\|struct CardFooter\|private struct StandardCard\|struct StandardCard" /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/beepbopboop/beepbopboop/Views/FeedItemView.swift
```

If `private`, replicate the footer inline (look at ~lines 100–140 of FeedItemView.swift for the pattern).

#### Step 3: Build to confirm no Swift errors

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/beepbopboop
xcodebuild -scheme beepbopboop -destination "platform=iOS Simulator,name=iPhone 16" build 2>&1 | grep -E "error:|BUILD"
```

#### Step 4: Commit

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6
git add beepbopboop/beepbopboop/Views/EntertainmentCards.swift
git commit -m "feat: add EntertainmentCard SwiftUI view for entertainment display_hint"
```

---

### Task 5: Wire `EntertainmentCard` into `FeedItemView.swift`

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift` (lines 8, 58–66)

#### Step 1: Add `.entertainment` to the shadow-style list (line 8)

Change:
```swift
if [.outfit, .weather, .scoreboard, .matchup, .standings].contains(post.displayHintValue) {
```
To:
```swift
if [.outfit, .weather, .scoreboard, .matchup, .standings, .entertainment].contains(post.displayHintValue) {
```

#### Step 2: Add the case to the `cardContent` switch (after `.standings:` case)

Add before `default:`:
```swift
case .entertainment:
    if let card = EntertainmentCard(post: post) as? EntertainmentCard {
        card
    } else {
        StandardCard(post: post)
    }
```

Wait — `EntertainmentCard` is not optional; it always renders (it falls back internally). Simpler:

```swift
case .entertainment:
    EntertainmentCard(post: post)
```

#### Step 3: Build

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/beepbopboop
xcodebuild -scheme beepbopboop -destination "platform=iOS Simulator,name=iPhone 16" build 2>&1 | grep -E "error:|BUILD"
```

Expected: `BUILD SUCCEEDED`.

#### Step 4: Commit

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6
git add beepbopboop/beepbopboop/Views/FeedItemView.swift
git commit -m "feat: route entertainment display_hint to EntertainmentCard in FeedItemView"
```

---

### Task 6: Skill file — `beepbopboop-celebrity`

**Files:**
- Create: `.claude/skills/beepbopboop-celebrity/SKILL.md`

#### Step 1: Create the skill directory and file

```bash
mkdir -p /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/.claude/skills/beepbopboop-celebrity
```

Create `.claude/skills/beepbopboop-celebrity/SKILL.md` with this content:

```markdown
---
name: beepbopboop-celebrity
description: Create entertainment news posts — celebrity news, red carpet, awards, social moments (sourced, not tabloid)
argument-hint: "[trending | awards | {celebrity name} | red carpet | social moment]"
allowed-tools: WebFetch, WebSearch, Bash
---

# BeepBopBoop Celebrity & Entertainment Skill

You create factual entertainment news posts from approved sources. You are **not** a gossip column. Every post must trace to a named, verifiable event, statement, or announcement.

## Important

- Only approved sources (see Step CE1). No tabloids.
- Facts only: no speculation, no anonymous sources ("sources say"), no rumour
- Name the subject in the title — factual, specific, no clickbait
- Kill list: "stuns fans", "breaks the internet", "we can't get over", "slays", "iconic"

---

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required: `BEEPBOPBOOP_API_URL`, `BEEPBOPBOOP_AGENT_TOKEN`
Optional: `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY`, `BEEPBOPBOOP_IMGUR_CLIENT_ID`

---

## Step CE1: Source selection (quality gatekeeping)

Approved sources — use RSS feeds or direct fetch:

| Source | RSS / URL |
|---|---|
| Entertainment Weekly | `https://ew.com/feed/` |
| Variety | `https://variety.com/feed/` |
| People | `https://people.com/feed/` |
| Hollywood Reporter | `https://www.hollywoodreporter.com/feed/` |
| Pitchfork (music crossover) | `https://pitchfork.com/rss/` |

**Not approved:** TMZ, Daily Mail, National Enquirer, Page Six, celebrity gossip sites.

If the user specifies a celebrity name or topic, use `WebSearch` to find recent coverage from approved sources only.

---

## Step CE2: Fetch + filter

Fetch one or two RSS feeds relevant to the topic:

```bash
# Example: fetch People RSS
curl -s "https://people.com/feed/" 2>/dev/null | grep -E "<title>|<link>|<pubDate>" | head -60
```

Or use `WebFetch` on the feed URL with prompt: "List the 10 most recent article titles, URLs, and dates."

Filter to stories from the last 48 hours. Prefer:
- Award announcements
- Confirmed project announcements
- Public appearances / red carpet events
- Direct quotes from the subject
- Milestone moments (album drops, box office records)

Reject:
- Breakup/relationship rumours
- Unconfirmed gossip
- Stories citing only "sources close to"
- Clickbait with no factual anchor

Select one story with: verified facts, clear subject, real-world outcome.

---

## Step CE3: Fetch article content

`WebFetch` the article URL with prompt:
"Extract: headline, key facts (who, what, when), any direct quotes from the subject, main photo URL if accessible, publication name, author, and publication date."

---

## Step CE4: Compose post

**Title format:** `"{What happened}"` — factual, specific. Name the person.
- Good: `"Zendaya Named TIME's Entertainer of the Year"`
- Bad: `"Zendaya Stuns Fans with Major News"`

**Body:** Core facts in 2 sentences max. Include a quote if compelling. Add one sentence of context (what project they're known for, why this moment matters).

**Banned phrases:** "stuns fans", "breaks the internet", "we can't get over", "slays", "iconic", "literally", rhetorical questions, emoji in headlines, speculation.

Determine category:
- `"award"` — win, nomination, honour, recognition
- `"project"` — new film, album, show, book, collaboration announced or released
- `"appearance"` — red carpet, premiere, event, public appearance
- `"social"` — statement, announcement, social media moment with public significance
- `"news"` — everything else verifiable

---

## Step CE5: Build `external_url` JSON

```json
{
  "subject": "Zendaya",
  "subjectImageUrl": "https://static.people.com/...",
  "headline": "Zendaya Named TIME's Entertainer of the Year",
  "source": "People",
  "sourceUrl": "https://people.com/zendaya-time-...",
  "publishedAt": "2026-04-16T14:30:00Z",
  "category": "award",
  "quote": "\"I feel incredibly grateful. This year has been... a lot,\" Zendaya said.",
  "relatedProject": "Dune: Part Two",
  "tags": ["awards", "zendaya", "time magazine", "entertainment"]
}
```

- `subjectImageUrl`: Use Unsplash if no article image is accessible (`WebSearch "zendaya portrait unsplash"`)
- `tags`: 3–5 lowercase, no spaces. Always include subject's name and category.
- `publishedAt`: ISO-8601. Use article date, not current time.
- `quote`: Include only verbatim quotes attributed to the subject by name. Strip attribution suffix from the JSON string — keep just the quote text.

---

## Step CE6: Publish

Load API config from Step 0. POST to `$BEEPBOPBOOP_API_URL/posts`:

```json
{
  "title": "<headline from Step CE4>",
  "body": "<2-sentence body from Step CE4>",
  "post_type": "article",
  "display_hint": "entertainment",
  "visibility": "public",
  "locality": "<source name, e.g. People>",
  "external_url": "<JSON string from Step CE5>",
  "labels": ["entertainment", "celebrity", "<category>", "<subject name lowercase>"]
}
```

Use `beepbopgraph` if available, otherwise `curl`:

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '<JSON body>'
```

Confirm `201 Created` response. Report the post title and ID.

---

## Quality checklist before publishing

- [ ] Source is on the approved list
- [ ] All facts are confirmed (no "reportedly", "sources say")
- [ ] Title names the subject and states what happened
- [ ] No banned phrases in title or body
- [ ] `external_url` JSON is valid and has `subject`, `headline`, `source`
- [ ] Tags include subject name and category
```

#### Step 2: Commit

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6
git add .claude/skills/beepbopboop-celebrity/
git commit -m "feat: add beepbopboop-celebrity skill with quality-gated entertainment sources"
```

---

### Task 7: Final build verification + PR

#### Step 1: Run backend tests

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/backend
go test ./... 2>&1 | tail -20
```

Expected: all PASS.

#### Step 2: Run iOS build

```bash
cd /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/busy-merkle-dfa3c6/beepbopboop
xcodebuild -scheme beepbopboop -destination "platform=iOS Simulator,name=iPhone 16" build 2>&1 | grep -E "error:|BUILD"
```

Expected: `BUILD SUCCEEDED`.

#### Step 3: Post progress comment on issue #62

```bash
gh issue comment 62 --body "$(cat <<'EOF'
## Implementation complete ✓

All components implemented on branch `claude/busy-merkle-dfa3c6`:

- **`.claude/skills/beepbopboop-celebrity/SKILL.md`** — quality-gated skill with approved source list (EW, Variety, People, THR, Pitchfork), 6-step flow, quality checklist
- **`EntertainmentData.swift`** — Codable model with `categoryBadgeColor` and `categoryLabel` computed properties
- **`Post.swift`** — `.entertainment` added to `DisplayHintValue`, `entertainmentData` computed accessor
- **`EntertainmentCards.swift`** — photo-forward editorial card: hero image, category badge, source badge, subject overlay, headline, quote strip, related project chip, tags
- **`FeedItemView.swift`** — `entertainment` wired up in card dispatch
- **`post.go`** — `entertainment` in `ValidDisplayHints` + `structuredHint` check + JSON validation
- **`post_test.go`** — 3 new tests + sync/sorted map tests updated

Backend tests: PASS. iOS build: SUCCEEDED.
EOF
)"
```

#### Step 4: Create PR

```bash
gh pr create \
  --title "feat: Celebrity/Entertainment skill and EntertainmentCard iOS view" \
  --body "$(cat <<'EOF'
## Summary

- Adds `beepbopboop-celebrity` skill: quality-gated entertainment news from EW, Variety, People, THR, Pitchfork with a 6-step flow and kill list against gossip
- Adds `EntertainmentData` Codable model with category-adaptive badge colors
- Adds `entertainment` display hint as a structured-JSON hint (like `scoreboard`/`matchup`) with backend validation requiring `subject`, `headline`, `source`
- Adds `EntertainmentCard`: photo-forward editorial card with hero image, floating category/source badges, subject name overlay, headline, quote strip, related project chip, and tag row
- Wires `entertainment` into `FeedItemView` dispatch

## Test plan

- [ ] `go test ./internal/handler/` passes (3 new entertainment tests + existing suite)
- [ ] iOS build succeeds: `xcodebuild -scheme beepbopboop ... build`
- [ ] `entertainment` display hint accepted by backend lint endpoint
- [ ] `entertainment` without `external_url` correctly rejected
- [ ] `EntertainmentCard` renders photo, badges, headline, quote strip
- [ ] Falls back to `StandardCard` when `external_url` JSON is malformed

Closes #62

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
