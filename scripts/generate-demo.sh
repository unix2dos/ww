#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLS_DIR="$ROOT_DIR/.tools"
TOOLS_BIN_DIR="$TOOLS_DIR/bin"
ASSETS_DIR="$ROOT_DIR/docs/assets"
ASCIINEMA_BIN="$TOOLS_BIN_DIR/asciinema"
FZF_BIN="$TOOLS_BIN_DIR/fzf"
HELPER_BIN="$TOOLS_BIN_DIR/ww-helper-demo"
EXPECT_BIN="/usr/bin/expect"
CAST_FILE="$ASSETS_DIR/ww-demo.cast"
SVG_FILE="$ASSETS_DIR/ww-demo.svg"

mkdir -p "$TOOLS_BIN_DIR" "$ASSETS_DIR"

if [[ ! -x "$EXPECT_BIN" ]]; then
  echo "expect is required but was not found at $EXPECT_BIN" >&2
  exit 1
fi

if [[ ! -x "$ASCIINEMA_BIN" ]]; then
  cargo install --locked --root "$TOOLS_DIR" asciinema --version 3.2.0
fi

install -m 0755 "$ROOT_DIR/scripts/demo-fzf.sh" "$FZF_BIN"

go build -buildvcs=false -o "$HELPER_BIN" ./cmd/ww-helper

TMP_DIR="/tmp/ww-demo-site"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

rm -rf "$TMP_DIR"

DEMO_REPO="$TMP_DIR/repo"
DEMO_HOME="$TMP_DIR/home"
STATE_HOME="$TMP_DIR/state"
mkdir -p "$DEMO_REPO" "$DEMO_HOME" "$STATE_HOME"

git init -b main "$DEMO_REPO" >/dev/null
git -C "$DEMO_REPO" config user.name "ww demo"
git -C "$DEMO_REPO" config user.email "ww-demo@example.com"
printf '# ww demo repo\n' >"$DEMO_REPO/README.md"
git -C "$DEMO_REPO" add README.md
git -C "$DEMO_REPO" commit -m "init demo repo" >/dev/null
git -C "$DEMO_REPO" worktree add "$DEMO_REPO/.worktrees/feat-a" -b feat-a >/dev/null
git -C "$DEMO_REPO" worktree add "$DEMO_REPO/.worktrees/hotfix" -b hotfix >/dev/null

export WW_DEMO_REPO="$DEMO_REPO"
export WW_DEMO_HELPER_BIN="$HELPER_BIN"
export WW_DEMO_SHELL_FILE="$ROOT_DIR/shell/ww.sh"
export WW_DEMO_HOME="$DEMO_HOME"
export WW_DEMO_STATE_HOME="$STATE_HOME"
export WW_DEMO_PATH="$TOOLS_BIN_DIR:/usr/bin:/bin:/usr/sbin:/sbin"
export WW_DEMO_KEYSTROKE_DELAY_MS="${WW_DEMO_KEYSTROKE_DELAY_MS:-75}"
export WW_DEMO_STEP_DELAY_MS="${WW_DEMO_STEP_DELAY_MS:-450}"
export WW_DEMO_FZF_FOCUS_DELAY_MS="${WW_DEMO_FZF_FOCUS_DELAY_MS:-500}"
export WW_DEMO_FZF_QUERY_SETTLE_MS="${WW_DEMO_FZF_QUERY_SETTLE_MS:-400}"
export WW_DEMO_CONFIRM_DELAY_MS="${WW_DEMO_CONFIRM_DELAY_MS:-350}"

IDLE_TIME_LIMIT="${WW_DEMO_IDLE_TIME_LIMIT:-1.1}"

"$ASCIINEMA_BIN" rec \
  --quiet \
  --overwrite \
  --headless \
  --return \
  --window-size 110x28 \
  --idle-time-limit "$IDLE_TIME_LIMIT" \
  --output-format asciicast-v2 \
  --command "$EXPECT_BIN -f $ROOT_DIR/scripts/demo-record.exp" \
  "$CAST_FILE"

npm_config_loglevel=error npx --yes svg-term-cli@2.1.1 \
  --in "$CAST_FILE" \
  --out "$SVG_FILE" \
  --window

echo "Generated:"
echo "  $CAST_FILE"
echo "  $SVG_FILE"
