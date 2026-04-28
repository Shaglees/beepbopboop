# User skills protocol

Tracking issues:
- [#283 niche skill creation flow](https://github.com/Shaglees/beepbopboop/issues/283)
- [#282 user skills survive plugin updates](https://github.com/Shaglees/beepbopboop/issues/282)
- [#285 conversational user prefs](https://github.com/Shaglees/beepbopboop/issues/285)
- [#281 plugin packaging](https://github.com/Shaglees/beepbopboop/issues/281)
- [#284 post-skill provenance tag](https://github.com/Shaglees/beepbopboop/issues/284)

This document is the shared contract between three codebases — the iOS app, the BeepBopBoop backend, and openclaw — for creating user-owned skills from inside the app and getting them loaded by the agent. It does not prescribe internal implementation; it pins down the wire format, storage layout, and lifecycle so each side can build in parallel.

## Goals

- A user can describe a niche need in the iOS app ("local high school football for [town]"), and a working skill exists in their next openclaw cycle.
- User-authored skills and conversational preferences (#285) are owned by the user and survive shipped-skill updates (#282).
- Generation runs server-side; the iOS app is thin; openclaw stays untouched between cycles.
- Sync is async by design — no realtime expectation.

## Non-goals

- Realtime feedback ("see your skill working immediately"). User sees results next cycle.
- In-app skill editing of raw `SKILL.md`. Capture is conversational / structured intake; the file is generated.
- A general-purpose plugin marketplace. This is scoped to the user's own skills.
- Cross-user sharing. Out of scope for v1.

## Actors

```
+----------+        POST /skills/user        +-------------------+
|  iOS app | ------------------------------> |                   |
+----------+                                 |  BeepBopBoop      |
                                             |  backend          |
                                             |                   |
                                             |  - skill-builder  |
                                             |  - storage        |
                                             |  - manifest       |
                                             |                   |
+-----------+   GET /skills/user/manifest    |                   |
|  openclaw | <----------------------------- |                   |
| bootstrap |   GET /skills/user/files/...   |                   |
+-----------+ -----------------------------> +-------------------+
       |
       v
  .claude/skills/_user/<skill>/
       |
       v
  launches Claude Code
```

Three runtimes:

1. **iOS app.** Captures intent. Calls the backend.
2. **Backend (this repo, Go).** Owns the skill-builder agent (server-side Claude API call), persistent storage, and the manifest endpoint.
3. **openclaw.** Pulls the user's skill set into `.claude/skills/_user/` at bootstrap, then launches Claude Code.

The skill-builder is a server-side agent, not a Claude Code skill. It runs on the backend with backend-managed credentials.

## The hard constraint

Claude Code reads every `SKILL.md` in `.claude/skills/` at process start and inserts the frontmatter `description` into the system prompt. Files written into `.claude/skills/` *after* the process has started are invisible until the next launch.

Therefore: **the sync must run before Claude Code starts, not from inside it.** A SessionStart hook is too late. openclaw's bootstrap sequence has to be:

```
openclaw bootstrap:
  1. authenticate as user (agent token)
  2. sync user skills from backend into .claude/skills/_user/
  3. launch Claude Code
```

This is the single most important constraint and dictates almost every other decision below.

## Storage layout (on-disk, openclaw side)

```
.claude/skills/                  # plugin-owned (#281)
  beepbopboop-post/
  beepbopboop-football/
  ...
.claude/skills/_user/            # user-owned, never touched by plugin install
  <user-skill-name>/             # full user-authored skill (#283)
    SKILL.md
    MODE_*.md
    reference/
  beepbopboop-football/          # extension to a shipped skill (#285)
    preferences.md               # auto-loaded after the shipped SKILL.md
```

Rules:

- Plugin install (#281) owns everything under `.claude/skills/` *except* `_user/`.
- Sync owns everything under `.claude/skills/_user/`. Files not present in the manifest are deleted on sync.
- The two namespaces never collide. A user skill named `beepbopboop-football` lives at `_user/beepbopboop-football/` and is treated as an extension overlay; a user-only skill named `local-hs-football` lives at `_user/local-hs-football/` as a standalone skill.

## Storage layout (backend side)

Backend is the source of truth for `_user/`. Suggested model — final shape is the backend's call:

```
user_skills
  user_id          uuid     # owner
  skill_name       text     # e.g. "local-hs-football" or "beepbopboop-football"
  version          integer  # bumped on each successful skill-builder run
  kind             enum     # "standalone" | "extension"
  created_at       timestamp
  updated_at       timestamp

user_skill_files
  user_id          uuid
  skill_name       text
  path             text     # relative path under the skill dir, e.g. "SKILL.md", "MODE_brief.md"
  sha256           text
  size_bytes       integer
  content          bytea    # or object storage URL
  updated_at       timestamp
```

A user skill is a `(user_id, skill_name)` pair. Files live underneath. Version is bumped atomically when the skill-builder writes a new revision.

## API contract

All three endpoints are authenticated with the existing agent token mechanism. Request/response are JSON unless noted. Paths assume the existing API base.

### POST /skills/user

Submit user intent. Returns immediately; generation is async.

**Request:**

```json
{
  "intent": "local high school football for Springfield, IL — score recaps and matchup previews",
  "kind": "standalone",
  "extends": null,
  "hints": {
    "location": "Springfield, IL",
    "frequency": "weekly"
  }
}
```

- `intent` (required): free-form user description, captured from the iOS intake screen.
- `kind`: `"standalone"` for a brand-new skill (#283), `"extension"` for prefs on a shipped skill (#285). Default `"standalone"`.
- `extends`: when `kind == "extension"`, the shipped skill name (e.g. `"beepbopboop-football"`). Required for extensions, ignored otherwise.
- `hints`: optional structured side-channel from the intake screen. Backend treats these as additional context for the skill-builder.

**Response (202 Accepted):**

```json
{
  "skill_name": "local-hs-football",
  "status": "queued",
  "submitted_at": "2026-04-28T18:02:00Z"
}
```

- `status`: `"queued"` initially. Polling not required — openclaw discovers the skill via the manifest endpoint at next bootstrap.
- The backend may rename the user's intended skill name to avoid collisions; the response is authoritative.

**Failure cases:**

- `400` — intent missing or empty.
- `400` — `kind == "extension"` with `extends` referring to a non-existent shipped skill.
- `409` — user has hit the per-user skill cap (TBD; suggest 50 for v1).
- `429` — rate limit (suggest 5 submissions per user per hour for v1).

### GET /skills/user/manifest

Lists every user skill the caller owns, with file-level metadata.

**Response (200):**

```json
{
  "user_id": "u_abc123",
  "skills": [
    {
      "name": "local-hs-football",
      "version": 3,
      "kind": "standalone",
      "extends": null,
      "files": [
        {"path": "SKILL.md",       "sha256": "...", "size": 2104},
        {"path": "MODE_recap.md",  "sha256": "...", "size": 1822},
        {"path": "reference/sources.md", "sha256": "...", "size": 940}
      ]
    },
    {
      "name": "beepbopboop-football",
      "version": 1,
      "kind": "extension",
      "extends": "beepbopboop-football",
      "files": [
        {"path": "preferences.md", "sha256": "...", "size": 312}
      ]
    }
  ]
}
```

- Manifest is authoritative for the `_user/` namespace. Skills not listed must be deleted from disk on sync.
- `version` is monotonically increasing per skill. openclaw can use it as a quick check before re-fetching files, but the canonical change signal is `sha256` per file.
- A skill that is still being built (queued or in-progress) **must not appear in the manifest** until at least one valid file revision exists. No half-synced skills.

### GET /skills/user/files/{skill_name}/{path}

Fetches a single file by path. `path` may contain `/` (URL-encoded).

**Response (200):**

- `Content-Type: text/markdown` (or whatever the file's actual type is).
- `ETag: "<sha256>"` so openclaw can avoid re-downloading unchanged files on subsequent syncs.
- Body: the raw file contents.

**Failure cases:**

- `404` — skill or file does not exist.
- `403` — caller does not own the skill.

## Sync protocol (openclaw side)

```
sync():
  manifest = GET /skills/user/manifest
  on_disk  = scan(.claude/skills/_user/)

  for each skill in manifest.skills:
    target_dir = .claude/skills/_user/<skill.name>/
    for each file in skill.files:
      local = on_disk[skill.name][file.path]
      if local is None or local.sha256 != file.sha256:
        body = GET /skills/user/files/<skill.name>/<file.path>
        atomically write target_dir/<file.path> with body
    delete any local files under target_dir not present in skill.files

  for each skill on disk not in manifest.skills:
    rm -rf .claude/skills/_user/<skill_name>/

  if any sync step failed:
    log error, leave on-disk state as-is, proceed to launch Claude Code
```

Atomicity:

- File writes are write-to-temp-then-rename within the same directory.
- Whole-skill deletion happens only after manifest fetch succeeds. If the manifest call fails, openclaw proceeds with the previous on-disk state (degraded but functional).
- openclaw never touches anything outside `.claude/skills/_user/`.

Sync runs once per openclaw bootstrap. There is no in-cycle re-sync.

## Identity binding

- iOS authenticates against the backend with the user's account credentials and acquires an agent token (existing mechanism per `.claude/connection-details.md`).
- The same agent token is provisioned to openclaw at bootstrap, scoped to the same user.
- All three endpoints accept the agent token via the existing auth middleware. The user_id is derived from the token, never from the request body.

The mapping iOS-user ↔ openclaw-user must be 1:1 and stable. If a user has multiple openclaw instances (rare, but possible), they all share the same skill set.

## Failure modes

| Failure | Behavior |
|---|---|
| iOS submit fails network | iOS retries with backoff; user sees "couldn't save, try again" |
| Backend skill-builder errors | Skill not added to manifest. Backend logs and retries on a queue. iOS may surface "still building" in a future enhancement. |
| openclaw manifest fetch fails | Bootstrap proceeds with previous on-disk state. Logged. |
| openclaw individual file fetch fails | That file's previous on-disk version is retained. Skill may be partially stale; logged. |
| Skill-builder produces invalid SKILL.md (missing frontmatter, etc.) | Backend rejects internally and does not bump version. Manifest continues to point at the previous valid version. |
| User deletes a skill in iOS | Backend removes from `user_skills`. Manifest no longer lists it. openclaw deletes on next sync. |

The system is built around "no half-synced state ever appears in the manifest." Validity is enforced at the backend before the manifest changes.

## Out of scope (for this protocol)

- The skill-builder agent's internals — prompt design, source-discovery logic, sibling-skill inheritance. That is a backend implementation detail.
- The iOS intake UI. The protocol only specifies what gets sent on submit.
- Conflict resolution between a user extension and a shipped-skill update (e.g. user's `preferences.md` references a source the shipped skill no longer uses). Tracked separately in #285.
- Engagement attribution / post provenance. Tracked in #284.
- Cross-user sharing of user skills. Future work.

## Open questions

- **Per-user cap.** What's a reasonable upper bound on user skills? Suggest 50.
- **File size cap.** Per-file? Per-skill? Suggest 64 KB per file, 256 KB per skill, to keep openclaw context manageable.
- **Manifest caching.** Should the manifest endpoint support `If-None-Match` so openclaw can skip the body when nothing changed? Cheap to add and worth it for sessions where nothing has changed.
- **Async submit feedback.** v1 returns `queued` and never tells iOS when ready. Should there be a `GET /skills/user/<name>` for status polling, or do we lean on next-cycle visibility as the only signal? Lean: skip for v1, add if iOS UX demands it.
- **Plugin packaging interaction.** When the plugin (#281) installs / updates, it must respect `.claude/skills/_user/`. Worth a smoke test in CI.
- **Pre-launch hook in openclaw.** This protocol assumes openclaw exposes a pre-launch step we can wire sync into. If not, this whole design needs a different shape — confirm before implementing the openclaw side.

## Phasing

Suggested order:

1. **Spec sign-off** (this doc).
2. **Backend endpoints + storage**, with a stub skill-builder that produces a fixed example skill. End-to-end wiring without the AI piece.
3. **openclaw bootstrap sync** against the stubbed backend. Verifies the on-disk lifecycle.
4. **Real skill-builder agent** on the backend.
5. **iOS intake screen.**
6. **Conversational pref capture (#285)** layered on top once the standalone-skill flow is solid.
