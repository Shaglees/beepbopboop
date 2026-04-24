# User Profile System Design

_2026-04-24_

## Problem

The user model is nearly empty (id, firebase_uid, created_at). Personal details like display name, location, timezone, and interests are either missing entirely or scattered across local config files that only exist on the agent machine. Skills can't pull a user profile from the server. The iOS app has no profile object. This means:

- Skills rely on a local config file for personalization, which breaks on other machines
- The iOS app can't display who the user is
- Interests are stored only as embedding vectors — no plaintext, no editability
- There's no way for the system to learn or adapt to changing user interests over time

## Approach

Extend the `users` table with identity fields. Create three new tables for rich interests, lifestyle tags, and content preferences. Add a `GET /user/profile` endpoint that both iOS and skills consume. Build an iOS onboarding flow that populates the profile. Add a background worker that infers interests from engagement and a feedback mechanism that respects interest seasonality.

---

## 1. Database Schema

### 1a. `users` table extensions

```sql
ALTER TABLE users
  ADD COLUMN display_name       TEXT NOT NULL DEFAULT '',
  ADD COLUMN avatar_url         TEXT NOT NULL DEFAULT '',
  ADD COLUMN timezone           TEXT NOT NULL DEFAULT 'UTC+0',
  ADD COLUMN home_location      TEXT NOT NULL DEFAULT '',
  ADD COLUMN home_lat           DOUBLE PRECISION,
  ADD COLUMN home_lon           DOUBLE PRECISION,
  ADD COLUMN profile_updated_at TIMESTAMPTZ;
```

`timezone` stores UTC offset strings (e.g. `UTC+0`, `UTC-7`, `UTC+5:30`) — unambiguous for agents and humans.

`profile_updated_at` is null until the user completes onboarding. Null = show onboarding flow.

### 1b. `user_interests` table

```sql
CREATE TABLE user_interests (
  id            TEXT PRIMARY KEY,
  user_id       TEXT NOT NULL REFERENCES users(id),
  category      TEXT NOT NULL,
  topic         TEXT NOT NULL,
  source        TEXT NOT NULL CHECK (source IN ('user', 'inferred')),
  confidence    DOUBLE PRECISION NOT NULL DEFAULT 1.0,
  dismissed     BOOLEAN NOT NULL DEFAULT FALSE,
  paused_until  TIMESTAMPTZ,
  last_asked_at TIMESTAMPTZ,
  times_asked   INT NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_interests_user ON user_interests(user_id);
```

Each interest is a rich object:

| Field | Purpose |
|-------|---------|
| `category` | Top-level grouping: sports, food, music, science, travel, fitness, pets, fashion, entertainment, tech |
| `topic` | Specific interest within category: NBA, ramen, indie rock, JWST |
| `source` | `user` = explicitly declared, `inferred` = derived from engagement |
| `confidence` | 0.0–1.0. User-declared start at 1.0. Inferred start at their computed score. "Less of this" sets to 0.3. |
| `dismissed` | User explicitly dismissed an inferred interest |
| `paused_until` | Seasonal pause — interest is dormant until this date. Skills skip it. |
| `last_asked_at` | When the system last asked about declining engagement |
| `times_asked` | How many times the system has asked. Max 3, then stop asking. |

### 1c. `user_lifestyle_tags` table

```sql
CREATE TABLE user_lifestyle_tags (
  id           TEXT PRIMARY KEY,
  user_id      TEXT NOT NULL REFERENCES users(id),
  tag_category TEXT NOT NULL,
  tag_value    TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, tag_category, tag_value)
);

CREATE INDEX idx_user_lifestyle_user ON user_lifestyle_tags(user_id);
```

Tag categories: `diet`, `fitness`, `pets`, `family`.

Examples:
- `{tag_category: "diet", tag_value: "vegetarian"}`
- `{tag_category: "pets", tag_value: "dog_owner"}`
- `{tag_category: "family", tag_value: "parent_of_8yo"}`

Skills use these for content filtering (food skill skips meat restaurants for vegetarians, pet skill knows they have a dog).

### 1d. `user_content_prefs` table

```sql
CREATE TABLE user_content_prefs (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES users(id),
  category    TEXT,
  depth       TEXT NOT NULL DEFAULT 'standard' CHECK (depth IN ('brief', 'standard', 'detailed')),
  tone        TEXT NOT NULL DEFAULT 'casual' CHECK (tone IN ('casual', 'informative', 'playful')),
  max_per_day INT,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, category)
);

CREATE INDEX idx_user_content_prefs_user ON user_content_prefs(user_id);
```

`category` is nullable — null row = global defaults, non-null = per-category override.

`max_per_day` is nullable — null = no cap.

---

## 2. API Endpoints

