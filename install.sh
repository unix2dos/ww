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

Installs the helper binary and appends a managed block that exposes `ww`
from the chosen shell rc file.
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
  local ww_bin="$2"

  mkdir -p "$(dirname "$rc_file")"
  touch "$rc_file"
  strip_managed_block "$rc_file" "$OLD_RC_MARKER_BEGIN" "$OLD_RC_MARKER_END"
  strip_managed_block "$rc_file" "$RC_MARKER_BEGIN" "$RC_MARKER_END"

  {
    printf '%s\n' "$RC_MARKER_BEGIN"
    printf '%s\n' "ww() {"
    printf '%s\n' "  local ww_bin=\"$ww_bin\""
    printf '%s\n' "  local target"
    printf '%s\n' "  target=\"\$(\"\$ww_bin\" \"\$@\")\""
    printf '%s\n' "  local status=\$?"
    printf '%s\n' "  if [ \$status -ne 0 ]; then"
    printf '%s\n' "    [ -n \"\$target\" ] && printf '%s\\n' \"\$target\""
    printf '%s\n' "    return \$status"
    printf '%s\n' "  fi"
    printf '%s\n' "  case \"\$target\" in"
    printf '%s\n' "    *\$'\\n'*)"
    printf '%s\n' "      [ -n \"\$target\" ] && printf '%s\\n' \"\$target\""
    printf '%s\n' "      return 0"
    printf '%s\n' "      ;;"
    printf '%s\n' "  esac"
    printf '%s\n' "  if [ -n \"\$target\" ] && [ -d \"\$target\" ]; then"
    printf '%s\n' "    cd \"\$target\" || return \$?"
    printf '%s\n' "    return 0"
    printf '%s\n' "  fi"
    printf '%s\n' "  [ -n \"\$target\" ] && printf '%s\\n' \"\$target\""
    printf '%s\n' "  return 0"
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
  else
    cd "$REPO_ROOT"
    go build -o "$bin_path" ./cmd/wt
  fi
  rm -f "$BIN_DIR/wt"
}

parse_args "$@"
install_binary
RC_TARGET="$(choose_rc_file)"
clean_managed_blocks <<EOF
$(collect_rc_files)
EOF
append_shell_wrapper "$RC_TARGET" "$BIN_DIR/ww"

printf 'Installed helper binary to %s\n' "$BIN_DIR/ww"
printf 'Installed ww shell function via %s\n' "$RC_TARGET"
printf 'Updated shell rc: %s\n' "$RC_TARGET"
printf '\n'
printf 'Reload your shell first: source %s\n' "$RC_TARGET"
printf 'Use `ww` to switch the current shell directory.\n'
printf 'Use `ww --fzf` for fzf selection.\n'
printf '`ww` changes directory in your current shell.\n'
