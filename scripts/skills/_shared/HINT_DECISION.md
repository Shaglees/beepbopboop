# Shared: picking the right `display_hint`

`display_hint` determines which iOS card the post renders as. Picking the wrong hint produces a technically-valid post that looks broken (e.g. a fabricated date badge on an evergreen recommendation, or a "place" with an invisible CTA).

**Always** call `GET /posts/hints` at the start of a session and consult the `renders` block on each candidate hint before picking. That block tells you what the client will actually draw. If you skip that step, use this decision tree.

## Decision tree

```
Does the post have a real, specific date/time?
├── Yes → Is the user going there repeatedly on a schedule?
│   ├── Yes → display_hint: "calendar"  (set scheduled_at; ALSO include a date token in body)
│   └── No  → display_hint: "event"     (set scheduled_at; ALSO include a date token in body)
└── No → Is the post about a specific location (trail, cafe, venue, park)?
    ├── Yes → display_hint: "place"  (booking/info URL in external_url renders as a CTA)
    └── No → Is the post linking out to an off-platform article/news story?
        ├── Yes → display_hint: "article"  (external_url required)
        └── No  → display_hint: "card"      (generic discovery)
```

### Date handling for event / calendar / concert

The iOS client reads `scheduled_at` **first**, then falls back to extracting a date from the title or body. That means:

- `scheduled_at` alone is enough for the badge to render correctly.
- If you don't have a machine-readable timestamp, put a human-readable date in the body (e.g. `"Saturday May 10 at 2pm"`) — the client will pull it out via `NSDataDetector`.
- Doing both is best: the skill can't tell which deployment of the iOS client the viewer is on, and older builds that predate scheduled_at will only honor body text.


## Red-flag rules (automatic wrong-hint detection)

These are the exact bugs we shipped on 2026-04-20. If any of these are true, pick a different hint:

