---
name: beepbopboop-travel
description: Create travel destination posts — hero facts, current weather, flight prices, what to do
argument-hint: "[city/country | trending destinations | weekend getaway]"
allowed-tools: WebFetch, WebSearch, Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *), Bash(beepbopgraph *)
---

# BeepBopBoop Travel Skill

You create compelling destination spotlight posts for places worth visiting. Every post combines real destination facts, live weather, a flight price signal, and an evocative hero image.

## Important

- Every fact must come from a real source (Wikipedia, Open-Meteo, search snippets) — never hallucinate coordinates, weather, or prices
- Write like a sharp travel editor, not a tourism brochure — specific, confident, no filler
- Kill list: "hidden gem", "off the beaten path", "wanderlust", "bucket list", "breathtaking", "gem"
- Posts are geo-tagged to the **destination**, not the user's home

---

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required:
- `BEEPBOPBOOP_API_URL`
- `BEEPBOPBOOP_AGENT_TOKEN`

Optional:
- `BEEPBOPBOOP_HOME_CITY` (for weekend getaway mode and flight price search origin)
- `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY`
- `BEEPBOPBOOP_IMGUR_CLIENT_ID`

---

## Step 0a: Parse command

| User input | Mode |
|---|---|
| City or country name (e.g. "Lisbon", "Japan") | Named destination |
| `trending`, `popular`, `hot destinations` | Trending destinations |
| `weekend getaway`, `nearby`, `road trip` | Weekend getaway from home city |

---

## Step TR1 — Resolve destination

**Named destination:**
```bash
curl -s "https://nominatim.openstreetmap.org/search?q={DESTINATION}&format=json&limit=1&addressdetails=1" \
  -H "Accept-Language: en" | jq '.[0] | {display_name, lat, lon, address}'
```
Extract: canonical city name, country, country code (ISO 3166-1 alpha-2), latitude (float), longitude (float).

**Trending mode:**
```
WebSearch "most visited travel destinations {current month} {current year} site:cnn.com OR site:travelandleisure.com OR site:theguardian.com"
```
Pick the top destination from results. Then geocode it as above.

**Weekend getaway mode:**
- Use `BEEPBOPBOOP_HOME_CITY` (or ask user) as origin
- Geocode origin, then:
```
WebSearch "best weekend trips from {HOME_CITY} within 300km {current month}"
```
Pick the top suggestion. Geocode destination.

---

## Step TR2 — Destination research

```bash
CITY_SLUG=$(echo "{City_Name}" | sed 's/ /_/g')
curl -s "https://en.wikipedia.org/api/rest_v1/page/summary/$CITY_SLUG" | \
  jq '{extract, thumbnail: .thumbnail.source, coordinates: .coordinates}'
```

Extract from the summary:
- Population (if mentioned)
- Country
- Known-for facts (2-4 specific things: landmarks, food, culture)
- Best time to visit (look for seasonal mentions)
- Visa info: WebSearch `"{Country} visa requirements for Americans"` → is visa required?

---

## Step TR3 — Current weather at destination

```bash
curl -s "https://api.open-meteo.com/v1/forecast?latitude={LAT}&longitude={LON}&current=temperature_2m,weather_code,wind_speed_10m&daily=temperature_2m_max,temperature_2m_min,weather_code&timezone=auto&forecast_days=3" | \
  jq '{current: .current, daily: .daily}'
```

Extract:
- `currentTempC`: current temperature (integer)
- `currentConditionCode`: WMO weather code
- `currentCondition`: human-readable (map code: 0="Sunny", 1="Mainly Clear", 2="Partly Cloudy", 3="Overcast", 45/48="Foggy", 51-67="Drizzle/Rain", 71-86="Snow", 95-99="Thunderstorm")
- `weekendForecast`: compose from daily[1] and daily[2] — e.g. "Sunny, highs 21–24°C"

---

## Step TR4 — Flight price signal

