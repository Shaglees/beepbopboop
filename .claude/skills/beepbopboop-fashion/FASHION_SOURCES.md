# Fashion Sources

Curated sources for fashion trend research, product discovery, and style inspiration. Filter by the user's style archetypes and budget tier before scraping.

## How to use

1. Read the user's `FASHION_STYLE` and `FASHION_BUDGET` from config
2. Match style tags to source entries below — each source lists which archetypes it serves
3. Fetch 2-3 sources per run (rotate to keep content fresh)
4. Extract: trend name, key pieces, color palette, silhouette notes, product links
5. Use extracted style descriptors to build image gen prompts
6. Use product links for "Shop the look" sections

---

## Trend & Editorial Sources

### Highsnobiety
- URL: https://www.highsnobiety.com/fashion/
- Style archetypes: streetwear, contemporary, smart-casual
- Budget tier: moderate, premium
- What to fetch: trending articles, "best of" lists, new drops, style guides
- Fetch: `WebFetch "https://www.highsnobiety.com/fashion/"` → extract article titles, images, links

### GQ Style
- URL: https://www.gq.com/style
- Style archetypes: smart-casual, classic, contemporary
- Budget tier: moderate, premium, luxury
- What to fetch: style guides, seasonal trend reports, outfit breakdowns
- Fetch: `WebFetch "https://www.gq.com/style"` → extract recent articles

### Hypebeast
- URL: https://hypebeast.com/fashion
- Style archetypes: streetwear, contemporary, avant-garde
- Budget tier: moderate, premium
- What to fetch: drops, collabs, brand news, trend roundups
- Fetch: `WebFetch "https://hypebeast.com/fashion"` → extract articles

### MR PORTER — The Journal
- URL: https://www.mrporter.com/en-us/journal/
- Style archetypes: classic, smart-casual, premium-minimalist
- Budget tier: premium, luxury
- What to fetch: style guides, seasonal edits, "how to wear" features
- Fetch: `WebFetch "https://www.mrporter.com/en-us/journal/"` → extract articles

### SSENSE Editorial
- URL: https://www.ssense.com/en-us/editorial
- Style archetypes: avant-garde, contemporary, designer
- Budget tier: premium, luxury
- What to fetch: trend essays, designer spotlights, seasonal lookbooks
- Fetch: `WebFetch "https://www.ssense.com/en-us/editorial"` → extract articles

### Permanent Style
- URL: https://www.permanentstyle.com/
- Style archetypes: classic, minimalist, smart-casual, tailoring
- Budget tier: premium, luxury
- What to fetch: fit guides, fabric deep-dives, wardrobe building
- Fetch: `WebFetch "https://www.permanentstyle.com/"` → extract recent posts
- Note: Excellent for proportion/fit advice — especially useful for personalizing by height/build

### Put This On
- URL: https://putthison.com/
- Style archetypes: classic, smart-casual, Americana
- Budget tier: moderate, premium
- What to fetch: outfit inspiration, brand recommendations, style rules
- Fetch: `WebFetch "https://putthison.com/"` → extract recent posts

---

## Retailer / Product Sources

Use these to find specific products with real images and prices. Match to user's `FASHION_BUDGET` and `FASHION_BRANDS`.

### Budget tier: budget

| Retailer | URL | API/Fetch | Notes |
|----------|-----|-----------|-------|
| Zara | https://www.zara.com/us/en/man-new-in-l811.html | WebFetch | New arrivals, trend-forward basics |
| H&M | https://www2.hm.com/en_us/men/new-arrivals.html | WebFetch | Affordable trend pieces |
| Uniqlo | https://www.uniqlo.com/us/en/men/new-arrivals | WebFetch | Quality basics, minimalist staples |

### Budget tier: moderate

| Retailer | URL | API/Fetch | Notes |
|----------|-----|-----------|-------|
| COS | https://www.cos.com/en_usd/men/whats-new.html | WebFetch | Minimalist, Scandi-inspired |
| Lululemon | https://shop.lululemon.com/c/men/_/N-7tu | WebFetch | Athleisure, performance wear |
| Reigning Champ | https://reigningchamp.com/collections/mens-new-arrivals | WebFetch | Premium basics, athletic luxury |
| A.P.C. | https://www.apc.fr/men/new-arrivals | WebFetch | French minimalism |
| Nike | https://www.nike.com/w/new-mens-3n82yznik1 | WebFetch | Sneakers, athletic, collabs |

### Budget tier: premium

