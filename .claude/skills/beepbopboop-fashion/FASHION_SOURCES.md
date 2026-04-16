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

> **Only use brands from this verified list.** All are Shopify-based and return product names, prices, image URLs, and product page URLs via WebFetch. Major retailers (Zara, Nike, SSENSE, MR PORTER, etc.) block scraping and should NOT be used for product sourcing — reference them only in editorial context.

### How Shopify scraping works

Product data is embedded in the initial HTML as JSON (`collectionView`, `window.igProductData`, or `collection_viewed` events). Pattern:
- **Product URLs:** `https://{domain}/products/{handle}`
- **Image URLs:** `https://{domain}/cdn/shop/files/{filename}` (prefix `https:` if path starts with `//`)
- **Prices:** Usually in cents (divide by 100) or as formatted strings

### Budget–Moderate ($30–$150)

| Brand | Scrape URL | Style | Notes |
|-------|-----------|-------|-------|
| **Everlane** | `https://www.everlane.com/collections/mens-new-arrivals` | Minimalist basics | USD. Clean essentials, transparent pricing |
| **Alex Mill** | `https://www.alexmill.com/collections/mens-new-arrivals` | Smart-casual | USD. NY-based, elevated basics |
| **Bonobos** | `https://www.bonobos.com/shop/new-arrivals` | Smart-casual | USD. Great chinos and trousers. Data in JS state (Imgix CDN) |
| **Saturdays NYC** | `https://www.saturdaysnyc.com/collections/new-arrivals` | Contemporary casual | USD. Surf-meets-city aesthetic |
| **Battenwear** | `https://www.battenwear.com/collections/all` | Americana, outdoor | USD. Made in USA. Sale prices available via `compare_at_price` |
| **Gitman Bros** | `https://www.gitman.com/collections/all` | Classic, shirting | USD. Heritage shirtmaker since 1978 |

### Moderate ($80–$250)

| Brand | Scrape URL | Style | Notes |
|-------|-----------|-------|-------|
| **Stussy** | `https://www.stussy.com/collections/new-arrivals` | Streetwear | USD. OG streetwear, always relevant |
| **Portuguese Flannel** | `https://www.portugueseflannel.com/collections/all` | Smart-casual, linen | EUR (prices in cents). Camp collars, linen specialists |
| **3sixteen** | `https://www.3sixteen.com/collections/all` | Americana, denim | USD. Premium denim, made in USA |
| **Oliver Spencer** | `https://www.oliverspencer.co.uk/collections/new-arrivals` | Contemporary British | GBP. Relaxed tailoring, great knitwear |
| **Woolrich** | `https://www.woolrich.com/us/en/men/new-arrivals` | Outdoor heritage | USD. Outerwear specialists since 1830 |

### Premium ($150–$400)

| Brand | Scrape URL | Style | Notes |
|-------|-----------|-------|-------|
| **Sunspel** | `https://us.sunspel.com/collections/mens` | Minimalist, premium basics | USD. James Bond's t-shirt brand. Polos, knits, underwear |
| **Officine Generale** | `https://us.officinegenerale.com/collections/men` | Parisian smart-casual | USD. Relaxed French tailoring |
| **Aime Leon Dore** | `https://www.aimeleondore.com/collections/new-arrivals` | Contemporary streetwear | USD. Queens, NY. 70+ SKUs. New Balance collabs |
| **Private White VC** | `https://www.privatewhitevc.com/collections/new-arrivals` | Heritage outerwear | GBP. Manchester-made, military heritage |
| **S.N.S. Herning** | `https://www.sns-herning.com/collections/all` | Knitwear | EUR. Danish fisherman knit specialists |
| **Filippa K** | `https://www.filippa-k.com/en/man/new-arrivals` | Scandi minimalist | USD. Stockholm-based, clean lines |
| **Standard & Strange** | `https://standardandstrange.com/collections/all` | Multi-brand (heritage) | USD. Curates Kapital, Mister Freedom, Iron Heart, etc. |

### Luxury ($300+)

| Brand | Scrape URL | Style | Notes |
|-------|-----------|-------|-------|
| **Lemaire** | `https://www.lemaire.fr/collections/new-arrivals-men-unisex` | Avant-garde minimalist | EUR. Also: `/collections/men-shirts`, `/collections/men-pants` |
| **Acne Studios** | `https://www.acnestudios.com/us/en/man/new-arrivals/` | Contemporary Scandi | USD. Names + prices work; images are lazy-loaded (use product page for images) |

