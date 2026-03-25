#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO="${WT_HOMEBREW_REPO:-unix2dos/ww}"
CHECKSUMS_PATH="${WT_HOMEBREW_CHECKSUMS_PATH:-}"

usage() {
  cat <<'EOF'
Usage: bash scripts/generate-homebrew-formula.sh <version> [output-path]

Generates a Homebrew formula for the given release version.

Checksum resolution order:
  1. WT_HOMEBREW_CHECKSUMS_PATH
  2. ./dist/checksums.txt
  3. https://github.com/<repo>/releases/download/<version>/checksums.txt

Environment overrides:
  WT_HOMEBREW_REPO            Override GitHub repo, default unix2dos/ww
  WT_HOMEBREW_CHECKSUMS_PATH  Read checksums from a local file
EOF
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

resolve_checksums() {
  local local_dist_checksums="$REPO_ROOT/dist/checksums.txt"

  if [ -n "$CHECKSUMS_PATH" ]; then
    cat "$CHECKSUMS_PATH"
    return
  fi

  if [ -f "$local_dist_checksums" ] && grep -q "ww-${VERSION}-" "$local_dist_checksums"; then
    cat "$local_dist_checksums"
    return
  fi

  need_cmd curl
  curl -fsSL "https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
}

checksum_for_archive() {
  local archive="$1"

  printf '%s\n' "$CHECKSUMS_DATA" | awk -v target="$archive" '
    $2 == target { print $1; found = 1; exit }
    END { if (found != 1) exit 1 }
  '
}

emit_branch() {
  local keyword="$1"
  local url="$2"
  local checksum="$3"
  local first="$4"

  if [ "$first" = "1" ]; then
    printf '  if %s\n' "$keyword"
  else
    printf '  elsif %s\n' "$keyword"
  fi
  printf '    url "%s"\n' "$url"
  printf '    sha256 "%s"\n' "$checksum"
}

if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  usage
  exit 0
fi

[ "$#" -ge 1 ] || {
  usage >&2
  exit 2
}

VERSION="$1"
OUTPUT_PATH="${2:-$REPO_ROOT/Formula/ww.rb}"
VERSION_NO_V="${VERSION#v}"
CHECKSUMS_DATA="$(resolve_checksums)"

DARWIN_ARM64_ARCHIVE="ww-${VERSION}-darwin-arm64.tar.gz"
DARWIN_AMD64_ARCHIVE="ww-${VERSION}-darwin-amd64.tar.gz"
LINUX_ARM64_ARCHIVE="ww-${VERSION}-linux-arm64.tar.gz"
LINUX_AMD64_ARCHIVE="ww-${VERSION}-linux-amd64.tar.gz"

darwin_arm64_checksum=""
darwin_amd64_checksum=""
linux_arm64_checksum=""
linux_amd64_checksum=""

if darwin_arm64_checksum="$(checksum_for_archive "$DARWIN_ARM64_ARCHIVE" 2>/dev/null)"; then :; else darwin_arm64_checksum=""; fi
if darwin_amd64_checksum="$(checksum_for_archive "$DARWIN_AMD64_ARCHIVE" 2>/dev/null)"; then :; else darwin_amd64_checksum=""; fi
if linux_arm64_checksum="$(checksum_for_archive "$LINUX_ARM64_ARCHIVE" 2>/dev/null)"; then :; else linux_arm64_checksum=""; fi
if linux_amd64_checksum="$(checksum_for_archive "$LINUX_AMD64_ARCHIVE" 2>/dev/null)"; then :; else linux_amd64_checksum=""; fi

if [ -z "$darwin_arm64_checksum$darwin_amd64_checksum$linux_arm64_checksum$linux_amd64_checksum" ]; then
  echo "no supported ww release archives found for $VERSION" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT_PATH")"

{
  cat <<EOF
class Ww < Formula
  desc "Fast worktree switching for safer parallel work"
  homepage "https://github.com/$REPO"
  version "$VERSION_NO_V"

EOF

  first_branch=1
  if [ -n "$darwin_arm64_checksum" ]; then
    emit_branch "OS.mac? && Hardware::CPU.arm?" "https://github.com/$REPO/releases/download/$VERSION/$DARWIN_ARM64_ARCHIVE" "$darwin_arm64_checksum" "$first_branch"
    first_branch=0
  fi
  if [ -n "$darwin_amd64_checksum" ]; then
    emit_branch "OS.mac? && Hardware::CPU.intel?" "https://github.com/$REPO/releases/download/$VERSION/$DARWIN_AMD64_ARCHIVE" "$darwin_amd64_checksum" "$first_branch"
    first_branch=0
  fi
  if [ -n "$linux_arm64_checksum" ]; then
    emit_branch "OS.linux? && Hardware::CPU.arm?" "https://github.com/$REPO/releases/download/$VERSION/$LINUX_ARM64_ARCHIVE" "$linux_arm64_checksum" "$first_branch"
    first_branch=0
  fi
  if [ -n "$linux_amd64_checksum" ]; then
    emit_branch "OS.linux? && Hardware::CPU.intel?" "https://github.com/$REPO/releases/download/$VERSION/$LINUX_AMD64_ARCHIVE" "$linux_amd64_checksum" "$first_branch"
  fi

  cat <<'EOF'
  end

  def install
    bin.install "bin/ww-helper"
    libexec.install "shell/ww.sh"
    doc.install "README.md"
  end

  def caveats
    <<~EOS
      `ww` changes the current shell directory, so Homebrew installs the helper and shell library
      but leaves shell activation to you.

      Add these lines to your shell rc file:

        export WW_HELPER_BIN="#{opt_bin}/ww-helper"
        source "#{opt_libexec}/ww.sh"

      For zsh:
        echo 'export WW_HELPER_BIN="#{opt_bin}/ww-helper"' >> ~/.zshrc
        echo 'source "#{opt_libexec}/ww.sh"' >> ~/.zshrc

      For bash:
        echo 'export WW_HELPER_BIN="#{opt_bin}/ww-helper"' >> ~/.bashrc
        echo 'source "#{opt_libexec}/ww.sh"' >> ~/.bashrc
    EOS
  end

  test do
    assert_path_exists libexec/"ww.sh"
    assert_match "Usage: ww-helper", shell_output("#{bin}/ww-helper help")
  end
end
EOF
} >"$OUTPUT_PATH"
