# Wave 4: New Features — Design Spec

**Goal:** Ship community local news, fashion try-on with user photos, and interest-driven calendar events as feed content.

**Issues:** #189 (community local news skill), #190 (local news rendering), #191 (fashion try-on panel), #156 (interest-driven calendar layer)

**Follow-up issues (server-load reduction):** #222 (skill-side materialization), #223 (webhook/push ingest), #224 (client-side event rendering)

**Architecture:** Three independent sub-systems sharing the existing feed, hint catalog, and publish pipeline. Local news gets a new skill + adaptive card. Fashion try-on extends the existing skill + card. Calendar events use a server worker with Go templates and existing card types.

**Implementation order:** A (Local News) → B (Fashion Try-On) → C (Interest Calendar)

---

## Sub-system A: Community Local News (#189, #190)

### A1. Backend — News Source Registry

#### `news_sources` table

```sql
CREATE TABLE news_sources (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    url          TEXT NOT NULL UNIQUE,
    feed_url     TEXT,
    area_label   TEXT NOT NULL,
    latitude     DOUBLE PRECISION NOT NULL,
    longitude    DOUBLE PRECISION NOT NULL,
    radius_km    DOUBLE PRECISION NOT NULL DEFAULT 25.0,
    topics       TEXT[] NOT NULL DEFAULT '{}',
    trust_score  SMALLINT NOT NULL DEFAULT 50,
    fetch_method TEXT NOT NULL DEFAULT 'rss',
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_news_sources_geo ON news_sources (latitude, longitude);
CREATE INDEX idx_news_sources_active ON news_sources (active) WHERE active = TRUE;
```

Fields:
- `area_label`: human-readable display label ("Dublin, Ireland") — not used for matching
- `latitude/longitude`: centroid of the source's coverage area — all geo queries use coordinates
- `radius_km`: coverage radius. Query matches when distance(source, query) < source.radius_km + query.radius_km
- `topics`: Postgres text array for topic filtering
- `trust_score`: 0–100. Sources below 30 are excluded from automatic fetching. Manual or crowd-sourced.
- `fetch_method`: `rss` | `scrape` | `api` — tells the skill how to fetch content

#### Repository: `NewsSourceRepo`

- `List(lat, lon, radiusKm float64, topics []string) ([]NewsSource, error)` — Haversine distance filter + optional topic intersection
- `Create(src NewsSource) error` — insert with conflict on URL (upsert)
- `Get(id string) (*NewsSource, error)`

#### Endpoints (agent-auth)

- `GET /news-sources?lat=53.35&lon=-6.26&radius_km=50&topics=sports` — list nearby sources
- `POST /news-sources` — register/update a source (skill discovery flow)
- `GET /news-sources/{id}` — get source details

### A2. Display Hint — `local_news`

New entry in `hints.go`:

```go
{
    Hint:           "local_news",
    PostType:       "article",
    StructuredJSON:  true,
    RequiredFields: []string{
        "external_url:content_kind",
        "external_url:source_name",
        "external_url:source_url",
    },
    Example: `{
        "content_kind": "article",
        "source_name": "Dublin Inquirer",
        "source_url": "https://dublininquirer.com",
        "source_logo_url": "https://example.com/logo.png",
        "thumbnail_url": "https://example.com/thumb.jpg",
        "article_url": "https://dublininquirer.com/2026/04/25/housing-report",
        "embed_url": null,
        "duration_seconds": null,
        "locality": "Dublin, Ireland",
        "published_at": "2026-04-25T10:00:00Z"
    }`,
    Renders: CardRender{
        Card:          "LocalNewsCard",
        UsesFields:    []string{"title", "body", "external_url", "images"},
        IgnoresFields: []string{},
    },
    PickWhen:  "Content from a local publication, community news source, or local video segment",
    AvoidWhen: "National/international news without a clear local source",
}
```

