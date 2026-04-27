# Drops & new releases (DR1–DR3)

**Trigger:** `drops`, `new releases`, `collabs`.

## DR1: Scan for drops

```
WebSearch "new fashion drops this week <MONTH> <YEAR>"
WebSearch "<FASHION_BRANDS> new release <MONTH> <YEAR>"
WebSearch "sneaker drops this week <MONTH> <YEAR>"
```

Brand-specific sources:

```
WebFetch "https://hypebeast.com/fashion"  → extract drops/releases
WebFetch "https://www.highsnobiety.com/fashion/"  → extract drops
```

## DR2: Filter by relevance

Select drops matching the user's:
- `FASHION_BRANDS` (direct match = always include)
- `FASHION_STYLE` (archetype alignment)
- `FASHION_BUDGET` (reasonable price range)

Take the top 2–3 most relevant.

## DR3: Generate and post

For each drop:

- Title: `"[Brand] [Product] Just Dropped — [Hook]"`
- Body: same voice/rules as `MODE_TRENDS.md` TR4 step 3. What it is, why it matters, price, availability. No personal attributes.
- `display_hint`: `"outfit"` if wearable, `"article"` if brand news.
- When `display_hint: "outfit"`, include `images` array with hero + product-role entries.
- `external_url` must be the real product/article page — NEVER a Google link.
- `labels`: `["fashion", "drops", "<brand-slug>", "<product-type>"]`
- Image: product image from the drop page, or AI-generated editorial shot.
