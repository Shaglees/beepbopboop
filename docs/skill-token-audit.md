# BeepBopBoop Skill Token Audit

Tracking issue: [#180 BeepBopBoop skill review](https://github.com/Shaglees/beepbopboop/issues/180)

## TL;DR

- **36 skills** live in `.claude/skills/` plus the repo-level `CLAUDE.md` / `AGENTS.md`.
- The top offender is **`beepbopboop-post/SKILL.md`** at **1,555 lines / ~82 KB** — roughly **24% of every skill byte** in the repo. It is loaded in full whenever the skill is invoked, even when the user only needs one mode (e.g. `calendar`, `brief`, `digest`).
- Two other BeepBopBoop content skills — **`beepbopboop-news`** (446 lines / ~17 KB) and **`beepbopboop-fashion`** (441 lines / ~17 KB) — follow the same anti-pattern: one giant SKILL.md that bundles every mode together.
- Several design-system skills (`harden`, `delight`, `overdrive`, `frontend-design`) are 300+ lines each. They're less critical because they're invoked less often, but they're still bigger than they need to be.
- Only the **description** frontmatter of each skill is always in context. The full SKILL.md body is loaded **on invocation**, but for high-frequency skills (e.g. `beepbopboop-post` gets invoked for every post / batch run) that's effectively "always on" during sessions that do any authoring.

## How skills are loaded

Based on Cursor / Claude Code skill handling, the cost model is:

1. **Per-session, always:** the `description` (and sometimes `argument-hint`) from the frontmatter of every skill ships with the system prompt. This is cheap — all 36 descriptions together are a few hundred tokens.
2. **On invocation:** the **entire SKILL.md body** is read and added to context. Additional files in the skill dir are only read if SKILL.md explicitly points to them (`Read X when Y`).
3. **Long-lived context cost:** once a skill is invoked in a session, it stays in the conversation history until truncated. Batch/authoring sessions often invoke `beepbopboop-post` multiple times, which means the 82 KB body is effectively pinned for the rest of the session.

**Key implication:** The highest-leverage optimization is not trimming descriptions — it's making SKILL.md small enough that loading it is cheap, and pushing per-mode detail into sibling reference files that are only read when that mode is actually selected.

## Size ranking (lines, SKILL.md only)

```
  1555  beepbopboop-post
   446  beepbopboop-news
   441  beepbopboop-fashion
   356  harden
   305  delight
   267  optimize
   266  beepbopboop-movies
   261  beepbopboop-music
   256  beepbopboop-soccer
   254  beepbopboop-fitness
   247  onboard
   223  beepbopboop-travel
   212  beepbopboop-pets
   204  polish
   203  adapt
   201  beepbopboop-basketball
   197  beepbopboop-science
   196  beepbopboop-baseball
   195  beepbopboop-food
   192  beepbopboop-football
   184  clarify
   176  animate
   166  beepbopboop-celebrity
   146  frontend-design
   144  colorize
   143  overdrive
   127  audit
   126  arrange
   123  distill
   121  critique
   118  bolder
   117  typeset
   104  quieter
    93  extract
    72  normalize
    70  teach-impeccable
```

## Anti-patterns observed

### 1. "Kitchen sink" SKILL.md (biggest problem)

`beepbopboop-post/SKILL.md` defines 14+ modes (`init`, `calendar`, `batch`, `weather`, `compare`, `seasonal`, `deals`, `follow-up`, `discovery`, `digest`, `brief`, `sports`, `fashion`, default local flow) in a single file. Any invocation loads:

- Every mode's step-by-step flow (~1000 lines)
- Every display hint's contract
- Every image-source priority
- Every shell snippet
- Every writing-quality example

A typical invocation only needs **one or two** of these sections. This is the canonical target for decomposition.

### 2. Sub-skill logic copy/pasted across SKILL.md files

`beepbopboop-news`, `beepbopboop-post`, and `beepbopboop-fashion` each duplicate:

- The "load config" step (including the full list of `BEEPBOPBOOP_*` env vars)
- The "publish via curl" snippet
- The "dedup via `beepbopgraph check`" contract
- Label-format rules
- Unsplash / Imgur image snippets
- Sports schedule routing logic (news skill duplicates what `beepbopboop-post` references via `SPORTS_SOURCES.md`)

That duplication means a batch run that delegates from `beepbopboop-post` → `beepbopboop-news` pays for two copies of the publishing contract.

### 3. Inline code snippets instead of references

Many skills include long, near-identical shell or CSS snippets inline (e.g. `harden`'s text-overflow CSS, `optimize`'s lazy-load patterns). These could live in a `reference/` directory and be referenced only when the agent actually needs to emit code.

### 4. Inline examples that could be reference-only

The big BeepBopBoop skills end with 200-400 lines of "examples" sections. These are useful for one-shot reference but rarely needed when the agent already knows the pattern. Examples are prime candidates for `EXAMPLES.md` sibling files.

### 5. `beepbopboop-post` includes `SPORTS_SOURCES.md` inline references AND keeps a condensed sports list in SKILL.md

The skill already correctly points at `SPORTS_SOURCES.md` for long-form detail — good. But it also keeps a trimmed version of the same mapping inline. Cleaning this up saves a small but real chunk.

## Recommended subskill pattern

For any skill with multiple modes, adopt this layout:

```
.claude/skills/<skill>/
  SKILL.md              # <~250 lines: description, mode router, shared contracts
  COMMON_PUBLISH.md     # shared publish+dedup+labels flow (referenced from modes)
  MODE_<name>.md        # one file per mode (batch, calendar, brief, digest, ...)
  EXAMPLES.md           # reference examples, only read when debugging/onboarding
  <REFERENCE>.md        # long tables (e.g. SPORTS_SOURCES.md, FASHION_SOURCES.md)
```

Rules for `SKILL.md`:

1. Keep the description short and action-oriented (already the case).
2. Keep a **mode router table**: keyword → which `MODE_*.md` to read.
3. Keep the **shared contract** sections (config keys, publish envelope, label format) if they're truly shared across every mode — otherwise move them to `COMMON_PUBLISH.md` and have each mode read it.
4. Every mode header in SKILL.md is now a **3-5 line summary** plus `Read MODE_<name>.md and follow the steps there`.
5. No inline examples in SKILL.md beyond a single canonical one; everything else goes to `EXAMPLES.md`.

The agent still has enough context to route correctly from the description + router table. It only pays the full per-mode cost when that mode is actually selected.

## Impact estimate

If we apply this pattern to the top three skills:

| Skill                     | Current SKILL.md | Post-refactor SKILL.md | Per-invocation savings (typical) |
|---------------------------|------------------|------------------------|----------------------------------|
| `beepbopboop-post`        | 1555 lines       | ~250 lines             | ~1300 lines / ~70 KB             |
| `beepbopboop-news`        | 446 lines        | ~180 lines             | ~266 lines / ~10 KB              |
| `beepbopboop-fashion`     | 441 lines        | ~180 lines             | ~261 lines / ~10 KB              |

"Typical" = a run that only exercises one or two modes. Batch runs that touch more modes pay a little more because they read multiple `MODE_*.md` files, but the total is still well under the current one-giant-file cost because each mode file is self-contained and doesn't drag in the other modes.

## Prioritized plan

### Phase 1 (this PR): Top offender + pattern

1. Split `beepbopboop-post/SKILL.md` into a router + per-mode reference files.
2. Move shared publish/dedup/label contract into `COMMON_PUBLISH.md`.
3. Leave mode-specific rule tables inside each `MODE_*.md`.
4. Document the pattern in `docs/skill-refactor-migration.md` so future skills and contributors follow it.

### Phase 2 (follow-up issues)

5. Apply the same split to `beepbopboop-news` (HN / PH / RSS / Reddit / Substack / Sports / Trending / Interest each become a mode file).
6. Apply the same split to `beepbopboop-fashion` (Trends / Outfit / Drops / Seasonal / Capsule / Init).
7. Pull the duplicated "load config" and "publish via curl" blurbs out of all three skills and into a single top-level `.claude/skills/_shared/` directory that every BeepBopBoop skill references.
8. For design-system skills (`harden`, `delight`, `overdrive`, `frontend-design`, `optimize`, `adapt`, `onboard`), extract inline code snippets to `reference/` and keep SKILL.md to decision-making text.

### Phase 3 (optional)

9. Evaluate splitting each `beepbopboop-<sport>` skill (basketball, baseball, football, soccer) into a shared `SPORTS_COMMON.md` + small sport-specific files, since they currently duplicate ESPN API scaffolding.

## Acceptance for this issue

This audit + Phase 1 refactor is what ships in PR. Phase 2/3 get tracked as follow-up issues.

## Measurement methodology

```bash
wc -l .claude/skills/*/SKILL.md | sort -n
wc -c .claude/skills/*/SKILL.md | sort -n
```

Line / byte counts taken from `main` on 2026-04-21.

For absolute-token estimation we use the conservative rule-of-thumb `~4 chars/token`:

- `beepbopboop-post` SKILL.md ≈ 82,842 B ≈ **~21,000 tokens** per full load
- `beepbopboop-news` SKILL.md ≈ 17,178 B ≈ **~4,300 tokens**
- `beepbopboop-fashion` SKILL.md ≈ 17,493 B ≈ **~4,400 tokens**

A Phase 1 split that cuts `beepbopboop-post` to a ~10 KB router reclaims **~18,000 tokens per invocation** in the common case.
