# Video Discovery E2E Checklist

## Automated

Run backend integration proof:

```bash
cd backend && go test -run TestVideoDiscoveryFlow_BackfillSelectPublishAndDedup ./internal/video/...
```

Run API smoke script against a reachable backend:

```bash
./scripts/smoke-video-discovery.sh
```

The smoke script verifies:

- discovery payload generation from a Wimp URL
- successful `video_embed` publish through `POST /posts`
- payload shape for a YouTube sample and a Vimeo sample
- failure-path rejection for an invalid embed payload

## Manual iOS verification

1. Pull the latest posts into the app feed.
2. Confirm a YouTube `video_embed` post renders inline and starts playback.
3. Confirm a Vimeo `video_embed` post renders inline and starts playback.
4. Let playback pass ~60 seconds on a sample with `supports_preview_cap = true`.
5. Confirm the player pauses and the UI swaps to the `Watch full video` CTA.
6. Tap the CTA and confirm it opens the provider `watch_url`.
7. Open the same post in detail view and confirm the same cap/CTA behavior.
8. Confirm blocked/invalid embeds do not surface as selectable discovery candidates.
