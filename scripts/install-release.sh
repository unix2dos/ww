#!/usr/bin/env bash

set -euo pipefail

REPO="${WT_RELEASE_REPO:-unix2dos/wt}"
API_URL="${WT_RELEASE_API_URL:-https://api.github.com/repos/$REPO/releases/latest}"
TARBALL_URL="${WT_TARBALL_URL:-}"
VERSION="${WT_VERSION:-}"
TMPDIR_WT=""

usage() {
  cat <<'EOF'
Usage: curl -fsSL https://github.com/unix2dos/wt/releases/latest/download/install-release.sh | bash
       curl -fsSL https://github.com/unix2dos/wt/releases/latest/download/install-release.sh | bash -s -- [install.sh args]

Environment overrides:
  WT_VERSION           Install a specific version tag, e.g. v0.1.0
  WT_RELEASE_REPO      Override GitHub repo, default unix2dos/wt
  WT_RELEASE_API_URL   Override latest release API endpoint
  WT_TARBALL_URL       Override archive URL directly
EOF
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

detect_os() {
  case "$(uname -s)" in
    Darwin) printf '%s\n' "darwin" ;;
    Linux) printf '%s\n' "linux" ;;
    *)
      echo "unsupported operating system: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    arm64|aarch64) printf '%s\n' "arm64" ;;
    x86_64|amd64) printf '%s\n' "amd64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

latest_version() {
  curl -fsSL "$API_URL" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

download_url() {
  local version="$1"
  local os="$2"
  local arch="$3"
  printf 'https://github.com/%s/releases/download/%s/wt-%s-%s-%s.tar.gz\n' "$REPO" "$version" "$version" "$os" "$arch"
}

main() {
  if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
    usage
    exit 0
  fi

  need_cmd curl
  need_cmd tar
  need_cmd bash

  local os arch version url archive extract_dir

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$VERSION"

  if [ -z "$TARBALL_URL" ]; then
    if [ -z "$version" ]; then
      version="$(latest_version)"
    fi
    if [ -z "$version" ]; then
      echo "failed to resolve release version" >&2
      exit 1
    fi
    url="$(download_url "$version" "$os" "$arch")"
  else
    url="$TARBALL_URL"
  fi

  TMPDIR_WT="$(mktemp -d)"
  trap 'rm -rf "$TMPDIR_WT"' EXIT

  archive="$TMPDIR_WT/wt.tar.gz"
  curl -fsSL "$url" -o "$archive"
  tar -xzf "$archive" -C "$TMPDIR_WT"

  extract_dir="$(find "$TMPDIR_WT" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
  if [ -z "$extract_dir" ] || [ ! -f "$extract_dir/install.sh" ]; then
    echo "invalid wt release archive" >&2
    exit 1
  fi

  (cd "$extract_dir" && bash ./install.sh "$@")
}

main "$@"
