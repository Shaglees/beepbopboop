# LLM Model Comparison Testing v2

Config-driven pipeline for testing BeepBopBoop skills across LLMs. Each model runs in an isolated Docker container via Hermes Agent. Results are captured as JSON and scored across 7 quality dimensions.

## Prerequisites

- **Docker** running locally
- **Backend** running and reachable (localhost:8080 or LAN IP)
- **OpenRouter API key** at `~/.apikeys/openrouter` (or `OPENROUTER_API_KEY` env var)
- **Go toolchain** for building `beepbopgraph`
- **Config** at `~/.config/beepbopboop/config` with `BEEPBOPBOOP_API_URL` and `BEEPBOPBOOP_AGENT_TOKEN`
- **Python 3** (stdlib only, no pip install needed)

## Quick Start

```bash
# Run all pending models + generate report
bash scripts/llm_model_compare.sh

# Run a single model
bash scripts/llm_model_compare.sh --model deepseek

# Capture existing posts (no container runs) + report
bash scripts/llm_model_compare.sh --capture-only

# Report only (from existing results/*.json)
bash scripts/llm_model_compare.sh --report-only

# Report with LLM judge scoring (adds Profile, Sophistication, Image dimensions)
bash scripts/llm_model_compare.sh --report-only --compare-model anthropic/claude-sonnet-4.6

# Clean up test data
bash scripts/llm_model_compare.sh --cleanup
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `--model <key>` | Run a single model (ignores status in models.conf) |
| `--skip-setup` | Skip test user creation in DB |
| `--cleanup` | Delete all test posts and exit |
| `--parallel` | Run containers simultaneously |
| `--report-only` | Skip runs, generate report from existing `results/*.json` |
| `--capture-only` | Fetch posts for ALL models via API, save JSON, then report |
| `--compare-model <id>` | Use this OpenRouter model as LLM judge for subjective scoring |
| `-h, --help` | Show help with model list |

## Models

Models are defined in `scripts/llm-compare/models.conf` (pipe-delimited):

```
# key|openrouter_id|display_name|status
sonnet46|anthropic/claude-sonnet-4.6|Claude Sonnet 4.6|done
deepseek|deepseek/deepseek-v3.2|DeepSeek V3.2|pending
qwen-coder|qwen/qwen3-coder|Qwen3 Coder|pending
glm|z-ai/glm-4.7|GLM 4.7|pending
gemini|google/gemini-3.1-pro-preview|Gemini 3.1 Pro|pending
grok|x-ai/grok-4.1-fast|Grok 4.1 Fast|pending
minimax|minimax/minimax-m2.7|MiniMax M2.7|pending
```

- `status=pending` → model will be run when no `--model` flag is given
- `status=done` → model is skipped but its `results/*.json` is included in reports
- To add a model: add a line to `models.conf`. The script generates `docker-compose.yml` automatically.
- To re-run a completed model: change its status to `pending` or use `--model <key>`

## Quality Scoring (7 Dimensions)

### Deterministic (always computed)

| Dimension | Weight | Scope | What it measures |
|-----------|--------|-------|-----------------|
| **Rule Following** | 2x | Per-post | Labels (3-8, lowercase-hyphenated), image, valid post_type/hint, body>50ch, title |
| **Render Expectation** | 3x | Per-post | Does the post have fields the iOS card needs? Checked against `expectations.json` |
| **Content Diversity** | 2x | Per-batch | Hint diversity, ≥2 post types, label concentration ≤60%, max 3 consecutive same-type |
| **Non-Repetition** | 1x | Per-batch | Title uniqueness, body variety, label-set variety |

### LLM Judge (requires `--compare-model`)

| Dimension | Weight | Scope | What it measures |
|-----------|--------|-------|-----------------|
| **Profile Matching** | 2x | Per-batch | Alignment with test user's interests, location, family |
| **Sophistication** | 1x | Per-post | Multi-image, lat/lon, structured JSON, rich schema usage |
| **Image Quality** | 1x | Per-post | Direct URLs (not generators), quality sources |

**Total** = weighted average normalized to 0–10.

## Report Tables

**Table 1 — Model Comparison**: Side-by-side scores for all models across all dimensions.

**Table 2 — Problem Post Types**: Hints that score poorly across ALL models → indicates skill design issues (not model issues).

**Table 3 — Problem Areas**: Dimensions where models struggle most, with best/worst model per dimension.

## Test User Profile

All models get the same test user profile for fair comparison:

```json
{
  "location": "Austin, TX",
  "interests": ["sports", "food", "music", "technology"],
  "family": {"partner": true, "kids": [{"age": 4}, {"age": 7}]}
}
```

## File Structure

```
scripts/
  llm_model_compare.sh              # Config-driven orchestrator
  LLM_MODEL_COMPARE.md              # This file
  llm-compare/
    models.conf                      # Model registry (add/remove models here)
    expectations.json                # iOS render requirements per display_hint
    llm_compare_report.py            # Quality scoring + report generation
    Dockerfile                       # Hermes Agent + skills + beepbopgraph
    entrypoint.sh                    # Container entrypoint
    hermes-config.yaml               # Hermes configuration
    hermes-env                       # OpenRouter key template
    .gitignore                       # Excludes results/, docker-compose.yml, binary, skills
    docker-compose.yml               # Auto-generated from models.conf (gitignored)
    results/                         # Captured post JSON per model (gitignored)
      sonnet46.json
      deepseek.json
      ...
```

## Workflow

1. Edit `models.conf` to set which models to test
2. Run `bash scripts/llm_model_compare.sh` — builds, runs pending models, captures JSON, reports
3. Review Table 2 for skill design issues
4. Optionally re-run with `--compare-model` for subjective scoring
5. Mark completed models as `done` in `models.conf`

## Troubleshooting

**No pending models**: All models in `models.conf` have `status=done`. Change to `pending` or use `--model <key>`.

**Docker not running**: Start Docker Desktop or `dockerd`.

**Backend unreachable from container**: Containers use `host.docker.internal:host-gateway`. On Linux, ensure Docker 20.10+.

**OpenRouter rate limits**: Run sequentially (default). Retry individual models with `--model <key>`.

**Build fails on beepbopgraph**: Ensure Go is installed: `cd backend && go build ./cmd/beepbopgraph`.

**No posts appear**: Check `/tmp/bbp-run-<key>.log`. Common issues: model can't follow skill instructions, API URL wrong inside container.

**Report shows N/A for subjective scores**: Add `--compare-model <openrouter_id>` to enable LLM judge scoring.
