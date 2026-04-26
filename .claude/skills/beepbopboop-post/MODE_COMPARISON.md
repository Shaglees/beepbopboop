# Comparison mode (CP1–CP3)

**Trigger:** `compare ...`, `best ... ranked`, `top N ... in`, `vs`, or from batch Phase 2.

---

## Required `external_url` JSON — copy and fill this in

The `comparison` hint **requires** a structured JSON string in `external_url`. Without it, iOS falls back to a plain StandardCard and the comparison layout is never shown.

```json
{
  "title": "Austin's 5 best BBQ spots, ranked",
  "items": [
    { "name": "Franklin Barbecue", "verdict": "Best brisket in Texas", "detail": "$$ · 900 E 11th St" },
    { "name": "La Barbecue", "verdict": "Best beef rib", "detail": "$$ · 2027 E Cesar Chavez" },
    { "name": "Micklethwait Craft Meats", "verdict": "Best sides", "detail": "$$ · 1309 Rosewood Ave" }
  ]
}
```

**Required keys:**
- `title` — short heading for the comparison card
- `items` — array, each with at minimum `name` (string) and `verdict` (string)

**Optional per-item keys:** `detail`, `score`, `price`, `address`, `image_url`

**Always attempt the comparison hint.** After researching the options in CP2, you will have names, specialties and prices for each place — that is everything needed to fill in `items`. Use `display_hint: article` only if the comparison is too abstract to produce named items (e.g. "compare Austin vs. Dallas as a city").

---

## CP1: Parse comparison subject

Extract:
- **Subject:** what to compare (e.g., "coffee roasters", "pizza places")
- **Location:** optional override (default: `BEEPBOPBOOP_DEFAULT_LOCATION`)

Geocode the location using `BASE_LOCAL.md` Step 1.

## CP2: Research options

1. POI discovery (`BASE_LOCAL.md` Step 2) with larger radius (3000m) and limit (10).
2. Research the top 5 POIs: reviews, specialties, prices, hours via WebSearch + WebFetch.
3. Cross-reference with `WebSearch "best <SUBJECT> <LOCATION> <YEAR>"` for local rankings.

## CP3: Generate comparison post

Generate **1 discovery post** with a ranking/comparison format:

- Title signals a curated ranking: "<LOCATION>'s 5 best <SUBJECT>, ranked by someone who's tried them all"
- Body names specific places, what they're best at, prices where available, one-line verdicts
- `post_type: "discovery"`
- **`display_hint: "comparison"`** with the required `external_url` JSON from the STOP block above

Then proceed to `COMMON_PUBLISH.md`.

### Example

> **Title:** "Victoria's 5 best coffee roasters, ranked by someone who's tried them all"
> **Body:** "Bows & Arrows on Fort Street wins on single-origin range — their Ethiopian Yirgacheffe is worth the $6. Discovery Coffee on Government is the safe pick with the best pastry selection. Habit on Pandora does the best cortado in town at $4.50."
> **external_url:** `{"title":"Victoria coffee roasters ranked","items":[{"name":"Bows & Arrows","verdict":"Best single-origin","detail":"$6"},{"name":"Discovery Coffee","verdict":"Best pastry","detail":"$5"},{"name":"Habit Coffee","verdict":"Best cortado","detail":"$4.50"}]}`
