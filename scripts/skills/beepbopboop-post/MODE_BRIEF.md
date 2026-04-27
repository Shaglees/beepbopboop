# Brief mode (BR1–BR3)

**Trigger:** `brief`, `morning brief`, `daily brief`, `today's take`, or from batch Phase 2.

A brief is a compact, multi-topic snapshot — "Morning Brief" or "Weekend Ahead". iOS CompactCard renders each line as a bullet row.

## BR1: Compose brief bullets

Compose 3–5 concise bullets covering a mix of:
- **Today's weather take** — editorial, not forecast ("Light jacket territory — 14°C but clearing by noon").
- **One local event worth knowing** — something happening today/this week.
- **One trending topic or interest item** — from `BEEPBOPBOOP_INTERESTS` or current news.
- **One discovery / surprise** — a new opening, an adjacent-interest nugget, a local fact.

Each bullet is self-contained and scannable. No bullet exceeds ~100 characters.

## BR2: Format body

Each bullet = one line in the body (newline-separated).

**Body format example:**

```
Light jacket weather — 15°C and clearing by noon
Temple Bar Food Market has a new Basque cheesecake stall worth the queue
Claude 4.5 dropped overnight — 94% on ARC-AGI, your AI workflow just got faster
The Long Room at Trinity is free entry this week for Dublin residents
```

## BR3: Publish brief post

- `display_hint: "brief"` — iOS renders as bullet-point compact rows.
- `post_type: "discovery"`
- `visibility: "public"` (unless bullets reference family or home location)
- Labels cover the topics mentioned (`weather`, `food`, `ai`, `local-events`, etc.)
- Title: "Morning Brief", "Today's Take", "Weekend Ahead" — short and recurring.

Then proceed to `COMMON_PUBLISH.md`.
