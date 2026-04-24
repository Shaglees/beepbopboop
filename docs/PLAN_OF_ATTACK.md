# Plan of Attack — Open Issues

_Last updated: 2026-04-24_

27 open issues, organized into 5 waves by priority and dependency.

---

## Wave 1: Bug Fixes & Build Health (ship-blocking)

Get the app into a clean, working state before adding anything new.

| # | Issue | Effort | Notes |
|---|-------|--------|-------|
| **186** | Post details fail to open in For You / Personal feeds | S | `.sheet()` modifier placement issue in paged TabView feeds. May already be fixed by PR #195 merge — verify on simulator first. |
| **187** | Basketball scores missing from feed panel view | S | Scoreboard card not rendering in feed; data is there in detail view. Likely a display_hint routing gap. |
| **205** | Fix 39 Xcode build warnings | M | 36 unnecessary try/catch (mechanical), 2 deprecated Map API, 1 UIScreen.main. No functional impact but noisy. |

**Exit criteria:** 0 bugs, 0 warnings on clean build, all card types render in feed.

---

## Wave 2: Skills Infrastructure (unblock content pipeline)

Six tightly coupled issues that fix the plumbing between skills, backend, and iOS. Do these together — they're all facets of one problem: "skills produce posts that iOS can't render correctly."

| # | Issue | Effort | Depends on | Notes |
|---|-------|--------|------------|-------|
| **201** | Single tested display-hint contract | L | — | **Do first.** Define the canonical table of all hints, their required fields, iOS model types, and sample payloads. This is the source of truth for everything below. |
| **197** | external_url JSON string vs object confusion | S | #201 | Once the contract exists, update skill docs to match. Backend stores as string; skills must `JSON.stringify()`. |
| **198** | Lint permits payloads that fail iOS decoding | M | #201 | Extend `POST /posts/lint` to validate structured payloads against the contract. Add missing field checks for FoodData, TravelData, ScienceData, etc. |
| **200** | Post router doesn't dispatch to specialty skills | M | #201 | Add dispatch table so `/post` routes to food, music, movies, pets, science, travel, fitness, celebrity, sports skills. |
| **199** | Uncovered display hints need generators | M | #200 | After routing works, audit which hints have no skill generator and either write one or mark as manual-only. |
| **203** | Preflight for config keys, CLIs, APIs | S | #200 | Add a preflight check so skills fail fast if required env vars / API keys / endpoints are missing. |
| **202** | Portable onboarding for OpenClaw runtimes | S | #203 | Make skill setup work outside the current dev machine. Lower priority — only matters for distribution. |

**Exit criteria:** Any skill can publish a post that iOS renders as a rich card with no decoding errors. Lint catches bad payloads before they hit the feed.

---

## Wave 3: Feed Architecture (user-facing quality)

Improve what users actually see when they open the app.

| # | Issue | Effort | Depends on | Notes |
|---|-------|--------|------------|-------|
| **188** | Consolidate feeds | XL | Wave 2 | Big design decision. Currently 5 feeds (Personal, Community, For You, Following, Saved). Proposal is to unify into fewer surfaces with spread logic. Needs design brainstorm before implementation. |
| **185** | User content spread settings | L | #188 | Depends on feed consolidation decision. Replaces hard-coded Omega/Beta allocation with user-configurable spread. Backend + iOS settings UI. |
| **180** | Skill review & decomposition | L | Wave 2 | Carve large skills into focused sub-skills. Specifically: extract Local Multimodal Indexer as dedicated compute engine. |

**Exit criteria:** Clear feed architecture, user-configurable content mix, skills are modular.

---

## Wave 4: New Features (growth)

Build out new capabilities once the foundation is solid.

| # | Issue | Effort | Notes |
|---|-------|--------|-------|
| **189** | Community Local News skill | L | Publication registry + recurring discovery. Needs #190 resolved first. |
| **190** | Local news rendering (article vs video vs hybrid) | M | Design decision on how news appears in feed. |
| **191** | Fashion try-on panel (OpenAI image-2) | L | Second panel in fashion card with AI-generated try-on previews. Fun but not urgent. |
| **156** | Interest-driven Calendar layer | XL | Server-side calendar events that skills populate. Big architectural addition — calendar as a parallel surface to the feed. |

**Exit criteria:** At least local news (#189/#190) shipped. Others are stretch goals.

---

## Wave 5: ML Pipeline (long-term)

These are all Weights & Biases integration issues plus the foundational ML work. They're important for feed quality long-term but don't block current UX.

| # | Issue | Effort | Notes |
|---|-------|--------|-------|
| **39** | ML-powered feed: embeddings + learned ranking | XL | Foundation. Post embeddings → user embeddings → ranking model. Partially shipped (embeddings exist, ranker exists). |
| **41** | User embedding computation | L | Aggregate engagement signals into user vectors. Worker exists but needs W&B tracking. |
| **143** | W&B: Training data pipeline metrics | M | Track data quality, volume, freshness. |
| **144** | W&B: Post embedding pipeline tracking | M | Track embedding generation, model versions, drift. |
| **145** | W&B: User embedding pipeline tracking | M | Track user vector computation, coverage, freshness. |
| **146** | W&B: Two-tower model experiment tracking | L | Track model training runs, loss curves, eval metrics. |
| **147** | W&B: Inference serving monitoring | M | Track latency, throughput, model version in production. |
| **148** | W&B: A/B test result tracking | M | Push experiment results to W&B dashboards. |
| **149** | W&B: Continuous learning pipeline | L | Auto-retrain on new data, track model drift. |
| **155** | ForYou ML / TDD hardening | M | Non-blocking follow-ups from earlier ML work. |

**Exit criteria:** W&B dashboards live for all ML subsystems. Continuous retrain loop operational.

---

## Recommended Order

```
Week 1:  Wave 1 (#186, #187, #205)
Week 2:  Wave 2 (#201 → #197 → #198 → #200 → #199 → #203 → #202)
Week 3:  Wave 3 (#188 brainstorm, #185, #180)
Week 4+: Wave 4 (#190 → #189, #191, #156)
Ongoing: Wave 5 (ML pipeline, pick off as capacity allows)
```

## Effort Key

- **S** = Small (< 1 hour)
- **M** = Medium (1-4 hours)
- **L** = Large (half day to full day)
- **XL** = Extra large (multi-day, needs design/brainstorm first)
