#!/usr/bin/env bash
set -euo pipefail
base=https://github.com/owenthereal/upterm/releases/download/v0.36.0
mkdir -p "$RUNNER_TEMP/upterm-download"
cd "$RUNNER_TEMP/upterm-download"
curl -fsSL "$base/upterm_linux_arm64.tar.gz" -o upterm_linux_arm64.tar.gz
curl -fsSL "$base/checksums.txt" -o checksums.txt
grep ' upterm_linux_arm64.tar.gz$' checksums.txt | sha256sum -c -
tar -xzf upterm_linux_arm64.tar.gz
sudo install -m 755 upterm /usr/local/bin/upterm
cd "$GITHUB_WORKSPACE"
upterm host --detach --accept --name harmonyos-debug --join-timeout 15m \
  --authorized-keys scripts/harmonyos-tcg/debug/authorized_keys \
  --known-hosts scripts/harmonyos-tcg/debug/known_hosts \
  --output json -- bash --noprofile --norc > emulator-diagnostics/debug-connection.json
