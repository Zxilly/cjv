# Tests and checks

Choose the verification commands that match the scope of your change. Unit and CLI tests check flag parsing and output formatting, while `lifecycle` tests exercise the production installation entry points against local `httptest.Server` manifests and archives. Standard tests do not connect to live upstream servers. Read [Architecture](architecture.md) for package boundaries.

## Go builds, tests, and linters

### Standard Go verification

Run these commands from the repository root directory:

```bash
go build ./...
go test -race -count=1 -timeout 300s ./...
go vet ./...
golangci-lint run
```

- During local iteration, replace `./...` with the specific packages that you changed.
- `-race` enables the data-race detector, and `-count=1` disables cached test results.
- Format all changed Go files with `gofmt`, and run `gofmt -l .` to check for unformatted files. Linter rules live in `.golangci.yml`.

### Mirror build verification

If you change download logic, self-update code, or shared interfaces in `config` or `selfupdate`, test the `-tags=mirror` variant as well:

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

### End-to-end CLI integration tests

Integration tests compile the real `cjv` binary and run it in isolated child processes:

```bash
go test -race -tags integration -count=1 ./tests/integration/
```

When writing tests, always isolate the user home directory, `CJV_HOME`, and system fallback settings (using `configtest`). Setting `CJV_HOME` alone does not isolate `~/.cjv/settings.toml`. `PATH` setup tests use temporary shell profile files on Unix and save/restore the user `PATH` registry value on Windows.

## Where to place regression tests

Add new tests to the package that owns the behavior under test:

| Behavior under test | Target package and needed assertions |
| --- | --- |
| CLI flags, repeated arguments, JSON output | `internal/cli` and its `*cmd` subpackages: use `clitest.Root`, isolated `io.Writer` buffers, and assert a single result or error |
| Install, update, rollback, uninstall | `internal/lifecycle`: use a mock local distribution, and assert final directory contents, `settings.toml` references, and crash recovery |
| Host and cross-target isolation | `internal/toolchain`, `internal/lifecycle`: assert installation identity, host platform, and exact release matching |
| Component file ownership | `internal/component`: assert shared SDK files, path-traversal rejection, symlinks, and backup restoration |
| Process interruption and journals | `internal/fstx` and `lifecycle` recovery tests: assert real journal replay after an interrupted child process exits |
| HTTP resume, cancellation, retries | `internal/dist`: assert HTTP headers (`Range`), SHA-256 verification, and total request counts |
| Runtime environment and child processes | `internal/env`, `internal/process`: assert merged environment variables, child exit codes, and signal forwarding |

Always assert observable outcomes. Use `testutil.ProgressRecorder` to check progress event types rather than matching translated human-readable strings. For failure recovery tests, check that blocked recovery keeps backup files intact and that rejected paths never change files outside `<CJV_HOME>`.

## Installer scripts and web frontend

### Installer script tests

Run the Unix installer test suite in Bash:

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

