# MODE_POSTER — real poster / promo image from upstream research

Used when Step 3 of the caller's research already surfaced a canonical image — movie poster from TMDB, album cover from Spotify, team logo for a matchup, product shot for a drop, venue website's hero image.

## Rules

1. **Only `.jpg`, `.png`, `.webp` URLs.** No `url` that returns HTML, no generic CDN thumbnails with a short TTL.
2. **Already permanent?** TMDB `image.tmdb.org`, Spotify `i.scdn.co`, YouTube `i.ytimg.com`, team-logo CDNs — use directly, no re-upload.
3. **Ephemeral?** Anything behind a signed URL or behind a hot-linking ban (news sites often ban hot-linking) — re-upload to imgur first.

## Re-upload recipe

```bash
curl -s -L -o /tmp/bbp_poster.jpg "SOURCE_URL"
POSTER_IMG=$(curl -s -X POST "https://api.imgur.com/3/image" \
  -H "Authorization: Client-ID $BEEPBOPBOOP_IMGUR_CLIENT_ID" \
  -F "image=@/tmp/bbp_poster.jpg" \
  -F "type=file" | jq -r '.data.link // empty')
rm -f /tmp/bbp_poster.jpg
```

## Common sources that do NOT need rehosting

| Domain | Note |
|---|---|
| `image.tmdb.org/...` | Permanent. Use `/w500/` for compact. |
| `i.scdn.co/image/...` | Spotify CDN; permanent. |
| `i.ytimg.com/vi/<id>/hqdefault.jpg` | Permanent; use for `video_embed.thumbnail_url`. |
| `commons.wikimedia.org/...` | Permanent. |
| `upload.wikimedia.org/wikipedia/commons/thumb/...` | Permanent. |

## Common sources that DO need rehosting

| Domain | Why |
|---|---|
| `places.googleapis.com/.../media?...` | Signed URL with short TTL. |
| Newspaper/CNN/BBC article images | Hot-link protection. |
| Facebook/Instagram CDN | Short-lived. |

## Exit

Return `{ image_url: "<URL>", source: "poster" }` on success. If the poster URL is dead or hot-link-blocked, drop back to `MODE_REAL.md` / `MODE_AI.md`.
