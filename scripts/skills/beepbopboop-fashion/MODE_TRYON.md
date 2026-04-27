# Mode: Fashion Try-On

Generate an AI outfit preview using the user's uploaded bodyshot photo.

## Step 1: Check User Photo

```bash
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  "$BEEPBOPBOOP_API_URL/user/photos/bodyshot" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN")
```

If `STATUS` is `404`: user has no bodyshot. **Fall back to standard outfit mode** — read `MODE_OUTFIT.md` instead and stop here.

If `STATUS` is `200`: download the photo:
```bash
curl -s "$BEEPBOPBOOP_API_URL/user/photos/bodyshot" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -o /tmp/bodyshot.jpg
```

## Step 2: Fetch Trends + Preferences

Reuse the existing fashion skill trend fetching (same as standard outfit mode):
- Read user fashion prefs from config: `BEEPBOPBOOP_FASHION_STYLES`, `BEEPBOPBOOP_FASHION_BUDGET`, `BEEPBOPBOOP_FASHION_BRANDS`
- Search current trends matching those preferences

## Step 3: Compose Outfit Description

Write a detailed text prompt describing the outfit:
- Style direction from preferences + trends
- Specific garments (top, bottom, shoes, accessories)
- Colors and materials
- Season-appropriate

## Step 4: Generate Try-On Image

Call OpenAI image generation with the bodyshot as reference:
- Input: /tmp/bodyshot.jpg + text prompt describing the outfit
- Output: AI-generated image of outfit on figure resembling user

**If image generation fails:** fall back to standard outfit post with text description only.

## Step 5: Upload Image

Use the `beepbopboop-images` skill/pipeline to upload the generated image to Imgur.

**If upload fails:** use the direct image URL from the generation service (may be temporary).

## Step 6: Compose Post

```json
{
  "title": "Try-On: <outfit description>",
  "body": "<2-3 sentences about the outfit, why it works, where to wear it>",
  "post_type": "discovery",
  "display_hint": "outfit",
  "image_url": "<hosted image URL>",
  "external_url": "{\"image_variant\":\"tryon\",\"outfit_items\":[...],\"style\":\"...\"}",
  "labels": ["fashion", "try-on", "<style>"]
}
```

## Step 7: Lint + Publish

Follow `../_shared/PUBLISH_ENVELOPE.md`.
