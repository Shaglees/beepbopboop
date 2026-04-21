#!/usr/bin/env bash
# Source from other scripts in this directory:
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   # shellcheck disable=SC1091
#   source "$SCRIPT_DIR/load-repo-env.sh"
#
# Loads repo-root .env if present (KEY=value lines). Uses set -a so variables export.

if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
  _beepbop_repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
  if [[ -f "${_beepbop_repo_root}/.env" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "${_beepbop_repo_root}/.env"
    set +a
  fi
  unset _beepbop_repo_root
fi
