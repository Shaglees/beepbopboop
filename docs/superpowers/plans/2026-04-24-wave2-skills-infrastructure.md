# Wave 2: Skills Infrastructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Any skill can publish a post that iOS renders as a rich card with no decoding errors. Lint catches bad payloads with actionable fix instructions. All specialty skills are reachable from the main router.

**Architecture:** hints.go is the single contract. Tests validate that every catalog example passes lint. Lint warnings are upgraded to actionable patch instructions. The main router skill gets a dispatch table mapping topics to specialty skills. New skills (gaming, creators) fill coverage gaps.

**Tech Stack:** Go (backend tests + validators), Claude Code skills (markdown), shell (curl/jq for skill publish flows)

**Spec:** `docs/superpowers/specs/2026-04-24-wave2-skills-infrastructure-design.md`

---

## File Structure

### Backend (Go)

| File | Responsibility |
|---|---|
| `backend/internal/handler/hints_test.go` | Modify: add iOS decode test + metadata completeness test |
| `backend/internal/handler/post.go` | Modify: upgrade warning messages to actionable patches, add creator_spotlight validator |
| `backend/internal/handler/hints.go` | Modify: add `generator` field to hintDescriptor, mark feedback as system-only |

### Skills (Markdown)

| File | Responsibility |
|---|---|
| `.claude/skills/_shared/PUBLISH_ENVELOPE.md` | Modify: add canonical structured external_url section |
| `.claude/skills/beepbopboop-post/SKILL.md` | Modify: add dispatch table for specialty skills |
| `.claude/skills/beepbopboop-post/INIT_WIZARD.md` | Modify: add config-file bootstrap path |
| `.claude/skills/beepbopboop-food/SKILL.md` | Modify: fix external_url from raw object to JSON string |
| `.claude/skills/beepbopboop-travel/SKILL.md` | Modify: fix external_url from raw object to JSON string |
| `.claude/skills/beepbopboop-fitness/SKILL.md` | Modify: clarify external_url stringification |
| `.claude/skills/beepbopboop-gaming/SKILL.md` | Create: game_release + game_review skill |
| `.claude/skills/beepbopboop-creators/SKILL.md` | Create: creator_spotlight skill |

---

## Task 1: Add iOS Decode Contract Test (#201)

**Files:**
- Modify: `backend/internal/handler/hints_test.go`

The existing `TestHints_ExamplesLintClean` (line 123) validates examples pass lint. We need a companion test that validates the structured JSON in each example can decode into the same shape iOS expects.

- [ ] **Step 1: Write the iOS decode test**

Add after `TestHints_EventHintNoWarningWhenDateInBody` (line 424):

```go
// TestHints_ExamplesDecodeForStructuredHints validates that every structured
// hint's example has an external_url that decodes into the shape iOS expects.
// This catches "lint passes but iOS card returns nil" drift.
func TestHints_ExamplesDecodeForStructuredHints(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	for _, entry := range hr.DisplayHints {
		if !entry.StructuredJSON {
			continue
		}
		entry := entry
		t.Run(entry.Hint, func(t *testing.T) {
			// Decode the example to get external_url
			var example struct {
				ExternalURL string `json:"external_url"`
			}
			if err := json.Unmarshal(entry.Example, &example); err != nil {
				t.Fatalf("cannot decode example: %v", err)
			}
			if example.ExternalURL == "" {
				t.Fatalf("structured hint %q example has empty external_url", entry.Hint)
			}

			// Verify external_url is valid JSON (not a raw object in the request —
			// it should be a JSON string that contains JSON)
			var raw json.RawMessage
			if err := json.Unmarshal([]byte(example.ExternalURL), &raw); err != nil {
				t.Fatalf("external_url for %q is not valid JSON: %v\nvalue: %s", entry.Hint, err, example.ExternalURL)
			}

			// Verify all required_fields that start with "external_url:" are present
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(example.ExternalURL), &parsed); err != nil {
				t.Fatalf("external_url for %q cannot be parsed as object: %v", entry.Hint, err)
			}
			for _, rf := range entry.RequiredFields {
				if len(rf) > len("external_url:") && rf[:len("external_url:")] == "external_url:" {
					key := rf[len("external_url:"):]
					if _, ok := parsed[key]; !ok {
						t.Errorf("hint %q example external_url missing required field %q", entry.Hint, key)
					}
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -run TestHints_ExamplesDecodeForStructuredHints -v`

Expected: PASS for all structured hints. If any fail, the catalog example in `hints.go` needs fixing — update the example to include the missing field.

- [ ] **Step 3: Add metadata completeness test**

Add after the decode test:

```go
// TestHints_MetadataComplete ensures every hint has the documentation
// fields skills need: description, required_fields, example, renders,
// pick_when, and avoid_when. Prevents undocumented hints from shipping.
func TestHints_MetadataComplete(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	for _, e := range hr.DisplayHints {
		t.Run(e.Hint, func(t *testing.T) {
			if e.Description == "" {
				t.Errorf("hint %q has empty description", e.Hint)
			}
			if len(e.RequiredFields) == 0 {
				t.Errorf("hint %q has no required_fields", e.Hint)
			}
			if len(e.Example) == 0 {
				t.Errorf("hint %q has empty example", e.Hint)
			}
			if e.Renders == nil || e.Renders.Card == "" {
				t.Errorf("hint %q missing renders.card", e.Hint)
			}
			if e.PickWhen == "" {
				t.Errorf("hint %q missing pick_when guidance", e.Hint)
			}
			if e.AvoidWhen == "" {
				t.Errorf("hint %q missing avoid_when guidance", e.Hint)
			}
		})
	}
}
```

