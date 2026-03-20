#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$SCRIPT_DIR"
OLD_RC_MARKER_BEGIN="# wt shell wrapper begin"
OLD_RC_MARKER_END="# wt shell wrapper end"
RC_MARKER_BEGIN="# ww shell wrapper begin"
RC_MARKER_END="# ww shell wrapper end"
INSTALL_SHELL=""
RC_FILE=""
BIN_DIR="$HOME/.local/bin"

usage() {
  cat <<'EOF'
Usage: bash install.sh [--shell zsh|bash] [--rc-file PATH] [--bin-dir PATH]

Installs `ww-helper`, copies `ww.sh`, and appends a managed block that
sources the shell-first `ww` function from the chosen shell rc file.
EOF
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

collect_rc_files() {
  local -a rc_files=()
  local candidate existing seen

  for candidate in "$RC_FILE" "$HOME/.zshrc" "$HOME/.bashrc" "${ZDOTDIR:-}/.zshrc"; do
    [ -n "$candidate" ] || continue
    seen=0
    if [ "${#rc_files[@]}" -gt 0 ]; then
      for existing in "${rc_files[@]}"; do
        if [ "$existing" = "$candidate" ]; then
          seen=1
          break
        fi
      done
    fi
    [ "$seen" -eq 1 ] && continue
    rc_files+=("$candidate")
  done

  if [ "${#rc_files[@]}" -gt 0 ]; then
    printf '%s\n' "${rc_files[@]}"
  fi
}

clean_managed_blocks() {
  local rc_file

  while IFS= read -r rc_file; do
    [ -n "$rc_file" ] || continue
    strip_managed_block "$rc_file" "$OLD_RC_MARKER_BEGIN" "$OLD_RC_MARKER_END"
    strip_managed_block "$rc_file" "$RC_MARKER_BEGIN" "$RC_MARKER_END"
  done
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
  local ww_helper_bin="$2"
  local ww_shell_lib="$3"

  mkdir -p "$(dirname "$rc_file")"
  touch "$rc_file"
  strip_managed_block "$rc_file" "$OLD_RC_MARKER_BEGIN" "$OLD_RC_MARKER_END"
  strip_managed_block "$rc_file" "$RC_MARKER_BEGIN" "$RC_MARKER_END"

  {
    printf '%s\n' "$RC_MARKER_BEGIN"
    printf '%s\n' "WW_HELPER_BIN=\"$ww_helper_bin\""
    printf '%s\n' "source \"$ww_shell_lib\""
    printf '%s\n' "$RC_MARKER_END"
  } >>"$rc_file"
}

install_artifacts() {
  local helper_path="$BIN_DIR/ww-helper"
  local shell_path="$BIN_DIR/ww.sh"

  mkdir -p "$BIN_DIR"
  if [ -x "$REPO_ROOT/bin/ww-helper" ]; then
    cp "$REPO_ROOT/bin/ww-helper" "$helper_path"
    chmod +x "$helper_path"
  else
    cd "$REPO_ROOT"
    go build -buildvcs=false -o "$helper_path" ./cmd/ww-helper
  fi
  cp "$REPO_ROOT/shell/ww.sh" "$shell_path"
  chmod +x "$shell_path"
  rm -f "$BIN_DIR/wt"
  rm -f "$BIN_DIR/ww"
}

parse_args "$@"
install_artifacts
RC_TARGET="$(choose_rc_file)"
clean_managed_blocks <<EOF
$(collect_rc_files)
EOF
append_shell_wrapper "$RC_TARGET" "$BIN_DIR/ww-helper" "$BIN_DIR/ww.sh"

printf 'Installed helper binary to %s\n' "$BIN_DIR/ww-helper"
printf 'Installed shell library to %s\n' "$BIN_DIR/ww.sh"
printf 'Installed ww shell function via %s\n' "$RC_TARGET"
printf 'Updated shell rc: %s\n' "$RC_TARGET"
printf '\n'
printf 'Reload your shell first: source %s\n' "$RC_TARGET"
printf 'Use `ww` to switch the current shell directory.\n'
printf '`ww` uses fzf when available and falls back to the built-in selector.\n'
printf 'Use `ww list` to print worktrees without changing directory.\n'
printf 'Use `ww new <name>` to create and enter a new worktree.\n'
printf '`ww` changes directory in your current shell.\n'
