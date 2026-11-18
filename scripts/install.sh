#!/usr/bin/env bash
set -euo pipefail

_TMP_DIR=""

cleanup() {
  if [ -n "${_TMP_DIR}" ] && [ -d "${_TMP_DIR}" ]; then
    rm -rf "${_TMP_DIR}"
  fi
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "Error: unsupported OS '$(uname -s)'. Only Linux and macOS are supported." >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)
      echo "Error: unsupported architecture '$(uname -m)'. Only amd64 and arm64 are supported." >&2
      exit 1
      ;;
  esac
}

resolve_tag() {
  if [ -n "${OPUS_VERSION:-}" ]; then
    echo "v${OPUS_VERSION#v}"
    return 0
  fi

  local api_url="https://api.github.com/repos/kilip/opus/releases?per_page=1"
  local resp=""
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    resp=$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" "$api_url") || {
      echo "Error: failed to query GitHub releases. Set OPUS_VERSION and retry." >&2
      exit 1
    }
  else
    resp=$(curl -fsSL "$api_url") || {
      echo "Error: failed to query GitHub releases. Set OPUS_VERSION and retry." >&2
      exit 1
    }
  fi

  local tag
  tag=$(printf '%s\n' "$resp" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/')
  if [ -z "$tag" ]; then
    echo "Error: could not determine latest release tag. Set OPUS_VERSION and retry." >&2
    exit 1
  fi
  echo "$tag"
}

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    echo "Error: neither sha256sum nor shasum is installed." >&2
    exit 1
  fi
}

main() {
  local os arch tag version archive base_url install_dir
  os=$(detect_os)
  arch=$(detect_arch)
  tag=$(resolve_tag)
  version="${tag#v}"
  archive="opus_${version}_${os}_${arch}.tar.gz"

  # ponytail: OPUS_DOWNLOAD_BASE hook overrides release URL for CI fixture tests
  base_url="${OPUS_DOWNLOAD_BASE:-https://github.com/kilip/opus/releases/download/${tag}}"
  base_url="${base_url%/}"

  _TMP_DIR=$(mktemp -d)
  trap cleanup EXIT

  curl -fsSL "${base_url}/${archive}" -o "${_TMP_DIR}/${archive}"
  curl -fsSL "${base_url}/checksums.txt" -o "${_TMP_DIR}/checksums.txt"

  local expected actual
  expected=$(awk -v f="$archive" '$2 == f {print $1; exit}' "${_TMP_DIR}/checksums.txt")
  if [ -z "$expected" ]; then
    echo "Error: checksum entry for '${archive}' not found in checksums.txt." >&2
    exit 1
  fi

  actual=$(sha256_file "${_TMP_DIR}/${archive}")
  if [ "$actual" != "$expected" ]; then
    echo "Error: checksum mismatch for '${archive}' (expected ${expected}, got ${actual})." >&2
    exit 1
  fi

  tar -xzf "${_TMP_DIR}/${archive}" -C "${_TMP_DIR}"
  if [ ! -f "${_TMP_DIR}/opus" ]; then
    echo "Error: extracted archive does not contain 'opus' binary." >&2
    exit 1
  fi

  install_dir="${OPUS_INSTALL_DIR:-$HOME/.local/bin}"
  mkdir -p "$install_dir"

  local tmp_bin
  tmp_bin=$(mktemp "${install_dir}/.opus.XXXXXX")
  cp "${_TMP_DIR}/opus" "$tmp_bin"
  chmod 755 "$tmp_bin"
  mv -f "$tmp_bin" "${install_dir}/opus"

  "${install_dir}/opus" --version

  case ":${PATH}:" in
    *":${install_dir}:"*) ;;
    *)
      echo ""
      echo "${install_dir} is not in your PATH. Add it with:"
      echo "export PATH=\"${install_dir}:\$PATH\""
      ;;
  esac

  local dep
  for dep in git gh agy; do
    if ! command -v "$dep" >/dev/null 2>&1; then
      echo "Warning: '${dep}' not found in PATH (required on worker machines)." >&2
    fi
  done
}

main "$@"