- [ ] **Step 4: Run all hints tests**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -run TestHints -v`

Expected: All PASS. Fix any catalog entries in `hints.go` that fail metadata completeness.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/hints_test.go
git commit -m "test(hints): add iOS decode + metadata completeness contract tests (#201)"
```

---

## Task 2: Add creator_spotlight Validator (#201, #198)

**Files:**
- Modify: `backend/internal/handler/post.go`

creator_spotlight is listed as a structured hint but has no validator — it's the only structured hint without one.

- [ ] **Step 1: Write a test for creator_spotlight validation**

Add to `backend/internal/handler/post_test.go`:

```go
func TestPostHandler_LintCreatorSpotlight_RequiresDesignation(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	body := []byte(`{
		"title": "Local artist spotlight",
		"body": "A ceramicist making waves in the community.",
		"post_type": "discovery",
		"display_hint": "creator_spotlight",
		"locality": "Dublin",
		"external_url": "{\"links\":{\"instagram\":\"@clay_maker\"}}",
		"labels": ["creator"]
	}`)

	req := httptest.NewRequest(http.MethodPost, "/posts/lint", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.LintPost(rec, req)

	var result struct {
		Valid    bool `json:"valid"`
		Warnings []struct {
			Field string `json:"field"`
			Code  string `json:"code"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Should pass (valid=true) but warn about missing designation
	if !result.Valid {
		t.Fatalf("expected valid=true, got false")
	}
	found := false
	for _, w := range result.Warnings {
		if w.Field == "external_url.designation" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for missing designation; got %+v", result.Warnings)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -run TestPostHandler_LintCreatorSpotlight -v`

Expected: FAIL — no warning is generated because there's no validator.

- [ ] **Step 3: Add the validator**

Add to `backend/internal/handler/post.go` after `validateScienceData` (around line 1085):

```go
// --- Creator spotlight data validation ---

type creatorDataValidation struct {
	Designation  *string                `json:"designation"`
	Links        map[string]interface{} `json:"links"`
	NotableWorks *string                `json:"notable_works"`
	Tags         []string               `json:"tags"`
	Source       *string                `json:"source"`
	AreaName     *string                `json:"area_name"`
}

func validateCreatorData(externalURL string, errs *[]validationIssue, warns *[]validationIssue) {
	var c creatorDataValidation
	if err := json.Unmarshal([]byte(externalURL), &c); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "external_url must be valid JSON for creator_spotlight hint"})
		return
	}
	if c.Designation == nil || *c.Designation == "" {
		*warns = append(*warns, validationIssue{
			Field:   "external_url.designation",
			Code:    "recommended",
			Message: "Add \"designation\": \"<role>\" (e.g. \"ceramicist\", \"muralist\", \"indie musician\") to your external_url JSON. CreatorSpotlightCard displays this prominently.",
		})
	}
	if c.Links == nil || len(c.Links) == 0 {
		*warns = append(*warns, validationIssue{
			Field:   "external_url.links",
			Code:    "recommended",
			Message: "Add \"links\": {\"instagram\": \"@handle\", \"website\": \"https://...\"} to your external_url JSON. Supported keys: website, instagram, bandcamp, etsy, substack, soundcloud, behance.",
		})
	}
	if c.AreaName == nil || *c.AreaName == "" {
		*warns = append(*warns, validationIssue{
			Field:   "external_url.area_name",
			Code:    "recommended",
			Message: "Add \"area_name\": \"<neighborhood or city>\" to your external_url JSON for local context.",
		})
	}
}
```

Then add the dispatch case in `validatePost` (around line 376, before the closing `}`):

```go
		case "creator_spotlight":
			validateCreatorData(req.ExternalURL, &errs, &warns)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -run TestPostHandler_LintCreatorSpotlight -v`

Expected: PASS

- [ ] **Step 5: Run all tests to check nothing broke**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -v`

Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/post.go backend/internal/handler/post_test.go
git commit -m "feat(lint): add creator_spotlight validator with actionable warnings (#201, #198)"
```

---

## Task 3: Upgrade Lint Warnings to Actionable Patches (#198)

**Files:**
- Modify: `backend/internal/handler/post.go` (lines 840-1085)

Upgrade every per-hint validator's warning messages to include: what to add, the field path, and a concrete example value. Also change the `code` from `"missing"` to `"recommended"`.

- [ ] **Step 1: Upgrade food validator warnings (lines 849-854)**

Replace the warning messages in `validateFoodData`:

```go
	if fd.Rating == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.rating", Code: "recommended", Message: "Add \"rating\": <number 1.0-5.0> to your external_url JSON for a richer RestaurantCard render. Example: \"rating\": 4.3"})
	}
	if fd.Latitude == nil || fd.Longitude == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.latitude", Code: "recommended", Message: "Add \"latitude\": <float> and \"longitude\": <float> to your external_url JSON. RestaurantCard shows a map pin when coordinates are present. Example: \"latitude\": 53.3498, \"longitude\": -6.2603"})
	}
```

Also add warnings for iOS fields not currently checked:

```go
	if fd.ReviewCount == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.reviewCount", Code: "recommended", Message: "Add \"reviewCount\": <integer> to your external_url JSON. RestaurantCard shows review count next to rating. Example: \"reviewCount\": 127"})
	}
	if len(fd.Cuisine) == 0 {
		*warns = append(*warns, validationIssue{Field: "external_url.cuisine", Code: "recommended", Message: "Add \"cuisine\": [\"Italian\", \"Pizza\"] to your external_url JSON. RestaurantCard displays cuisine tags."})
	}
```

- [ ] **Step 2: Upgrade travel validator warnings (lines 1057-1059)**

```go
	if t.Latitude == nil || t.Longitude == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.latitude", Code: "recommended", Message: "Add \"latitude\": <float> and \"longitude\": <float> to your external_url JSON. DestinationCard shows a map when coordinates are present. Example: \"latitude\": 48.8566, \"longitude\": 2.3522"})
	}
```

Also add `knownFor` warning:

```go
	// Add knownFor field to travelDataValidation struct first
	if t.KnownFor == nil || *t.KnownFor == "" {
		*warns = append(*warns, validationIssue{Field: "external_url.knownFor", Code: "recommended", Message: "Add \"knownFor\": \"<description>\" to your external_url JSON. DestinationCard uses this as a subtitle. Example: \"knownFor\": \"Art museums and riverside walks\""})
	}
```

Update the `travelDataValidation` struct to include `KnownFor`:

```go
type travelDataValidation struct {
	City      *string  `json:"city"`
	Country   *string  `json:"country"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	KnownFor  *string  `json:"knownFor"`
}
```

- [ ] **Step 3: Upgrade fitness validator warning (line 870)**

```go
	if f.Activity == nil || *f.Activity == "" {
		*warns = append(*warns, validationIssue{Field: "external_url.activity", Code: "recommended", Message: "Add \"activity\": \"<activity name>\" to your external_url JSON. FitnessCard uses this as the card title. Example: \"activity\": \"Morning HIIT Circuit\""})
	}
