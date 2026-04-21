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
| Dated concert/festival | `event` (generic) | `concert` (structured with artist info) |
| Recipe / how-to with no CTA | `article` (requires external_url you don't have) | `card` |
| Stats/standings post | `article` | `standings` (structured) |

## Structured hints (`structured_json: true`)

These hints store JSON in `external_url`, not a URL. The server validates the shape. If you're not sure you can produce the right structure, fall back to the simpler hint (`article`, `card`, `place`) instead of guessing.

- Sports: `scoreboard`, `matchup`, `standings`, `box_score`, `player_spotlight`
- Content: `movie`, `show`, `album`, `concert`, `game_release`, `game_review`
- Location+metadata: `restaurant`, `destination`, `pet_spotlight`
- Other: `weather`, `entertainment`, `science`, `fitness`, `feedback`, `creator_spotlight`, `video_embed`

For each, `GET /posts/hints` returns a lint-clean example you can copy and mutate. Always start from that example.

## Lint feedback loop

The server lint warns when an event hint will render without a visible date:

- `event_without_date` — you picked `event`/`calendar`/`concert` without a `scheduled_at` and without a date token in the title or body. Move to `place`/`card` or add the date.

Warnings do not block publish — your post still ships — but treat them as errors in skill code.

## One-line check before publishing

> _"If I look at this post as a feed card, does the card type match the content shape?"_

Pull up the `renders.card` value from `/posts/hints` for your chosen hint. If the answer is "no", go back to the decision tree.