| Red flag | Wrong hint | Right hint |
|---|---|---|
| Evergreen trail / cafe / recommendation with no date | `event` (client will fabricate a date badge from createdAt) | `place` |
| Place post whose main CTA is a booking URL | `place` with `external_url` set (silently dropped by PlaceCard) | `place` with URL **inlined in body text**, or `restaurant` / `destination` which render CTAs |
| News article with title+source | `card` | `article` (required external_url, proper source display) |
| Dated concert/festival | `event` (generic) | `concert` (structured — `external_url` JSON with `artist`, `venue`, `date` is **required**; `scheduled_at` alone is not enough) |
| Recipe / how-to with no CTA | `article` (requires external_url you don't have) | `card` |
| Stats/standings post | `article` | `standings` (structured) |
| A vs B / ranked list with named items | `comparison` without `external_url` JSON (iOS shows StandardCard fallback) | `comparison` **with** `external_url: {"title":"...","items":[{"name":"...","verdict":"..."}]}` — or fall back to `article` if you can't produce the JSON |
| matchup post using `gameTime` key | `matchup` with `gameTime` in external_url | `matchup` with `date` (ISO string) — key must be `date` not `gameTime` |
| matchup post without `league` | `matchup` missing `league` key | Add `"league": "NBA"` (or NFL / MLB / NHL) — it is required |
| restaurant post without `cuisine` | `restaurant` external_url missing `cuisine` | Add `"cuisine": "..."` — required alongside `name` |
| entertainment post with `subject`/`headline` | `entertainment` with wrong keys | Use `title` + `type` only — not `subject`, `headline`, `category`, or `tags` |
| fitness post with `activity`/`intensity` | `fitness` with wrong keys | Use `title` + `type` only — not `activity`, `intensity`, `duration_min`, or `notes` |
| destination post with `city` key | `destination` with `city` | Use `name` — not `city`, `location`, or `place` |

## Structured hints — required `external_url` JSON schemas

These hints require `external_url` to be a **JSON string** (not a URL). The iOS client parses it to render the structured card. If the keys are wrong or missing, the card silently falls back to a plain StandardCard.

**Before publishing any structured hint:** verify your `external_url` JSON contains ALL the required keys listed below. If you cannot produce all required keys, use the listed fallback hint instead.

---

### Sports

**`matchup`** — upcoming or live game preview
```json
{
  "sport": "basketball",
  "league": "NBA",
  "date": "2026-04-26",
  "home": {"name": "Thunder", "abbr": "OKC"},
  "away": {"name": "Mavericks", "abbr": "DAL"}
}
```
> ⚠️ **The field is called `date`, NOT `gameTime`**. If you are thinking "gameTime", put that value in the `date` field instead. `gameTime` is NOT a valid key and will cause the card to render as a plain StandardCard with no game info. `league` is required (NBA / NFL / MLB / NHL / etc.). Fallback: `event`

**`scoreboard`** — final or in-progress game score
```json
{
  "sport": "basketball",
  "league": "NBA",
  "status": "Final",
  "home": {"name": "Thunder", "score": 112, "abbr": "OKC"},
  "away": {"name": "Mavericks", "score": 104, "abbr": "DAL"}
}
```
> Fallback: `card`

**`standings`** — league standings table
```json
{
  "league": "NBA",
  "season": "2025-26",
  "teams": [{"rank": 1, "name": "Thunder", "wins": 68, "losses": 14}]
}
```
> Fallback: `card`

**`box_score`** — detailed game stats
```json
{
  "sport": "basketball",
  "teams": ["Thunder", "Mavericks"],
  "quarters": [28, 31, 27, 26]
}
```
> Fallback: `card`

**`player_spotlight`** — player feature
```json
{
  "name": "Shai Gilgeous-Alexander",
  "team": "OKC Thunder",
  "position": "Guard",
  "stats": {"points": 32.7, "assists": 6.4}
}
```
> Fallback: `card`

---

### Entertainment / Music

**`concert`** — live music event
```json
{
  "artist": "Billie Eilish",
  "venue": "Moody Center",
  "date": "2026-05-03",
  "ticketUrl": "https://..."
}
```
> ⚠️ `external_url` JSON is REQUIRED — `scheduled_at` alone is not enough. Without the JSON, the artist name and venue will NOT appear on the card. Fallback: `event`

**`album`** — music release
```json
{
  "title": "HIT ME HARD AND SOFT",
  "artist": "Billie Eilish"
}
```
> Fallback: `card`

**`movie`** — film feature
```json
{
  "title": "Mission: Impossible — The Final Reckoning"
}
```
> Fallback: `article`

**`show`** — TV show feature
```json
{
  "title": "Severance"
}
```
> Fallback: `article`

**`game_release`** — upcoming game
```json
{
  "title": "Grand Theft Auto VI",
  "platform": "PS5 / Xbox Series X"
}
```
> Fallback: `card`

**`game_review`** — game review
```json
{
  "title": "Grand Theft Auto VI",
  "score": 9.5
}
```
> Fallback: `card`

---

### Location + Metadata

**`restaurant`** — restaurant or food venue
```json
{
  "name": "Franklin Barbecue",
  "cuisine": "BBQ / Central Texas"
}
```
> ⚠️ `cuisine` is required — do NOT omit it. `latitude`/`longitude` are optional extras. Fallback: `place`

**`destination`** — travel destination
```json
{
  "name": "Banff National Park",
  "country": "Canada"
}
```
> ⚠️ Key is `name`, NOT `city`, `location`, or `place`. Fallback: `place`

**`pet_spotlight`** — pet feature / adoption
```json
{
  "name": "Biscuit",
  "type": "dog"
}
```
> Fallback: `card`

---

### Lifestyle

**`entertainment`** — general entertainment content (non-music, non-film)
```json
{
  "title": "Austin Bandcamp Friday Picks",
  "type": "music"
}
```
> ⚠️ Keys are `title` and `type`. Do NOT use `subject`, `headline`, `category`, or `tags`. Valid `type` values: `music`, `film`, `tv`, `podcast`, `event`, `other`. Fallback: `card`

**`fitness`** — fitness or health content
```json
{
  "title": "5K Training Plan — Week 4",
  "type": "run"
}
```
> ⚠️ Keys are `title` and `type`. Do NOT use `activity`, `intensity`, `duration_min`, or `notes`. Valid `type` values: `run`, `workout`, `yoga`, `cycling`, `swim`, `other`. Fallback: `card`

**`weather`** — weather update
```json
{
  "location": "Austin, TX",
  "temp_f": 84,
  "condition": "Sunny"
}
```
> Fallback: `card`

**`deal`** — sale or offer
```json
{
  "title": "REI Anniversary Sale",
  "original_price": "$189",
  "sale_price": "$119"
}
```
> Fallback: `card`

---

### People / Community

**`creator_spotlight`** — creator or account feature
```json
{
  "name": "Austin Eastciders"
}
```
> Fallback: `card`

**`comparison`** — ranked list of 3 or more named items

> ⚠️ **NEVER use `comparison` for head-to-head between just 2 subjects** (e.g. "Asahi Linux vs Ubuntu", "iOS vs Android", "Austin vs Dallas"). Two-subject analysis → use `article`. `comparison` is ONLY for ranked lists of **3 or more specific named options** where each gets a verdict.
>
> | Content | Correct hint |
> |---|---|
> | "5 best BBQ spots in Austin, ranked" | `comparison` |
> | "Asahi Linux vs Ubuntu — which to install" | `article` |
> | "Top 4 running trails near downtown Austin" | `comparison` |
> | "iOS vs Android for privacy" | `article` |

```json
{
  "title": "Austin's 5 best BBQ spots, ranked",
  "items": [
    {"name": "Franklin Barbecue", "verdict": "Best brisket"},
    {"name": "La Barbecue", "verdict": "Best beef rib"},
    {"name": "Micklethwait", "verdict": "Best sides"}
  ]
}
```
> Each item requires `name` and `verdict`. Minimum 3 items. Fallback: `article`

## Lint feedback loop

The server lint warns when an event hint will render without a visible date:

- `event_without_date` — you picked `event`/`calendar`/`concert` without a `scheduled_at` and without a date token in the title or body. Move to `place`/`card` or add the date.

Warnings do not block publish — your post still ships — but treat them as errors in skill code.

## One-line check before publishing

> _"If I look at this post as a feed card, does the card type match the content shape?"_

Pull up the `renders.card` value from `/posts/hints` for your chosen hint. If the answer is "no", go back to the decision tree.