```

Also add `duration_min` warning — add field to struct and check:

```go
type fitnessDataValidation struct {
	Activity    *string `json:"activity"`
	DurationMin *int    `json:"duration_min"`
}
```

```go
	if f.DurationMin == nil {
		*warns = append(*warns, validationIssue{Field: "external_url.duration_min", Code: "recommended", Message: "Add \"duration_min\": <integer> to your external_url JSON. FitnessCard shows workout duration. Example: \"duration_min\": 30"})
	}
```

- [ ] **Step 4: Upgrade remaining validators**

Apply the same pattern to all other validators with `"missing"` code warnings:

- `validateGameData` (line ~525): sport warning → `"recommended"` + example
- `validateMusicData` (lines 924-932): title, venue, date warnings → `"recommended"` + examples
- `validateMediaData` (line 956): type warning → `"recommended"` + example
- `validatePlayerData` (line 981): team warning → `"recommended"` + example
- `validateBoxScoreData` (line 1015): sport warning → `"recommended"` + example
- `validateScienceData`: add tags warning (not currently checked):

```go
	// Add to scienceDataValidation struct:
	Tags []string `json:"tags"`

	// Add check:
	if len(s.Tags) == 0 {
		*warns = append(*warns, validationIssue{Field: "external_url.tags", Code: "recommended", Message: "Add \"tags\": [\"astronomy\", \"NASA\"] to your external_url JSON. ScienceCard shows topic tags. Example: \"tags\": [\"physics\", \"quantum\"]"})
	}
```

- `validatePetData`: add species, breed, name warnings:

```go
type petDataValidation struct {
	Type    *string `json:"type"`
	Name    *string `json:"name"`
	Species *string `json:"species"`
	Breed   *string `json:"breed"`
}

	if p.Name == nil || *p.Name == "" {
		*warns = append(*warns, validationIssue{Field: "external_url.name", Code: "recommended", Message: "Add \"name\": \"<pet name>\" to your external_url JSON. PetSpotlightCard displays the pet's name prominently. Example: \"name\": \"Biscuit\""})
	}
	if p.Species == nil || *p.Species == "" {
		*warns = append(*warns, validationIssue{Field: "external_url.species", Code: "recommended", Message: "Add \"species\": \"dog\" or \"cat\" to your external_url JSON. Example: \"species\": \"dog\""})
	}
	if p.Breed == nil || *p.Breed == "" {
		*warns = append(*warns, validationIssue{Field: "external_url.breed", Code: "recommended", Message: "Add \"breed\": \"<breed>\" to your external_url JSON. Example: \"breed\": \"Golden Retriever\""})
	}
```

- [ ] **Step 5: Run all tests**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -v`

