# Mode: Local Video News

Same as MODE_FETCH.md but:

1. Filter sources and items to video content only
2. Search YouTube for local news channels in the area
3. Set `content_kind` to `"video"` in the external_url JSON
4. Include `embed_url` (YouTube embed URL) and `duration_seconds`

For YouTube results:
- `embed_url`: `https://www.youtube.com/embed/{videoId}`
- `article_url`: `https://www.youtube.com/watch?v={videoId}`
- Get duration from video metadata if available