### Blocked — do NOT scrape (use for editorial reference only)

These sites block WebFetch or return empty JS-rendered pages:
Zara, H&M, COS, Uniqlo, ASOS, Gap, A.P.C., Nike, Adidas, New Balance, MR PORTER, SSENSE, END Clothing, Farfetch, Norse Projects, Carhartt WIP, Our Legacy, Margaret Howell, J.Crew, AllSaints, Todd Snyder, Kith

You can still **mention** these brands and use **WebSearch** to find their products, but you cannot scrape their product pages directly. For product images and prices from these brands, use WebSearch results or link to the product page without scraping it.

---

## Image Generation

### Prompt construction

When building image gen prompts from trend research:

1. **Extract style descriptors** from articles: silhouette (relaxed, tailored, oversized), color palette (earth tones, monochrome, muted pastels), fabric (linen, cotton, wool blend), mood (effortless, polished, rugged)
2. **Add specific garments** from product research: "unstructured linen blazer, wide-leg cotton trousers, white leather sneakers"
3. **Set the scene**: editorial photography style, natural light, urban/outdoor setting
4. **No personal attributes** — never include age, height, build, or hair color. Focus on the clothes.

### Prompt template

```
Editorial fashion photograph, [specific outfit description from products].
[Style mood from trend research]. Shot in [setting], natural light,
shallow depth of field. No text, no logos, no UI elements, no faces.
```

> **Do NOT include personal attributes** (age, height, build, hair color) in image prompts. Focus on the garments, styling, and mood. Crop or frame to avoid faces — think flatlay, detail shots, or torso-down editorial.

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

### Images array for outfit posts

Outfit posts use a multi-image `images` array. Populate it during research and publishing:

| Role | Source | Used in iOS |
|------|--------|-------------|
| `hero` | AI-generated outfit render or editorial photo | Full-bleed feed card, top of detail collage |
| `detail` | Additional editorial shots, styling close-ups | Interleaved inline images in lookbook detail |
| `product` | Product page thumbnails from retailers | Feed card scroll row, "Shop the look" rows |

- Always include at least 1 `hero` image
- Product images: set `caption` to the brand/product short name (shown below thumbnail)
- Detail images are optional but make the lookbook detail view much richer
- The `image_url` field should duplicate the hero URL for backwards compatibility

4. **Unsplash fallback** — editorial photography, no personalization
   - Search: `"mens fashion [trend keyword] [season]"`
   - Use `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY`

---

## Style Archetype Reference

Maps user's `FASHION_STYLE` tags to source weighting and prompt modifiers:

| Style tag | Scrape brands | Editorial sources | Prompt modifiers | Price sweet spot |
|-----------|--------------|-------------------|-------------------|-----------------|
| `minimalist` | Everlane, Sunspel, Filippa K | Permanent Style, GQ | "clean lines, neutral palette, understated" | moderate-premium |
| `smart-casual` | Alex Mill, Officine Generale, Sunspel | GQ, Put This On | "polished but relaxed, no tie, tailored" | moderate-premium |
| `streetwear` | Stussy, Aime Leon Dore, Saturdays NYC | Highsnobiety, Hypebeast | "relaxed fit, sneakers, layered, graphic" | moderate |
| `classic` | Gitman Bros, Private White VC, Sunspel | Permanent Style, Put This On | "timeless, well-fitted, quality fabrics" | premium |
| `contemporary` | Officine Generale, Lemaire, Filippa K | Highsnobiety, GQ | "modern silhouettes, trend-aware, elevated" | moderate-premium |
| `athleisure` | Everlane, Saturdays NYC, Bonobos | GQ | "performance fabrics, clean sneakers, joggers" | moderate |
| `avant-garde` | Lemaire, Acne Studios, S.N.S. Herning | SSENSE Editorial, Hypebeast | "experimental, oversized, asymmetric, dark" | premium-luxury |
| `americana` | 3sixteen, Battenwear, Standard & Strange | Put This On | "workwear, denim, boots, heritage" | moderate |

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