Expected: All PASS. The catalog examples in `hints.go` may now generate more warnings — that's fine since they're advisory. If any `TestHints_ExamplesLintClean` subtests fail (errors, not warnings), fix the catalog example.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/post.go
git commit -m "feat(lint): upgrade all validators to actionable patch warnings (#198)"
```

---

## Task 4: Add Generator Field to Hint Catalog (#199)

**Files:**
- Modify: `backend/internal/handler/hints.go`
- Modify: `backend/internal/handler/hints_test.go`

Mark each hint with its generator: which skill produces it, or "system" for system-only hints.

- [ ] **Step 1: Add Generator field to hintDescriptor**

In `hints.go`, update the `hintDescriptor` struct (line 37):

```go
type hintDescriptor struct {
	Hint           string          `json:"hint"`
	Description    string          `json:"description"`
	PostType       string          `json:"post_type"`
	StructuredJSON bool            `json:"structured_json"`
	RequiredFields []string        `json:"required_fields"`
	Example        json.RawMessage `json:"example"`
	Renders        *hintRenderInfo `json:"renders,omitempty"`
	PickWhen       string          `json:"pick_when,omitempty"`
	AvoidWhen      string          `json:"avoid_when,omitempty"`
	Generator      string          `json:"generator,omitempty"`
}
```

- [ ] **Step 2: Add generator values to each hint in buildHintCatalog**

Add `Generator` to each entry in `buildHintCatalog()`. Use these values:

| Hint | Generator |
|---|---|
| card, place, article, event, calendar, deal, digest, brief, comparison | `beepbopboop-post` |
| weather | `beepbopboop-post` |
| outfit | `beepbopboop-fashion` |
| scoreboard, matchup, standings, player_spotlight, box_score | `beepbopboop-news` (sports delegation) |
| entertainment | `beepbopboop-celebrity` |
| movie, show | `beepbopboop-movies` |
| album, concert | `beepbopboop-music` |
| restaurant | `beepbopboop-food` |
| destination | `beepbopboop-travel` |
| science | `beepbopboop-science` |
| pet_spotlight | `beepbopboop-pets` |
| fitness | `beepbopboop-fitness` |
| game_release, game_review | `beepbopboop-gaming` |
| creator_spotlight | `beepbopboop-creators` |
| feedback | `system` |
| video_embed | `beepbopboop-post` |

Example for one entry:

```go
{
	Hint:           "feedback",
	// ... existing fields ...
	Generator:      "system",
},
```

- [ ] **Step 3: Add test for generator field**

Add to `hints_test.go`:

```go
func TestHints_GeneratorFieldPresent(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	for _, e := range hr.DisplayHints {
		if e.Generator == "" {
			t.Errorf("hint %q missing generator field", e.Hint)
		}
	}
}
```

Update the `hintEntry` struct in `hints_test.go` to include `Generator`:

```go
type hintEntry struct {
	Hint           string          `json:"hint"`
	Description    string          `json:"description"`
	PostType       string          `json:"post_type"`
	StructuredJSON bool            `json:"structured_json"`
	RequiredFields []string        `json:"required_fields"`
	Example        json.RawMessage `json:"example"`
	Renders        *hintRenders    `json:"renders,omitempty"`
	PickWhen       string          `json:"pick_when,omitempty"`
	AvoidWhen      string          `json:"avoid_when,omitempty"`
	Generator      string          `json:"generator,omitempty"`
}
```

- [ ] **Step 4: Run tests**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -run TestHints -v`

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/hints.go backend/internal/handler/hints_test.go
git commit -m "feat(hints): add generator field to hint catalog (#199)"
```

---

## Task 5: Fix external_url Docs in Skills (#197)

**Files:**
- Modify: `.claude/skills/_shared/PUBLISH_ENVELOPE.md`
- Modify: `.claude/skills/beepbopboop-food/SKILL.md`
- Modify: `.claude/skills/beepbopboop-travel/SKILL.md`
- Modify: `.claude/skills/beepbopboop-fitness/SKILL.md`

- [ ] **Step 1: Add canonical structured external_url section to PUBLISH_ENVELOPE.md**

After line 109 ("For structured hints..."), replace the brief note with a full section:

```markdown
### Structured external_url — canonical pattern

For hints where `structured_json: true` in the hint catalog, `external_url` carries a **JSON string** (not a raw object). The backend field is `ExternalURL string` — if you send a raw object, JSON decoding fails before lint even runs.

**Correct pattern (using jq):**

```bash
# Build your data object
DATA_JSON='{"name":"Ramen House","rating":4.5,"cuisine":["Japanese","Ramen"],"latitude":53.34,"longitude":-6.26}'

# Stringify it for the outer JSON envelope
EXTERNAL_URL=$(echo "$DATA_JSON" | jq -c . | jq -Rs .)

# Use in payload — note $EXTERNAL_URL already has outer quotes from jq -Rs
PAYLOAD=$(jq -n \
  --arg title "Best Ramen in Dublin" \
  --arg body "Rich tonkotsu broth..." \
  --argjson ext "$EXTERNAL_URL" \
  '{title: $title, body: $body, external_url: $ext, display_hint: "restaurant", post_type: "place"}')
```

**Result in the JSON body:**
```json
{"external_url": "{\"name\":\"Ramen House\",\"rating\":4.5}"}
```

**Common mistake (raw object — will fail):**
```json
{"external_url": {"name": "Ramen House", "rating": 4.5}}
```
```

- [ ] **Step 2: Fix beepbopboop-food SKILL.md**

Find the publish step that has `"external_url": {FOOD_DATA_JSON}` and replace with:

```markdown
"external_url": $(echo "$FOOD_DATA_JSON" | jq -c . | jq -Rs .),
```

Add a note: "See `../_shared/PUBLISH_ENVELOPE.md` § Structured external_url for the canonical pattern."

- [ ] **Step 3: Fix beepbopboop-travel SKILL.md**

Find `"external_url": {TRAVEL_JSON_STRING}` and replace with:

```markdown
"external_url": $(echo "$TRAVEL_JSON" | jq -c . | jq -Rs .),
```

- [ ] **Step 4: Fix beepbopboop-fitness SKILL.md**

Find `"external_url": <FITNESS_JSON_STRING>` and replace with:

```markdown
"external_url": $(echo "$FITNESS_JSON" | jq -c . | jq -Rs .),
```

- [ ] **Step 5: Commit**

```bash
git add .claude/skills/_shared/PUBLISH_ENVELOPE.md .claude/skills/beepbopboop-food/SKILL.md .claude/skills/beepbopboop-travel/SKILL.md .claude/skills/beepbopboop-fitness/SKILL.md
git commit -m "fix(skills): standardize external_url as JSON string across all skills (#197)"
```

---

## Task 6: Add Dispatch Table to Main Router (#200)

**Files:**
- Modify: `.claude/skills/beepbopboop-post/SKILL.md`

- [ ] **Step 1: Add dispatch table after Step 0a routing table**

After the mode routing table (around line 87, before Step 0b), add a new section:

```markdown
### Step 0a-2: Specialty skill dispatch

If the user's idea matches a specialty topic below, **delegate to the named skill** instead of handling internally. This ensures the post gets the correct structured `display_hint` and rich card rendering.

