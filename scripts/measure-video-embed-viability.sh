#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INPUT_FILE="${1:-$ROOT/scripts/video-embed-sample-urls.txt}"

JSON_OUT="${TMPDIR:-/tmp}/video-embed-spike.json"
MD_OUT="${TMPDIR:-/tmp}/video-embed-spike.md"

pushd "$ROOT/backend" >/dev/null
go run ./cmd/videoembedspike --input-file "$INPUT_FILE" --format json >"$JSON_OUT"
popd >/dev/null

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required" >&2
  exit 1
fi

sample_size="$(jq '.sample_size' <"$JSON_OUT")"
recommendation="$(jq -r '.recommendation' <"$JSON_OUT")"
provider_count="$(jq '.providers | length' <"$JSON_OUT")"

if [[ "$sample_size" -le 0 ]]; then
  echo "error: sample_size must be > 0" >&2
  exit 1
fi
if [[ -z "$recommendation" || "$recommendation" == "null" ]]; then
  echo "error: recommendation missing" >&2
  exit 1
fi
if [[ "$provider_count" -le 0 ]]; then
  echo "warning: no providers reported (all samples may be no_live_embed/error)" >&2
fi

{
  echo "# Video Embed Spike"
  echo
  echo "- Sample size: $sample_size"
  echo "- No live embed pages: $(jq '.no_live_embed_count' <"$JSON_OUT")"
  echo "- Recommendation: $recommendation"
  echo
  echo "## Provider Matrix"
  jq -r '.providers | to_entries[] | "- `\(.key)`: ok=\(.value.ok) blocked=\(.value.blocked) gone=\(.value.gone) unknown=\(.value.unknown) preview_cap=\(.value.policy.supports_preview_cap) fallback=\(.value.policy.fallback_behavior)"' <"$JSON_OUT"
} >"$MD_OUT"

echo "JSON report: $JSON_OUT"
cat "$JSON_OUT"
echo
echo "Markdown report: $MD_OUT"
cat "$MD_OUT"
