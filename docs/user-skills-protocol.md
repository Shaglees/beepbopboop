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
+----------+        (intent + frequency)     |  BeepBopBoop      |
                                             |  backend          |
                                             |                   |
                                             |  - skill-builder  |
                                             |  - storage        |
                                             |  - spread updater |
                                             |                   |
+-----------+         GET /user/profile      |                   |
| openclaw  | <----------------------------- |                   |
| (running  |   (profile.user_skills =       |                   |
|  skill,   |    [{name, version, files...}])|                   |
|  daily)   |                                |                   |
|           |   GET /skills/user/files/...   |                   |
|           | -----------------------------> +-------------------+
+-----------+
       |
       v
  .claude/skills/_user/<skill>/   (next-run effective)
```

Three runtimes:

1. **iOS app.** Captures intent + frequency (slider: every-day → every-month). Calls `POST /skills/user`.
2. **Backend (this repo, Go).** Owns the skill-builder, persistent storage, and the spread auto-updater. Returns `user_skills` on the agent variant of `/user/profile` so a running skill can install pending entries.
3. **openclaw.** Runs Claude Code daily. The shipped skills' existing `_shared/CONTEXT_BOOTSTRAP.md` step calls `/user/profile`; if `profile.user_skills` is non-empty, the running skill curls the file endpoints and writes `.claude/skills/_user/<name>/`.

The skill-builder is server-side. It runs on the backend with backend-managed credentials, not as a Claude Code skill.

## How install timing works

Claude Code scans `.claude/skills/` and loads skill descriptions into the system prompt **before** SessionStart hooks run, so a SessionStart hook that creates a new skill folder is too late for that session. We work around this by piggybacking on the profile fetch:

- A **new** user-skill folder created during a session won't be invocable until the **next** openclaw run. The user accepts a one-cycle latency on first install (see `docs/skill-prompting-playbook.md` — daily cadence).
- An **edit** to a file inside an *already-installed* user-skill folder is picked up live by Claude Code's skill-directory watcher and is effective immediately in the current session.
- An **extension** (`_user/<shipped-skill>/preferences.md`) is read by the matching shipped skill during its own context-load step, so it takes effect in the current session as long as the file is on disk before the shipped skill runs.

This means we never need a pre-launch wrapper or separate sync binary — the running skill is the installer, and the install runs as part of the bootstrap step that already calls `/user/profile`.

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
- The running skill writes to `.claude/skills/_user/` based on `profile.user_skills`. v1 is **install-only**: files in the manifest are written / updated; files or folders absent from the manifest are NOT auto-deleted (deletes are a future feature). User-side cleanup is therefore manual until the delete contract is added.
- The two namespaces never collide. A user-skill named `beepbopboop-football` lives at `_user/beepbopboop-football/` and is treated as an extension overlay (a `preferences.md` consumed by the matching shipped skill); a user-only skill named `local-hs-football` lives at `_user/local-hs-football/` as a standalone skill.

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

`POST /skills/user` uses Firebase auth (iOS user submits). The two read endpoints (`/skills/user/manifest`, `/skills/user/files/...`) and the agent variant of `/user/profile` use agent auth (openclaw reads). Request / response are JSON unless noted. Paths assume the existing API base.

### POST /skills/user

Submit user intent. Returns immediately. The current backend builds the skill synchronously (stub builder); the response shape preserves the async-ready `status` field for when the real builder lands.

**Request:**

```json
{
  "intent": "local high school football for Springfield, IL — score recaps and matchup previews",
  "kind": "standalone",
  "extends": null,
  "frequency_per_month": 14,
  "hints": {
    "location": "Springfield, IL"
  }
}
```

- `intent` (required): free-form user description, captured from the iOS intake screen.
- `kind`: `"standalone"` for a brand-new skill (#283), `"extension"` for prefs on a shipped skill (#285). Default `"standalone"`.
- `extends`: when `kind == "extension"`, the shipped-skill name (e.g. `"beepbopboop-football"`). Required for extensions, ignored otherwise.
- `frequency_per_month`: integer 1-30 from the iOS skill-builder slider (30 = "every day", 1 = "every month"). The backend uses it to allocate a slot in the user's spread on **standalone** skills (extensions don't change the spread). Missing / zero defaults to `7` (weekly). Out-of-range values are clamped.
- `hints`: optional structured side-channel for the skill-builder. Frequency lives in its own top-level field, not here.

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

## GET /user/profile (agent variant) — install trigger

The agent variant of `/user/profile` carries the user-skills manifest inline as `user_skills`. This is the **trigger and source of truth** for installs — there is no separate sync runner.

```json
{
  "identity": { "...": "..." },
  "interests": [],
  "lifestyle": [],
  "content_prefs": [],
  "profile_initialized": true,
  "user_skills": [
    {
      "name": "local-hs-football",
      "version": 3,
      "kind": "standalone",
      "extends": null,
      "files": [
        {"path": "SKILL.md",      "sha256": "...", "size": 2104},
        {"path": "MODE_brief.md", "sha256": "...", "size": 1822}
      ]
    }
  ]
}
```

The bootstrap step in `_shared/CONTEXT_BOOTSTRAP.md` interprets `user_skills` and installs each entry by fetching `/skills/user/files/{name}/{path}` (using `If-None-Match` to short-circuit unchanged files) and writing under `.claude/skills/_user/<name>/`. See that doc for the install loop.

If the field is absent or empty, there is nothing to install — the skill proceeds to its mode-specific work.

`/skills/user/manifest` (the standalone manifest endpoint) remains available for tools that want a fully detailed view (admin / debugging / iOS "what skills did I create" UIs). The piggyback on `/user/profile` is the production install path; the standalone manifest is a convenience.

## Identity binding

- iOS authenticates against the backend with Firebase. `POST /skills/user` derives `user_id` from the Firebase UID.
- openclaw authenticates with an agent token. The agent's `user_id` (via `agents.user_id`) scopes both `/user/profile` and the `/skills/user/...` reads.
- The mapping iOS-user ↔ openclaw-agent's user must be 1:1 and stable. If a user has multiple agents, they all see the same skill set.

`user_id` is always derived from auth state, never from the request body.

## Failure modes

| Failure | Behavior |
|---|---|
| iOS submit fails network | iOS retries with backoff; user sees "couldn't save, try again" |
| Backend skill-builder errors | Skill not added to manifest. Backend logs and retries on a queue. iOS may surface "still building" in a future enhancement. |
| `/user/profile` fetch fails on the agent | Bootstrap step logs and proceeds without installing — existing on-disk skills remain usable. |
| Individual file fetch fails | That file's previous on-disk copy is retained. Skill may be partially stale; logged. |
| Skill-builder produces invalid SKILL.md (missing frontmatter, etc.) | Backend rejects internally and does not bump version. `profile.user_skills` continues to advertise the previous valid version. |
| Spread auto-update fails on `POST /skills/user` | Logged but non-fatal — the skill is installed, the spread just isn't allocated a new slot. User can fix via `PUT /settings/spread`. |
| User deletes a skill in iOS | Backend deletes the row. The next `/user/profile` fetch will not list it. v1 leaves the on-disk copy in `_user/` (no auto-cleanup); a future delete contract handles removal. |

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
- **Per-file `If-None-Match` is implemented; manifest-level isn't.** `GET /skills/user/files/...` honors `If-None-Match` (sha256 ETag). If `/user/profile` becomes expensive, we could add an ETag there too; not worth it yet.
- **Async submit feedback.** Stub builder returns `status: "ready"` synchronously. When the real builder is async, `status: "queued"` is the contract; iOS may want a `GET /skills/user/<name>` poll endpoint at that point.
- **Plugin packaging interaction.** When the plugin (#281) installs / updates, it must not touch `.claude/skills/_user/`. Worth a smoke test once #281 is real.
- **Auto-delete on iOS-side delete.** v1 leaves on-disk copies after iOS-side deletes. A delete contract (likely "manifest entries gain a `deleted_at`, agent removes the folder when seen") is a clean follow-up.

## Phasing

Status as of this revision:

1. ✅ **Spec sign-off** (PR #286).
2. ✅ **Backend endpoints + storage** with stub skill-builder (PR #287).
3. ✅ **Profile piggyback + spread auto-update + bootstrap install step** (this PR). Replaces the originally planned "openclaw bootstrap sync" — there is no separate sync runner; the running skill is the installer.
4. **Real skill-builder agent** on the backend. Replaces the stub with a Claude API call that does source discovery, sibling-skill inheritance, and proper SKILL.md generation.
5. **iOS intake screen.** Slider for `frequency_per_month`, intent text field, kind picker.
6. **Conversational pref capture (#285).** Layered on top of the standalone flow.
7. **Delete contract.** Auto-cleanup on disk when a user deletes a skill in iOS (see Open questions).
