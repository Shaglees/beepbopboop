# Wave 3: Feed Architecture & Skill Cleanup — Design Spec

**Goal:** User-configurable content mix with a full dashboard, plus completing the skill decomposition from #180.

**Issues:** #185 (spread settings), #180 Phase 3 (skill cleanup), #188 (close as done)

**Architecture:** Backend-driven spread targets stored in `user_settings`, exposed via REST endpoints, rendered as a slider-based settings UI in iOS. Skills read spread guidance from the API — no skill-side spread logic. Parallel track: complete skill decomposition (sport shared lib, init rename, hints cache).

---

## 1. Backend — Spread Settings Storage & API

### Data Model

Add `spread_targets` JSONB column to the existing `user_settings` table.

Schema:

```json
{
  "verticals": {
    "sports": { "weight": 0.25, "pinned": true },
    "food": { "weight": 0.15, "pinned": false },
    "music": { "weight": 0.10, "pinned": false },
    "travel": { "weight": 0.10, "pinned": false },
    "science": { "weight": 0.08, "pinned": false },
    "gaming": { "weight": 0.07, "pinned": false },
    "creators": { "weight": 0.05, "pinned": false },
    "fashion": { "weight": 0.05, "pinned": false },
    "movies": { "weight": 0.05, "pinned": false },
    "pets": { "weight": 0.05, "pinned": false },
    "news": { "weight": 0.05, "pinned": false }
  },
  "omega": "sports",
  "auto_adjust": true,
  "updated_at": "2026-04-24T00:00:00Z"
}
```

Fields:

- `weight`: 0.0–1.0 per vertical. All weights must sum to 1.0.
- `pinned`: If true, Hermes auto-adjustment skips this vertical.
- `omega`: The primary anchor category. Always gets at least 1 slot in batch mode.
- `auto_adjust`: Master toggle for engagement-driven weight shifts.

Default targets are seeded from the user's onboarding interests. If no interests were selected, use an even distribution across all verticals.

### Endpoints

**`GET /settings/spread`**

Returns current targets plus 30-day actual allocation and per-vertical status.

Response:

```json
{
  "targets": { "sports": 0.25, "food": 0.15, "music": 0.10, "travel": 0.10, "science": 0.08, "gaming": 0.07, "creators": 0.05, "fashion": 0.05, "movies": 0.05, "pets": 0.05, "news": 0.05 },
  "omega": "sports",
  "pinned": ["sports"],
  "auto_adjust": true,
  "actual_30d": { "sports": 0.23, "food": 0.16, "music": 0.08, "travel": 0.11, "science": 0.09, "gaming": 0.06, "creators": 0.05, "fashion": 0.04, "movies": 0.06, "pets": 0.05, "news": 0.07 },
  "status": {
    "sports": "on_target",
    "food": "on_target",
    "music": "below_target",
    "travel": "on_target",
    "science": "on_target",
    "gaming": "on_target",
    "creators": "on_target",
    "fashion": "on_target",
    "movies": "on_target",
    "pets": "on_target",
    "news": "on_target"
  }
}
```

Status logic: if actual is within ±3% of target → `on_target`, below → `below_target`, above → `above_target`.

`actual_30d` is computed from posts created in the last 30 days, grouped by primary label. Cached and recomputed hourly by Hermes.

**`PUT /settings/spread`**

Update targets. Request body:

```json
{
  "targets": { "sports": 0.25, "food": 0.15, ... },
  "omega": "sports",
  "pinned": ["sports"],
  "auto_adjust": true
}
```

Validation:
- All weights must be >= 0 and sum to 1.0 (±0.01 tolerance for float rounding).
- `omega` must be a key in `targets`.
- At least one vertical must be unpinned if `auto_adjust` is true.

**`GET /settings/spread/history`**

Returns 30-day daily breakdown for the trend chart.

Response:

```json
{
  "days": [
    {
      "date": "2026-04-24",
      "target": { "sports": 0.25, "food": 0.15 },
      "actual": { "sports": 0.22, "food": 0.16 }
    }
  ]
}
```

Computed from daily post counts by label. Stored as a materialized daily snapshot by Hermes (one row per day in a `spread_history` table).

### Hermes Auto-Adjustment

The existing hourly cron (`ComputeFromEngagement()`) gains spread-awareness:

1. Read user's `spread_targets`.
2. If `auto_adjust` is false, skip.
3. For each non-pinned vertical, compare 7-day actual allocation to target.
4. If actual < target and positive engagement signals exist for that vertical → nudge weight up by 2%.
5. If actual > target and negative engagement signals exist (less/not_for_me reactions) → nudge weight down by 2%.
6. Re-normalize all non-pinned weights to maintain sum = 1.0.
7. Write updated targets back to `user_settings`.
8. Write a daily snapshot row to `spread_history`.

Maximum shift per run: ±2% per vertical. This keeps the feed evolution gradual.

### Spread-Guidance Endpoint Change

`POST /posts/spread-guidance` currently returns hardcoded defaults. Change it to:

1. Read user's `spread_targets` from `user_settings`.
2. If no targets exist, return defaults (even distribution).
3. Return the same response shape — skills see no API change.

---

## 2. iOS — Content Mix Settings UI

### Location

New "Content Mix" section inside the existing Settings/Profile screen, below the existing settings sections.

### UI Design: Slider-Based List

Components:

