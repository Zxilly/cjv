# Testing and checks

Choose your validation commands according to the scope of your change. CLI unit tests cover flag parsing and output rendering, while installation and update behavior is tested through production `lifecycle` entry points using local `httptest.Server` fixtures for manifests and archives. Ordinary unit and integration tests never depend on live upstream distribution servers.

## Go

From the repository root, run the standard build, test, and lint suite:

```bash
go build ./...
go test -race -count=1 -timeout 300s ./...
go vet ./...
golangci-lint run
```

During local iteration, you can replace `./...` with the specific packages you touched. `-race` enables Go's race detector, and `-count=1` disables cached test results. Format all modified Go files with `gofmt`, and use `gofmt -l .` to check for unformatted files. Linter rules are configured in `.golangci.yml`.

Whenever you touch download logic, self-update code, or shared interfaces in `config` or `selfupdate`, also validate the `mirror` build variant:

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

End-to-end CLI integration tests compile and invoke the real `cjv` binary in isolated child processes:

```bash
go test -race -tags integration -count=1 ./tests/integration/
```

When writing tests, always isolate the user home directory, `CJV_HOME`, and system fallback settings (using `configtest`). Remember that setting `CJV_HOME` alone does **not** isolate `~/.cjv/settings.toml`. `PATH` setup tests use temporary shell profile files on Unix, and guard and restore the user `PATH` registry key on Windows.

## Where regression tests belong

Place new regression tests in the package that owns the behavior under test:

| Behavior | Location and assertions |
| --- | --- |
| CLI arguments, repeated flags, JSON output | `internal/cli` and its `*cmd` packages: run against real command trees (`clitest.Root`) with isolated writers, asserting one result or error |
| Install, update, rollback, remove | `internal/lifecycle`: test against a local mock distribution, asserting final directory contents, settings references, and failure recovery |
| Host-target isolation | `internal/toolchain`, `internal/lifecycle`: assert installation identity, host platform, and exact release matching |
| Component ownership | `internal/component`: assert shared SDK files, path-escaping manifests, symlinks, and backup restoration |
| Process interruption and journals | `internal/fstx` and `lifecycle` regressions: assert actual journal recovery after an interrupted child process exits |
| HTTP resume, cancellation, retries | `internal/dist`: assert HTTP requests, `Range` headers, SHA-256 hashes, and total request counts |
| Runtime environment and processes | `internal/env`, `internal/process`: assert supplied base environments, real child exit codes, and signal forwarding |

Always assert observable outcomes. Use `testutil.ProgressRecorder` to verify progress event kinds rather than matching localized text strings. In failure-recovery tests, verify that blocked recovery retains backup directories and that rejected paths leave files outside `<CJV_HOME>` untouched.

## Installers and frontend

To run the Unix shell installer test suite:

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

To run the PowerShell installer test suite:

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"
./tests/install-scripts/install-ps1.ps1
```

In CI, these suites also run across `sh`, `zsh`, `fish`, and Windows PowerShell 5.1.

For the landing page in `web/`, run:

```bash
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm build
pnpm coverage
pnpm test:built
```

Vitest runs component tests in real browsers via Playwright:

- `VITEST_BROWSER` selects `chromium`, `firefox`, or `webkit`.
- `pnpm test:integration` runs every simulated OS and architecture combination in each browser.
- `pnpm coverage` combines unit tests with the coverage thresholds configured in `web/vitest.config.ts`.
- After `pnpm build`, `pnpm test:built` serves the built bundle and verifies installation paths using the browser's actual platform, content visibility when font downloads are pending, and layout overflow on both desktop and narrow mobile viewports (`PLAYWRIGHT_BROWSER` selects its browser).

## Documentation, workflows, and dependencies

Before submitting a pull request, run the documentation, workflow, and module checks from the repository root:

```bash
npx --yes markdownlint-cli2@0.22.1
python3 scripts/build-docs.py
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
go mod verify
```

After changing Go dependencies, run `go mod tidy` and review the diff in `go.mod` and `go.sum`—CI re-runs `go mod tidy` and requires a clean working tree. Vulnerability scanning in `.github/workflows/ci.yml` runs the pinned `govulncheck` version on the current stable Go release; check the project's supported Go versions whenever upgrading the scanner.

`scripts/build-docs.py` builds all four mdBook manuals (`user-guide` and `dev-guide` in Chinese and English) with the shared theme and verifies all internal page links, cross-book `/book/` links, static assets, and `#anchor` fragments. Both PR checks and production deployments use this exact entry point; see [Documentation site](documentation.md).