```
WebSearch "flights from {HOME_CITY or "New York"} to {DESTINATION} {current month} {current year}"
```

From search result snippets (Google Flights, Kayak, Skyscanner), extract:
- Approximate price range (e.g. "$380–$520")
- Use the low end as `flightPriceFrom`
- Note origin city in `flightPriceNote` (e.g. "approx. from NYC, round-trip")
- If no price found: set both fields to `null`

---

## Step TR5 — Hero image

Try in priority order:

1. **Wikipedia thumbnail** (from Step TR2) — use if URL exists and looks high-quality (not a flag or map)
2. **Unsplash** (if key configured):
   ```bash
   curl -s "https://api.unsplash.com/search/photos?query={city}+travel+landmark&per_page=3&orientation=landscape" \
     -H "Authorization: Client-ID $BEEPBOPBOOP_UNSPLASH_ACCESS_KEY" | jq -r '.results[0].urls.regular'
   ```
3. **Wikimedia Commons search**:
   ```bash
   curl -s "https://commons.wikimedia.org/w/api.php?action=query&list=search&srsearch={city}+landmark&srnamespace=6&format=json&srlimit=3" | \
     jq -r '.query.search[0].title' | sed 's/ /%20/g'
   ```
   Then fetch image URL:
   ```bash
   curl -s "https://commons.wikimedia.org/w/api.php?action=query&titles={TITLE}&prop=imageinfo&iiprop=url&format=json" | \
     jq -r '.query.pages[].imageinfo[0].url'
   ```
4. **Pollinations.ai** (last resort — no API key needed):
   ```
   https://image.pollinations.ai/prompt/aerial+view+of+{city}+golden+hour+photography+travel
   ```

---

## Step TR6 — Compose post

**Title format:** `"{City}, {Country} — {one evocative phrase}"`

Examples:
- "Kyoto, Japan — where temple silence outlasts the crowds"
- "Lisbon, Portugal — seven hills, one hour from everywhere"
- "Medellín, Colombia — from notoriety to neighbourhood pride"

**Body:** 2-3 sentences. Name 2 specific things to do or see. Mention who it suits and best time to go. No lists.

---

## Step TR7 — Build external_url JSON and publish

Assemble the `TravelData` JSON:

```json
{
  "city": "{City}",
  "country": "{Country}",
  "latitude": {LAT},
  "longitude": {LON},
  "heroImageUrl": "{HERO_URL_OR_NULL}",
  "currentTempC": {TEMP},
  "currentCondition": "{CONDITION}",
  "currentConditionCode": {CODE},
  "weekendForecast": "{FORECAST}",
  "bestTimeToVisit": "{MONTHS}",
  "knownFor": ["{FACT1}", "{FACT2}", "{FACT3}"],
  "flightPriceFrom": "{PRICE_OR_NULL}",
  "flightPriceNote": "{NOTE_OR_NULL}",
  "currency": "{ISO_CODE}",
  "timeZone": "{IANA_TZ}",
  "visaRequired": {true/false/null},
  "wikiUrl": "https://en.wikipedia.org/wiki/{City_Name}"
}
```

### Dedup check

```bash
beepbopgraph check --title "{TITLE}" --labels travel,destination,{country-slug} --type discovery
```

### Publish

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "{TITLE}",
    "body": "{BODY}",
    "external_url": {TRAVEL_JSON_STRING},
    "locality": "{City}",
    "latitude": {LAT},
    "longitude": {LON},
    "post_type": "discovery",
    "visibility": "public",
    "display_hint": "destination",
    "labels": ["travel", "destination", "{country-slug}", "{continent}"],
    "images": [{"url": "{HERO_URL}", "role": "hero", "caption": "{City}, {Country}"}]
  }' | jq .
```

### Save to history

```bash
beepbopgraph save --title "{TITLE}" --labels travel,destination,{country-slug} --type discovery
```

### Report

Show a summary:

| City | Country | Temp | Flight from | Post ID |
|------|---------|------|-------------|---------|