1. **Summary bar** — horizontal stacked bar showing the overall mix by color. Each vertical gets a distinct color. Appears at the top of the Content Mix section.

2. **Vertical list** — one row per content vertical:
   - Left: emoji icon + vertical name
   - Omega badge: green "Ω Primary" pill on the omega vertical
   - Right: percentage label + pin toggle icon
   - Pin icon is solid when pinned, faded when unpinned
   - Tapping a row opens a detail sheet with a slider (0–100%) and an option to set as omega

3. **Auto-adjust toggle** — at the bottom of the list. "Auto-adjust from engagement" with an on/off switch.

4. **Status indicators** — each row shows a subtle colored dot:
   - Green: on target
   - Orange: below target
   - Blue: above target

### Data Flow

- On appear: `GET /settings/spread` → populate sliders, summary bar, status dots
- On slider change: debounce 500ms → `PUT /settings/spread` with new weights. Re-normalize other non-pinned weights proportionally when one slider moves.
- Pin toggle: immediate `PUT /settings/spread`.
- Auto-adjust toggle: immediate `PUT /settings/spread`.

### Models

```swift
struct SpreadTargets: Codable {
    let targets: [String: Double]
    let omega: String
    let pinned: [String]
    let autoAdjust: Bool
    let actual30d: [String: Double]
    let status: [String: String]

    enum CodingKeys: String, CodingKey {
        case targets, omega, pinned
        case autoAdjust = "auto_adjust"
        case actual30d = "actual_30d"
        case status
    }
}
```

---

## 3. Skill Changes — Spread-Guidance Integration

### MODE_BATCH.md

Step BT2 (allocation) changes from hardcoded buckets to API-driven:

1. Call `GET /settings/spread` to fetch user's target weights.
2. Use `targets` map to determine how many of N batch slots go to each vertical.
3. `omega` vertical always gets at least 1 slot.
4. Fill remaining slots proportionally by weight.
5. Diversity scorecard validation stays the same — references user's targets instead of hardcoded values.

### WEIGHT_COMPUTATION.md

Update to document:
- How Hermes reads `spread_targets` and applies auto-adjustment.
- What "pinned" means (weight locked from auto-adjustment).
- The ±2% per-run nudge cap.

### No other skill changes

Skills don't read spread guidance directly. They call `POST /posts/spread-guidance` which now reads from the user's stored targets. The response shape is unchanged.

---

## 4. Skill Decomposition — #180 Phase 3

### 4a. Sport Shared Library

Create `.claude/skills/_shared/SPORTS_COMMON.md` containing:
- Score formatting patterns (box score layout, standings table, stat lines)
- Team data fetching (team lookup, roster, schedule)
- Shared sport-post publishing conventions (labels, display hints, structured JSON patterns)

Update `beepbopboop-soccer`, `beepbopboop-basketball`, `beepbopboop-football`, `beepbopboop-baseball` to reference `../_shared/SPORTS_COMMON.md` for shared patterns instead of duplicating them.

### 4b. INIT_WIZARD.md → MODE_INIT.md Rename

Rename `beepbopboop-post/INIT_WIZARD.md` to `beepbopboop-post/MODE_INIT.md` for naming consistency with other mode files. Update all references in:
- `beepbopboop-post/SKILL.md`
- `beepbopboop-post/README.md` (if it references the file)

### 4c. Client-Side Hints Cache

Skills currently call `GET /posts/hints` on every invocation. Add caching:

1. On fetch, write response to `~/.cache/beepbopboop/hints.json` with a `fetched_at` timestamp.
2. On next invocation, check `fetched_at`. If < 24 hours old, read from cache.
3. If cache is missing or stale, fetch from API and update cache.
4. Document the cache pattern in `_shared/CONTEXT_BOOTSTRAP.md` (where hints fetching is defined).

Cache format:

```json
{
  "fetched_at": "2026-04-24T12:00:00Z",
  "hints": [ ... ]
}
```

### 4d. Design-System Skill Splits

The design-system skills (teach-impeccable, etc.) are third-party plugins, not BeepBopBoop code. Skip this item — it's outside our control.

---

## 5. Testing Strategy

### Backend Tests

- `TestSpreadHandler_GetDefault` — returns even distribution when no targets stored
- `TestSpreadHandler_GetWithTargets` — returns stored targets + computed actual_30d + status
- `TestSpreadHandler_PutValidation` — rejects weights that don't sum to 1.0, missing omega, all-pinned with auto_adjust
- `TestSpreadHandler_History` — returns 30-day daily snapshots
- `TestHermes_AutoAdjust` — verifies ±2% nudge cap, pinned verticals unchanged, re-normalization

### iOS Tests

- `SpreadTargets` model decoding from API response
- Weight re-normalization logic when one slider moves

### Skill Tests

- Verify `MODE_BATCH.md` references the spread API instead of hardcoded values (manual review)
- Verify hints cache reads/writes correctly (manual test with stale/fresh cache)

---

## 6. Migration & Defaults

- Add `spread_targets` JSONB column to `user_settings` (nullable, defaults to NULL).
- When NULL, endpoints return default even distribution.
- No data migration needed — targets are seeded lazily on first `GET /settings/spread` call from the iOS settings screen.
- Add `spread_history` table: `(user_id, date, targets JSONB, actuals JSONB)`. Hermes writes one row per user per day.