`content_kind` values:
- `article` — text article with optional thumbnail
- `video` — video segment with embed/watch URL and duration
- `hybrid` — article with an embedded video component

### A3. Skill — `beepbopboop-local-news`

New skill at `.claude/skills/beepbopboop-local-news/SKILL.md`.

#### Modes

| Mode | Trigger | Behavior |
|---|---|---|
| `discover` | "find local news sources" | Web search for publications near user location, propose via `POST /news-sources` |
| `fetch` | "local news" / batch dispatch | Query registry, fetch RSS/scrape, score by recency + topic relevance, compose top 3–5 posts |
| `video` | "local video news" | Same as fetch but filtered to video content (YouTube local channels, publication video segments) |

#### Fetch flow

1. Load config (`_shared/CONFIG.md`) — need `BEEPBOPBOOP_HOME_LAT/LON`
2. `GET /news-sources?lat=...&lon=...&radius_km=50` to get nearby active sources
3. For each source: fetch RSS feed (if `feed_url` set) or scrape homepage
4. Normalize each item: title, summary, URL, thumbnail, published_at, content_kind
5. Score by: recency (last 48h preferred), topic overlap with user interests, source trust_score
6. Top 3–5 → compose `local_news` posts with structured JSON external_url
7. Lint → dedup → publish via `_shared/PUBLISH_ENVELOPE.md`

#### Batch integration

`MODE_BATCH.md` gains a routing entry: when spread targets allocate a "news" slot, batch can delegate to `beepbopboop-local-news` for local content. The batch planner decides the split between local news and national/interest news from `beepbopboop-news`.

### A4. iOS — `LocalNewsCard`

New file: `beepbopboop/Views/LocalNewsCards.swift`

Single adaptive card view that switches layout based on `content_kind` from the structured JSON:

**Article layout:**
- Thumbnail (leading, 80x80 rounded) or top hero image if large
- Source badge: logo + name + locality pill
- Headline (title, 2 lines max)
- Body preview (2 lines)
- Published timestamp
- Tap → opens `article_url` in Safari/in-app browser

**Video layout:**
- Large thumbnail (full width, 16:9 aspect) with play button overlay
- Duration badge (bottom-right of thumbnail, e.g. "3:42")
- Source badge below thumbnail
- Headline
- Tap → opens `embed_url` in WebView or `watch_url` in Safari

**Hybrid layout:**
- Article layout at top
- Secondary video thumbnail row below body (smaller, with play icon + duration)
- Two tap targets: article area → article_url, video area → embed_url

**All variants show:**
- Trust indicator: subtle green dot for `trust_score > 70`, nothing otherwise. The skill includes `"trust_score": 85` in the `external_url` JSON — the card reads it directly.

#### FeedItemView routing

Add `case .localNews:` to the display hint switch in `FeedItemView.swift`, routing to `LocalNewsCard`.

---

## Sub-system B: Fashion Try-On (#191)

### B1. Backend — User Photo Storage

#### Schema changes on `users` table

```sql
ALTER TABLE users ADD COLUMN headshot_data BYTEA;
ALTER TABLE users ADD COLUMN headshot_type TEXT;
ALTER TABLE users ADD COLUMN bodyshot_data BYTEA;
ALTER TABLE users ADD COLUMN bodyshot_type TEXT;
```

- `headshot_data`: 360x360 JPEG, ~50–150KB
- `bodyshot_data`: 360x720 JPEG, ~80–200KB
- `*_type`: MIME type, always `image/jpeg` after server-side conversion
- Nullable — user hasn't uploaded yet
- Deleted when user row is deleted — no separate cleanup

#### Repository: `UserPhotoRepo`

- `SaveHeadshot(userID string, data []byte, contentType string) error` — resize to 360x360, convert to JPEG, store
- `SaveBodyshot(userID string, data []byte, contentType string) error` — resize to 360x720, convert to JPEG, store
- `GetHeadshot(userID string) ([]byte, string, error)` — returns (data, contentType, error)
- `GetBodyshot(userID string) ([]byte, string, error)`
- `DeletePhoto(userID string, photoType string) error` — nulls the relevant columns