## Real downloads and CI

### HarmonyOS emulator tests

The separate `.github/workflows/hmos.yml` workflow boots the official HarmonyOS 6.1.1 (API 24) PC emulator on an x86_64 Linux KVM runner. It cross-compiles a standalone self-signing test binary and executes it both unsigned and signed, and then runs the complete `ohos`, `config`, `env`, `fsops`, `target`, `cjverr`, `i18n`, and `retry` package test suites inside the emulator using `go test -exec`. Standard host CI continues to cover the remaining packages (including HTTP server tests). Note that this emulator workflow executes x86_64 binaries only, not ARM64.

Two helpers under `scripts/harmonyos/` coordinate the emulator runs:

- **`scripts/harmonyos/ci.py`** owns the workflow's three stages: `build` compiles the host tools, `install` installs the pinned emulator and Ubuntu dependencies, and `test` runs the unsigned, signed, exit-code, and package checks. It invokes programs directly via Python's `subprocess` without Bash orchestration. After running `build`, you can run `python3 scripts/harmonyos/emulator.py python3 scripts/harmonyos/ci.py test` to reproduce the CI test sequence locally.
- **`scripts/harmonyos/exec`** is a host Go executable for `go test -exec` that transfers a compiled test binary through HDC, maps its package directory to a guest source checkout, supplies isolated `HOME` and `TMPDIR` directories, and propagates the guest exit code. Set `HDC`, `HDC_TARGET`, `HMOS_TEST_SOURCE`, and `HMOS_TEST_ROOT` (which must live under `/data/local/tmp`). Set `HMOS_SIGN` to sign a private copy before transfer, or leave it empty to test unsigned execution. `scripts/harmonyos/emulator.py` prepares the guest checkout and environment variables, runs the supplied command, and stops the emulator cleanly.

With the official HarmonyOS CLI unpacked under `HMOS_EMULATOR_HOME` and the tools from [Build from source](building.md#openharmony) built:

```bash
go build -o /tmp/cjv-hmos-exec ./scripts/harmonyos/exec
python3 scripts/harmonyos/emulator.py env GOOS=openharmony GOARCH=amd64 CGO_ENABLED=0 \
  "$HMOS_GO" test -exec /tmp/cjv-hmos-exec -p 1 -count=1 ./internal/ohos/...
```

In this emulator image, the HDC shell runs as `uid 2000` with SELinux `Enforcing`. It permits unsigned executables and symlinks, but denies hard links: only tests that require successful hard-link creation skip on a verified permission error (`EPERM`/`EACCES`), while deduplication tests still verify their copy fallback behavior. Because this HDC shell policy does not establish the HiShell policy on physical HarmonyOS PCs, release builds and SDK preparation retain platform-gated self-signing.

### Live downloads and CI scope

Live component download smoke tests fetch real upstream artifacts and run separately from ordinary unit tests:

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

`.github/workflows/smoke.yml` runs this smoke test on a schedule. The main `.github/workflows/ci.yml` workflow runs on every pull request and `master` push, using `scripts/ci-scope.py` to select expensive jobs according to which files changed (changes to workflow files or scope rules run all checks). New commits pushed to a PR cancel older in-progress runs, whereas `master` pushes never cancel.

Normal frontend CI checks run simulated platforms across all three browsers (`chromium`, `firefox`, `webkit`) plus built-page smoke tests on Linux/Windows Chromium and macOS WebKit. Manual and weekly scheduled runs select all jobs and add native Linux ARM64, Windows ARM64, and macOS Intel checks. The HarmonyOS workflow uses the same path-scoping and PR cancellation policy.

To run the isolated CI scope and release-script fixture tests locally (requires `bash`, `zip`, `unzip`, `tar`, and `sha256sum`):

```bash
python3 scripts/test_ci_scope.py
bash tests/release-scripts/release-scripts.sh
```
