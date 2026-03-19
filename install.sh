#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$SCRIPT_DIR"
RC_MARKER_BEGIN="# ww shell wrapper begin"
RC_MARKER_END="# ww shell wrapper end"
INSTALL_SHELL=""
RC_FILE=""
BIN_DIR="$HOME/.local/bin"

usage() {
  cat <<'EOF'
Usage: bash install.sh [--shell zsh|bash] [--rc-file PATH] [--bin-dir PATH]

Installs the helper binary and appends a managed block that exposes `ww`
from the chosen shell rc file.
EOF
}

strip_managed_block() {
  local rc_file="$1"
  local tmp

  [ -f "$rc_file" ] || return 0

  tmp="$(mktemp)"
  awk -v begin="$RC_MARKER_BEGIN" -v end="$RC_MARKER_END" '
    $0 == begin { skip = 1; next }
    $0 == end { skip = 0; next }
    skip != 1 { print }
  ' "$rc_file" >"$tmp"
  mv "$tmp" "$rc_file"
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

append_shell_wrapper() {
  local rc_file="$1"

  mkdir -p "$(dirname "$rc_file")"
  touch "$rc_file"
  strip_managed_block "$rc_file"

  {
    printf '%s\n' "$RC_MARKER_BEGIN"
    printf '%s\n' "ww() {"
    printf '%s\n' "  local target"
    printf '%s\n' "  target=\"\$(command ww \"\$@\")\" || return \$?"
    printf '%s\n' "  [ -n \"\$target\" ] || return 1"
    printf '%s\n' "  cd \"\$target\" || return \$?"
    printf '%s\n' "}"
    printf '%s\n' "$RC_MARKER_END"
  } >>"$rc_file"
}

install_binary() {
  local bin_path="$BIN_DIR/ww"

  mkdir -p "$BIN_DIR"
  if [ -x "$REPO_ROOT/bin/ww" ]; then
    cp "$REPO_ROOT/bin/ww" "$bin_path"
    chmod +x "$bin_path"
    return
  fi

  cd "$REPO_ROOT"
  go build -o "$bin_path" ./cmd/wt
}

parse_args "$@"
install_binary

RC_TARGET="$(choose_rc_file)"
append_shell_wrapper "$RC_TARGET"

printf 'Installed helper binary to %s\n' "$BIN_DIR/ww"
printf 'Installed ww shell function via %s\n' "$RC_TARGET"
printf 'Updated shell rc: %s\n' "$RC_TARGET"
printf '\n'
printf 'Reload your shell first: source %s\n' "$RC_TARGET"
printf 'Use `ww` to switch the current shell directory.\n'
printf 'Use `ww --fzf` for fzf selection.\n'
printf '`ww` changes directory in your current shell.\n'