Run the PowerShell installer test suite:

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"
./tests/install-scripts/install-ps1.ps1
```

CI runs these suites across `sh`, `bash`, `zsh`, `fish`, `pwsh`, and Windows PowerShell 5.1.

### Web landing page tests

Run these commands inside `web/`:

```bash
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm build
pnpm coverage
pnpm test:built
```

- Vitest runs component tests in real browsers via Playwright. Set `VITEST_BROWSER` to select `chromium`, `firefox`, or `webkit`.
- `pnpm test:integration` tests every simulated OS and architecture combination across browsers.
- `pnpm coverage` runs the unit test suite and enforces the coverage thresholds in `web/vitest.config.ts`.
- `pnpm test:built` serves the production `dist/` output and checks platform detection, content visibility when font downloads hang, and layout overflow on desktop and mobile viewports. Set `PLAYWRIGHT_BROWSER` to select the browser. Read [Landing page](web.md) for frontend architecture.

## Documentation, workflows, and dependencies

Run these checks from the repository root before opening a pull request:

```bash
npx --yes markdownlint-cli2@0.22.1
python3 scripts/build-docs.py
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
go mod verify
```

- **Markdown and mdBook**: `markdownlint-cli2` checks Markdown formatting (`.markdownlint-cli2.jsonc`), and `scripts/build-docs.py` builds all four books (`user-guide` and `dev-guide` in Chinese and English) and checks every internal link, asset, and heading anchor. Read [Documentation site](documentation.md).
- **Go dependencies**: After changing dependencies, run `go mod tidy` and inspect `go.mod` and `go.sum`. CI runs `go mod tidy` and fails if the files are dirty. Security checks in `.github/workflows/ci.yml` run the pinned `govulncheck` version on the current stable Go release.
- **Release script fixtures**: On an environment with `bash`, `zip`, `unzip`, `tar`, and `sha256sum`, test the CI scope filter and release archive scripts:

```bash
python3 scripts/test_ci_scope.py
bash tests/release-scripts/release-scripts.sh
```

## HarmonyOS emulator and live download tests

### HarmonyOS emulator CI and local runs

The `.github/workflows/hmos.yml` workflow boots the official HarmonyOS 6.1.1 (API 24) x86_64 PC emulator on a Linux KVM runner. It cross-compiles a standalone self-signing test binary, runs it both unsigned and signed, and then executes the full test suites for `ohos`, `config`, `env`, `fsops`, `target`, `cjverr`, `i18n`, and `retry` inside the emulator via `go test -exec`. Standard host CI covers the remaining packages (including HTTP server tests). This emulator workflow runs x86_64 binaries only, not ARM64.

Two tools under `scripts/harmonyos/` drive the emulator tests:

- `scripts/harmonyos/ci.py` runs the three workflow stages directly through Python `subprocess`: `build` (compiles host tools), `install` (installs the pinned emulator and Ubuntu packages), and `test` (runs the unsigned, signed, exit-code, and package test suites). After running `build`, you can run `python3 scripts/harmonyos/emulator.py python3 scripts/harmonyos/ci.py test` to reproduce the full CI test sequence locally.
- `scripts/harmonyos/exec` is a host Go wrapper for `go test -exec`. It pushes each test binary into the emulator over HDC, maps the package working directory into the guest source checkout, creates isolated `HOME` and `TMPDIR` directories under `/data/local/tmp`, and forwards the guest exit code. Set `HDC`, `HDC_TARGET`, `HMOS_TEST_SOURCE`, and `HMOS_TEST_ROOT` (under `/data/local/tmp`). Set `HMOS_SIGN` to sign a copy of the binary before transfer, or leave `HMOS_SIGN` empty to test unsigned execution.

With the official HarmonyOS CLI unpacked at `HMOS_EMULATOR_HOME` and the tools from [Build from source](building.md#openharmony) ready, run a package test suite in the emulator like this:

```bash
go build -o /tmp/cjv-hmos-exec ./scripts/harmonyos/exec
python3 scripts/harmonyos/emulator.py env GOOS=openharmony GOARCH=amd64 CGO_ENABLED=0 \
  "$HMOS_GO" test -exec /tmp/cjv-hmos-exec -p 1 -count=1 ./internal/ohos/...
```

In this emulator image, the HDC shell runs as `uid 2000` with SELinux `Enforcing`. It allows unsigned binaries and symlinks, but blocks hard links (`link`). Tests that need hard-link creation skip when they confirm `EPERM`/`EACCES`, while deduplication tests still check the copy fallback. Because physical HarmonyOS PCs enforce stricter HiShell code-signing policies than the HDC shell, release builds and SDK extraction keep mandatory OpenHarmony self-signing enabled.

### Live upstream smoke tests and CI scope

Live component download smoke tests fetch real archives from upstream servers and do not run during standard `go test` runs:

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

- `.github/workflows/smoke.yml` runs this smoke test on a schedule.
- `.github/workflows/ci.yml` runs on every pull request and `master` push. It uses `scripts/ci-scope.py` to select heavy jobs according to the changed file paths; changes to workflows or `ci-scope.py` trigger all jobs.
- Pushing new commits to a pull request cancels superseded workflow runs, whereas `master` pushes never cancel in-progress runs.
- Manual (`workflow_dispatch`) and weekly scheduled runs execute the full matrix, including native Linux ARM64, Windows ARM64, and macOS Intel browser checks. Read [Releases](releasing.md) for release workflow details.