| Topic Keywords | Delegate To | Produces Hints |
|---|---|---|
| restaurant, food, dining, cuisine, cafe, brunch, new opening | `beepbopboop-food` | `restaurant` |
| movie, film, TV show, streaming, what to watch, Netflix, series | `beepbopboop-movies` | `movie`, `show` |
| album, artist, concert, music, new release, Spotify, playlist | `beepbopboop-music` | `album`, `concert` |
| pet, adoption, dog, cat, shelter, rescue, breed | `beepbopboop-pets` | `pet_spotlight` |
| science, space, NASA, research, discovery, physics, biology | `beepbopboop-science` | `science` |
| travel, destination, flight, trip, vacation, city guide | `beepbopboop-travel` | `destination` |
| workout, fitness, exercise, gym, yoga, running, HIIT | `beepbopboop-fitness` | `fitness` |
| celebrity, red carpet, awards, entertainment news, gossip | `beepbopboop-celebrity` | `entertainment` |
| video game, gaming, game release, game review, Steam, PS5, Xbox | `beepbopboop-gaming` | `game_release`, `game_review` |
| local artist, maker, creator, spotlight, craftsperson | `beepbopboop-creators` | `creator_spotlight` |
| basketball, NBA, WNBA | `beepbopboop-basketball` | `scoreboard`, `matchup`, `standings`, `player_spotlight`, `box_score` |
| baseball, MLB | `beepbopboop-baseball` | `scoreboard`, `matchup`, `standings`, `box_score` |
| football, NFL, Super Bowl | `beepbopboop-football` | `scoreboard`, `matchup`, `standings`, `player_spotlight` |
| soccer, Premier League, Champions League, La Liga, MLS | `beepbopboop-soccer` | `scoreboard`, `matchup`, `standings` |

**Priority:** Check specialty dispatch BEFORE falling through to Step 0b (local vs interest). A query like "best ramen near me" should go to `beepbopboop-food`, not `BASE_LOCAL.md`.

**In batch mode (MODE_BATCH.md):** When building the content plan at BT3, classify each post idea against this table. Route matching ideas to the specialty skill. Only use generic modes for ideas that don't match any specialty.
```

- [ ] **Step 2: Update the existing routing table entries**

In the mode routing table (lines 70-87), add entries for topics that currently fall through:

```markdown
| `food`, `restaurant`, `dining`, `where to eat` | Food | **Delegate to `beepbopboop-food`** |
| `movie`, `what to watch`, `streaming`, `TV` | Movies | **Delegate to `beepbopboop-movies`** |
| `music`, `album`, `concert`, `playlist` | Music | **Delegate to `beepbopboop-music`** |
| `pets`, `adoption`, `dog`, `cat` | Pets | **Delegate to `beepbopboop-pets`** |
| `science`, `space`, `NASA`, `research` | Science | **Delegate to `beepbopboop-science`** |
| `travel`, `destination`, `trip`, `vacation` | Travel | **Delegate to `beepbopboop-travel`** |
| `fitness`, `workout`, `exercise`, `gym` | Fitness | **Delegate to `beepbopboop-fitness`** |
| `celebrity`, `entertainment news`, `red carpet` | Celebrity | **Delegate to `beepbopboop-celebrity`** |
| `gaming`, `video game`, `game release` | Gaming | **Delegate to `beepbopboop-gaming`** |
| `creator`, `local artist`, `maker spotlight` | Creators | **Delegate to `beepbopboop-creators`** |
```

- [ ] **Step 3: Commit**

```bash
git add .claude/skills/beepbopboop-post/SKILL.md
git commit -m "feat(router): add specialty skill dispatch table (#200)"
```

---

## Task 7: Create beepbopboop-gaming Skill (#199)

**Files:**
- Create: `.claude/skills/beepbopboop-gaming/SKILL.md`

- [ ] **Step 1: Create the skill directory**

```bash
mkdir -p /Users/shanegleeson/Repos/beepbopboop/.claude/skills/beepbopboop-gaming
```

- [ ] **Step 2: Write SKILL.md**

Create `.claude/skills/beepbopboop-gaming/SKILL.md` with this content:

````markdown
---
name: beepbopboop-gaming
description: Create video game posts — new releases, reviews, upcoming titles using RAWG/Steam
argument-hint: <game title or topic> [locality]
allowed-tools: Bash(curl *), Bash(jq *), Bash(cat *), Bash(beepbopgraph *), WebSearch, WebFetch
---

# BeepBopBoop Gaming Skill

Generate `game_release` and `game_review` posts for video games.

## Step 0: Load configuration

Read `../_shared/CONFIG.md` and follow it.
Read `../_shared/CONTEXT_BOOTSTRAP.md` and execute the four parallel fetches.

### Required env vars
- `BEEPBOPBOOP_API_URL` (required)
- `BEEPBOPBOOP_AGENT_TOKEN` (required)

### Optional env vars
- `RAWG_API_KEY` — enables RAWG API for richer game metadata. Free tier: 20k req/month at https://rawg.io/apidocs

## Step GM1: Identify the game or topic

Parse the user's request:
- **Specific game title** → search for that game
- **"upcoming"** or **"new releases"** → fetch upcoming/recently released games
- **Genre** (e.g., "indie", "RPG", "FPS") → search by genre
- **Platform** (e.g., "PS5", "Switch", "PC") → filter by platform

## Step GM2: Fetch game data

### If RAWG_API_KEY is available:

```bash
# Search for a specific game
GAMES=$(curl -s "https://api.rawg.io/api/games?key=$RAWG_API_KEY&search=$(echo "$QUERY" | jq -Rr @uri)&page_size=5")

