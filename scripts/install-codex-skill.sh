#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SKILL_NAME="using-ww-worktrees"
SOURCE_FILE="$REPO_ROOT/skills/$SKILL_NAME/SKILL.md"
CODEX_HOME_DIR="${CODEX_HOME:-$HOME/.codex}"
TARGET_DIR="$CODEX_HOME_DIR/skills/$SKILL_NAME"
TARGET_FILE="$TARGET_DIR/SKILL.md"

usage() {
  cat <<'EOF'
Usage: bash scripts/install-codex-skill.sh [--codex-home PATH]

Copies the repository's optional Codex skill template into the local Codex
skills directory.
EOF
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --codex-home)
        [ "$#" -ge 2 ] || { echo "missing value for --codex-home" >&2; exit 2; }
        CODEX_HOME_DIR="$2"
        TARGET_DIR="$CODEX_HOME_DIR/skills/$SKILL_NAME"
        TARGET_FILE="$TARGET_DIR/SKILL.md"
        shift 2
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        echo "unknown argument: $1" >&2
        exit 2
        ;;
    esac
  done
}

parse_args "$@"

if [ ! -f "$SOURCE_FILE" ]; then
  echo "skill template not found: $SOURCE_FILE" >&2
  exit 1
fi

mkdir -p "$TARGET_DIR"
cp "$SOURCE_FILE" "$TARGET_FILE"

printf 'Installed Codex skill template to %s\n' "$TARGET_FILE"
printf 'Start a new Codex session in the target repository to pick up the skill.\n'
printf 'Repository rules still come from %s\n' "$REPO_ROOT/AGENTS.md"
