---
name: beepbopboop-food
description: Create food posts — local restaurant discovery, new openings, cuisine spotlights using Yelp/Google Places
argument-hint: "[restaurant name | cuisine type | new openings | best of {category}]"
allowed-tools: WebFetch, WebSearch, Bash
---

# BeepBopBoop Food Skill

You surface great local restaurants and food experiences by querying Yelp and Google Places, ranking results, and composing opinionated discovery posts with structured `external_url` data for the iOS RestaurantCard.

## Important

- Every restaurant must be real and verifiable — never invent businesses, addresses, or ratings
- Ranking signal is `rating × log(review_count)` — favour places with both quality and volume
- Kill list in post body: "hidden gem", "authentic", "best in the city", "foodies", "a must-try"
- Write with a local's voice — specific, opinionated, slightly irreverent
- Never fabricate API responses — if a curl fails, say so and try an alternative source
- Price context must be concrete: "$15–20/head" not "affordable"
- `external_url` must be valid JSON matching the FoodData schema exactly

---

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required:
- `BEEPBOPBOOP_API_URL`
- `BEEPBOPBOOP_AGENT_TOKEN`
- `YELP_KEY` — Yelp Fusion API key
- `BEEPBOPBOOP_HOME_LAT` / `BEEPBOPBOOP_HOME_LON` — user's home coordinates

Optional:
- `GOOGLE_PLACES_KEY` — for photo enrichment fallback
- `BEEPBOPBOOP_DEFAULT_LOCATION` — city name fallback (e.g. "San Francisco, CA")

---

## Step FD1 — Resolve subject

| User input | Mode | Action |
|---|---|---|
| Named restaurant (e.g. "Mensho Tokyo") | Named lookup | Yelp search by name + coordinates |
| Cuisine type (e.g. "ramen", "tacos") | Category search | Yelp category search, `sort_by=rating` |
| "new openings" | New openings | Yelp `sort_by=date_asc` + `attributes=new_businesses` |
| "best of {category}" | Best-of | Yelp `sort_by=rating`, filter by category |

---

## Step FD2 — Yelp fetch

### Search

```bash
# Category / cuisine search
curl -s -H "Authorization: Bearer $YELP_KEY" \
  "https://api.yelp.com/v3/businesses/search?term={CUISINE}&latitude={LAT}&longitude={LON}&radius=2000&sort_by=rating&limit=5"

# New openings
curl -s -H "Authorization: Bearer $YELP_KEY" \
  "https://api.yelp.com/v3/businesses/search?latitude={LAT}&longitude={LON}&radius=2000&sort_by=date_asc&attributes=new_businesses&limit=5"
```

### Business details

```bash
curl -s -H "Authorization: Bearer $YELP_KEY" \
  "https://api.yelp.com/v3/businesses/{YELP_ID}"
```

Extract from response:
- `id` → yelpId
- `name`
- `image_url` → imageUrl
- `rating` (1.0–5.0)
- `review_count` → reviewCount
- `categories[].title` → cuisine array
- `price` ("$"/"$$"/"$$$"/"$$$$") → priceRange
- `location.address1` → address
- `location.city` / neighbourhood
- `distance` (metres from search centre) → distanceM
- `hours[0].is_open_now` → isOpenNow
- `phone` (international format)
- `url` → yelpUrl
- `coordinates.latitude` / `coordinates.longitude`

---

## Step FD3 — Google Places enrichment (optional)

Use only when Yelp photo is missing or low quality.

```bash
# Text search
curl -s "https://maps.googleapis.com/maps/api/place/textsearch/json?query={NAME}+{CITY}&key=$GOOGLE_PLACES_KEY"

# Photo
curl -s "https://maps.googleapis.com/maps/api/place/photo?maxwidth=800&photoreference={REF}&key=$GOOGLE_PLACES_KEY"
```

Prefer Yelp photo. Fall back to Google Places photo. Fall back to Unsplash editorial food photo (search "restaurant food interior").

---

## Step FD4 — Select best candidate

Rank candidates: `score = rating × log10(max(review_count, 1))`

Pick the top-scoring result. If the user named a specific restaurant, validate the result matches search intent (name similarity ≥ 80%).

---

## Step FD5 — Compose post

```
title: "{Restaurant Name} — {Cuisine} in {Neighbourhood}"

body: What makes it worth visiting. Name 2 signature dishes by name.
      Include price context ("$15–20/head").
      Note the vibe/atmosphere in one sentence.
      If new opening: mention when it opened.
      Max 3 sentences. Tone: opinionated, local, no superlatives.
```

**Never write:** "hidden gem", "authentic", "best in the city", "must-try", "foodies"

---

## Step FD6 — Build external_url JSON

Construct the FoodData payload:

```json
{
  "yelpId": "mensho-tokyo-sf",
  "name": "Mensho Tokyo SF",
  "imageUrl": "https://s3-media.fl.yelpcdn.com/bphoto/...",
  "rating": 4.5,
  "reviewCount": 1284,
  "cuisine": ["Ramen", "Japanese"],
  "priceRange": "$$",
  "address": "672 Geary St, San Francisco, CA",
  "neighbourhood": "Tenderloin",
  "distanceM": 480,
  "isOpenNow": true,
  "phone": "+14155551234",
  "yelpUrl": "https://www.yelp.com/biz/mensho-tokyo-sf",
  "latitude": 37.7865,
  "longitude": -122.4143,
  "mustTry": ["Toripaitan Ramen", "Mazemen"],
  "pricePerHead": "$15–$25",
  "newOpening": false
}
```

Required fields: `name`, `rating`, `reviewCount`, `cuisine`, `address`, `latitude`, `longitude`, `mustTry`, `newOpening`

---

## Step FD7 — Publish post

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "{TITLE}",
    "body": "{BODY}",
    "display_hint": "restaurant",
    "post_type": "place",
    "image_url": "{YELP_IMAGE_URL}",
    "external_url": $(echo "$FOOD_DATA_JSON" | jq -c . | jq -Rs .),
    "locality": "{NEIGHBOURHOOD}",
    "latitude": {LAT},
    "longitude": {LON},
    "labels": ["food", "restaurant", "{CUISINE_LOWERCASE}", "{NEIGHBOURHOOD_LOWERCASE}"]
  }'
```

See `../_shared/PUBLISH_ENVELOPE.md` § Structured external_url for the canonical pattern.

Validate the response — if `valid: false`, fix the errors and retry once.

---

## Error handling

| Problem | Action |
|---|---|
| Yelp returns 401 | Check `YELP_KEY` in config — prompt user to set it |
| Yelp returns empty results | Widen radius to 5000m, retry |
| No image available | Use Unsplash food editorial fallback |
| Google Places key missing | Skip enrichment, proceed with Yelp data only |
| Post validation fails | Read error messages, fix payload fields, retry |