### Firebase-auth (iOS client)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/user/profile` | Full profile: identity + interests + lifestyle + content prefs |
| `PUT` | `/user/profile` | Update identity fields (name, avatar, timezone, location) |
| `PUT` | `/user/interests` | Bulk set declared interests (replaces existing `source='user'` rows) |
| `POST` | `/user/interests/{id}/promote` | Promote an inferred interest to `source='user'` |
| `POST` | `/user/interests/{id}/dismiss` | Dismiss an inferred interest |
| `POST` | `/user/interests/{id}/pause` | Pause interest with a `paused_until` date |
| `PUT` | `/user/lifestyle` | Bulk set lifestyle tags |
| `PUT` | `/user/content-prefs` | Set content delivery preferences |

### Agent-auth (skills)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/user/profile` | Same response as above, read-only. Skills call this as a pre-task. |

### Response shape for `GET /user/profile`

```json
{
  "identity": {
    "display_name": "Shane",
    "avatar_url": "https://...",
    "timezone": "UTC-7",
    "home_location": "San Francisco",
    "home_lat": 37.77,
    "home_lon": -122.42
  },
  "interests": [
    {
      "id": "int_abc123",
      "category": "sports",
      "topic": "NBA",
      "source": "user",
      "confidence": 1.0,
      "paused_until": null
    },
    {
      "id": "int_def456",
      "category": "food",
      "topic": "ramen",
      "source": "inferred",
      "confidence": 0.72,
      "paused_until": null
    }
  ],
  "lifestyle": [
    {"category": "diet", "value": "vegetarian"},
    {"category": "pets", "value": "dog_owner"}
  ],
  "content_prefs": [
    {"category": null, "depth": "standard", "tone": "casual", "max_per_day": null},
    {"category": "sports", "depth": "detailed", "tone": "informative", "max_per_day": 5}
  ],
  "profile_initialized": true
}
```

`profile_initialized` is derived: true when `display_name` is non-empty and at least 1 interest exists with `source='user'`.

Dismissed interests and interests with `paused_until` in the future are excluded from the response by default. iOS can request them with `?include_inactive=true` for the profile editing screen.

---

## 3. iOS Onboarding Flow

Triggered when `profile_initialized` is false after sign-up or first login.

### Step 1: Name & Avatar
Text field for display name. Optional photo picker for avatar (uploaded to storage, URL saved via `PUT /user/profile`).

### Step 2: Location & Timezone
Request location permission. Auto-fill home location name and UTC offset from device's `TimeZone.current` (converted to UTC offset string). User can manually adjust.

### Step 3: Notifications
Push notification permission prompt with context: "Get your daily digest and live score alerts." Configure digest hour. Toggle calendar sync. Writes to existing `PUT /user/settings`.

### Step 4: Interests
Category grid (Sports, Food, Music, Science, Travel, Fitness, Pets, Fashion, Entertainment, Tech). When the user taps a category, it expands into a **carousel of real card mockups** showing what that skill actually produces:

- Tapping "Sports" → carousel of scoreboard card, matchup card, player spotlight
- Tapping "Food" → carousel of restaurant card, deal card, seasonal dish card
- Tapping "Music" → carousel of album card, concert card

This gives the user a concrete preview of what they'll receive rather than an abstract category name. Within each category, the user can select specific topics (e.g. NBA, Premier League within Sports). Minimum 3 categories selected.

Writes to `PUT /user/interests` with `source='user'`, `confidence=1.0`.

After interests are saved, the existing `POST /user/interests` embedding pipeline runs to generate the user embedding vector from the plaintext interest list.

### Step 5: Content Frequency
"How much content per day?" — slider or segmented control mapping to a target range. The backend stores this as the global `max_per_day` in `user_content_prefs` (category=null). When the agent calls `GET /posts/stats`, the response includes the user's target frequency so batch mode knows how many posts to aim for, replacing the local `BATCH_MIN`/`BATCH_MAX` config values.

### Step 6: Lifestyle Tags (optional, skippable)
Toggleable pills grouped by category:
- Diet: vegetarian, vegan, gluten-free, halal, kosher
- Fitness: runner, cyclist, gym, yoga, swimmer
- Pets: dog owner, cat owner
- Family: parent (with child age input), couple

### Step 7: Content Preferences (optional, skippable)
Depth slider (brief ↔ detailed) and tone picker (casual / informative / playful). Sensible defaults: standard depth, casual tone.

Each step writes to its respective endpoint immediately so partial progress is saved. If the user kills the app mid-onboarding and returns, they resume where they left off (check which fields are already populated).

---

## 4. Skill Pre-Task Profile Fetch

### Bootstrap change

The shared bootstrap (`_shared/CONTEXT_BOOTSTRAP.md`) currently fetches 4 endpoints in parallel:
- `GET /posts/hints`
- `GET /posts/stats`
- `GET /reactions/summary`
- `GET /events/summary`

Add a 5th parallel fetch:
```bash
PROFILE=$(curl -s -H "$AUTH" "$API/user/profile")
```

Pin into working memory alongside the existing data.

### Batch mode integration

