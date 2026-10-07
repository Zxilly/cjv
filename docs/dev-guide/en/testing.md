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
| Arguments, repeated invocations, JSON | `internal/cli`: real command trees, isolated writers, one result or error |
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
pnpm test
pnpm coverage
```

Vitest uses real browsers. `VITEST_BROWSER` selects chromium, firefox, or webkit. Platform integration tests run through `pnpm test:integration`, with CI supplying runner-specific `VITE_EXPECTED_*` values. Coverage thresholds live in `web/vitest.config.ts`.

## Documentation, workflows, and dependencies

```bash
npx --yes markdownlint-cli2@0.22.1
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
go mod verify
```

After dependency changes, run `go mod tidy` and review `go.mod` and `go.sum`. CI reruns tidy and requires no changes. Use the vulnerability scanner version in `.github/workflows/ci.yml`; it currently pins `govulncheck@v1.3.0` to avoid a Go 1.26 compatibility issue.

Documentation changes also require both language builds; see [documentation](documentation.md).

## Real downloads and CI

Live component download tests run separately and fetch real artifacts:

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

`.github/workflows/smoke.yml` schedules this test. The main `ci.yml` runs on PRs and master pushes, checking Go, mirror, builds and tests across five platforms, frontend browser matrices, installers, lint, and dependencies. Workflow files define current platforms and tool versions. See [documentation](documentation.md) and [releases](releasing.md) for site and release automation.
