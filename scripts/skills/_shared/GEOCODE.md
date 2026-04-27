# Shared: geocoding + label-saturation lint

Two small guardrails that came out of an actual daily run — both were silent failure modes that no existing skill documented.

## Geocoding (Nominatim) — fallback ladder

Nominatim (`nominatim.openstreetmap.org`) is great for locality queries but consistently returns `[]` for specific civic addresses outside major cities (e.g. `2620 Mica Place, Victoria BC` came back empty three different ways during today's run). Every skill that wants coordinates should try these in order and stop at the first hit:

1. **Exact address with format=jsonv2**
   ```bash
   curl -s -A "BeepBopBoop/1.0 (contact@beepbopboop.app)" \
     "https://nominatim.openstreetmap.org/search?format=jsonv2&addressdetails=1&limit=1&q=2620+Mica+Place,+Victoria,+BC"
   ```

2. **Strip the civic number, keep the street + city**
   `Mica Place, Victoria, BC`

3. **Strip the street, keep the locality + region**
   `Victoria, BC`

4. **Fall back to the user's default locality** (from config) with no coordinates.

Rules:

- Always send a unique `User-Agent` — Nominatim throttles and will return HTTP 403 otherwise.
- Rate-limit yourself to 1 request/second. Add `sleep 1` between levels.
- Never send raw home addresses in public posts — round coordinates to 3 decimal places (~110m) for anything outside commercial venues.
- If all four levels fail, post without `latitude`/`longitude` and rely on `locality` for visible location. The server lint will still warn on missing locality *and* coords; that's expected.

### Quick helper

```bash
geocode_nominatim() {
  local q="$1"
  curl -sL -A "BeepBopBoop/1.0 (contact@beepbopboop.app)" \
    "https://nominatim.openstreetmap.org/search?format=jsonv2&limit=1&q=$(jq -rn --arg q "$q" '$q|@uri')" \
    | jq -r '.[0] | if . then "\(.lat),\(.lon)" else empty end'
}
```

## Label-saturation lint (don't reuse today's most-posted labels)

`GET /posts/stats` returns a `top_labels` rollup per window. Before attaching a label, check whether it's already saturated this week.

Rule of thumb:

- **Drop any candidate label whose 7d count ≥ 80th percentile of all labels in `top_labels[7d]`.**
- **Drop any candidate label appearing in `reactions_summary.less` with `less >= 1 and more == 0`.**
- If that leaves < 3 labels, top up with label candidates ranked by `reactions_summary.more`.

Pseudocode (bash + `jq`), assuming you've already cached `/tmp/bbp_stats.json` and `/tmp/bbp_reactions.json`:

```bash
saturated=$(jq -r '.top_labels["7d"] | [.[].count] | sort | .[(length*0.8|floor)]' /tmp/bbp_stats.json)
less_labels=$(jq -r '.labels | to_entries | map(select(.value.less >= 1 and .value.more == 0)) | .[].key' /tmp/bbp_reactions.json)

# For each candidate in $LABELS, drop if its 7d count >= $saturated or if it's in $less_labels
```

### Why this exists

During the daily run we posted "spring" on a hike despite `spring` being the #2 most-posted label of the week. The stats were fetched but the lint wasn't run. This rule forces it.

### Integration point

This lint belongs **inside the compose step, before `/posts/lint`**. The server-side lint doesn't know which labels are saturated for this user; the skill does.

## Checklist

Before publishing any post with a `locality` or `labels[]`:

- [ ] Ran geocode ladder and stopped at first hit, or accepted no-coords outcome
- [ ] Compared candidate labels against 7d saturation percentile from `/posts/stats`
- [ ] Dropped any label flagged in `reactions_summary.less`
- [ ] Verified final labels still include at least one interest-relevant token
