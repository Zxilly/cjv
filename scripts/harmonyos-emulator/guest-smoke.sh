#!/system/bin/sh
set -eu

# HDC's shell defaults to HOME=/; use an isolated writable CI home.
export HOME=/data/local/tmp/cjv-ci/home
export CJV_HOME="$HOME/.cjv"
export TMPDIR=/data/local/tmp/cjv-ci/tmp
mkdir -p "$HOME" "$CJV_HOME" "$TMPDIR"

cat > /data/local/tmp/cjv-ci/env.sh <<'EOF'
export HOME=/data/local/tmp/cjv-ci/home
export CJV_HOME="$HOME/.cjv"
export TMPDIR=/data/local/tmp/cjv-ci/tmp
EOF

printf 'cjv-filesystem-check\n' > "$TMPDIR/write-check"
mv "$TMPDIR/write-check" "$TMPDIR/rename-check"
test "$(cat "$TMPDIR/rename-check")" = cjv-filesystem-check
rm "$TMPDIR/rename-check"

id
uname -m
printf 'HOME=%s\nCJV_HOME=%s\nTMPDIR=%s\n' "$HOME" "$CJV_HOME" "$TMPDIR"
printf 'CJV_ENV_READY\n'
