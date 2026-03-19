#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OLD_RC_MARKER_BEGIN="# wt shell wrapper begin"
OLD_RC_MARKER_END="# wt shell wrapper end"
RC_MARKER_BEGIN="# ww shell wrapper begin"
RC_MARKER_END="# ww shell wrapper end"
INSTALL_SHELL=""
RC_FILE=""
BIN_DIR="$HOME/.local/bin"

usage() {
  cat <<'EOF'
Usage: bash uninstall.sh [--shell zsh|bash] [--rc-file PATH] [--bin-dir PATH]

Removes the installed helper binary and deletes the managed shell block that
exposes `ww`.
EOF
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --shell)
        [ "$#" -ge 2 ] || { echo "missing value for --shell" >&2; exit 2; }
        INSTALL_SHELL="$2"
        shift 2
        ;;
      --rc-file)
        [ "$#" -ge 2 ] || { echo "missing value for --rc-file" >&2; exit 2; }
        RC_FILE="$2"
        shift 2
        ;;
      --bin-dir)
        [ "$#" -ge 2 ] || { echo "missing value for --bin-dir" >&2; exit 2; }
        BIN_DIR="$2"
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

choose_rc_file() {
  if [ -n "$RC_FILE" ]; then
    printf '%s\n' "$RC_FILE"
    return
  fi

  case "$INSTALL_SHELL" in
    zsh) printf '%s\n' "$HOME/.zshrc"; return ;;
    bash) printf '%s\n' "$HOME/.bashrc"; return ;;
    "") ;;
    *)
      echo "unsupported shell: $INSTALL_SHELL" >&2
      exit 2
      ;;
  esac

  if [ -n "${ZDOTDIR:-}" ] && [ -f "${ZDOTDIR}/.zshrc" ]; then
    printf '%s\n' "${ZDOTDIR}/.zshrc"
    return
  fi

  case "${SHELL:-}" in
    */zsh)
      printf '%s\n' "$HOME/.zshrc"
      return
      ;;
    */bash)
      printf '%s\n' "$HOME/.bashrc"
      return
      ;;
  esac

  if [ -f "$HOME/.zshrc" ]; then
    printf '%s\n' "$HOME/.zshrc"
    return
  fi

  if [ -f "$HOME/.bashrc" ]; then
    printf '%s\n' "$HOME/.bashrc"
    return
  fi

  printf '%s\n' "$HOME/.zshrc"
}

strip_managed_block() {
  local rc_file="$1"
  local begin="$2"
  local end="$3"
  local tmp

  [ -f "$rc_file" ] || return 0

  tmp="$(mktemp)"
  awk -v begin="$begin" -v end="$end" '
    $0 == begin { skip = 1; next }
    $0 == end { skip = 0; next }
    skip != 1 { print }
  ' "$rc_file" >"$tmp"
  mv "$tmp" "$rc_file"
}

parse_args "$@"

rm -f "$BIN_DIR/ww" "$BIN_DIR/wt"
RC_TARGET="$(choose_rc_file)"
strip_managed_block "$RC_TARGET" "$OLD_RC_MARKER_BEGIN" "$OLD_RC_MARKER_END"
strip_managed_block "$RC_TARGET" "$RC_MARKER_BEGIN" "$RC_MARKER_END"

printf 'Removed helper binary from %s and %s\n' "$BIN_DIR/ww" "$BIN_DIR/wt"
printf 'Cleaned shell rc: %s\n' "$RC_TARGET"
