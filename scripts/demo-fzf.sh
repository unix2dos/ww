#!/usr/bin/env bash

set -euo pipefail

tty_path="${FZF_TTY:-/dev/tty}"
exec 3<>"$tty_path"

declare -a candidates=()
declare -a filtered=()

while IFS= read -r line; do
  candidates+=("$line")
done

old_stty="$(stty -g <&3)"
query=""
selected=0

cleanup() {
  stty "$old_stty" <&3 >/dev/null 2>&1 || true
  printf '\033[?25h\033[?1049l' >&3 || true
}
trap cleanup EXIT

lower() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

filter_candidates() {
  filtered=()
  local lowered_query
  local candidate lowered_candidate

  lowered_query="$(lower "$query")"

  for candidate in "${candidates[@]}"; do
    lowered_candidate="$(lower "$candidate")"
    if [[ -z "$lowered_query" || "$lowered_candidate" == *"$lowered_query"* ]]; then
      filtered+=("$candidate")
    fi
  done

  if (( selected >= ${#filtered[@]} )); then
    selected=0
  fi
}

print_candidate() {
  local candidate="$1"
  local index status branch path

  IFS=$'\t' read -r index status branch path <<<"$candidate"
  printf "%s  %-6s %-8s %s" "$index" "${status:-}" "$branch" "$path"
}

render() {
  printf '\033[?1049h\033[H\033[2J\033[?25l' >&3
  printf 'Select a worktree> %s\n\n' "$query" >&3

  if (( ${#filtered[@]} == 0 )); then
    printf '  no matches\n' >&3
    return
  fi

  local i
  for i in "${!filtered[@]}"; do
    if (( i == selected )); then
      printf '> ' >&3
    else
      printf '  ' >&3
    fi
    print_candidate "${filtered[$i]}" >&3
    printf '\n' >&3
  done
}

stty -echo -icanon min 1 time 0 <&3
filter_candidates
render

while IFS= read -r -s -n1 key <&3; do
  case "$key" in
    "")
      break
      ;;
    $'\r'|$'\n')
      break
      ;;
    $'\177'|$'\b')
      query="${query%?}"
      selected=0
      ;;
    $'\003')
      exit 130
      ;;
    $'\033')
      rest=""
      IFS= read -r -s -n2 -t 0.01 rest <&3 || true
      case "$rest" in
        "[A")
          if (( selected > 0 )); then
            ((selected--))
          fi
          ;;
        "[B")
          if (( selected + 1 < ${#filtered[@]} )); then
            ((selected++))
          fi
          ;;
        *)
          exit 130
          ;;
      esac
      ;;
    *)
      query+="$key"
      selected=0
      ;;
  esac

  filter_candidates
  render
done

if (( ${#filtered[@]} == 0 )); then
  exit 130
fi

printf '%s\n' "${filtered[$selected]}"