# Or fetch upcoming releases
GAMES=$(curl -s "https://api.rawg.io/api/games?key=$RAWG_API_KEY&dates=$(date +%Y-%m-%d),$(date -v+3m +%Y-%m-%d)&ordering=-added&page_size=10")
```

Extract from response: `name`, `released`, `metacritic`, `platforms[].platform.name`, `genres[].name`, `background_image`, `description_raw`, `slug`.

### If no RAWG_API_KEY:

Use WebSearch to find game information:
```
WebSearch: "<game title> release date platforms metacritic 2026"
```

Extract: title, release date, platforms, genres, review scores from search results.

### Steam store fallback:

```bash
# Search Steam
STEAM=$(curl -s "https://store.steampowered.com/api/storeappdetails?appids=<APP_ID>")
```

## Step GM3: Classify hint type

- Game not yet released OR released within last 7 days → `game_release`
- Game released > 7 days ago with review scores available → `game_review`

## Step GM4: Build structured external_url

### For game_release:

```bash
GAME_DATA=$(jq -n \
  --arg title "$TITLE" \
  --arg status "$STATUS" \
  --arg releaseDate "$RELEASE_DATE" \
  --argjson platforms "$PLATFORMS_JSON" \
  --argjson genres "$GENRES_JSON" \
  --arg description "$DESCRIPTION" \
  --arg coverURL "$COVER_URL" \
  '{
    title: $title,
    status: $status,
    releaseDate: $releaseDate,
    platforms: $platforms,
    genres: $genres,
    description: $description,
    coverURL: $coverURL
  }')
```

Where:
- `status`: `"upcoming"` | `"released"` | `"early_access"`
- `releaseDate`: ISO date string (e.g., `"2026-06-15"`)
- `platforms`: array of strings (e.g., `["PC", "PS5", "Xbox Series X"]`)
- `genres`: array of strings (e.g., `["RPG", "Action"]`)

### For game_review:

```bash
GAME_DATA=$(jq -n \
  --arg title "$TITLE" \
  --arg status "released" \
  --arg releaseDate "$RELEASE_DATE" \
  --argjson platforms "$PLATFORMS_JSON" \
  --argjson genres "$GENRES_JSON" \
  --argjson metacriticScore "$METACRITIC" \
  --arg description "$DESCRIPTION" \
  --arg coverURL "$COVER_URL" \
  '{
    title: $title,
    status: $status,
    releaseDate: $releaseDate,
    platforms: $platforms,
    genres: $genres,
    metacriticScore: $metacriticScore,
    description: $description,
    coverURL: $coverURL
  }')
```

## Step GM5: Build the post

Write a compelling title and body:
- **Title:** Hook the reader. Not just the game name — add context. E.g., "Hollow Knight: Silksong Finally Has a Release Date" not "Hollow Knight: Silksong"
- **Body:** 2-3 sentences. What makes this game notable? What should the reader know? Include platform availability and price if known.

## Step GM6: Find or generate image

1. Use `coverURL` from RAWG/Steam if available (set as `image_url`)
2. If no cover URL: use WebSearch to find an official screenshot
3. Fallback: invoke `beepbopboop-images` subskill

## Step GM7: Publish

Stringify the external_url (see `../_shared/PUBLISH_ENVELOPE.md` § Structured external_url):

```bash
EXTERNAL_URL=$(echo "$GAME_DATA" | jq -c . | jq -Rs .)
```

Build payload and follow `../_shared/PUBLISH_ENVELOPE.md` steps P1-P4:
- `display_hint`: `"game_release"` or `"game_review"`
- `post_type`: `"article"`
- `visibility`: `"public"`
- `labels`: 3-6 from: game title (slugified), platform names, genre names, `"gaming"`, `"new-release"` or `"review"`
````

- [ ] **Step 3: Commit**

```bash
git add .claude/skills/beepbopboop-gaming/SKILL.md
git commit -m "feat(skills): add beepbopboop-gaming skill for game_release/game_review (#199)"
```

---

## Task 8: Create beepbopboop-creators Skill (#199)

**Files:**
- Create: `.claude/skills/beepbopboop-creators/SKILL.md`

- [ ] **Step 1: Create the skill directory**

```bash
mkdir -p /Users/shanegleeson/Repos/beepbopboop/.claude/skills/beepbopboop-creators
```

- [ ] **Step 2: Write SKILL.md**

Create `.claude/skills/beepbopboop-creators/SKILL.md`:

````markdown
---
name: beepbopboop-creators
description: Create creator spotlight posts — local artists, makers, musicians, craftspeople
argument-hint: <creator name or "local creators in <area>"> [locality]
allowed-tools: Bash(curl *), Bash(jq *), Bash(cat *), Bash(beepbopgraph *), WebSearch, WebFetch
---

# BeepBopBoop Creators Skill

Generate `creator_spotlight` posts highlighting local artists, makers, musicians, and craftspeople.

## Step 0: Load configuration

Read `../_shared/CONFIG.md` and follow it.
Read `../_shared/CONTEXT_BOOTSTRAP.md` and execute the four parallel fetches.

### Required env vars
- `BEEPBOPBOOP_API_URL` (required)
- `BEEPBOPBOOP_AGENT_TOKEN` (required)

### Optional env vars
- None — this skill uses web search for discovery.

## Step CR1: Identify the creator or discovery scope

Parse the user's request:
- **Specific creator name** → search for that person
- **"local creators in <area>"** → discover creators in the specified area
- **Category** (e.g., "ceramicists", "muralists", "indie musicians") → search by craft
- **No specific input** → use `BEEPBOPBOOP_DEFAULT_LOCATION` and `BEEPBOPBOOP_INTERESTS` to find relevant local creators

## Step CR2: Research the creator

Use WebSearch to find information:

```
WebSearch: "<creator name> <area> artist portfolio"
WebSearch: "local <craft> <area> 2026"
```

Look for:
- Name and designation (what they do)
- Online presence: website, Instagram, Bandcamp, Etsy, Substack, SoundCloud, Behance
- Notable works or achievements
- Neighborhood/area they're based in
- Source article or profile where you found them

**Important:** Only spotlight creators with verifiable online presence. Do not fabricate profiles.

## Step CR3: Build structured external_url

```bash
CREATOR_DATA=$(jq -n \
  --arg designation "$DESIGNATION" \
  --arg area_name "$AREA_NAME" \
  --arg source "$SOURCE_URL" \
  --arg notable_works "$NOTABLE_WORKS" \
  --argjson tags "$TAGS_JSON" \
  --argjson links "$LINKS_JSON" \
  '{
    designation: $designation,
    area_name: $area_name,
    source: $source,
    notable_works: $notable_works,
    tags: $tags,
    links: $links
  }')
