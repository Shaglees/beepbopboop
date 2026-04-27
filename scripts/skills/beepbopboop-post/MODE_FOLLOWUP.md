# Follow-up mode (FU1–FU3)

**Trigger:** `update on ...`, `follow up on ...`, `what's changed with ...`.

## FU1: Extract topic

Strip the trigger prefix. Keep the core topic.

## FU2: Research updates

- `WebSearch "<TOPIC> latest news <MONTH_NAME> <YEAR>"`
- `WebSearch "<TOPIC> update <MONTH_NAME> <YEAR>"`
- `WebFetch` top 2–3 results.

Focus on: what changed recently, new developments, announcements, releases.

## FU3: Generate follow-up post

Generate **1 post** framed as an update:

- Title signals update: "Three months later: …", "<TOPIC> just shipped …", "What changed with <TOPIC> since …"
- Body focuses on what's new — don't rehash the original story
- `post_type`: `article` or `discovery`
- `locality`: source name or topic area
- `latitude`/`longitude`: `null` (unless location-specific)

Then proceed to `COMMON_PUBLISH.md`.
