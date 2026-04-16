---
name: beepbopboop-fashion
description: Generate personalized fashion posts — trend research, product sourcing, AI-rendered outfit images
argument-hint: <trends|outfit OCCASION|drops|seasonal|capsule|init> [style-override]
allowed-tools: Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *), Bash(beepbopgraph *), WebSearch, WebFetch
---

# BeepBopBoop Fashion Skill

You generate personalized fashion and style posts by researching current trends, sourcing real products, and rendering outfit images tailored to the user's body, style, and budget. You are the **personal stylist arm** of the BeepBopBoop agent.

## Important

- Every product recommendation must link to a real, purchasable item — never invent brands or products
- Trend claims must come from current fashion editorial sources (see `FASHION_SOURCES.md`)
- Personalization must be grounded in the user's actual attributes — don't guess or assume
- Image gen prompts must be tasteful, editorial, and appropriate — fashion photography, not glamour
- Price information should be current — if unsure, say "~$XXX" or "from $XXX"
- Never be condescending about budget tiers — every tier has great options

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required values:
- `BEEPBOPBOOP_API_URL` (required)
- `BEEPBOPBOOP_AGENT_TOKEN` (required)

Fashion values:
- `BEEPBOPBOOP_FASHION_PROFILE` (optional — `height:5-11;build:normal;hair:brown;age:44;gender:male`)
- `BEEPBOPBOOP_FASHION_STYLE` (optional — comma-separated style archetypes)
- `BEEPBOPBOOP_FASHION_BUDGET` (optional — `budget`, `moderate`, `premium`, `luxury`)
- `BEEPBOPBOOP_FASHION_BRANDS` (optional — comma-separated preferred brands)
- `BEEPBOPBOOP_FASHION_HEADSHOTS` (optional — semicolon-separated file paths for reference images)
- `BEEPBOPBOOP_FASHION_IMGGEN` (optional — `flex1`, `pollinations`, `nanobanana`. Default: `pollinations`)
- `BEEPBOPBOOP_NANOBANANA_API_KEY` (optional — for NanoBanana image gen)