```

Where:
- `designation`: their primary role (e.g., `"ceramicist"`, `"muralist"`, `"indie folk musician"`)
- `area_name`: neighborhood or city (e.g., `"Stoneybatter, Dublin"`)
- `source`: URL where you found information about them
- `notable_works`: brief description of key works
- `tags`: array of strings for discovery (e.g., `["ceramics", "handmade", "local-art"]`)
- `links`: object with any of: `website`, `instagram`, `bandcamp`, `etsy`, `substack`, `soundcloud`, `behance`

Example links object:
```json
{
  "website": "https://example.com",
  "instagram": "@creator_handle",
  "etsy": "https://etsy.com/shop/creator"
}
```

## Step CR4: Build the post

Write a compelling title and body:
- **Title:** Focus on what makes them interesting, not just their name. E.g., "Dublin Ceramicist Turns Demolished Buildings Into Glazes" not "Meet Jane Doe"
- **Body:** 2-3 sentences. What's their story? What do they make? Why should the reader care? Include where to find/follow them.

## Step CR5: Find or generate image

1. If creator has a portfolio/Instagram with public images: use WebSearch to find a representative image URL
2. Fallback: invoke `beepbopboop-images` subskill with the creator's work as the prompt

## Step CR6: Publish

Stringify the external_url:

```bash
EXTERNAL_URL=$(echo "$CREATOR_DATA" | jq -c . | jq -Rs .)
```

Build payload and follow `../_shared/PUBLISH_ENVELOPE.md` steps P1-P4:
- `display_hint`: `"creator_spotlight"`
- `post_type`: `"discovery"`
- `visibility`: `"public"`
- `labels`: 3-6 from: creator name (slugified), designation, area, craft category, `"creator-spotlight"`, `"local-art"`
````

- [ ] **Step 3: Commit**

```bash
git add .claude/skills/beepbopboop-creators/SKILL.md
git commit -m "feat(skills): add beepbopboop-creators skill for creator_spotlight (#199)"
```

---

## Task 9: Add Preflight Step 0 (#203)

**Files:**
- Modify: `.claude/skills/beepbopboop-post/SKILL.md`
- Modify: `.claude/skills/_shared/CONFIG.md`

- [ ] **Step 1: Add preflight section to SKILL.md**

Add between "Step 0: Load configuration" (line 54) and "Step 0d: Bootstrap server context" (line 58):

```markdown
## Step 0-pre: Preflight checks

Before generating any content, verify the environment is ready. **If any required check fails, stop and report the issue.**

### Required checks (fail if missing):

```bash
# 1. Backend reachable
HINTS_CHECK=$(curl -s -o /dev/null -w "%{http_code}" "$BEEPBOPBOOP_API_URL/posts/hints")
if [ "$HINTS_CHECK" != "200" ]; then
  echo "PREFLIGHT FAIL: Backend unreachable at $BEEPBOPBOOP_API_URL (HTTP $HINTS_CHECK)"
  exit 1
fi

# 2. Auth valid
AUTH_CHECK=$(curl -s -o /dev/null -w "%{http_code}" "$BEEPBOPBOOP_API_URL/posts?limit=1" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN")
if [ "$AUTH_CHECK" != "200" ]; then
  echo "PREFLIGHT FAIL: Auth token invalid (HTTP $AUTH_CHECK)"
  exit 1
fi

# 3. Required CLIs
for cmd in jq curl; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "PREFLIGHT FAIL: Required CLI '$cmd' not found"
    exit 1
  fi
done
```

### Optional capability matrix:

Check which specialty skills have their dependencies met. Print the result and use it when routing in batch mode — only route to skills that passed preflight.

| Skill | Check |
|---|---|
| beepbopboop-food | `YELP_KEY` set |
| beepbopboop-movies | `TMDB_KEY` set |
| beepbopboop-music | `SPOTIFY_TOKEN` or `LASTFM_KEY` set |
| beepbopboop-gaming | `RAWG_API_KEY` set (optional — falls back to web search) |
| beepbopboop-travel | no external deps (web search) |
| beepbopboop-science | no external deps (web search) |
| beepbopboop-pets | no external deps (Petfinder is free) |
| beepbopboop-fitness | no external deps |
| beepbopboop-celebrity | no external deps (web search) |
| beepbopboop-creators | no external deps (web search) |
| beepbopboop-fashion | no external deps (web search + AI image gen) |
| beepbopboop-news | no external deps |
| beepbopboop-images | `BEEPBOPBOOP_IMGUR_CLIENT_ID` set (for re-hosting) |

