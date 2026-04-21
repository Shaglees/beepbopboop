# Skill refactor migration guide

Tracking issue: [#180 BeepBopBoop skill review](https://github.com/Shaglees/beepbopboop/issues/180).

This document explains the "router + mode files" pattern we adopted for large skills, how to apply it to new skills, and what's still pending.

## The pattern

Large skills suffered from one giant `SKILL.md` that bundled every mode, shared contract, and example together. Any invocation loaded all of it — even when the user only needed one mode.

The new layout:

```
.claude/skills/<skill>/
  SKILL.md              # Router: description, mode table, Step 0 config
  COMMON_PUBLISH.md     # Shared publish / dedup / label / report flow (optional, but reused across modes)
  BASE_LOCAL.md         # Shared "default path" flow for modes that fall back to it
  MODE_<name>.md        # One file per mode (each self-contained)
  FAMILY_CONTEXT.md     # Cross-cutting rule file (if applicable)
  EXAMPLES.md           # Reference examples (only read when debugging / learning)
  <REFERENCE>.md        # Long tables (e.g. SPORTS_SOURCES.md, FASHION_SOURCES.md)
  README.md             # Human docs — not loaded by the agent
```

### `SKILL.md` is the router

It contains:

1. The YAML frontmatter (`name`, `description`, `argument-hint`, `allowed-tools`).
2. One or two paragraphs of "Important" context.
3. A **file map** table: which mode lives in which file.
4. Step 0 — load configuration. This is small and shared across all modes.
5. Step 0a — parse command and route. Use a table keyed on user-input pattern that says "read `MODE_<X>.md` and follow the steps there."
6. Optionally, a brief "Publishing" section that points at `COMMON_PUBLISH.md`.

`SKILL.md` does **not** implement any mode logic. It only routes.

### Mode files

Each `MODE_*.md` is self-contained:

- Starts with the trigger list.
- Walks through that mode's steps.
- Ends with "Then proceed to `COMMON_PUBLISH.md`" (or equivalent).

The agent reads exactly one (or a few, for batch) mode files per invocation — never all of them.

### Cross-skill reuse

When multiple skills share a contract (publish envelope, dedup command, label format), **one skill owns the canonical copy** and the others reference it. We use `beepbopboop-post/COMMON_PUBLISH.md` as the owner; `beepbopboop-news` and `beepbopboop-fashion` reference it via relative paths.

If the shared contract outgrows a single skill, move it to `.claude/skills/_shared/` and have every skill reference it. (Not yet done — tracked as Phase 2.)

## What's been applied so far (Phase 1)

- `beepbopboop-post` SKILL.md: 1555 → 131 lines. Modes extracted to 9 `MODE_*.md` files plus `BASE_LOCAL.md`, `COMMON_PUBLISH.md`, `FAMILY_CONTEXT.md`, and `EXAMPLES.md`.
- `beepbopboop-news` SKILL.md: 446 → 83 lines. Modes extracted to `MODE_SOURCES.md`, `MODE_SPORTS.md`, `MODE_INTEREST.md`, `MODE_TRENDING.md`. Publish contract references `../beepbopboop-post/COMMON_PUBLISH.md`.
- `beepbopboop-fashion` SKILL.md: 441 → 118 lines. Modes extracted to `MODE_INIT.md`, `MODE_TRENDS.md`, `MODE_OUTFIT.md`, `MODE_DROPS.md`, `MODE_SEASONAL.md`, `MODE_CAPSULE.md`.

Estimated typical-case token savings per invocation:

- `beepbopboop-post`: ~18,000 tokens
- `beepbopboop-news`: ~3,000 tokens
- `beepbopboop-fashion`: ~3,000 tokens

## What's still to do (Phase 2)

Tracked as follow-up issues off of #180:

1. **Pull the "load config" blurb into `_shared/CONFIG.md`**, have every BeepBopBoop skill reference it. Currently duplicated across post / news / fashion.
2. **Pull the "publish via curl" snippet into `_shared/PUBLISH_ENVELOPE.md`**. Once every skill points here, the canonical publish contract has a single home.
3. **Split each `beepbopboop-<sport>` skill** (basketball, baseball, football, soccer) along the same pattern. Each currently has ~200 lines of ESPN-API scaffolding that duplicates logic in `beepbopboop-news/MODE_SPORTS.md`. Candidate: a shared `_shared/SPORTS_COMMON.md` + small sport-specific files.
4. **Split the design-system skills** (`harden`, `delight`, `overdrive`, `frontend-design`, `optimize`, `adapt`, `onboard`) at the section level. They aren't multi-mode, but they have long sections that could be broken into focused reference files so the agent reads only the relevant subset.
5. **Rename `INIT_WIZARD.md` → `MODE_INIT.md`** in `beepbopboop-post` (cosmetic, to match the rest of the pattern). Left as-is in Phase 1 to keep the refactor minimally invasive.

## Checklist for adding a new multi-mode skill

- [ ] `SKILL.md` is ≤ ~200 lines.
- [ ] YAML frontmatter keeps `description` action-oriented and ≤ 1 sentence.
- [ ] A file-map table lives near the top of `SKILL.md`.
- [ ] Each mode is in its own `MODE_*.md`.
- [ ] No mode implementation logic lives in `SKILL.md`.
- [ ] Shared logic (config, publish, labels) lives in exactly one file and is referenced, not copied.
- [ ] `EXAMPLES.md` holds reference examples; they do not live in `SKILL.md`.
- [ ] Cross-skill references use relative paths (`../skill-name/FILE.md`).
- [ ] `README.md` is for humans only; the agent doesn't need it.

## Verification

To re-measure sizes:

```bash
wc -l .claude/skills/*/SKILL.md | sort -n
wc -c .claude/skills/*/SKILL.md | sort -n
```

A healthy repo should have every `SKILL.md` under ~250 lines, with detail pushed to sibling `MODE_*.md` / reference files.
