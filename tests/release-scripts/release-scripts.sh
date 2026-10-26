#!/usr/bin/env bash
set -euo pipefail
repo=$(cd "$(dirname "$0")/../.." && pwd)
scratch=$(mktemp -d)
trap 'rm -rf -- "$scratch"' EXIT
mkdir -p "$scratch/archives" "$scratch/bin" "$scratch/source"

for platform in windows_amd64 darwin_amd64 darwin_arm64 linux_amd64 linux_arm64 openharmony_amd64 openharmony_arm64; do
  for variant in cjv cjv-mirror; do
    binary="$variant"
    if [[ "$platform" == windows_* ]]; then binary+='.exe'; fi
    printf '%s %s\n' "$variant" "$platform" > "$scratch/source/$binary"
    if [[ "$platform" == windows_* ]]; then
      (cd "$scratch/source" && zip -q "$scratch/archives/${variant}_${platform}.zip" "$binary")
    else
      tar -czf "$scratch/archives/${variant}_${platform}.tar.gz" -C "$scratch/source" "$binary"
    fi
  done
done
(cd "$scratch/archives" && sha256sum cjv_* cjv-mirror_* > checksums.txt)
export ARCHIVES_DIR="$scratch/archives" OUT_DIR="$scratch/output"
bash "$repo/scripts/extract-init-binaries.sh" > /dev/null
test "$(find "$OUT_DIR" -type f | wc -l)" -eq 14
test "$(cat "$OUT_DIR/mirror/openharmony_arm64/cjv-init")" = 'cjv-mirror openharmony_arm64'
test "$(cat "$OUT_DIR/official/windows_amd64/cjv-init.exe")" = 'cjv windows_amd64'

cp "$ARCHIVES_DIR/checksums.txt" "$scratch/valid-checksums.txt"
expect_rejected() {
  OUT_DIR="$scratch/rejected-output"
  export OUT_DIR
  if bash "$repo/scripts/extract-init-binaries.sh" > "$scratch/failure.log" 2>&1; then
    echo 'invalid release artifacts were accepted' >&2
    exit 1
  fi
  test ! -e "$OUT_DIR"
}
# Duplicate, missing, and malformed entries all fail before publishing files.
cat "$scratch/valid-checksums.txt" >> "$ARCHIVES_DIR/checksums.txt"
expect_rejected
sed '/cjv_windows_amd64.zip$/d' "$scratch/valid-checksums.txt" > "$ARCHIVES_DIR/checksums.txt"
expect_rejected
sed 's/^[a-f0-9]*/bad-digest/' "$scratch/valid-checksums.txt" > "$ARCHIVES_DIR/checksums.txt"
expect_rejected
cp "$scratch/valid-checksums.txt" "$ARCHIVES_DIR/checksums.txt"
printf 'tampered\n' >> "$ARCHIVES_DIR/cjv-mirror_openharmony_arm64.tar.gz"
expect_rejected

cat > "$scratch/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$GH_TEST_LOG"
case "$2" in
  view) printf 'v1.2.3\n' ;;
  download) test "$3" = "$GH_TEST_TAG" ;;
  *) exit 1 ;;
esac
EOF
chmod +x "$scratch/bin/gh"
export PATH="$scratch/bin:$PATH" GH_TEST_LOG="$scratch/gh.log" GH_TEST_TAG=v1.2.3
export GITHUB_REPOSITORY=example/cjv GITHUB_OUTPUT="$scratch/action-output"
RELEASE_TAG='' bash "$repo/scripts/download-release-archives.sh"
test "$(wc -l < "$GH_TEST_LOG")" -eq 2
grep -q '^release download v1.2.3 ' "$GH_TEST_LOG"
grep -q '^release_tag=v1.2.3$' "$GITHUB_OUTPUT"
: > "$GH_TEST_LOG"
GH_TEST_TAG=v1.0.0 RELEASE_TAG=v1.0.0 bash "$repo/scripts/download-release-archives.sh"
test "$(wc -l < "$GH_TEST_LOG")" -eq 1
grep -q '^release download v1.0.0 ' "$GH_TEST_LOG"
if RELEASE_TAG=invalid bash "$repo/scripts/download-release-archives.sh" 2>/dev/null; then exit 1; fi
test "$(wc -l < "$GH_TEST_LOG")" -eq 1
echo 'Release resolution and all 14 installer fixtures passed; invalid checksums rejected.'
