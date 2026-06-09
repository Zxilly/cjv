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
    _archive="${BINARY}_${PLATFORM}_${ARCH}.tar.gz"
    _url="${UPDATE_ROOT}/${_archive}"
    _tmpdir="$(mktemp -d)"
    # Clean up the temp dir on normal exit and on interruption (the bare EXIT
    # trap is not always run when sh is killed by a signal).
    # shellcheck disable=SC2064
    trap "cleanup '$_tmpdir'" EXIT
    # shellcheck disable=SC2064
    trap "cleanup '$_tmpdir'; exit 130" HUP INT TERM

    say "downloading cjv from $_url"
    download "$_url" "$_tmpdir/cjv.tar.gz"

    verify_checksum "$_tmpdir/cjv.tar.gz" "$_archive"

    tar -xzf "$_tmpdir/cjv.tar.gz" -C "$_tmpdir"

    say "running cjv init"
    # When invoked as `curl ... | sh`, the binary inherits the script pipe as
    # stdin. Reconnect the controlling terminal if there is one so interactive
    # setup can prompt; otherwise cjv init detects the non-tty stdin and
    # proceeds with a standard non-interactive install. The /dev/tty probe must
    # run in a subshell: a failed redirection on the `exec` special builtin
    # terminates a non-interactive POSIX shell outright, even inside an `if`
    # condition.
    if [ ! -t 0 ] && (exec </dev/tty) 2>/dev/null; then
        "$_tmpdir/$BINARY" init "$@" </dev/tty
    else
        "$_tmpdir/$BINARY" init "$@"
    fi
}

download() {
    if command -v curl > /dev/null 2>&1; then
        curl -sSfL "$1" -o "$2"
    elif command -v wget > /dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        err "need curl or wget to download cjv"
    fi
}

fetch_text() {
    if command -v curl > /dev/null 2>&1; then
        curl -sSfL "$1"
    elif command -v wget > /dev/null 2>&1; then
        wget -qO- "$1"
    else
        return 1
    fi
}

compute_sha256() {
    if command -v sha256sum > /dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum > /dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        return 1
    fi
}

# verify_checksum validates the downloaded archive against the release
# checksums.txt. A missing checksums file or sha256 tool degrades to a warning
# (older releases / minimal environments); a present-but-mismatched checksum is
# fatal so a tampered or corrupted download is never installed.
verify_checksum() {
    _file=$1
    _name=$2
    if ! _sums="$(fetch_text "${UPDATE_ROOT}/checksums.txt")"; then
        warn "could not download checksums.txt; skipping integrity verification"
        return 0
    fi
    _expected="$(printf '%s\n' "$_sums" | awk -v n="$_name" '$2 == n {print $1; exit}')"
    if [ -z "$_expected" ]; then
        warn "no checksum entry for $_name; skipping integrity verification"
        return 0
    fi
    if ! _actual="$(compute_sha256 "$_file")"; then
        warn "no sha256 tool available; skipping integrity verification"
        return 0
    fi
    if [ "$_actual" != "$_expected" ]; then
        err "checksum mismatch for $_name (expected $_expected, got $_actual)"
    fi
    say "checksum verified"
}

cleanup() {
    [ -n "${1:-}" ] && rm -rf "$1"
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
