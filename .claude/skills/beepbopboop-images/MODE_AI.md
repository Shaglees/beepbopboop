# MODE_AI — AI image generation

Use when no real image fits or real-image tiers all failed. Always re-host to imgur so the URL survives.

## Tier A1 — Pollinations / flux (general)

Requires `BEEPBOPBOOP_IMGUR_CLIENT_ID`. Optional `BEEPBOPBOOP_POLLINATIONS_TOKEN` for higher rate limits.

### Prompt rules

- 15–30 words, one scene.
- Editorial photography, natural light, candid.
- No text, logos, UI chrome, watermarks, people staring at camera.
- Respect the post's tone: cozy → warm desk lamp, sporty → arena crowd, techy → abstract nodes.

### Examples

| Topic | Prompt |
|---|---|
| Coffee | `Warm morning light through cafe window, single-origin pour-over on wooden counter, Pacific Northwest` |
| Market | `Outdoor farmers market stalls with colorful produce, morning crowd, spring sunshine` |
| Event | `Theatre marquee at dusk, warm glow from lobby windows, people arriving for evening show` |
| AI article | `Abstract visualization of neural network connections, dark background, glowing nodes, futuristic` |
| YouTube creator | `Content creator workspace, multiple monitors, camera setup, warm desk lamp, modern studio` |

### Generate + re-host

```bash
curl -s -L -o /tmp/bbp_post_image.jpg \
  "https://gen.pollinations.ai/image/URL_ENCODED_PROMPT?width=1024&height=768&model=flux&seed=-1&quality=medium&nologo=true"

AI_IMG=$(curl -s -X POST "https://api.imgur.com/3/image" \
  -H "Authorization: Client-ID $BEEPBOPBOOP_IMGUR_CLIENT_ID" \
  -F "image=@/tmp/bbp_post_image.jpg" \
  -F "type=file" | jq -r '.data.link // empty')

rm -f /tmp/bbp_post_image.jpg
```

**Never** embed `gen.pollinations.ai/...` directly in `image_url` — the iOS client will time out.

## Tier A2 — Flex.1 (fashion outfit render)

Used primarily by `beepbopboop-fashion/MODE_OUTFIT.md`. Produces a single hero render of a styled outfit. Specifics live in that mode file; keep this tier in mind when the caller passes `aesthetic_hint: outfit`.

## Tier A3 — Nanobanana (fashion detail shots)

Detail / product-level shots for the `outfit` display hint's `images[]` array (`role=detail` / `role=product`). Again specific to fashion — that skill owns the prompt templates.

## Exit

Return the first non-empty image URL. If all tiers fail, return `{ image_url: "", source: "" }` and let the caller fall back to "empty image + iOS gradient placeholder."
