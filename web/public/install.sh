#!/bin/sh
# cjv installer script
# Usage: curl -sSf https://cjv.zxilly.dev/install.sh | sh
# Or:    curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror -y
# Prerelease: ... | sh -s -- --version v0.0.0-hmos.1 -y
#
# `--mirror` switches the installer to download the cjv-mirror archive from
# GitCode (for environments without reliable GitHub access). `--version TAG`
# (or CJV_VERSION) selects an exact release, including prereleases. Explicit
# CJV_UPDATE_ROOT / CJV_GITHUB_ROOT / CJV_GITCODE_ROOT URLs take precedence.
# All other flags are forwarded to `cjv init`.

set -eu

main() {
    VERSION="${CJV_VERSION:-}"
    if [ "${CJV_MIRROR:-}" = "1" ]; then
        USE_MIRROR=1
    else
        USE_MIRROR=0
    fi

    remaining=$#
    installer_args=1
    while [ "$remaining" -gt 0 ]; do
        arg=$1
        shift
        remaining=$((remaining - 1))
        if [ "$installer_args" = "0" ]; then
            set -- "$@" "$arg"
            continue
        fi
        case "$arg" in
            --mirror) USE_MIRROR=1 ;;
            --version)
                [ "$remaining" -gt 0 ] || err "--version requires a release tag"
                VERSION=$1
                shift
                remaining=$((remaining - 1))
                validate_version
                ;;
            --version=*) VERSION=${arg#--version=}; validate_version ;;
            --) installer_args=0; set -- "$@" "$arg" ;;
            *) set -- "$@" "$arg" ;;
        esac
    done

    _release_path="latest/download"
    if [ -n "$VERSION" ]; then
        validate_version
        _release_path="download/$VERSION"
    fi
    if [ "$USE_MIRROR" = "1" ]; then
        BINARY="cjv-mirror"
        _root="${CJV_GITCODE_ROOT:-}"
        _default_root="https://gitcode.com/Zxilly/cjv/releases/$_release_path"
    else
        BINARY="cjv"
        _root="${CJV_GITHUB_ROOT:-}"
        _default_root="https://github.com/Zxilly/cjv/releases/$_release_path"
    fi
    UPDATE_ROOT="${CJV_UPDATE_ROOT:-$_root}"
    if [ -z "$UPDATE_ROOT" ]; then
        UPDATE_ROOT=$_default_root
    fi
    UPDATE_ROOT=${UPDATE_ROOT%/}

    detect_platform
    detect_arch

    if [ "$PLATFORM" = "darwin" ] && [ "$ARCH" = "amd64" ]; then
        warn "macOS x86_64 has limited support; some LTS and STS releases may not include prebuilt SDK for macOS x86_64."
    fi

    download_and_install "$@"
}

validate_version() {
    # A single URL-safe tag segment. Reject option-like values, path traversal,
    # query strings and shell whitespace rather than interpreting them as URLs.
    case "$VERSION" in
        ''|[!a-zA-Z0-9]*|*[!a-zA-Z0-9._+-]*) err "invalid release tag: $VERSION" ;;
    esac
}

