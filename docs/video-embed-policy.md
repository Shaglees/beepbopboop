# Video Embed Policy

## Method

Run the fixed-sample spike harness:

```bash
./scripts/measure-video-embed-viability.sh
```

That wrapper:

1. runs `go run ./backend/cmd/videoembedspike --input-file scripts/video-embed-sample-urls.txt --format json`
2. inspects archived Wimp pages with the same Wayback adapter used by the backfill
3. checks resulting provider embeds with the same `videohealth.HTTPChecker` used by the reconciliation worker
4. asserts:
   - `sample_size > 0`
   - at least one provider is represented
   - recommendation is present
5. renders a markdown summary from the exact same JSON payload

## Fixed Sample

The repo fixture `scripts/video-embed-sample-urls.txt` currently contains 5 archived Wimp page URLs:

- `https://www.wimp.com/flyingbike/`
- `https://www.wimp.com/a-blooper-reel-of-beatles-recordings/`
- `https://www.wimp.com/a-japanese-cover-of-country-roads/`
- `https://www.wimp.com/a-surprisingly-good-cover-of-nirvanas-dumb/`
- `https://www.wimp.com/betty-boop-in-the-old-man-of-the-mountain/`

## Latest Recorded Run

Sample size: `5`

No live embed pages: `1`

Provider matrix observed on the most recent run:

- `youtube`: `ok=0 blocked=1 gone=0 unknown=0 preview_cap=true fallback=drop`

The sample is intentionally small and reproducible, so treat the exact rates as a sanity check rather than a global estimate. The important result is directional:

- some archived Wimp pages have **no live third-party embed**
- some live provider embeds are **blocked**
- therefore we should **not** fall back to article-link cards in the feed when embed playback is blocked

## Policy Matrix

- `youtube`
  - `supports_preview_cap: true`
  - `fallback_behavior: drop`
- `vimeo`
  - `supports_preview_cap: true`
  - `fallback_behavior: drop`
- unknown providers
  - `supports_preview_cap: false`
  - `fallback_behavior: drop`

## Recommendation

Drop `blocked` / `gone` embeds from the feed. Do **not** fall back to article links for these candidates.

Allow first-minute preview caps for YouTube and Vimeo only, and treat provider support as an explicit policy decision rather than a guess in clients.