In `MODE_BATCH.md` (BT1), the profile feeds into content planning:
- **Declared interests** replace `BEEPBOPBOOP_INTERESTS` as the primary signal for what categories to generate. Active (not paused, not dismissed) interests with confidence > 0.5 drive the fill phase.
- **Lifestyle tags** filter content: food skill checks diet tags, pet skill checks pet tags, fitness skill checks activity tags.
- **Content prefs** shape writing: depth and tone per category. Global `max_per_day` replaces `BATCH_MIN`/`BATCH_MAX`.
- **Target frequency** from `content_prefs` (global `max_per_day`) is included in `GET /posts/stats` response so the batch knows how many posts to produce.

### Single-post mode integration

In `BASE_LOCAL.md`, when someone says "post about kids birthday presents", the skill reads the profile, sees `{tag_category: "family", tag_value: "parent_of_8yo"}`, and focuses on age-appropriate gifts rather than generic research.

### Fallback

Local config keys (`BEEPBOPBOOP_INTERESTS`, `BEEPBOPBOOP_HOME_ADDRESS`, etc.) remain as fallbacks — used only if the profile endpoint is unreachable or returns empty data. The server profile is the source of truth.

### Router clarity

Both batch and single-post flows go through the same bootstrap. The router (`SKILL.md` Step 0a) dispatches to the right mode. Both modes pin the profile into working memory. The key difference: batch plans a diverse multi-post spread using the full profile, while single-post uses the profile for contextual depth on one idea.

---

## 5. Interest Lifecycle

### Inferred interests (background worker)

A new worker (like the existing embedding/weather/sports workers) runs every 24 hours:

1. Query `post_events` and `reactions` per user over the last 30 days
2. Aggregate engagement by post label/category (saves, dwell time, "more" reactions)
3. For categories with strong signal not already in `user_interests`: insert with `source='inferred'` and a computed confidence score
4. For existing inferred interests with declining engagement: lower confidence. Remove if confidence drops below 0.1.
5. Skip any interest where `paused_until` is in the future

### Interest decay for user-declared interests

When engagement data shows clear disinterest in a user-declared interest for 30+ days (consistent "less" reactions, zero saves, low dwell), the system generates a **feedback panel** post using the existing `feedback` display hint:

"You haven't been engaging with sports posts recently. What would you like to do?"

**Options:**
- **"Still interested"** — no change, back off 90 days before asking again
- **"Pause for a while"** — user picks duration: 1 month, 2 months, 4 months, or "Until next season". Interest stays in profile but `paused_until` is set. Skills skip it. Auto-reactivates on the resume date with a welcome-back post ("Sports are back in your feed — here's what you missed").
- **"Less of this"** — reduce confidence to 0.3. Interest still appears occasionally but at lower priority.
- **"Remove it"** — hard delete from `user_interests`.

### Backoff rules

- After any response: don't ask about this interest again for 90 days
- No response after 14 days: treat as "still interested", back off 90 days
- Maximum 3 asks total per interest (`times_asked` column), then stop asking permanently and respect the original declaration
- Track via `last_asked_at` and `times_asked` on the `user_interests` row

### Seasonal patterns

The pause mechanism handles seasonality naturally:
- Sports offseasons: user pauses "NFL" for 4 months, auto-reactivates before the new season
- Gardening: user pauses in November, sets resume for March
- Holiday interests: pause after December, resume next November

On reactivation, the system posts a welcome-back card and resets confidence to 1.0.

---

## 6. Files to Modify

### Backend (new)
- `internal/database/migrations/NNN_user_profile.sql` — schema migration
- `internal/repository/user_interest_repo.go` — CRUD for `user_interests`
- `internal/repository/user_lifestyle_repo.go` — CRUD for `user_lifestyle_tags`
- `internal/repository/user_content_prefs_repo.go` — CRUD for `user_content_prefs`
- `internal/handler/profile.go` — `GET/PUT /user/profile`, interest/lifestyle/prefs endpoints
- `internal/interest/worker.go` — background interest inference worker

### Backend (modify)
- `internal/model/model.go` — extend `User` struct, add `UserInterest`, `LifestyleTag`, `ContentPref` models
- `internal/repository/user_repo.go` — add profile field reads/writes to existing user queries
- `internal/handler/onboarding.go` — update `POST /user/interests` to write plaintext interests alongside embeddings
- `cmd/server/main.go` — register new routes, start interest worker

### iOS (new)
- `Models/UserProfile.swift` — `UserProfile`, `UserInterest`, `LifestyleTag`, `ContentPref` structs
- `Views/Onboarding/` — onboarding flow views (7 steps)
- `Views/ProfileView.swift` — profile display and editing screen

### iOS (modify)
- `Services/APIService.swift` — add profile fetch/update methods
- `Services/AuthService.swift` — trigger profile check after sign-in
- `Views/FeedListView.swift` — gate on `profile_initialized`, show onboarding if false

### Skills (modify)
- `_shared/CONTEXT_BOOTSTRAP.md` — add `GET /user/profile` as 5th parallel fetch
- `beepbopboop-post/MODE_BATCH.md` — read profile interests/lifestyle/prefs in BT1
- `beepbopboop-post/BASE_LOCAL.md` — read profile for contextual depth
- `beepbopboop-post/SKILL.md` — pin profile into working memory in Step 0
