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
    rm|diff)
      "$ww_helper_bin" "$@"
      return $?
      ;;
    help|-h|--help)
      "$ww_helper_bin" --help
      return $?
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