Print availability:
```
Preflight complete:
  ✓ Backend reachable
  ✓ Auth valid
  ✓ jq, curl available
  Specialty skills:
    ✓ beepbopboop-food (YELP_KEY found)
    ✗ beepbopboop-movies (TMDB_KEY missing — movie/show cards unavailable)
    ✓ beepbopboop-music (SPOTIFY_TOKEN found)
    ...
```

In batch mode (MODE_BATCH.md), skip unavailable specialty skills and note in the final report why they were skipped.
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/beepbopboop-post/SKILL.md
git commit -m "feat(skills): add preflight checks for env vars, CLIs, backend (#203)"
```

---

## Task 10: Portable Onboarding (#202)

**Files:**
- Modify: `.claude/skills/beepbopboop-post/INIT_WIZARD.md`
- Modify: `.claude/skills/_shared/CONFIG.md`

- [ ] **Step 1: Add config-file bootstrap to CONFIG.md**

At the top of `_shared/CONFIG.md`, add a section before the existing content:

```markdown
## Config file bootstrap (non-interactive)

If running in an environment without `AskUserQuestion` (OpenClaw, Codex, etc.), the config can be provided as a file:

**Config file location:** `~/.config/beepbopboop/config`

**Required keys (must be present or the skill will stop):**
```
BEEPBOPBOOP_API_URL=http://192.168.1.x:8080
BEEPBOPBOOP_AGENT_TOKEN=<agent-token>
```

**Optional keys:**
```
BEEPBOPBOOP_DEFAULT_LOCATION=Dublin, Ireland
BEEPBOPBOOP_HOME_LAT=53.3498
BEEPBOPBOOP_HOME_LON=-6.2603
BEEPBOPBOOP_INTERESTS=basketball,food,science,gaming
BEEPBOPBOOP_FAMILY=partner:Alex:na:cooking;child:Sam:8:legos,swimming
BEEPBOPBOOP_SOURCES=hn,substack:stratechery
BEEPBOPBOOP_CALENDAR_URL=https://calendar.google.com/...
BEEPBOPBOOP_SCHEDULE=Mon|batch|;Wed|weather|;Fri|batch|
BEEPBOPBOOP_BATCH_MIN=8
BEEPBOPBOOP_BATCH_MAX=15
BEEPBOPBOOP_UNSPLASH_ACCESS_KEY=...
BEEPBOPBOOP_IMGUR_CLIENT_ID=...
BEEPBOPBOOP_GOOGLE_PLACES_KEY=...
YELP_KEY=...
TMDB_KEY=...
RAWG_API_KEY=...
SPOTIFY_TOKEN=...
```

**If config file is missing AND `AskUserQuestion` is not available:**
Print the template above and stop with:
```
Config file not found at ~/.config/beepbopboop/config.
Create it with at minimum BEEPBOPBOOP_API_URL and BEEPBOPBOOP_AGENT_TOKEN, then re-run.
```
```

- [ ] **Step 2: Update INIT_WIZARD.md to support non-interactive mode**

Add at the top of `INIT_WIZARD.md`, before the existing wizard steps:

```markdown
## Non-interactive bootstrap

If running in an environment without `AskUserQuestion` (OpenClaw, Codex, etc.):

1. Check if `~/.config/beepbopboop/config` exists
2. If yes: source it and skip the wizard — all config keys are loaded
3. If no: print the config template from `../_shared/CONFIG.md` and stop

The interactive wizard below writes to the same `~/.config/beepbopboop/config` file, so both paths converge.

---

## Interactive wizard (Claude Code)

*The following steps use `AskUserQuestion` and are only available in Claude Code.*
```

- [ ] **Step 3: Commit**

```bash
git add .claude/skills/beepbopboop-post/INIT_WIZARD.md .claude/skills/_shared/CONFIG.md
git commit -m "feat(skills): add non-interactive config bootstrap for portable onboarding (#202)"
```

---

## Task 11: Final Integration Test & Cleanup

**Files:**
- All modified files from Tasks 1-10

- [ ] **Step 1: Run full backend test suite**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./... -v`

Expected: All PASS

- [ ] **Step 2: Verify skill files are well-formed**

```bash
# Check all skill files exist and have frontmatter
for skill in beepbopboop-gaming beepbopboop-creators; do
  if [ -f ".claude/skills/$skill/SKILL.md" ]; then
    echo "✓ $skill"
  else
    echo "✗ $skill MISSING"
  fi
done
```

- [ ] **Step 3: Verify dispatch table entries match existing skills**

```bash
# List all beepbopboop skills
ls -d .claude/skills/beepbopboop-*/
```

Cross-reference against the dispatch table in SKILL.md — every skill in the table should exist as a directory.

- [ ] **Step 4: Close issues**

```bash
gh issue close 201 -c "Contract tests added in hints_test.go: iOS decode test, metadata completeness test"
gh issue close 197 -c "external_url guidance standardized across all skills to use JSON string pattern"
gh issue close 198 -c "Lint warnings upgraded to actionable patch instructions with examples"
gh issue close 200 -c "Dispatch table added to beepbopboop-post SKILL.md for all specialty skills"
gh issue close 199 -c "Added beepbopboop-gaming and beepbopboop-creators skills. feedback marked as system-only."
gh issue close 203 -c "Preflight Step 0-pre added to beepbopboop-post SKILL.md"
gh issue close 202 -c "Config-file bootstrap added for non-interactive onboarding"
```

- [ ] **Step 5: Final commit if any cleanup was needed**

```bash
git add -A
git status
# Only commit if there are changes
git commit -m "chore: Wave 2 cleanup and integration verification"
```
