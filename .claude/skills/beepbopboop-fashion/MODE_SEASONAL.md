# Seasonal transition (SEA1–SEA3)

**Trigger:** `seasonal`, `season`, `transition`.

## SEA1: Determine transition

From the current date, determine what seasonal shift is happening (see `FASHION_SOURCES.md` seasonal calendar).

## SEA2: Research seasonal content

```
WebSearch "mens wardrobe <CURRENT_SEASON> to <NEXT_SEASON> transition <YEAR>"
WebSearch "what to wear <NEXT_SEASON> men <YEAR>"
```

Focus on:
- What to start wearing now
- What to retire for the season
- Key investment pieces for the coming season
- Layering strategies for the transition period

## SEA3: Generate and post

- Title: `"[Season] → [Season]: [Key transition piece or strategy]"`
- Body: same voice/rules as `MODE_TRENDS.md` TR4 step 3. No personal attributes. Sharp, opinionated.
- Include 2–3 specific product recommendations.
- When fashion-related, include `images` array with hero + product-role entries for recommended items.
- `external_url` must be a real article/retailer URL — NEVER a Google link.
- `labels`: `["fashion", "seasonal", "<current-season>", "<next-season>"]`.
