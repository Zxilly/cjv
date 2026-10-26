# Testing and checks

Choose validation according to the change. CLI tests cover parsing and output. Installation behavior is tested through production `lifecycle` entry points, using `httptest.Server` for manifests and archives. Ordinary tests do not depend on live distributions.

## Go

From the repository root:

```bash
go build ./...
go test -race -count=1 -timeout 300s ./...
go vet ./...
golangci-lint run
```

Replace `./...` with affected packages during development. `-race` detects races; `-count=1` disables test caching. Format changed files with `gofmt`, and use `gofmt -l .` to list unformatted files. Lint rules live in `.golangci.yml`.

For download, self-update, or shared-interface changes, also validate mirror builds:

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

CLI end-to-end tests compile and run the actual binary:

```bash
go test -race -tags integration -count=1 ./tests/integration/
```

Isolate the user home, `CJV_HOME`, and fallback settings in tests. Setting only `CJV_HOME` does not isolate user `settings.toml`. PATH tests use temporary shell files; Windows tests guard and restore the registry PATH.

## Where regression tests belong

| Behavior | Location and assertions |
| --- | --- |
| Arguments, repeated invocations, JSON | `internal/cli` and its command packages: real command trees (`clitest.Root`), isolated writers, one result or error |
| Install, update, remove | `internal/lifecycle`: local distribution, final contents, settings references, and recovery |
| Host-target isolation | `internal/toolchain`, `internal/lifecycle`: identity, platform, and release matching |
| Component ownership | `internal/component`: shared files, escaping manifests, links, and backup restoration |
| Process interruption | `internal/fstx`, lifecycle regressions: actual journal recovery after child exit |
| Resume, cancellation, retries | `internal/dist`: HTTP requests, Range, hashes, and request counts |
| Environment and processes | `internal/env`, `internal/process`: supplied environments, real child exit codes, and signals |

Assert observable results. Use `testutil.ProgressRecorder` to check progress event kinds rather than translated messages. Verify that failed recovery retains backups and rejected paths leave external files untouched.

## Installers and frontend

Unix installer:

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

PowerShell installer:

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"
./tests/install-scripts/install-ps1.ps1
```

CI also covers sh, zsh, fish, and Windows PowerShell 5.1. For the frontend, run in `web/`:

```bash
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm build
pnpm coverage
pnpm test:built
```

Vitest uses real browsers. `VITEST_BROWSER` selects chromium, firefox, or webkit; `pnpm test:integration` runs every simulated platform in each browser. `pnpm coverage` combines unit tests and coverage checks. After building, `pnpm test:built` checks installation paths using the actual browser platform, content visibility with pending font downloads, and desktop/narrow-screen overflow. `PLAYWRIGHT_BROWSER` selects its browser. Coverage thresholds live in `web/vitest.config.ts`.

## Documentation, workflows, and dependencies

```bash
npx --yes markdownlint-cli2@0.22.1
python3 scripts/build-docs.py
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
go mod verify
```

After dependency changes, run `go mod tidy` and review `go.mod` and `go.sum`. CI reruns tidy and requires no changes. Vulnerability checks use the pinned `govulncheck` version in `.github/workflows/ci.yml` with the current stable Go release. Check the project's supported Go versions when upgrading the scanner.

The shared documentation script builds all four books and checks internal pages, assets, and anchors. Pull requests and deployments use the same entry point; see [documentation](documentation.md).

## Real downloads and CI

### HarmonyOS emulator tests

The separate `hmos.yml` workflow boots the official HarmonyOS 6.1.1/API 24 PC emulator on an x86_64 Linux KVM runner. It cross-compiles a standalone self-signing test binary and executes it both unsigned and signed. It then runs the complete `ohos`, `config`, `env`, `fsops`, `target`, `cjverr`, `i18n`, and `retry` package tests with `go test -exec`. Host CI continues to cover the remaining packages, including HTTP server tests. This workflow does not execute ARM64 binaries.

`scripts/harmonyos/ci.py` owns the workflow's three steps: `build` compiles the tools, `install` installs the pinned emulator and Ubuntu dependencies, and `test` runs the unsigned, signed, exit-code, and package checks. It invokes programs directly through Python's `subprocess`, without Bash orchestration. Run `python3 scripts/harmonyos/emulator.py python3 scripts/harmonyos/ci.py test` after `build` to reproduce the CI test sequence locally.

`scripts/harmonyos/exec` is a host Go executable that transfers a binary through HDC, maps its package directory to a guest source checkout, supplies isolated `HOME` and `TMPDIR` directories, and propagates the guest exit code. Set `HDC`, `HDC_TARGET`, `HMOS_TEST_SOURCE`, and `HMOS_TEST_ROOT`; the latter must be below `/data/local/tmp`. Set `HMOS_SIGN` to sign a private copy before transfer, or leave it empty to test unsigned execution. `scripts/harmonyos/emulator.py` prepares the guest checkout and these variables, runs the supplied command, then stops the emulator.

With the official CLI unpacked under `HMOS_EMULATOR_HOME` and the tools from [building](building.md):

```bash
go build -o /tmp/cjv-hmos-exec ./scripts/harmonyos/exec
python3 scripts/harmonyos/emulator.py env GOOS=openharmony GOARCH=amd64 CGO_ENABLED=0 \
  "$HMOS_GO" test -exec /tmp/cjv-hmos-exec -p 1 -count=1 ./internal/ohos/...
```

The HDC shell runs as uid 2000 with SELinux enforcing. It permits unsigned executables and symlinks in this image, but denies hard links: only tests that require successful hard-link creation skip on a verified permission error; deduplication still tests its fallback behavior. This emulator result does not establish the HiShell policy on physical PCs, so releases and SDK preparation retain platform-gated self-signing.

### Live downloads

Live component download tests run separately and fetch real artifacts:

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

`.github/workflows/smoke.yml` schedules this test. The main `ci.yml` runs on PRs and master pushes, checking Go, mirror, builds and tests across five platforms, frontend browser matrices, installers, lint, and dependencies. Workflow files define current platforms and tool versions. See [documentation](documentation.md) and [releases](releasing.md) for site and release automation.

Main CI always responds to PRs and master pushes, then selects expensive jobs by changed paths using `scripts/ci-scope.py`. Workflow or scope-rule changes run all checks. New PR updates cancel old runs; master pushes do not. Normal frontend checks run simulated platforms in all three browsers and built-page smoke tests on Linux/Windows Chromium and macOS WebKit. Manual and weekly runs select all jobs and add native Linux ARM64, Windows ARM64, and macOS Intel checks. HarmonyOS uses the same path classification and PR cancellation policy in its separate workflow.

Run isolated release-script fixtures with Bash, zip, unzip, tar, and sha256sum installed:

```bash
python3 scripts/test_ci_scope.py
bash tests/release-scripts/release-scripts.sh
```
