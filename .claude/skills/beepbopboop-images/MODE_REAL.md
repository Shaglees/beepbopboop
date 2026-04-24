# MODE_REAL — real-image tiers

Real images beat AI images for almost every post. Try these in order for a post where a "real" image is plausible.

## Tier R1 — Wikimedia Commons (geographic only)

No key; requires `User-Agent`. Coordinates are `LAT|LON` (pipe-separated, URL-encoded as `%7C`).

**R1a — geosearch (500m radius)**

```bash
WC_IMG=$(curl -s -H "User-Agent: BeepBopBoop/1.0 (contact@beepbopboop.app)" \
  "https://commons.wikimedia.org/w/api.php?action=query&format=json&generator=geosearch&ggsprimary=all&ggsnamespace=6&ggsradius=500&ggscoord=LAT%7CLON&ggslimit=5&prop=imageinfo&iilimit=1&iiprop=url&iiurlwidth=1024" \
  | jq -r '[.query.pages[] | select(.imageinfo[0].thumburl)] | sort_by(.index) | .[0].imageinfo[0].thumburl // empty')
```

**R1b — text search fallback**

```bash
if [ -z "$WC_IMG" ]; then
  WC_IMG=$(curl -s -H "User-Agent: BeepBopBoop/1.0 (contact@beepbopboop.app)" \
    "https://commons.wikimedia.org/w/api.php?action=query&format=json&generator=search&gsrnamespace=6&gsrsearch=PLACE_NAME+CITY&gsrlimit=5&prop=imageinfo&iilimit=1&iiprop=url&iiurlwidth=1024" \
    | jq -r '[.query.pages[] | select(.imageinfo[0].thumburl)] | sort_by(.index) | .[0].imageinfo[0].thumburl // empty')
fi
```

Use `thumburl` (1024px), not `url`. Permanent CDN. Strongest for landmarks / parks / museums.

## Tier R2 — Panoramax (geographic only)

Street-level outdoor imagery, no auth.

```bash
PX_IMG=$(curl -s "https://api.panoramax.xyz/api/search?place_position=LON,LAT&place_distance=0-100&limit=1" \
  | jq -r '.features[0].assets.sd.href // empty')
```

**Coordinate order is `LON,LAT`** (GeoJSON) — flipping these is the most common bug. Use `sd` asset (2048px). Dense coverage in EU, sparse in North America.

## Tier R3 — Google Places → imgur rehost (geographic + venue-specific)

Requires `BEEPBOPBOOP_GOOGLE_PLACES_KEY` AND `BEEPBOPBOOP_IMGUR_CLIENT_ID`.

**Find place → photo name:**

```bash
GP_PHOTO_NAME=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Goog-Api-Key: $BEEPBOPBOOP_GOOGLE_PLACES_KEY" \
  -H "X-Goog-FieldMask: places.photos" \
  -d "{\"textQuery\": \"PLACE_NAME CITY\"}" \
  "https://places.googleapis.com/v1/places:searchText" \
  | jq -r '.places[0].photos[0].name // empty')
```

**Download + re-upload (Google photo URLs are signed/temporary):**

```bash
if [ -n "$GP_PHOTO_NAME" ] && [ -n "$BEEPBOPBOOP_IMGUR_CLIENT_ID" ]; then
  curl -s -L -o /tmp/bbp_google_photo.jpg \
    "https://places.googleapis.com/v1/${GP_PHOTO_NAME}/media?key=$BEEPBOPBOOP_GOOGLE_PLACES_KEY&maxWidthPx=1024"
  GP_IMG=$(curl -s -X POST "https://api.imgur.com/3/image" \
    -H "Authorization: Client-ID $BEEPBOPBOOP_IMGUR_CLIENT_ID" \
    -F "image=@/tmp/bbp_google_photo.jpg" \
    -F "type=file" | jq -r '.data.link // empty')
  rm -f /tmp/bbp_google_photo.jpg
fi
```

Cost: ~$0.04/place (free $200/month ≈ 5000 lookups). Best venue coverage globally.

## Tier R4 — Unsplash (any post)

Requires `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY`. Great for non-geographic or abstract topics.

```bash
UN_IMG=$(curl -s "https://api.unsplash.com/search/photos?query=SEARCH_KEYWORDS&per_page=1&orientation=landscape" \
  -H "Authorization: Client-ID $BEEPBOPBOOP_UNSPLASH_ACCESS_KEY" \
  | jq -r '.results[0].urls.regular // empty')
```

**Keyword rules:** 2–4 concrete visual nouns. Include locale when it sharpens relevance. Examples:

| Topic | Keywords |
|---|---|
| Coffee | `cafe coffee latte morning` |
| Cherry blossoms | `cherry blossom street spring pink` |
| Hockey game | `ice hockey arena crowd` |
| AI article | `artificial intelligence technology abstract` |
| Farmers market | `farmers market produce outdoor morning` |
| Theatre show | `theatre stage performance spotlight` |
| Park/hiking | `hiking trail nature forest` |
| Restaurant | `restaurant dining table food` |

## Exit

Return the first non-empty URL among `$WC_IMG`, `$PX_IMG`, `$GP_IMG`, `$UN_IMG`. If all empty and `fallback_ok`, read `MODE_AI.md`.
