#!/usr/bin/env bash
# Extract release archives into web/dist/dl/{official,mirror}/{platform}/cjv-init[.exe].
# Used by the Pages workflow and as a local-test helper.
#
# Inputs:
#   ARCHIVES_DIR  Directory containing the downloaded archives (default: ./archives)
#   OUT_DIR       Output base directory (default: web/dist/dl)
#
# Archive naming follows goreleaser:
#   cjv_<os>_<arch>.zip|tar.gz         -> official/<os>_<arch>/cjv-init[.exe]
#   cjv-mirror_<os>_<arch>.zip|tar.gz  -> mirror/<os>_<arch>/cjv-init[.exe]

set -euo pipefail

ARCHIVES_DIR="${ARCHIVES_DIR:-./archives}"
OUT_DIR="${OUT_DIR:-web/dist/dl}"

PLATFORMS=(
  "windows_amd64 zip"
  "darwin_amd64 tar.gz"
  "darwin_arm64 tar.gz"
  "linux_amd64 tar.gz"
  "linux_arm64 tar.gz"
)

extract_one() {
  local variant="$1" platform="$2" ext="$3"
  local archive_prefix init_basename src
  if [[ "$variant" == "official" ]]; then
    archive_prefix="cjv"
  else
    archive_prefix="cjv-mirror"
  fi
  if [[ "$platform" == windows_* ]]; then
    init_basename="cjv-init.exe"
    src="${archive_prefix}.exe"
  else
    init_basename="cjv-init"
    src="${archive_prefix}"
  fi
  local archive="${ARCHIVES_DIR}/${archive_prefix}_${platform}.${ext}"
  local out_dir="${OUT_DIR}/${variant}/${platform}"
  mkdir -p "${out_dir}"
  local tmp
  tmp=$(mktemp -d)

  if [[ "$ext" == "zip" ]]; then
    unzip -q -o "${archive}" -d "${tmp}"
  else
    tar -xzf "${archive}" -C "${tmp}"
  fi

  if [[ ! -f "${tmp}/${src}" ]]; then
    rm -rf "${tmp}"
    echo "missing ${src} inside ${archive}" >&2
    return 1
  fi

  install -m 0755 "${tmp}/${src}" "${out_dir}/${init_basename}"
  rm -rf "${tmp}"
  echo "extracted ${variant}/${platform}/${init_basename}"
}

main() {
  for entry in "${PLATFORMS[@]}"; do
    read -r platform ext <<<"${entry}"
    extract_one official "${platform}" "${ext}"
    extract_one mirror "${platform}" "${ext}"
  done
}

main "$@"
