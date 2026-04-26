# LLM Model Comparison Pipeline

End-to-end testing pipeline that runs the BeepBopBoop posting skill across multiple LLMs and scores the output quality. Each model runs in an isolated Docker container with [Hermes Agent](https://github.com/anthropics/hermes), produces posts via the beepbopboop-post skill, and gets scored across 7 quality dimensions.

## How It Works

```
models.conf → docker-compose.yml → Docker containers (1 per model)
                                         ↓
                                    Hermes Agent + beepbopboop-post skill
                                         ↓
                                    Posts via backend API
                                         ↓
                                    results/{model}.json
                                         ↓
                                    llm_compare_report.py → scored report
```

1. **Setup**: Script reads `models.conf`, generates `docker-compose.yml`, creates test users in the DB with identical profiles (Austin, TX; interests: sports, food, music, technology)
2. **Run**: Each model gets its own Docker container running Hermes with the beepbopboop-post skill. The skill fetches the user profile from the server API (`GET /user/profile`) and generates 8-10 posts.
3. **Capture**: After each container finishes (or times out), posts are fetched via the API and saved as JSON.
4. **Score**: A Python report scores each model's posts across 7 dimensions and generates 3 comparison tables.

## Prerequisites

| Requirement | Why | Check |
|-------------|-----|-------|
| Docker | Containers for each model | `docker info` |
| Backend running | API for post creation + user profiles | `curl http://<IP>:8080/posts` |
| OpenRouter API key | Routes to different LLM providers | `~/.apikeys/openrouter` or `OPENROUTER_API_KEY` env |
| Go toolchain | Builds `beepbopgraph` binary for containers | `go version` |
| Python 3 | Report generation (stdlib only) | `python3 --version` |
| BBP config | Backend connection details | `~/.config/beepbopboop/config` |

The BBP config file needs at minimum:
```
BEEPBOPBOOP_API_URL=http://<your-lan-ip>:8080
BEEPBOPBOOP_AGENT_TOKEN=<your-agent-token>
```

## Quick Start

```bash
# 1. Make sure backend + DB are running
cd backend && docker-compose up -d

# 2. Run all pending models
bash scripts/llm_model_compare.sh

# 3. Or run just one model to test
bash scripts/llm_model_compare.sh --model sonnet46
```

## Usage

### Run all pending models
```bash
bash scripts/llm_model_compare.sh
```
Runs every model in `models.conf` with `status=pending`, captures results, generates report.

### Run a single model
```bash
bash scripts/llm_model_compare.sh --model deepseek
```
Ignores status — always runs the specified model.

### Generate report from existing results
```bash
bash scripts/llm_model_compare.sh --report-only
```
No containers started. Uses whatever JSON files exist in `results/`.

### Report with LLM judge scoring
```bash
bash scripts/llm_model_compare.sh --report-only --compare-model anthropic/claude-haiku-4.5-20251001
```
Adds 3 subjective dimensions (Profile Matching, Sophistication, Image Quality) scored by the judge model via OpenRouter.

### Capture posts without running containers
```bash
bash scripts/llm_model_compare.sh --capture-only
```
Fetches posts for ALL models (including `status=done`) from the API and saves to `results/`. Useful when posts already exist from a previous run.

### Clean up test data
```bash
bash scripts/llm_model_compare.sh --cleanup
```
Deletes all test posts from the database.

### Skip test user setup
```bash
bash scripts/llm_model_compare.sh --model sonnet46 --skip-setup
```
Skips DB inserts if test users already exist from a previous run.

### Set timeout per model
```bash
MODEL_TIMEOUT=900 bash scripts/llm_model_compare.sh --model deepseek
```
Default is 600 seconds (10 min). Some models need longer.

## CLI Reference

| Flag | Description |
|------|-------------|
| `--model <key>` | Run a single model (ignores status in models.conf) |
| `--skip-setup` | Skip test user creation in DB |
| `--cleanup` | Delete all test posts from DB and exit |
| `--parallel` | Run containers simultaneously (experimental) |
| `--report-only` | Skip runs, generate report from existing `results/*.json` |
| `--capture-only` | Fetch posts for ALL models via API, save JSON, then report |
| `--compare-model <id>` | OpenRouter model ID for LLM judge scoring |
| `-h, --help` | Show help with model list |

## Managing Models

Models are defined in `models.conf`:

```
# key|openrouter_id|display_name|status
sonnet46|anthropic/claude-sonnet-4.6|Claude Sonnet 4.6|done
deepseek|deepseek/deepseek-v3.2|DeepSeek V3.2|pending
```

- **`pending`** — model will be run on next execution
- **`done`** — model is skipped but results are included in reports
- **To add a model**: add a line. The script auto-generates `docker-compose.yml`.
- **To re-run a completed model**: change status to `pending` or use `--model <key>`
- **To remove a model**: delete the line

### Token computation

Each model gets a deterministic test token: `bbp_test_{key}_` padded with zeros to exactly 64 characters. The script computes the SHA-256 hash and inserts it into the DB. You don't need to manage tokens manually.

## Quality Scoring

### Deterministic dimensions (always computed)

| Dimension | Weight | What it checks |
|-----------|--------|----------------|
| Rule Following | 2x | 3-8 labels, lowercase-hyphenated, has image, valid post_type + display_hint, body > 50 chars, has title |
| Render Expectation | 3x | Does the post have all fields the iOS card needs? Validated against `expectations.json` |
| Content Diversity | 2x | Hint variety, ≥2 post types, label concentration ≤60%, max 3 consecutive same-type |
| Non-Repetition | 1x | Unique titles, varied bodies, varied label sets |

### LLM judge dimensions (requires `--compare-model`)

| Dimension | Weight | What it checks |
|-----------|--------|----------------|
| Profile Matching | 2x | Posts align with test user's location, interests, family |
| Sophistication | 1x | Multi-image, lat/lon, structured JSON, external_url usage |
| Image Quality | 1x | Direct URLs (not generators), quality sources (Wikimedia, Unsplash) |

### Report tables

**Table 1 — Model Comparison**: Side-by-side scores across all dimensions.

**Table 2 — Problem Post Types**: Hints that score poorly across ALL models — signals skill documentation issues (not model issues). If every model fails at a hint, the skill description for that hint needs improvement.

**Table 3 — Problem Areas**: Which dimensions are hardest, with best/worst model per dimension.

## Render Expectations

`expectations.json` maps each `display_hint` to the fields the iOS app needs to render that card type. For structured hints (scoreboard, matchup, standings, etc.), the post's `external_url` must contain valid JSON with specific keys.

Example: a `scoreboard` post needs:
```json
{
  "external_url": "{\"sport\":\"basketball\",\"league\":\"NBA\",\"status\":\"Final\",\"home\":{\"name\":\"...\",\"score\":110,\"logo\":\"...\"},\"away\":{...}}"
}
```

If the model doesn't produce the right JSON structure, the iOS app falls back to a generic `StandardCard` — wasted hint.

## Test User Profile

All models get the same profile for fair comparison:

```
Location:  Austin, TX (30.2672, -97.7431)
Interests: sports, food, music, technology
```

The profile is stored server-side (in `users`, `user_settings`, `user_interests` tables) and the skill fetches it via `GET /user/profile` at runtime (Step 0a in CONFIG.md).

## File Structure

```
scripts/
  llm_model_compare.sh              # Main orchestrator
  LLM_MODEL_COMPARE.md              # Feature overview doc
  llm-compare/
    README.md                        # This file
    models.conf                      # Model registry
    expectations.json                # iOS render requirements per display_hint
    llm_compare_report.py            # Scoring engine + report tables
    Dockerfile                       # Hermes Agent + skills image
    entrypoint.sh                    # Container entrypoint (config + launch)
    hermes-config.yaml               # Hermes base config
    hermes-env                       # OpenRouter key template
    .gitignore                       # Excludes results/, docker-compose.yml, binary
    docker-compose.yml               # Auto-generated (gitignored)
    results/                         # Post JSON per model (gitignored)
```

## Troubleshooting

### Backend unreachable from container
Containers use `host.docker.internal` to reach the host. Verify your backend is listening on `0.0.0.0:8080` (not just `127.0.0.1`). On Linux, requires Docker 20.10+.

### Model produces 0 posts
Check the container log at `/tmp/bbp-run-<key>.log`. Common causes:
- **Hermes security scanner blocking**: Fixed by `--yolo` flag in entrypoint. If you see "dotfile overwrite" or "plain HTTP URL" blocks, the `--yolo` flag is missing.
- **Model can't follow Step 0a**: Less capable models may not understand the instruction to fetch profile from server API. The entrypoint logs the profile it found — if it shows `location=(none)`, the API call failed.
- **API URL wrong inside container**: Should be `http://host.docker.internal:8080`, not `localhost`.

### Token authentication errors
The script auto-generates tokens and inserts SHA-256 hashes into the DB. If you get `{"error":"invalid or revoked token"}`, the hash may be stale. Run without `--skip-setup` to re-create test users.

### Container hangs past timeout
The default timeout is 600s (10 min). Set `MODEL_TIMEOUT=900` for slower models. The script kills the container and proceeds to capture whatever posts were created.

### Report shows "N/A" for some dimensions
Profile Matching, Sophistication, and Image Quality require `--compare-model <id>`. Without it, only deterministic dimensions are scored.

### `beepbopgraph` build fails
Ensure Go is installed and you're on the right branch:
```bash
cd backend && go build ./cmd/beepbopgraph
```

### Database container not found
The script expects the DB container to be named `backend-db-1`. If yours differs:
```bash
docker ps | grep postgres  # find the actual name
```

### macOS: `timeout` command not found
Not an issue — the script uses a portable bash wait loop instead of `timeout`/`gtimeout`.

## Known Limitations

- **No `DELETE /posts` API** (#241) — cleanup uses direct DB deletes, which requires the DB container to be accessible
- **No programmatic user creation API** (#236) — test users are created via raw SQL inserts
- **Hermes `-Q` quiet mode** (#240) — no stdout from the agent, making debugging hard. Check `/tmp/bbp-run-<key>.log` for entrypoint output only.
- **Less capable models struggle with Step 0a** (#235) — DeepSeek V3.2 produced 0 posts when required to fetch profile from server. May need to pre-populate config for weaker models as a fallback.
