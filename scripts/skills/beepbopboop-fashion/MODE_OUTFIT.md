# Outfit builder (OUT1–OUT4)

**Trigger:** `outfit <occasion>` (e.g., `outfit date night`, `outfit office casual`, `outfit weekend brunch`).

## OUT1: Parse occasion

| Occasion | Formality | Key pieces | Mood |
|----------|-----------|------------|------|
| `date night` | smart-casual to dressy | blazer, good shoes, dark denim or chinos | confident, put-together |
| `office` / `work` | business casual | chinos, button-down, clean sneakers or loafers | professional, not stuffy |
| `weekend` / `brunch` | casual | tee or henley, joggers or shorts, sneakers | relaxed, effortless |
| `wedding guest` | semi-formal to formal | suit or separates, dress shoes | elegant, appropriate |
| `travel` | comfort-smart | stretch chinos, layers, packable jacket | practical, still stylish |
| `outdoor` / `hike` | technical casual | performance layers, trail shoes | functional, gorpcore |
| `party` / `going out` | casual to smart | statement piece, dark colors, good shoes | expressive, fun |

## OUT2: Research current takes

```
WebSearch "<occasion> outfit ideas men <SEASON> <YEAR>"
WebSearch "<FASHION_STYLE[0]> <occasion> outfit"
```

WebFetch the top 1–2 results for outfit breakdowns.

## OUT3: Build the outfit

Select 3–5 pieces that form a complete outfit:
- Top (shirt / tee / sweater)
- Bottom (pants / shorts)
- Layer (jacket / blazer / cardigan) if appropriate
- Shoes
- Accessory (watch / bag / sunglasses) if relevant

For each piece, find a real product from retailers matching `FASHION_BUDGET` and `FASHION_BRANDS`.

## OUT4: Generate image and post

Follow `MODE_TRENDS.md` TR4 for image generation and posting, but:

- Title format: `"[Occasion] Look: [Key piece or vibe]"`
- Body uses the same voice and rules from TR4 step 3 — sharp, opinionated, no personal attributes.
- Include `images` array: hero image + product-role entries for each outfit piece (with `caption` set to brand/product name).
- `external_url` must be a real article/retailer URL — NEVER a Google link.
- Labels include the occasion: `["fashion", "outfit", "<occasion-slug>", "<season>"]`.
