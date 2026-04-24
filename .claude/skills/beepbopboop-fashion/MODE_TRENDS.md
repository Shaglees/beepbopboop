# Trend scan (TR1–TR4)

**Trigger:** `trends`, `trending fashion`, `what's in`.

## TR1: Load sources and determine season

Read `FASHION_SOURCES.md`. Determine current season:

```bash
date +%m
```

Cross-reference with the seasonal calendar in `FASHION_SOURCES.md` to set seasonal context.

## TR2: Research current trends

Select 2–3 editorial sources from `FASHION_SOURCES.md` matching the user's `FASHION_STYLE`:

```
WebFetch "<source_url>"
```

Extract from each:
- **Trend names** (e.g., "unstructured blazers", "quiet luxury", "gorpcore")
- **Key pieces** (specific garments driving the trend)
- **Color palettes**
- **Silhouette notes** (oversized, slim, relaxed, cropped)
- **Notable brands** leading the trend

Also run targeted searches:
```
WebSearch "mens fashion trends <MONTH> <YEAR>"
WebSearch "<FASHION_STYLE[0]> style trends <SEASON> <YEAR>"
```

## TR3: Find matching products

For the top 1–2 trends, find real products:

1. Check if any `FASHION_BRANDS` align with the trend.
2. WebSearch or WebFetch retailers from `FASHION_SOURCES.md` matching `FASHION_BUDGET`:
   ```
   WebSearch "site:<retailer> <trend keyword> men"
   ```
3. Extract: product name, brand, price, product page URL, product image URL.
4. Collect product image URLs — these become `product`-role entries in the `images` array. Each product image entry should have `caption` set to the brand/product short name.
5. Aim for 2–3 products per trend at the user's budget tier.
6. Find 1 budget alternative if user is `moderate` or above.

## TR4: Generate and post

For each trend worth posting (usually 1–2 per run):

### 1) Build image gen prompt

See `FASHION_SOURCES.md` → Image Generation → Prompt template.

### 2) Generate image using the configured `FASHION_IMGGEN`

**Pollinations (default):**

```bash
PROMPT="Editorial fashion photograph, [specific outfit description from products]. [Style mood from trend research]. Urban setting, natural light, shallow depth of field. No text, no logos, no faces."
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

The hero image becomes `{"url": "...", "role": "hero"}` in the images array AND goes in `image_url` (for backwards compat). If you can generate or source a second editorial shot, add it as `{"url": "...", "role": "detail"}`.

### 3) Compose post body

**Voice:** Sharp, opinionated friend who works in fashion — not a department store catalog. Concise, confident, slightly wry. Drop cultural references. Have a point of view. Never hedge with "could work" or "might look nice" — commit to the recommendation. Highsnobiety meets a group chat.

**Rules:**
- NEVER mention the reader's height, weight, build, age, or any physical attributes in the body text.
- DO use those attributes internally to choose products and silhouettes — just don't say "at 5'11"" or "for your build".
- The `**For you:**` section should read as styling advice, not a body assessment.
- Keep the intro paragraph to 2–3 punchy sentences max.
- Vary sentence length. Mix fragments with full sentences.

```
[2-3 sentence intro — what's happening, why it matters, cultural context]

**Trend:** [Crisp, specific — name the trend in 3-5 words]

**For you:** [Styling advice — how to wear it, what to pair it with, what to avoid. No body stats.]

**Try:** Brand Product ($price) · Brand Product ($price) · Brand Product ($price)

**Alt:** Brand Product ($price)
```

> Product URLs go in the `images` array as product-role entries, NOT as markdown links in the body. The iOS parser expects plain `Name ($Price)` format.

### 4) Dedup, publish, save

Follow the publish contract (see SKILL.md "Publishing" section). Key specifics:

> **IMPORTANT — `external_url` must be a real article URL** (e.g., `https://www.highsnobiety.com/p/unstructured-blazers-trend/`). NEVER use a Google search URL, Google AMP link, or any `google.com` domain. If you only have a Google link, follow it to get the real destination URL.

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "<TITLE>",
    "body": "<BODY>",
    "image_url": "<HERO_IMAGE_URL>",
    "external_url": "<REAL_SOURCE_ARTICLE_URL — not a Google link>",
    "locality": "<SOURCE_NAME>",
    "latitude": null,
    "longitude": null,
    "post_type": "article",
    "visibility": "personal",
    "display_hint": "outfit",
    "labels": ["fashion", "<trend-slug>", "<season>", "<style-archetype>", "<garment-type>"],
    "images": [
      {"url": "<HERO_IMAGE_URL>", "role": "hero"},
      {"url": "<DETAIL_IMAGE_URL>", "role": "detail", "caption": "<optional>"},
      {"url": "<PRODUCT_IMAGE_URL>", "role": "product", "caption": "<BRAND_NAME>", "link": "<PRODUCT_PAGE_URL>"}
    ]
  }' | jq .
```

Dedup check and save-to-history use the standard `beepbopgraph check` / `beepbopgraph save` contract documented in `../beepbopboop-post/COMMON_PUBLISH.md`.
