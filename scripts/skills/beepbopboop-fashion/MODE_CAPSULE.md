# Capsule wardrobe (CAP1–CAP3)

**Trigger:** `capsule`, `wardrobe`.

## CAP1: Assess context

Determine the capsule focus:
- Travel capsule (pack light, max versatility)
- Seasonal capsule (core pieces for the season)
- Work capsule (office-appropriate rotation)
- Weekend capsule (casual rotation)

## CAP2: Build the capsule

Create a 10–15 piece wardrobe that:
- Matches user's `FASHION_STYLE`
- Stays within `FASHION_BUDGET`
- Maximizes outfit combinations
- Includes pieces from `FASHION_BRANDS` where possible

Structure:
- 3–4 tops
- 2–3 bottoms
- 2 layers
- 2 pairs of shoes
- 1–2 accessories

Find real products for each slot.

## CAP3: Generate and post

- Title: `"[Context] Capsule: [N] Pieces, [M] Outfits"`
- Body: same voice/rules as `MODE_TRENDS.md` TR4 step 3. No personal attributes. Sharp, opinionated.
- Include `images` array: hero image + product-role entries for each capsule piece (with `caption` set to brand/product name).
- `external_url` must be a real article/retailer URL — NEVER a Google link.
- Image: AI-generated editorial shot of one key outfit from the capsule.
- `labels`: `["fashion", "capsule", "<context-slug>", "<season>"]`.
