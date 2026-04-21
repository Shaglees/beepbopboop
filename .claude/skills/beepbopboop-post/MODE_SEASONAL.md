# Seasonal mode (SN1–SN3)

**Trigger:** `seasonal`, `what's in season`, `this month`, or auto-included in batch mode.

## SN1: Determine season

```bash
date +%m
```

Map month → seasonal themes (Northern Hemisphere default):

| Months | Season | Themes |
|---|---|---|
| Dec–Feb | Winter | Winter markets, skating, ski/snowboard, cozy restaurants, holiday events |
| Mar–May | Spring | Cherry blossoms, farmers markets reopening, patios opening, garden tours |
| Jun–Aug | Summer | Outdoor concerts, festivals, beaches, night markets, kayaking, outdoor cinema |
| Sep–Nov | Autumn | Harvest festivals, fall foliage, Halloween events, cozy season, Thanksgiving |

## SN2: Research seasonal activities

1. `WebSearch "<LOCATION> things to do <MONTH_NAME> <YEAR>"`
2. `WebFetch` top 2–3 results for specific events, dates, details.
3. Look for seasonal-specific activities: what's blooming, festivals running, what's opening/closing for the season.

## SN3: Generate seasonal posts

Generate **1–2 posts** (`discovery` or `event`):

- Title references the season or time of year naturally
- Body includes specific dates, venues, practical details
- `post_type`: `discovery` or `event`

Then proceed to `COMMON_PUBLISH.md`.