Server-side resize uses Go's `image` stdlib + `golang.org/x/image/draw` for lanczos resampling.

#### Endpoints

**Firebase-auth (mobile client):**
- `PUT /user/photos/headshot` — multipart form upload, server resizes + stores
- `PUT /user/photos/bodyshot` — multipart form upload, server resizes + stores
- `GET /user/photos/headshot` — returns raw image bytes with `Content-Type` header
- `GET /user/photos/bodyshot` — returns raw image bytes
- `DELETE /user/photos/{type}` — nulls the column (`type` is `headshot` or `bodyshot`)

**Agent-auth (skill access, scoped to agent's owner):**
- `GET /user/photos/headshot` — read-only, for try-on generation
- `GET /user/photos/bodyshot` — read-only, for try-on generation

No agent-auth write/delete access to photos.

### B2. Skill — `MODE_TRYON.md`

New mode file in existing `beepbopboop-fashion` skill.

#### Flow

1. Check if user has a bodyshot: `GET /user/photos/bodyshot` — if 404, skip try-on and fall back to standard outfit mode
2. Fetch current fashion trends (reuse existing fashion skill trend fetching)
3. Read user's fashion preferences from config (`BEEPBOPBOOP_FASHION_STYLES`, `BEEPBOPBOOP_FASHION_BUDGET`, `BEEPBOPBOOP_FASHION_BRANDS`)
4. Compose outfit description matching preferences + trends
5. Call OpenAI image-2 API:
   - Input: user's bodyshot as reference image + text prompt describing the outfit
   - Output: AI-generated image of the outfit on a figure resembling the user
6. Upload generated image to Imgur via existing image pipeline (`beepbopboop-images`)
7. Compose post with `display_hint: "outfit"` and add `"image_variant": "tryon"` to the structured JSON
8. Lint → publish

#### Graceful degradation

- No bodyshot → skip try-on, generate standard outfit post
- OpenAI API failure → skip try-on, generate standard outfit post with text description
- Imgur upload failure → include OpenAI image URL directly (temporary, may expire)

### B3. iOS — Photo Upload UI & Card Changes

#### Settings: "My Photos" section

New section in `SettingsView.swift` (or `ProfileView.swift`), below Content Mix:

- Two upload slots: "Headshot" (360x360) and "Full Body" (360x720)
- Each slot shows: current photo thumbnail (if uploaded), "Upload" button (camera/library picker), "Remove" button
- Upload flow: `UIImagePickerController` → crop to target aspect ratio → `PUT /user/photos/{type}`
- Privacy note: small text "Photos are used for AI outfit previews and stored on your account. Delete anytime."

#### `OutfitCard` extension

When `image_variant` in the structured JSON equals `"tryon"`:
- Overlay a subtle label on the image: "AI try-on preview" (semi-transparent pill, bottom-left)
- No other layout changes — the card renders identically otherwise

---

## Sub-system C: Interest Calendar (#156)

### C1. Backend — Event Storage

#### `interest_calendar_events` table

```sql
CREATE TABLE interest_calendar_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_key     TEXT NOT NULL UNIQUE,
    domain        TEXT NOT NULL,
    title         TEXT NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ,
    timezone      TEXT NOT NULL DEFAULT 'UTC',
    status        TEXT NOT NULL DEFAULT 'scheduled',
    entity_type   TEXT NOT NULL,
    entity_ids    JSONB NOT NULL DEFAULT '{}',
    interest_tags TEXT[] NOT NULL DEFAULT '{}',
    payload       JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ice_domain_start ON interest_calendar_events (domain, start_time);
CREATE INDEX idx_ice_status ON interest_calendar_events (status) WHERE status = 'scheduled';
CREATE INDEX idx_ice_tags ON interest_calendar_events USING GIN (interest_tags);
```

Fields:
- `event_key`: idempotency key — `espn:event:401234567`, `tmdb:movie:999:release:US`. Re-ingesting updates, never duplicates.
- `domain`: `sports` | `entertainment`. Extensible to `music`, `gaming`, etc. later.
- `status`: `scheduled` → `live` → `final` | `cancelled`
- `entity_type`: `game` | `movie_release` | `tv_premiere`
- `entity_ids`: structured identifiers — `{"home_team":"lakers","away_team":"celtics","league":"nba","espn_event_id":"401234567"}` or `{"tmdb_id":999,"type":"movie"}`
- `interest_tags`: tags for user matching — `{"basketball","nba","lakers","celtics"}` or `{"action","sci-fi","mission-impossible"}`
- `payload`: full structured data for template rendering — team records, venue, broadcast, poster URL, cast, etc.

#### `calendar_post_log` table

```sql
CREATE TABLE calendar_post_log (
    event_key  TEXT NOT NULL,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    window     TEXT NOT NULL,
    post_id    UUID NOT NULL REFERENCES posts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (event_key, user_id, window)
);
```

Tracks which (event, user, window) combos have been published. Prevents duplicate posts.

#### Repository: `CalendarEventRepo`

- `Upsert(event CalendarEvent) error` — insert or update by event_key
- `Upcoming(domain string, from, to time.Time) ([]CalendarEvent, error)` — events in time range
- `ForUser(userID string, interests []string, from, to time.Time) ([]CalendarEvent, error)` — events matching user interests in time range
- `LogPost(eventKey, userID, window, postID string) error`
- `IsPublished(eventKey, userID, window string) (bool, error)`

### C2. Ingest Workers

#### Sports ingest (extend existing `sports.Worker`)

The existing sports worker polls ESPN every 10 minutes for live scores. Extend it to also write upcoming games to `interest_calendar_events`:

- On each poll, for games in `scheduled` status with `start_time` in the next 7 days:
  - Upsert to `interest_calendar_events` with `domain: "sports"`, `entity_type: "game"`
  - `interest_tags` derived from team slugs + league name
  - `payload`: home/away team names, abbreviations, records, colors, venue, broadcast info
- On status change (scheduled → live → final → cancelled): update the row

#### Entertainment ingest (new `entertainment.Worker`)

New worker, runs once daily (TMDB data changes slowly):

- `GET /movie/upcoming` from TMDB API — upcoming theatrical releases for user's region
- `GET /tv/on_the_air` — current and upcoming TV premieres
- For each item:
  - Upsert to `interest_calendar_events` with `domain: "entertainment"`, `entity_type: "movie_release"` or `"tv_premiere"`
  - `event_key`: `tmdb:movie:{id}:release:{region}` or `tmdb:tv:{id}:premiere:{region}`
  - `interest_tags`: derived from genres + franchise keywords
  - `payload`: title, poster URL, genres, runtime, cast top 3, overview, rating

Required config: `TMDB_KEY` (already in config spec as optional).

### C3. Materialization Worker

New `calendar.MaterializeWorker`, runs every 15 minutes.

#### Flow

For each user with declared interests:

1. Query upcoming events matching user's interests: `CalendarEventRepo.ForUser(userID, interests, now, now+24h)`
2. For each matched event, check publish windows:
   - **Sports preview**: T-24h to T-12h before `start_time`
   - **Sports imminent**: T-2h to T-0
   - **Entertainment preview**: T-7d to T-3d
   - **Entertainment release day**: T-24h to T+24h
3. For each applicable window: check `IsPublished(eventKey, userID, window)` — skip if already published
4. Select Go template by `(domain, entity_type, window)`, fill from event `payload`
5. Create post via `PostRepo.Create()` (direct, no HTTP round-trip):
   - Sports preview/imminent → `display_hint: "matchup"`, structured JSON from payload
   - Entertainment preview/release → `display_hint: "event"`, structured JSON from payload
6. Log to `calendar_post_log`

#### Templates

Embedded Go templates in the worker package:

**Sports preview:**
- Title: `{Away} @ {Home} — {StartTime formatted}`
- Body: `{Home} ({HomeRecord}) host {Away} ({AwayRecord}) at {Venue}. {BroadcastInfo}.`

**Sports imminent:**
- Title: `{Away} @ {Home} tips off in {TimeUntil}`
- Body: `{Home} ({HomeRecord}) vs {Away} ({AwayRecord}). {SeriesContext if applicable}.`

**Entertainment preview:**
- Title: `{Title} hits theaters {ReleaseDay}`
- Body: `{Overview, truncated to 2 sentences}. Starring {Cast top 3}. {Runtime}min, rated {Rating}.`

**Entertainment release day:**
- Title: `{Title} is out today`
- Body: `{Overview, truncated to 2 sentences}. Now playing at theaters near you.`

### C4. User Matching

Events are matched to users via:
1. User's declared interests (from `user_profiles.interests_declared`) intersected with event's `interest_tags`
2. For sports: user's followed teams (from `user_settings.followed_teams`) matched against `entity_ids` team slugs
3. At least one tag must match — no match, no post for that user

### C5. No New iOS Changes

Calendar posts use existing display hints:
- Sports → `matchup` → renders `MatchupCard` (already exists)
- Entertainment → `event` → renders `DateCard` (already exists)

No new card views, no new feed surfaces. Calendar events appear as time-sensitive cards in the existing feed.

---

## Testing Strategy

### Backend Tests

**News sources:**
- `TestNewsSourceRepo_ListByRadius` — sources within/outside radius
- `TestNewsSourceRepo_ListByTopics` — topic filtering
- `TestNewsSourceHandler_Create` — register new source
- `TestLocalNewsHint_LintValidPayload` — lint accepts all three content_kinds

**User photos:**
- `TestUserPhotoRepo_SaveAndGet` — round-trip headshot/bodyshot
- `TestUserPhotoRepo_Delete` — nulls columns
- `TestUserPhotoHandler_Upload` — multipart upload with resize
- `TestUserPhotoHandler_AgentReadOnly` — agent-auth can GET but not PUT/DELETE

**Calendar events:**
- `TestCalendarEventRepo_Upsert` — idempotent by event_key
- `TestCalendarEventRepo_ForUser` — interest tag matching
- `TestCalendarPostLog_Dedup` — same event+user+window not published twice
- `TestMaterializeWorker_SportsPreview` — publishes matchup post in preview window
- `TestMaterializeWorker_SkipsPublished` — no duplicate posts
- `TestEntertainmentWorker_IngestTMDB` — parses TMDB response, creates events

### iOS Tests

- `LocalNewsCard` snapshot tests for article/video/hybrid layouts
- `SpreadTargets` decoding (already exists from Wave 3)
- Photo upload flow (manual test — picker + upload + display)

### Skill Tests

- `beepbopboop-local-news` fetch mode produces lint-clean `local_news` payloads (manual test against local backend)
- `MODE_TRYON.md` gracefully degrades when no bodyshot exists (manual test)

---

## Migration & Defaults

- `news_sources` table: new table, no migration. Skills populate it via `POST /news-sources` during discovery runs.
- User photos: new nullable columns on `users`. No default data.
- `interest_calendar_events` + `calendar_post_log`: new tables. Workers populate them on first run.
- `local_news` hint: added to `hints.go` catalog. Existing posts unaffected.
- `TMDB_KEY`: optional config key, already documented. Entertainment ingest worker skipped if missing.

---

## Future Improvements (tracked as issues)

- **#222**: Move calendar materialization to skill-side — server becomes thin data store, skills compose posts during batch runs
- **#223**: Replace polling ingest with webhook/conditional requests — TMDB ETags, sports schedule-aware polling
- **#224**: Client-side event rendering — events as ephemeral feed items, never written to posts table
