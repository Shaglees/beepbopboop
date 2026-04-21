# Digest mode (DG1–DG3)

**Trigger:** `digest`, `roundup`, `weekly digest`, `summary`, or from batch Phase 2.

A digest is a multi-item roundup — "5 AI Developments This Week" or "Your Local Scene This Weekend". iOS CompactCard renders each line as a numbered row.

## DG1: Pick a digest topic

Pick one based on available data:

| Topic type | Example title | Data source |
|---|---|---|
| Interest-based roundup | "5 AI Developments This Week" | `BEEPBOPBOOP_INTERESTS` + WebSearch |
| Local scene roundup | "Your Local Scene This Weekend" | Event/place data from location |
| Sports roundup | "What Your Teams Did This Week" | `BEEPBOPBOOP_SPORTS_TEAMS` + ESPN API |
| Mixed weekly digest | "This Week in Your World" | Combine 2–3 sources |

Pick the topic with the richest data available. If the user hinted (`digest AI`), bias that direction.

## DG2: Research 4–7 items

- Each item = one line in the body (newline-separated)
- Self-contained nugget: title + one-sentence summary or key detail
- Scannable — the reader gets value just from skimming

**Body format example:**

```
Claude 4.5 scores 94% on ARC-AGI — Anthropic's latest reasoning model sets a new benchmark
Google DeepMind open-sources Gemma 3 with 2B parameter model — runs on a laptop
OpenAI ships GPT-5 Turbo with 2M context window — 10x previous limit
Meta releases Llama 4 Scout with mixture-of-experts — 109B active params from 400B total
Mistral launches Le Chat Enterprise with on-prem deployment — targeting regulated industries
```

## DG3: Publish digest post

- `display_hint: "digest"` — iOS renders as numbered compact rows.
- `post_type: "article"`
- `visibility: "public"`
- Labels include `digest`, the topic area, relevant sub-topics.
- Title signals a roundup: numbers, timeframes, or scope ("Your Local Scene…").

Then proceed to `COMMON_PUBLISH.md`.
