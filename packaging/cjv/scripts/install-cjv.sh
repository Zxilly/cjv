#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
MODULE_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

MODE=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --mode)
      MODE="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [ -z "$MODE" ]; then
  echo "--mode is required" >&2
  exit 1
fi

RELEASE_ENV="$MODULE_DIR/release.env"
if [ ! -f "$RELEASE_ENV" ]; then
  echo "release.env is missing: $RELEASE_ENV" >&2
  exit 1
fi

# shellcheck disable=SC1090
. "$RELEASE_ENV"

if [ -z "${CJV_VERSION:-}" ] || [ -z "${CJV_TAG:-}" ] || [ -z "${CJV_REPOSITORY:-}" ]; then
  echo "release.env must define CJV_VERSION, CJV_TAG and CJV_REPOSITORY" >&2
  exit 1
fi

BASE_URL="${CJV_RELEASE_BASE_URL:-}"
if [ -z "$BASE_URL" ]; then
  BASE_URL="https://github.com/${CJV_REPOSITORY}/releases/download"
fi

OS_NAME=$(uname -s 2>/dev/null || echo unknown)
ARCH_NAME=$(uname -m 2>/dev/null || echo unknown)

case "$OS_NAME/$ARCH_NAME" in
  Linux/x86_64) ASSET="cjv_linux_amd64.tar.gz" ;;
  Linux/aarch64|Linux/arm64) ASSET="cjv_linux_arm64.tar.gz" ;;
  Darwin/x86_64) ASSET="cjv_darwin_amd64.tar.gz" ;;
  Darwin/arm64) ASSET="cjv_darwin_arm64.tar.gz" ;;
  *)
    echo "platform $OS_NAME/$ARCH_NAME is not supported" >&2
    exit 1
    ;;
esac

case "$MODE" in
  build)
    DEST="$MODULE_DIR/target/release/bin/main"
    ;;
  install)
    HOME_DIR="${HOME:-}"
    if [ -z "$HOME_DIR" ]; then
      echo "HOME is not set; cannot resolve install destination" >&2
      exit 1
    fi
    DEST="$HOME_DIR/.cjpm/bin/cjv"
    ;;
  *)
    echo "unsupported mode: $MODE" >&2
    exit 1
    ;;
esac

ASSET_URL="$BASE_URL/$CJV_TAG/$ASSET"
CHECKSUMS_URL="$BASE_URL/$CJV_TAG/checksums.txt"
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/cjv-package.XXXXXX")
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT HUP TERM

download() {
  url="$1"
  output="$2"
  curl -fsSL "$url" -o "$output"
}

checksum_line() {
  asset="$1"
  file="$2"
  awk -v target="$asset" '
    {
      gsub(/\r/, "", $0)
      if (NF >= 2 && ($2 == target || $2 == "*" target)) {
        print $1
        exit 0
      }
    }
  ' "$file"
}

verify_checksum() {
  expected="$1"
  file="$2"
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$file" | awk "{print \$1}")
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$file" | awk "{print \$1}")
  else
    echo "sha256sum or shasum is required" >&2
    exit 1
  fi

  if [ "$actual" != "$expected" ]; then
    echo "checksum mismatch for $file" >&2
    echo "expected: $expected" >&2
    echo "actual:   $actual" >&2
    exit 1
  fi
}

ASSET_PATH="$TMP_DIR/$ASSET"
CHECKSUMS_PATH="$TMP_DIR/checksums.txt"

echo "Downloading $ASSET_URL"
download "$ASSET_URL" "$ASSET_PATH"
download "$CHECKSUMS_URL" "$CHECKSUMS_PATH"

EXPECTED=$(checksum_line "$ASSET" "$CHECKSUMS_PATH")
if [ -z "$EXPECTED" ]; then
  echo "checksum entry for $ASSET is missing from checksums.txt" >&2
  exit 1
fi

verify_checksum "$EXPECTED" "$ASSET_PATH"

EXTRACT_DIR="$TMP_DIR/extract"
mkdir -p "$EXTRACT_DIR"
tar -xzf "$ASSET_PATH" -C "$EXTRACT_DIR"

DOWNLOADED_BIN="$EXTRACT_DIR/cjv"
if [ ! -f "$DOWNLOADED_BIN" ]; then
  echo "expected extracted binary at $DOWNLOADED_BIN" >&2
  exit 1
fi

mkdir -p "$(dirname "$DEST")"
TEMP_DEST="$TMP_DIR/cjv"
cp "$DOWNLOADED_BIN" "$TEMP_DEST"
chmod +x "$TEMP_DEST"
mv "$TEMP_DEST" "$DEST"

echo "Installed cjv $CJV_VERSION to $DEST"
