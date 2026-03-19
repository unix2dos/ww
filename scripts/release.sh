#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT_DIR="${WT_RELEASE_OUT_DIR:-$REPO_ROOT/dist}"
TARGETS="${WT_RELEASE_TARGETS:-darwin/arm64 darwin/amd64 linux/arm64 linux/amd64}"

usage() {
  cat <<'EOF'
Usage: bash scripts/release.sh <version>

Builds release tarballs in dist/ (or WT_RELEASE_OUT_DIR) for the configured
WT_RELEASE_TARGETS matrix.
EOF
}

[ "$#" -eq 1 ] || { usage >&2; exit 2; }
VERSION="$1"

mkdir -p "$OUT_DIR"
rm -f "$OUT_DIR"/wt-"$VERSION"-*.tar.gz "$OUT_DIR"/checksums.txt "$OUT_DIR"/install-release.sh

cd "$REPO_ROOT"

for target in $TARGETS; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  ARTIFACT_DIR="wt-${VERSION}-${GOOS}-${GOARCH}"
  ARCHIVE_PATH="$OUT_DIR/${ARTIFACT_DIR}.tar.gz"
  STAGE_ROOT="$(mktemp -d)"
  STAGE_DIR="$STAGE_ROOT/$ARTIFACT_DIR"

  mkdir -p "$STAGE_DIR/bin" "$STAGE_DIR/shell"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$STAGE_DIR/bin/wt" ./cmd/wt
  cp README.md install.sh uninstall.sh "$STAGE_DIR/"
  cp shell/cwt.sh "$STAGE_DIR/shell/cwt.sh"

  (
    cd "$STAGE_ROOT"
    tar -czf "$ARCHIVE_PATH" "$ARTIFACT_DIR"
  )

  rm -rf "$STAGE_ROOT"
done

cp scripts/install-release.sh "$OUT_DIR/install-release.sh"
chmod +x "$OUT_DIR/install-release.sh"

if command -v sha256sum >/dev/null 2>&1; then
  (
    cd "$OUT_DIR"
    sha256sum wt-"$VERSION"-*.tar.gz > checksums.txt
  )
elif command -v shasum >/dev/null 2>&1; then
  (
    cd "$OUT_DIR"
    shasum -a 256 wt-"$VERSION"-*.tar.gz > checksums.txt
  )
fi

printf 'Release artifacts written to %s\n' "$OUT_DIR"
