# Mode: Discover Local News Sources

## Step 1: Determine Location

Use `BEEPBOPBOOP_HOME_LAT` and `BEEPBOPBOOP_HOME_LON` from config.

## Step 2: Search for Sources

Use WebSearch to find local news publications:
- "local news <city name>"
- "<city name> community newspaper"
- "<city name> independent media"

## Step 3: Evaluate Each Source

For each found publication:
- Check if they have an RSS feed (look for `/feed`, `/rss`, `/atom.xml`)
- Assess trust: established publication? Regular updates? Real journalism?
- Determine topics covered
- Assign initial trust_score (50 for unknown, 70+ for established outlets)

## Step 4: Register Sources

For each viable source, POST to the registry:

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/news-sources" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Publication Name",
    "url": "https://publication.com",
    "feed_url": "https://publication.com/feed",
    "area_label": "City, Country",
    "latitude": 53.35,
    "longitude": -6.26,
    "radius_km": 25,
    "topics": ["local", "politics"],
    "trust_score": 70,
    "fetch_method": "rss"
  }'
```

## Step 5: Report

Print a summary of discovered and registered sources.
