# Interest mode (INT1–INT3)

**Trigger:** any topic matching `BEEPBOPBOOP_INTERESTS`, or a direct topic query.

## INT1: Resolve interest context

- Parse the idea for topic area, specific creators/sources, timeframe.
- Cross-reference with `BEEPBOPBOOP_INTERESTS` from config.
- No geocoding needed.

## INT2: Research content

Search for recent content:

- **For topics:** `WebSearch "<TOPIC> latest news <MONTH> <YEAR>"`, `"<TOPIC> breakthroughs <MONTH> <YEAR>"`.
- **For creators:** `WebSearch "<CREATOR> latest blog post"`, `"<CREATOR> latest YouTube video <MONTH> <YEAR>"`.
- **For YouTube:** `WebSearch "<CHANNEL> latest video <MONTH> <YEAR>"`.

WebFetch top 2–3 results. Extract:
- Title, author/source, publication date, key points, URL.
- For YouTube: video title, channel, publish date, description summary.

## INT3: Classify and generate

**Classification:**
- YouTube video, video essay, podcast with video → `video`
- Blog post, news article, essay, newsletter → `article`

**Post fields:**
- `title` and `body`: follow writing quality standards (see `BASE_LOCAL.md`) — hook + deliver.
- `locality`: source/creator name (e.g., `"Simon Willison's Blog"`, `"Fireship on YouTube"`).
- `latitude`/`longitude`: `null`.
- `external_url`: direct content link.
- `post_type`: `article` or `video`.
- `display_hint`: `article` (or `card` for videos).

Then proceed to `COMMON_PUBLISH.md`.
