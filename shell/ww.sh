ww_print_help() {
  cat <<'EOF'
Usage:
  ww [switch] [<index>|<name>]
  ww list
  ww new <name>
  ww rm [--force] [--base <branch>] [<name>]
  ww help

Commands:
  switch  Select a worktree and change into it. Default when omitted.
  list    Print worktrees without changing directory.
  new     Create a worktree under ./.worktrees/<name> and enter it.
  rm      Remove a worktree and delete its branch only when merged.
  help    Show this help.

Notes:
  Uses fzf automatically when available.
  Falls back to the built-in selector otherwise.

Examples:
  ww
  ww 2
  ww switch feat-a
  ww new feat-demo
  ww rm feat-demo
EOF
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
      target="$("$ww_helper_bin" switch-path "$@")" || return $?
      ;;
    new)
      shift
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
    help|-h|--help)
      ww_print_help
      return 0
      ;;
    *)
      target="$("$ww_helper_bin" switch-path "$@")" || return $?
      ;;
  esac

  if [ -z "$target" ]; then
    return 0
  fi

  cd "$target" || return $?
}
