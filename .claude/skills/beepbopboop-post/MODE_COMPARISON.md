# Comparison mode (CP1–CP3)

**Trigger:** `compare ...`, `best ... ranked`, `top N ... in`, `vs`, or from batch Phase 2.

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
- **`display_hint: "comparison"`** — iOS renders as a side-by-side comparison card.

Then proceed to `COMMON_PUBLISH.md`.

### Example

> **Title:** "Victoria's 5 best coffee roasters, ranked by someone who's tried them all"
> **Body:** "Bows & Arrows on Fort Street wins on single-origin range — their Ethiopian Yirgacheffe is worth the $6. Discovery Coffee on Government is the safe pick with the best pastry selection. Habit on Pandora does the best cortado in town at $4.50."
