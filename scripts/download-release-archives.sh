#!/usr/bin/env bash
# Resolve latest once for ordinary website builds; release callers provide a tag.
set -euo pipefail

tag="${RELEASE_TAG:-}"
if [[ -z "$tag" ]]; then
  tag=$(gh release view --repo "${GITHUB_REPOSITORY:?}" --json tagName --jq .tagName)
fi
if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "website installers require a stable release tag: $tag" >&2
  exit 1
fi

archives="${ARCHIVES_DIR:-archives}"
mkdir -p "$archives"
gh release download "$tag" --repo "${GITHUB_REPOSITORY:?}" --dir "$archives" \
  --pattern 'cjv_*' --pattern 'cjv-mirror_*' --pattern checksums.txt
echo "Downloaded website installers from $tag"
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  printf 'release_tag=%s\n' "$tag" >> "$GITHUB_OUTPUT"
fi
