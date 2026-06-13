# Testing

cjv's tests are split into several layers: Go unit tests cover the individual packages under `internal/`, Go integration tests in `tests/integration/` build a real `cjv` binary and run commands end to end, the install-script integration tests run `install.sh` and `install.ps1` under a variety of shells, the landing page runs frontend tests in vitest's browser mode, and the smoke tests pull every component from the real distribution source on a daily schedule. Each layer has a corresponding CI job. This chapter goes through them layer by layer, explaining how to run each one locally and what it verifies.

All CI jobs are defined in `.github/workflows/ci.yml`, with smoke tests separately in `.github/workflows/smoke.yml`. You do not need to run the whole suite while developing; locally you usually run only the unit tests plus whichever layer you touched.

## Unit tests

Unit tests live in the same package as the code under test, with file names ending in `_test.go`, scattered throughout `internal/`, currently over a hundred files. They use the `assert` and `require` helpers from `github.com/stretchr/testify` for assertions and do not depend on the network: when a distribution source is needed, they use `MockDistServer` in `internal/testutil` to spin up an `httptest.Server` that returns a constructed `sdk-versions.json` and a packaged mock SDK.

Run all unit tests locally:

```bash
go test -race -count=1 ./...
```

`-race` enables the race detector, where most concurrency-related bugs show up. `-count=1` disables the test result cache, guaranteeing the tests actually run again instead of printing the previous cached results. CI's `go-test` job adds `-timeout 300s` and runs the tests on a matrix of five platforms: Linux x86_64, Linux arm64, macOS arm64, macOS Intel, and Windows. The `go-test` job also runs `go build ./...` and `go vet ./...` to make sure every package compiles and `vet` is clean.

To run a single package, replace `./...` with the package path:

```bash
go test -race -count=1 ./internal/config/
```

### The mirror variant

Some code is controlled by the `mirror` build tag (`internal/config/manifest_mirror.go`, `internal/selfupdate/update_mirror.go`, and others), providing an alternative implementation for the Chinese mainland mirror distribution. By default this path is not compiled in; you have to pass the tag explicitly. In `go-test`, CI separately verifies that the mirror variant compiles, that the `selfupdate` mirror tests pass, and that `vet` is clean:

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

When you touch `*_mirror.go` or the interfaces they depend on, remember to run again with `-tags=mirror`; a plain `go test ./...` does not compile these files.

## Go integration tests

The integration tests live in `tests/integration/`, and every file begins with `//go:build integration`, so the default `go test ./...` does not see them. This is deliberate: the integration tests build a `cjv` binary on the fly and then invoke it repeatedly, making them much slower than unit tests, and they should not be mixed into the fast everyday loop.

To run the integration tests, pass the tag explicitly:

```bash
go test -race -v -tags integration -count=1 ./tests/integration/
```

This is exactly the command CI's `go-integration` job runs on the same five-platform matrix. `TestMain` (in `helpers_test.go`) builds the binary once before any cases run, using `go build -o ... ./cmd/cjv`, and caches it, so every subsequent case reuses the same artifact instead of recompiling. Each case gets an isolated `CJV_HOME` via `t.TempDir()`, writes the mock server's `manifest_url` into `settings.toml`, places the cjv binary into `bin/` ahead of time, and then actually runs commands like `cjv install`, `cjv default`, `cjv which`, and `cjv completions`, asserting on their output and on-disk results. Every invocation passes `CJV_NO_PATH_SETUP=1` to keep the tests from modifying the real environment's PATH.

Platform-specific cases are split using combined build tags. PATH injection edits shell config files on Unix-like systems and the registry on Windows, so it is split into two files:

```go
//go:build integration && !windows
```

```go
//go:build integration && windows
```

On Windows the integration tests read and write the user PATH in the registry. Before running the tests, `runWithRegistryGuard` in `testmain_windows_test.go` saves the current value with `testutil.SaveRegistryPath` and restores it afterward with `Restore`, so the tests do not pollute the developer machine's registry. The same-named function on non-Windows platforms is an empty shim that just calls `m.Run()`.

## Install-script integration tests

The two install scripts the landing page provides, `web/public/install.sh` and `web/public/install.ps1`, have their own integration tests in `tests/install-scripts/`. These tests are themselves shell scripts, not Go tests: they actually run the install flow in a temporary directory and then assert that the files, directories, and PATH configuration all ended up where expected.

The Unix side is `install-sh.sh`, which uses an environment variable to choose which shell to run the script under test:

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

CI's `install-script-integration` job uses a matrix to run under `sh`, `bash`, `zsh`, and `fish`, covering Linux x86_64, Linux arm64, macOS arm64, and macOS Intel; shells that are not installed by default, such as `zsh` and `fish`, are installed first by the job with `apt-get` or `brew`. `CJV_INSTALL_TEST_MODE` controls which install mode is asserted.

