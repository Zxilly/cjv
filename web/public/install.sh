#!/bin/sh
# cjv installer script
# Usage: curl -sSf https://cjv.zxilly.dev/install.sh | sh
# Or:    curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror -y

set -eu

CJV_UPDATE_ROOT="${CJV_UPDATE_ROOT:-https://github.com/Zxilly/cjv/releases/latest/download}"

main() {
    # Support CJV_MIRROR env var
    if [ "${CJV_MIRROR:-}" = "1" ]; then
        set -- --mirror "$@"
    fi

    detect_platform
    detect_arch

    if [ "$PLATFORM" = "darwin" ] && [ "$ARCH" = "amd64" ]; then
        warn "macOS x86_64 has limited support; some LTS and STS releases may not include prebuilt SDK for macOS x86_64."
    fi

    download_and_install "$@"
}

detect_platform() {
    _os="$(uname -s)"
    case "$_os" in
        Linux)   PLATFORM="linux" ;;
        Darwin)  PLATFORM="darwin" ;;
        *)       err "unsupported platform: $_os" ;;
    esac
}

detect_arch() {
    _arch="$(uname -m)"
    case "$_arch" in
        x86_64|amd64)       ARCH="amd64" ;;
        aarch64|arm64)      ARCH="arm64" ;;
        *)                  err "unsupported architecture: $_arch" ;;
    esac
}

download_and_install() {
    _url="${CJV_UPDATE_ROOT}/cjv_${PLATFORM}_${ARCH}.tar.gz"
    _tmpdir="$(mktemp -d)"
    # shellcheck disable=SC2064
    trap "rm -rf '$_tmpdir'" EXIT

    say "downloading cjv from $_url"

    if command -v curl > /dev/null 2>&1; then
        curl -sSfL "$_url" -o "$_tmpdir/cjv.tar.gz"
    elif command -v wget > /dev/null 2>&1; then
        wget -qO "$_tmpdir/cjv.tar.gz" "$_url"
    else
        err "need curl or wget to download cjv"
    fi

    tar -xzf "$_tmpdir/cjv.tar.gz" -C "$_tmpdir"

    say "running cjv init"
    "$_tmpdir/cjv" init "$@"
}

say() {
    printf "cjv-install: %s\n" "$1"
}

warn() {
    say "warning: $1" >&2
}

err() {
    say "error: $1" >&2
    exit 1
}

main "$@"
