#!/bin/sh
# cjv installer script
# Usage: curl -sSf https://cjv.zxilly.dev/install.sh | sh
# Or:    curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror -y
#
# `--mirror` switches the installer to download the cjv-mirror archive from
# GitCode (for environments without reliable GitHub access). All other flags
# are forwarded to `cjv init`.

set -eu

CJV_GITHUB_ROOT="${CJV_GITHUB_ROOT:-https://github.com/Zxilly/cjv/releases/latest/download}"
CJV_GITCODE_ROOT="${CJV_GITCODE_ROOT:-https://gitcode.com/Zxilly/cjv/releases/latest/download}"

main() {
    if [ "${CJV_MIRROR:-}" = "1" ]; then
        USE_MIRROR=1
    else
        USE_MIRROR=0
    fi

    remaining=$#
    while [ "$remaining" -gt 0 ]; do
        arg=$1
        shift
        remaining=$((remaining - 1))
        case "$arg" in
            --mirror) USE_MIRROR=1 ;;
            *) set -- "$@" "$arg" ;;
        esac
    done

    if [ "$USE_MIRROR" = "1" ]; then
        BINARY="cjv-mirror"
        UPDATE_ROOT="${CJV_UPDATE_ROOT:-$CJV_GITCODE_ROOT}"
    else
        BINARY="cjv"
        UPDATE_ROOT="${CJV_UPDATE_ROOT:-$CJV_GITHUB_ROOT}"
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
    _url="${UPDATE_ROOT}/${BINARY}_${PLATFORM}_${ARCH}.tar.gz"
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
    "$_tmpdir/$BINARY" init "$@"
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