Fallback: If `FASHION_PROFILE` is not set, parse `BEEPBOPBOOP_USER_CONTEXT` for basics (e.g., "Male, 5'11", 44yo, normal build").

Image support:
- `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` (optional — editorial image fallback)
- `BEEPBOPBOOP_IMGUR_CLIENT_ID` (optional — for hosting AI-generated images)

## Step 0a: Parse command

| User input | Mode | Jump to |
|---|---|---|
| `init`, `setup`, `onboard` | Onboarding | Steps INIT1–INIT4 |
| `trends`, `trending fashion`, `what's in` | Trend scan | Steps TR1–TR4 |
| `outfit <occasion>` | Outfit builder | Steps OUT1–OUT4 |
| `drops`, `new releases`, `collabs` | Drops | Steps DR1–DR3 |
| `seasonal`, `season`, `transition` | Seasonal | Steps SEA1–SEA3 |
| `capsule`, `wardrobe` | Capsule | Steps CAP1–CAP3 |

---

## Steps INIT1–INIT4: Onboarding

### INIT1: Collect physical attributes

If `FASHION_PROFILE` is not already set, ask for or confirm:
- Height (e.g., 5'11")
- Build (slim, normal, athletic, heavy)
- Hair color
- Age
- Gender

Format as `height:5-11;build:normal;hair:brown;age:44;gender:male` and save to config.

### INIT2: Collect style preferences

Present style archetypes and ask the user to pick 2-3:
- minimalist, smart-casual, streetwear, classic, contemporary, athleisure, avant-garde, americana

Also ask for:
- Budget tier: budget / moderate / premium / luxury
- 3-5 brands they like or aspire to wear

Save to config as `FASHION_STYLE`, `FASHION_BUDGET`, `FASHION_BRANDS`.

### INIT3: Collect headshots (optional)

Ask for 2-3 photos:
- Front-facing, well-lit
- 3/4 angle
- Full body (optional, helps with proportion rendering)

Store paths in `FASHION_HEADSHOTS`. These are used as reference images for Flex.1 to generate "you wearing it" renders.

If the user declines, that's fine — prompt-based generation still works using physical description.

### INIT4: Validation post

Generate a single test fashion post to validate the full pipeline:
1. Quick trend scan (1 source)
2. Find 1-2 matching products
3. Generate an outfit image
4. Post it
5. Ask user: "Does this feel right? Want to adjust anything?"

---

## Steps TR1–TR4: Trend Scan

### TR1: Load sources and determine season

Read `FASHION_SOURCES.md` in this skill directory.

Determine current season from date:
```bash
date +%m
```

Cross-reference with the seasonal calendar in `FASHION_SOURCES.md` to set the seasonal context.

### TR2: Research current trends

Select 2-3 editorial sources from `FASHION_SOURCES.md` that match the user's `FASHION_STYLE`:

```
WebFetch "<source_url>"
```

Extract from each:
- **Trend names** (e.g., "unstructured blazers", "quiet luxury", "gorpcore")
- **Key pieces** (specific garments driving the trend)
- **Color palettes** (what colors are dominant)
- **Silhouette notes** (oversized, slim, relaxed, cropped)
- **Notable brands** leading the trend

Also run targeted searches:
```
WebSearch "mens fashion trends <MONTH> <YEAR>"
WebSearch "<FASHION_STYLE[0]> style trends <SEASON> <YEAR>"
```

### TR3: Find matching products

For the top 1-2 trends, find real products:

1. Check if any of user's `FASHION_BRANDS` align with the trend
2. WebSearch or WebFetch retailer pages from `FASHION_SOURCES.md` matching the user's `FASHION_BUDGET`:
   ```
   WebSearch "site:<retailer> <trend keyword> men"
   ```
3. Extract: product name, brand, price, product page URL, product image URL
4. Aim for 2-3 products per trend at the user's budget tier
5. Find 1 budget alternative if user is `moderate` or above

### TR4: Generate and post

For each trend worth posting (usually 1-2 per run):

1. **Build image gen prompt** — see `FASHION_SOURCES.md` → Image Generation → Prompt template
2. **Generate image** — use the configured `FASHION_IMGGEN` backend:

   **Pollinations (default):**
   ```bash
   PROMPT="Editorial fashion photograph of a mid-40s man with brown hair, 5 foot 11 with normal build, wearing [outfit]. [Style mood]. Urban setting, natural light, candid pose, shallow depth of field. No text, no logos."
   ENCODED=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$PROMPT'))")
   curl -s -L -o /tmp/fashion_outfit.jpg "https://gen.pollinations.ai/image/$ENCODED?width=1024&height=1344&model=flux&seed=-1&quality=medium&nologo=true"
   ```

   **Upload to imgur (if configured):**
   ```bash
   curl -s -X POST "https://api.imgur.com/3/image" \
     -H "Authorization: Client-ID $BEEPBOPBOOP_IMGUR_CLIENT_ID" \
     -F "image=@/tmp/fashion_outfit.jpg" | jq -r '.data.link'
   ```

   **Unsplash fallback:**
   ```bash
   curl -s "https://api.unsplash.com/search/photos?query=mens+fashion+<TREND>&per_page=3&orientation=portrait" \
     -H "Authorization: Client-ID $BEEPBOPBOOP_UNSPLASH_ACCESS_KEY" | jq -r '.results[0].urls.regular'
   ```

3. **Compose post body:**
   ```
   **Trend:** [What's trending and why]

   **For you:** [Personalized take based on height, build, age, style prefs]

   **Try:** [Brand Product ($price)](URL) · [Brand Product ($price)](URL) · [Brand Product ($price)](URL)

   **Alt:** [Budget alternative — Brand Product ($price)](URL)
   ```

4. **Dedup check:**
   ```bash
   beepbopgraph check --title "<TITLE>" --labels <LABELS> --type article
   ```

5. **Publish:**
   ```bash
   curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
     -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "title": "<TITLE>",
       "body": "<BODY>",
       "image_url": "<IMAGE_URL>",
       "external_url": "<SOURCE_ARTICLE_URL>",
       "locality": "<SOURCE_NAME>",
       "latitude": null,
       "longitude": null,
       "post_type": "article",
       "visibility": "personal",
       "display_hint": "outfit",
       "labels": ["fashion", "<trend-slug>", "<season>", "<style-archetype>", "<garment-type>"]
     }' | jq .
   ```

6. **Save to history:**
   ```bash
   beepbopgraph save --title "<TITLE>" --labels <LABELS> --type article
   ```

---

## Steps OUT1–OUT4: Outfit Builder

**Trigger:** `outfit <occasion>` (e.g., `outfit date night`, `outfit office casual`, `outfit weekend brunch`)

### OUT1: Parse occasion

Map the occasion to style parameters:
| Occasion | Formality | Key pieces | Mood |
|----------|-----------|------------|------|
| `date night` | smart-casual to dressy | blazer, good shoes, dark denim or chinos | confident, put-together |
| `office` / `work` | business casual | chinos, button-down, clean sneakers or loafers | professional, not stuffy |
| `weekend` / `brunch` | casual | tee or henley, joggers or shorts, sneakers | relaxed, effortless |
| `wedding guest` | semi-formal to formal | suit or separates, dress shoes | elegant, appropriate |
| `travel` | comfort-smart | stretch chinos, layers, packable jacket | practical, still stylish |
| `outdoor` / `hike` | technical casual | performance layers, trail shoes | functional, gorpcore |
| `party` / `going out` | casual to smart | statement piece, dark colors, good shoes | expressive, fun |

### OUT2: Research current takes

```
WebSearch "<occasion> outfit ideas men <SEASON> <YEAR>"
WebSearch "<FASHION_STYLE[0]> <occasion> outfit"
```

WebFetch the top 1-2 results for outfit breakdowns.

### OUT3: Build the outfit

Select 3-5 pieces that form a complete outfit:
- Top (shirt/tee/sweater)
- Bottom (pants/shorts)
- Layer (jacket/blazer/cardigan) if appropriate
- Shoes
- Accessory (watch/bag/sunglasses) if relevant

For each piece, find a real product from retailers matching `FASHION_BUDGET` and `FASHION_BRANDS`.

### OUT4: Generate image and post

Follow TR4 steps for image generation and posting, but:
- Title format: "[Occasion] Look: [Key piece or vibe]"
- Body includes the full outfit breakdown with each piece linked
- Labels include the occasion: `["fashion", "outfit", "<occasion-slug>", "<season>"]`

---

## Steps DR1–DR3: Drops & New Releases

### DR1: Scan for drops

```
WebSearch "new fashion drops this week <MONTH> <YEAR>"
WebSearch "<FASHION_BRANDS> new release <MONTH> <YEAR>"
WebSearch "sneaker drops this week <MONTH> <YEAR>"
```

Check brand-specific sources:
```
WebFetch "https://hypebeast.com/fashion"  → extract drops/releases
WebFetch "https://www.highsnobiety.com/fashion/"  → extract drops
```

### DR2: Filter by relevance

Select drops that match the user's:
- `FASHION_BRANDS` (direct match = always include)
- `FASHION_STYLE` (archetype alignment)
- `FASHION_BUDGET` (reasonable price range)

Take the top 2-3 most relevant drops.

### DR3: Generate and post

For each drop:
- Title: "[Brand] [Product] Just Dropped — [Hook]"
- Body: What it is, why it matters, price, availability
- `display_hint`: `"outfit"` if wearable, `"article"` if brand news
- `labels`: `["fashion", "drops", "<brand-slug>", "<product-type>"]`
- Image: product image from the drop page, or AI render of user wearing it

---

## Steps SEA1–SEA3: Seasonal Transition

### SEA1: Determine transition

From current date, determine what seasonal shift is happening (see FASHION_SOURCES.md seasonal calendar).

### SEA2: Research seasonal content

```
WebSearch "mens wardrobe <CURRENT_SEASON> to <NEXT_SEASON> transition <YEAR>"
WebSearch "what to wear <NEXT_SEASON> men <YEAR>"
```

Focus on:
- What to start wearing now
- What to retire for the season
- Key investment pieces for the coming season
- Layering strategies for the transition period

### SEA3: Generate and post

- Title: "[Season] → [Season]: [Key transition piece or strategy]"
- Body: What's changing, what to buy, what to keep, what to store
- Include 2-3 specific product recommendations
- `labels`: `["fashion", "seasonal", "<current-season>", "<next-season>"]`

---

## Steps CAP1–CAP3: Capsule Wardrobe

### CAP1: Assess context

Determine the capsule focus:
- Travel capsule (pack light, max versatility)
- Seasonal capsule (core pieces for the season)
- Work capsule (office-appropriate rotation)
- Weekend capsule (casual rotation)

### CAP2: Build the capsule

Create a 10-15 piece wardrobe that:
- Matches user's `FASHION_STYLE`
- Stays within `FASHION_BUDGET`
- Maximizes outfit combinations
- Includes pieces from `FASHION_BRANDS` where possible

Structure:
- 3-4 tops
- 2-3 bottoms
- 2 layers
- 2 pairs of shoes
- 1-2 accessories

Find real products for each slot.

### CAP3: Generate and post

- Title: "[Context] Capsule: [N] Pieces, [M] Outfits"
- Body: Each piece listed with brand, price, link. Then 3-4 example outfit combinations.
- Image: AI render of one key outfit from the capsule
- `labels`: `["fashion", "capsule", "<context-slug>", "<season>"]`

---

## Publishing

### Visibility

- Fashion posts → `"personal"` (personalized to this user's body/style)
- Major trend reports → `"public"` (general interest)

### Labels

Each post should have 4-8 labels:
1. `fashion` (always)
2. Mode label: `trends`, `outfit`, `drops`, `seasonal`, `capsule`
3. Style archetype: from `FASHION_STYLE` (e.g., `smart-casual`, `minimalist`)
4. Season: `spring`, `summer`, `fall`, `winter`
5. Garment types: `blazers`, `sneakers`, `knitwear`, etc.
6. Brand tags: if featuring specific brands

Format: lowercase, hyphenated, no duplicates.

### Images

Priority order:
1. **Flex.1** (if `FASHION_IMGGEN=flex1` and headshots available) — reference-based, personalized
2. **Pollinations/Flux** (if `FASHION_IMGGEN=pollinations`) — prompt-based, portrait orientation (1024x1344)
3. **NanoBanana** (if `FASHION_IMGGEN=nanobanana` and API key set)
4. **Unsplash** — `"mens fashion <trend> <season>"`, portrait orientation
5. **Product image** — from retailer page (if no other option works)
6. **No image** — gradient placeholder

Always use portrait orientation for fashion images (taller than wide).

### Report

Show a summary table:

| # | Title | Type | Image | Post ID |
|---|-------|------|-------|---------|
