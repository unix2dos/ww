ww_print_help() {
  cat <<'EOF'
Usage:
  ww [switch] [<index>|<name>]
  ww list
  ww new <name>
  ww rm [--force] [<name>]
  ww version
  ww help

Commands:
  switch   Select a worktree and change into it. Default when omitted.
  list     Print worktrees without changing directory.
  new      Create a worktree under ./.worktrees/<name> and enter it.
  rm       Remove one worktree.
  version  Print the binary and protocol version.
  help     Show this help.

Notes:
  Uses fzf automatically when available.
  Falls back to the built-in selector otherwise.

Examples:
  ww
  ww 2
  ww switch feat-a
  ww new feat-demo
  ww rm feat-demo
  ww --version
EOF
}

ww_has_json_flag() {
  local arg
  for arg in "$@"; do
    if [ "$arg" = "--json" ]; then
      return 0
    fi
  done
  return 1
}

ww_is_helper_only_new_flag() {
  case "${1-}" in
    --label|--label=*|--ttl|--ttl=*|--message|--message=*|-m)
      return 0
      ;;
  esac
  return 1
}

ww_validate_human_new_args() {
  local arg
  for arg in "$@"; do
    if ww_is_helper_only_new_flag "$arg"; then
      printf '%s\n' "ww new only supports creating a worktree." >&2
      printf '%s\n' "For metadata-aware automation, use ww-helper new-path ..." >&2
      return 2
    fi
  done
  return 0
}

ww() {
  local ww_helper_bin="${WW_HELPER_BIN:-ww-helper}"
  local target

  case "${1-}" in
    "" )
      target="$("$ww_helper_bin" switch-path)" || return $?
      ;;
    switch)
      shift
      if ww_has_json_flag "$@"; then
        "$ww_helper_bin" switch-path "$@"
        return $?
      fi
      target="$("$ww_helper_bin" switch-path "$@")" || return $?
      ;;
    new)
      shift
      ww_validate_human_new_args "$@" || return $?
      if ww_has_json_flag "$@"; then
        "$ww_helper_bin" new-path "$@"
        return $?
      fi
      target="$("$ww_helper_bin" new-path "$@")" || return $?
      ;;
    list)
      shift
      "$ww_helper_bin" list "$@"
      return $?
      ;;
    rm)
      "$ww_helper_bin" "$@"
      return $?
      ;;
    gc)
      printf '%s\n' "ww gc is not part of the human shell workflow." >&2
      printf '%s\n' "Use ww rm --cleanup for interactive cleanup, or ww-helper gc for automation." >&2
      return 2
      ;;
    help|-h|--help)
      ww_print_help
      return 0
      ;;
    version|-v|--version)
      shift
      "$ww_helper_bin" version "$@"
      return $?
      ;;
    *)
      if ww_has_json_flag "$@"; then
        "$ww_helper_bin" switch-path "$@"
        return $?
      fi
      target="$("$ww_helper_bin" switch-path "$@")" || return $?
      ;;
  esac

  if [ -z "$target" ]; then
    return 0
  fi

  cd "$target" || return $?
}
