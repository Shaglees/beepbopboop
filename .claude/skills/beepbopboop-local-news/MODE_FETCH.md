# Mode: Fetch Local News

## Step 1: Get Nearby Sources

```bash
curl -s "$BEEPBOPBOOP_API_URL/news-sources?lat=$BEEPBOPBOOP_HOME_LAT&lon=$BEEPBOPBOOP_HOME_LON&radius_km=50" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN"
```

If empty: suggest running discover mode first, then stop.

## Step 2: Fetch Content from Each Source

For each source with `feed_url`:
- Fetch the RSS feed (use WebFetch)
- Parse items: title, link, description, pubDate
- Filter to items published in last 48 hours

For sources without `feed_url`:
- Fetch the main `url` with WebFetch
- Extract top stories from the page

## Step 3: Score and Rank

Score each item by:
- **Recency**: items from last 6h score highest, 6-24h medium, 24-48h lowest
- **Source trust**: multiply by `trust_score / 100`
- **Topic relevance**: bonus if item topics overlap user's interests (from profile)

Pick top 3-5 items.

## Step 4: Compose Posts

For each selected item, compose a post:

```json
{
  "title": "<headline, max 100 chars>",
  "body": "<2-3 sentence summary>",
  "post_type": "article",
  "display_hint": "local_news",
  "image_url": "<thumbnail if available>",
  "external_url": "<JSON string — see below>",
  "locality": "<source area_label>",
  "labels": ["news", "local", "<topic>"]
}
```

The `external_url` must be a **JSON string** (stringified JSON, not an object) with this shape:
```json
{
  "content_kind": "article",
  "source_name": "Source Name",
  "source_url": "https://source.com",
  "source_logo_url": null,
  "thumbnail_url": "https://example.com/thumb.jpg",
  "article_url": "https://source.com/article",
  "embed_url": null,
  "duration_seconds": null,
  "locality": "City, Country",
  "published_at": "2026-04-25T10:00:00Z",
  "trust_score": 80
}
```

## Step 5: Lint and Publish

Follow `../_shared/PUBLISH_ENVELOPE.md`.