detect_platform() {
    _os="$(uname -s)"
    case "$_os" in
        OpenHarmony|HarmonyOS|OHOS|openharmony|harmonyos|ohos)
            PLATFORM="openharmony"
            ;;
        Linux)
            # OpenHarmony can use a Linux kernel. Check positive userland
            # signals before choosing Linux; Android is a different ABI.
            _userland="$(uname -o 2>/dev/null)" || _userland=
            case "$_userland" in
                Android|android) err "unsupported platform: Android" ;;
                OpenHarmony|HarmonyOS|OHOS|openharmony|harmonyos|ohos)
                    PLATFORM="openharmony"
                    return
                    ;;
            esac
            if command -v param >/dev/null 2>&1 && _fullname="$(param get const.ohos.fullname </dev/null 2>/dev/null)"; then
                case "$_fullname" in
                    OpenHarmony-[0-9]*|OpenHarmony\ [0-9]*|HarmonyOS-[0-9]*|HarmonyOS\ [0-9]*)
                        PLATFORM="openharmony"
                        return
                        ;;
                esac
            fi
            PLATFORM="linux"
            ;;
        Darwin) PLATFORM="darwin" ;;
        *) err "unsupported platform: $_os" ;;
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
    if [ "$PLATFORM" = "openharmony" ]; then
        # Native terminals need not provide /tmp. Use a caller-owned location,
        # and resolve aliases before rejecting the filesystem root.
        _tmpbase="${TMPDIR:-${HOME:-}}"
        case "$_tmpbase" in
            /*) ;;
            *) err "OpenHarmony requires TMPDIR or HOME to name an existing absolute writable directory other than /" ;;
        esac
        if [ ! -d "$_tmpbase" ]; then
            err "OpenHarmony requires TMPDIR or HOME to name an existing absolute writable directory other than /"
        fi
        _tmpbase="$(CDPATH= cd "$_tmpbase" && pwd -P)" || err "could not resolve the OpenHarmony temporary directory"
        [ "$_tmpbase" != / ] || err "OpenHarmony temporary directory must not be the filesystem root; set TMPDIR or HOME"
        if [ ! -w "$_tmpbase" ] || [ ! -x "$_tmpbase" ]; then
            err "OpenHarmony requires TMPDIR or HOME to name an existing absolute writable directory other than /"
        fi
        _tmpdir="$(mktemp -d "$_tmpbase/cjv-install.XXXXXXXXXX")" || err "could not create an OpenHarmony temporary directory under $_tmpbase"
    else
        _tmpdir="$(mktemp -d)"
    fi
    # Clean up the temp dir on normal exit and on interruption (the bare EXIT
    # trap is not always run when sh is killed by a signal).
    trap 'cleanup "$_tmpdir"' EXIT
    trap 'cleanup "$_tmpdir"; exit 130' HUP INT TERM

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
        _hash_output="$(sha256sum "$1")" || return 2
    elif command -v shasum > /dev/null 2>&1; then
        _hash_output="$(shasum -a 256 "$1")" || return 2
    else
        return 1
    fi
    # Toybox on native HarmonyOS does not necessarily include awk. Preserve
    # every output line so unexpected extra hashes still cause a mismatch.
    while IFS=' 	' read -r _hash _hash_filename; do
        printf '%s\n' "$_hash"
    done <<EOF
$_hash_output
EOF
}

# verify_checksum validates the downloaded archive against the release
# checksums.txt. OpenHarmony prereleases require a valid checksum and a working
# SHA-256 tool before extraction or execution. Older stable platforms retain
# their warning-only fallback for missing checksums/tools. Release packaging
# must self-sign native OpenHarmony binaries before producing these checksums;
# the downloaded cjv cannot bootstrap its own executable signature.
verify_checksum() {
    _file=$1
    _name=$2
    if ! _sums="$(fetch_text "${UPDATE_ROOT}/checksums.txt")"; then
        checksum_unavailable "could not download checksums.txt"
        return 0
    fi
    _expected=
    _matches=0
    # A here-document keeps the loop in this shell and handles a final line
    # without a newline. read -r treats filenames as data, never shell input.
    while IFS=' 	' read -r _sum _sum_name _sum_extra; do
        case "$_sum_name" in
            "$_name"|"*$_name")
                _matches=$((_matches + 1))
                [ "$_matches" -eq 1 ] && [ -z "$_sum_extra" ] || err "invalid checksum entry for $_name"
                _expected=$_sum
                ;;
        esac
    done <<EOF
$_sums
EOF
    if [ -z "$_expected" ]; then
        checksum_unavailable "no checksum entry for $_name"
        return 0
    fi
    # Also rejects duplicate entries, truncated hashes and unexpected output.
    [ "${#_expected}" -eq 64 ] || err "invalid checksum entry for $_name"
    case "$_expected" in
        *[!a-fA-F0-9]*) err "invalid checksum entry for $_name" ;;
    esac
    # Normalize only the validated hex alphabet using shell builtins. Native
    # HarmonyOS Toybox may omit both awk and tr.
    _hex=$_expected
    _expected=
    while [ -n "$_hex" ]; do
        _hex_tail=${_hex#?}
        _hex_char=${_hex%"$_hex_tail"}
        case "$_hex_char" in
            A) _hex_char=a ;;
            B) _hex_char=b ;;
            C) _hex_char=c ;;
            D) _hex_char=d ;;
            E) _hex_char=e ;;
            F) _hex_char=f ;;
        esac
        _expected=$_expected$_hex_char
        _hex=$_hex_tail
    done
    if _actual="$(compute_sha256 "$_file")"; then
        :
    else
        case "$?" in
            1) checksum_unavailable "no sha256 tool available" ;;
            *) err "SHA-256 tool failed for $_name" ;;
        esac
        return 0
    fi
    if [ "$_actual" != "$_expected" ]; then
        err "checksum mismatch for $_name (expected $_expected, got $_actual)"
    fi
    say "checksum verified"
}

checksum_unavailable() {
    if [ "$PLATFORM" = "openharmony" ]; then
        err "$1; refusing to run an unverified OpenHarmony archive"
    fi
    warn "$1; skipping integrity verification"
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
