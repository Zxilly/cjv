#!/usr/bin/env bash
set -euo pipefail
for GOOS in linux darwin windows; do
  for GOARCH in amd64 arm64; do
    echo "Building ${GOOS}/${GOARCH}..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o /dev/null ./cmd/cjv/
  done
done