| Retailer | URL | API/Fetch | Notes |
|----------|-----|-----------|-------|
| MR PORTER | https://www.mrporter.com/en-us/mens/whats-new | WebFetch | Curated premium menswear |
| END. | https://www.endclothing.com/us/new-arrivals | WebFetch | Streetwear + designer mix |
| SSENSE | https://www.ssense.com/en-us/men/new-arrivals | WebFetch | Designer, contemporary |
| Norse Projects | https://www.norseprojects.com/collection/men-new-arrivals | WebFetch | Scandi minimalism |

### Budget tier: luxury

| Retailer | URL | API/Fetch | Notes |
|----------|-----|-----------|-------|
| Matches Fashion | https://www.matchesfashion.com/us/mens/new-in | WebFetch | Designer luxury |
| Farfetch | https://www.farfetch.com/shopping/men/new-in/items.aspx | WebFetch | Multi-brand luxury |

---

## Image Generation

### Prompt construction

When building image gen prompts from trend research:

1. **Extract style descriptors** from articles: silhouette (relaxed, tailored, oversized), color palette (earth tones, monochrome, muted pastels), fabric (linen, cotton, wool blend), mood (effortless, polished, rugged)
2. **Add user attributes** from `FASHION_PROFILE`: height, build, hair color, age range
3. **Add specific garments** from product research: "wearing a COS unstructured linen blazer, Lululemon ABC joggers, white leather sneakers"
4. **Set the scene**: editorial photography style, natural light, urban/outdoor setting, candid feel

### Prompt template

```
Editorial fashion photograph of a [age-range] [gender] with [hair] hair,
[height] with [build] build, wearing [specific outfit description from products].
[Style mood from trend research]. Shot in [setting], natural light,
candid pose, shallow depth of field. No text, no logos, no UI elements.
```

### Image gen backends (in priority order)

1. **Flex.1 via OpenClaw/Hermes** — best quality, supports reference images (headshots)
   - Use when `FASHION_IMGGEN=flex1` or headshots are available
   - Pass headshot as reference image for likeness
   - Hermes tool: `image_generate`

2. **Pollinations AI / Flux** — free, prompt-only
   - Use when `FASHION_IMGGEN=pollinations` (default)
   - Endpoint: `https://gen.pollinations.ai/image/URL_ENCODED_PROMPT?width=1024&height=1344&model=flux&seed=-1&quality=medium&nologo=true`
   - Note: portrait orientation (1024x1344) for fashion
   - Upload result to imgur if `BEEPBOPBOOP_IMGUR_CLIENT_ID` configured

3. **NanoBanana** — user-provided API key
   - Use when `FASHION_IMGGEN=nanobanana` and `NANOBANANA_API_KEY` is set

4. **Unsplash fallback** — editorial photography, no personalization
   - Search: `"mens fashion [trend keyword] [season]"`
   - Use `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY`

---

## Style Archetype Reference

Maps user's `FASHION_STYLE` tags to source weighting and prompt modifiers:

| Style tag | Priority sources | Prompt modifiers | Price sweet spot |
|-----------|-----------------|-------------------|-----------------|
| `minimalist` | COS, Norse Projects, Permanent Style | "clean lines, neutral palette, understated" | moderate-premium |
| `smart-casual` | GQ, MR PORTER, Put This On | "polished but relaxed, no tie, tailored" | moderate-premium |
| `streetwear` | Highsnobiety, Hypebeast, END. | "relaxed fit, sneakers, layered, graphic" | moderate |
| `classic` | Permanent Style, Put This On, MR PORTER | "timeless, well-fitted, quality fabrics" | premium |
| `contemporary` | SSENSE, Highsnobiety, GQ | "modern silhouettes, trend-aware, elevated" | moderate-premium |
| `athleisure` | Lululemon, Reigning Champ, Nike | "performance fabrics, clean sneakers, joggers" | moderate |
| `avant-garde` | SSENSE, Farfetch, Hypebeast | "experimental, oversized, asymmetric, dark" | premium-luxury |
| `americana` | Put This On, J.Crew, Filson | "workwear, denim, boots, heritage" | moderate |

---

## Seasonal Calendar

| Month | Northern hemisphere | Key content |
|-------|-------------------|-------------|
| Jan-Feb | Winter → Spring transition | Layering, transitional outerwear, spring preview |
| Mar-Apr | Spring arrivals | Light jackets, linen, sneakers, color refresh |
| May-Jun | Spring → Summer | Shorts, linen everything, lightweight, vacation |
| Jul-Aug | Summer + Fall preview | Swim, summer sales, fall trend previews |
| Sep-Oct | Fall arrivals | Layers, boots, knitwear, outerwear, earth tones |
| Nov-Dec | Winter arrivals | Heavy outerwear, holiday dressing, gift guides |
