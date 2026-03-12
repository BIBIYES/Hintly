#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-BIBIYES/Hintly}"
BINARY_NAME="${BINARY_NAME:-hint}"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

resolve_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "linux-amd64" ;;
    aarch64|arm64) echo "linux-arm64" ;;
    armv7l|armv7) echo "linux-armv7" ;;
    i386|i686) echo "linux-386" ;;
    *)
      echo "Unsupported Linux architecture: $(uname -m)" >&2
      echo "Supported: amd64, arm64, armv7, 386" >&2
      exit 1
      ;;
  esac
}

normalize_tag() {
  local raw="$1"
  if [ "$raw" = "latest" ]; then
    local tag
    tag="$(
      curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -n 1
    )"
    if [ -z "$tag" ]; then
      echo "Failed to resolve latest release tag from ${REPO}" >&2
      exit 1
    fi
    echo "$tag"
    return
  fi

  case "$raw" in
    v*) echo "$raw" ;;
    *) echo "v${raw}" ;;
  esac
}

install_binary() {
  local from="$1"
  local to_dir="$2"
  local mode="${3:-normal}"

  mkdir -p "$to_dir"
  if [ -w "$to_dir" ]; then
    install -m 0755 "$from" "$to_dir/$BINARY_NAME"
    echo "$to_dir/$BINARY_NAME"
    return
  fi

  if [ "$mode" = "fallback_allowed" ] && command -v sudo >/dev/null 2>&1; then
    sudo install -m 0755 "$from" "$to_dir/$BINARY_NAME"
    echo "$to_dir/$BINARY_NAME"
    return
  fi

  return 1
}

main() {
  if [ "$(uname -s)" != "Linux" ]; then
    echo "This install script only supports Linux." >&2
    exit 1
  fi

  need_cmd curl
  need_cmd tar
  need_cmd install
  need_cmd mktemp

  local arch
  arch="$(resolve_arch)"
  local tag
  tag="$(normalize_tag "$VERSION")"

  local asset="hint_${tag}_${arch}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${tag}/${asset}"

  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  echo "Downloading: ${url}"
  curl -fL "$url" -o "$tmpdir/$asset"

  tar -xzf "$tmpdir/$asset" -C "$tmpdir"
  if [ ! -f "$tmpdir/$BINARY_NAME" ]; then
    echo "Binary '${BINARY_NAME}' not found in release asset ${asset}" >&2
    exit 1
  fi

  local existing="false"
  if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    existing="true"
  fi

  local installed_path=""
  if [ "${INSTALL_DIR}" = "/usr/local/bin" ]; then
    if ! installed_path="$(install_binary "$tmpdir/$BINARY_NAME" "$INSTALL_DIR" "fallback_allowed")"; then
      local user_bin="${HOME}/.local/bin"
      installed_path="$(install_binary "$tmpdir/$BINARY_NAME" "$user_bin")"
      echo "Installed to ${installed_path}"
      echo "Add to PATH if needed: export PATH=\"${HOME}/.local/bin:\$PATH\""
    fi
  else
    installed_path="$(install_binary "$tmpdir/$BINARY_NAME" "$INSTALL_DIR")"
  fi

  if [ -n "$installed_path" ] && [ "${INSTALL_DIR}" != "/usr/local/bin" ]; then
    echo "Installed to ${installed_path}"
  fi

  if [ "$existing" = "true" ]; then
    echo "Update complete: ${BINARY_NAME} ${tag}"
  else
    echo "Install complete: ${BINARY_NAME} ${tag}"
  fi
}

main "$@"
