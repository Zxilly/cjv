#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd "$SCRIPT_DIR/../.." && pwd)
INSTALL_SH="$REPO_ROOT/web/public/install.sh"

TEST_SHELL=${CJV_INSTALL_TEST_SHELL:-sh}
TEST_MODE=${CJV_INSTALL_TEST_MODE:-none}
TEST_SHELL_NAME=$(basename "$TEST_SHELL")

TMP_PARENT=${TMPDIR:-/tmp}
TMP_ROOT=$(mktemp -d "$TMP_PARENT/cjv-install-script.XXXXXX")
TMP_ROOT=$(CDPATH= cd "$TMP_ROOT" && pwd -P)

cleanup() {
    rm -rf "$TMP_ROOT"
}

trap cleanup EXIT HUP INT TERM

say() {
    printf "install-sh-test: %s\n" "$1"
}

fail() {
    printf "install-sh-test: error: %s\n" "$1" >&2
    exit 1
}

require_file() {
    if [ ! -f "$1" ]; then
        fail "expected file missing: $1"
    fi
}

require_dir() {
    if [ ! -d "$1" ]; then
        fail "expected directory missing: $1"
    fi
}

require_contains() {
    file=$1
    needle=$2
    if ! grep -F "$needle" "$file" >/dev/null 2>&1; then
        fail "expected $file to contain: $needle"
    fi
}

require_any_toolchain() {
    toolchains_dir=$1
    require_dir "$toolchains_dir"
    if ! find "$toolchains_dir" -mindepth 1 -maxdepth 1 -type d | grep . >/dev/null 2>&1; then
        fail "expected at least one installed toolchain under $toolchains_dir"
    fi
}

run_installer() {
    cjv_home=$1
    shift

    shell_path=$(command -v "$TEST_SHELL" 2>/dev/null || true)
    if [ -z "$shell_path" ]; then
        fail "$TEST_SHELL is not installed"
    fi

    mkdir -p "$cjv_home/.config/fish"

    say "running install.sh through $TEST_SHELL"
    (
        unset CJV_UPDATE_ROOT CJV_GITHUB_ROOT CJV_GITCODE_ROOT CJV_MIRROR CJV_FALLBACK_SETTINGS CJV_NO_PATH_SETUP
        export CJV_HOME="$cjv_home"
        export HOME="$cjv_home"
        export USERPROFILE="$cjv_home"
        export CJV_LANG=en
        export CJV_TOOLCHAIN=

        case "$(basename "$shell_path")" in
            fish)
                "$shell_path" -c 'sh $argv' "$INSTALL_SH" "$@"
                ;;
            *)
                "$shell_path" -c 'sh "$@"' cjv-install "$INSTALL_SH" "$@"
                ;;
        esac
    )
}

assert_base_install() {
    cjv_home=$1

    require_file "$cjv_home/bin/cjv"
    require_file "$cjv_home/bin/cjc"
    require_file "$cjv_home/bin/cjpm"
    require_file "$cjv_home/env"
}

assert_path_config() {
    cjv_home=$1
    bin_dir="$cjv_home/bin"
    marker="# cjv (managed by cjv, do not edit)"

    case "$TEST_SHELL_NAME" in
        fish)
            fish_config="$cjv_home/.config/fish/config.fish"
            require_file "$fish_config"
            require_contains "$fish_config" "$marker"
            require_contains "$fish_config" "fish_add_path"
            require_contains "$fish_config" "$bin_dir"
            ;;
        zsh)
            for rc in .zshrc .zprofile; do
                path="$cjv_home/$rc"
                require_file "$path"
                require_contains "$path" "$marker"
                require_contains "$path" "export PATH="
                require_contains "$path" "$bin_dir"
            done
            ;;
        *)
            for rc in .profile .bashrc; do
                path="$cjv_home/$rc"
                require_file "$path"
                require_contains "$path" "$marker"
                require_contains "$path" "export PATH="
                require_contains "$path" "$bin_dir"
            done
            ;;
    esac
}

run_none_case() {
    cjv_home="$TMP_ROOT/none"
    run_installer "$cjv_home" -y --default-toolchain none --no-modify-path
    assert_base_install "$cjv_home"
}

run_lts_case() {
    cjv_home="$TMP_ROOT/lts"
    run_installer "$cjv_home" -y --no-modify-path
    assert_base_install "$cjv_home"
    require_any_toolchain "$cjv_home/toolchains"
}

run_path_case() {
    if [ "${CI:-}" != "true" ] && [ "${CJV_INSTALL_TEST_ALLOW_PATH_SETUP:-}" != "1" ]; then
        say "skipping PATH setup case outside CI"
        return
    fi

    cjv_home="$TMP_ROOT/path"
    run_installer "$cjv_home" -y --default-toolchain none
    assert_base_install "$cjv_home"
    assert_path_config "$cjv_home"
}

case "$TEST_MODE" in
    none)
        run_none_case
        ;;
    lts)
        run_lts_case
        ;;
    path)
        run_path_case
        ;;
    all)
        run_none_case
        run_lts_case
        run_path_case
        ;;
    *)
        fail "unsupported CJV_INSTALL_TEST_MODE: $TEST_MODE"
        ;;
esac

say "ok"