The Windows side is `install-ps1.ps1`, which uses `CJV_INSTALL_TEST_POWERSHELL` to choose the PowerShell host:

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"; ./tests/install-scripts/install-ps1.ps1
```

On Windows, CI runs both the system's legacy `powershell.exe` (Windows PowerShell 5.1) and the newer `pwsh` (PowerShell 7+), because the two behave differently; `CJV_INSTALL_TEST_DEFAULT_TOOLCHAIN` controls which toolchains are installed by default.

## Landing page tests

The landing page lives in `web/` and is tested with vitest, with test files named `src/**/*.test.{ts,tsx}`. Unlike most frontend projects, this one runs in vitest's browser mode rather than jsdom: in `web/vitest.config.ts`, `browser.enabled` is `true`, the provider is `@vitest/browser-playwright`, and Playwright drives a real browser. This lets platform detection and APIs that only exist in a real browser, such as `navigator.userAgentData`, be tested faithfully. The browser is chosen by the `VITEST_BROWSER` environment variable (`chromium`, `firefox`, `webkit`), defaulting to `chromium`.

All frontend tests are under the `web/` directory and run with pnpm. First install the dependencies and the Playwright browsers:

```bash
cd web
pnpm install
pnpm exec playwright install chromium
```

Then run the tests:

```bash
pnpm test
```

`pnpm test` maps to `vitest run`, which runs all cases. It initializes via `src/test-setup.ts`: wiring up `@testing-library/jest-dom`'s assertions, stubbing `navigator.clipboard`, and pinning the i18n language to Chinese (the cases assert against the Chinese source strings).

Platform detection has a separate set of integration tests in `src/App.platform.integration.test.tsx`, which verify, across different operating systems and architectures, that the download recommendation the landing page gives is correct. They are run via `pnpm test:integration`:

```bash
pnpm test:integration
```

CI keeps this set of cases separate from the rest: the `web-test` job runs the remaining tests with `--exclude src/App.platform.integration.test.tsx`, while the platform integration tests are run by the `web-browser-integration` job on a matrix of chromium, firefox, and webkit times a full column of real runners (various OSes plus architectures). Each combination injects, via a set of `VITE_EXPECTED_*` environment variables, the platform, architecture, and download label that runner should be recognized as, and the tests then assert that the detection result matches. This confirms that on real Windows arm64, macOS Intel, Linux arm64, and other environments, platform identification and download recommendations are correct.

### Coverage

Coverage is collected using vitest's v8 provider:

```bash
cd web
pnpm coverage
```

`web/vitest.config.ts` sets four thresholds, and falling below any one of them fails:

```ts
thresholds: {
  lines: 80,
  functions: 80,
  branches: 80,
  statements: 80,
}
```

Coverage is measured over `src/**/*.{ts,tsx}`, but excludes the test files themselves, `test-setup.ts`, `test-utils.tsx`, `src/components/ui/**` (third-party UI components), `src/locales/**`, `src/main.tsx`, and `src/vite-env.d.ts`. CI's `web-coverage` job runs `pnpm coverage`, and the job goes red immediately if a threshold is not met. When you add logic to the landing page, remember to add the corresponding tests, otherwise you may push coverage below the threshold.

## Smoke tests

The smoke tests live in `tests/smoke/`, carry the `//go:build smoke` tag, and do not take part in any regular test run by default. The key difference from the integration tests is this: the integration tests use a mock server, while the smoke tests really go to the live distribution source to download every known component of LTS and STS, verifying that the manifest URLs, file hashes, and unpacking flow still hold up against real data. Such tests are slow and depend on external services, so they are not suitable for the PR loop.

They are run by `.github/workflows/smoke.yml` once a day at 08:00 UTC on a schedule, and can also be triggered manually with `workflow_dispatch`. To reproduce locally:

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

Because the download volume is large, the timeout is set to 45 minutes, and in CI the per-download timeout is relaxed via `CJV_DOWNLOAD_TIMEOUT` and throughput is improved with BBR. You generally do not need to run this in everyday development, but running it once after touching code that faces the real distribution source directly, such as component download, unpacking, or manifest parsing, can catch changes in the live data format early.

## Run what you changed

For everyday Go changes, run `go test -race -count=1 ./...` first; if you touched `*_mirror.go`, run again with `-tags=mirror`; if you changed end-to-end command-line behavior or PATH/proxy link logic, run the `-tags integration` tests; if you touched `install.sh` or `install.ps1`, run the corresponding install-script test locally under at least one shell; if you changed the landing page, go into `web/` and run `pnpm test`, and if platform detection is involved, also run `pnpm test:integration`, and after adding logic run `pnpm coverage` to check the thresholds. CI reruns every layer for you on the full platform matrix, but passing locally first saves the round-trip time of waiting on CI. For the full picture of each job and its trigger conditions, see [Continuous integration](ci.md).
