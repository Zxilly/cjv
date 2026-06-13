# Continuous Integration

All of cjv's automation lives under `.github/workflows/`, with four workflows in total:

- `ci.yml`: runs on every push to `master` and on every pull request, the main gate for day-to-day development.
- `pages.yml`: deploys the landing page and both books to GitHub Pages.
- `release.yml`: triggered when a `v*` tag is pushed, runs GoReleaser and mirrors to GitCode.
- `smoke.yml`: runs once daily on a schedule, verifying that real component downloads have not broken.

Each is described below, with the actual files in the repository as the source of truth.

## ci.yml

It is triggered by a push to the `master` branch or by any pull request:

```yaml
on:
  push:
    branches: [master]
  pull_request:
```

The jobs in the workflow are independent of each other and run in parallel. All jobs pin action versions by SHA, Go uses `go-version: stable` across the board, Node uses 22, and pnpm uses latest.

### go-test

Builds and tests the Go code on five runners: `ubuntu-24.04`, `ubuntu-24.04-arm`, `macos-26`, `macos-26-intel`, and `windows-2025`, all with `-race`. On each runner the following run in sequence:

```bash
go build ./...
go build -tags=mirror ./...
go test -race -count=1 -timeout 300s ./...
go test -tags=mirror -race -count=1 -timeout 300s ./internal/selfupdate/...
go vet ./...
go vet -tags=mirror ./...
```

The `mirror` build tag switches the download source to the mirror aimed at mainland China. It only affects the download-related code paths, so the mirror variant only tests `./internal/selfupdate/...`, while the build and vet steps still cover all packages. `-count=1` disables the test cache, ensuring every run actually runs.

### lint

Runs `golangci-lint` on `ubuntu-24.04`, using the official `golangci/golangci-lint-action` with `version: latest`. The lint rules are determined by the configuration file in the repository root; see [Linting and Formatting](linting.md) for details.

### markdown-lint

Runs `npx --yes markdownlint-cli2@0.22.1` with Node to check the Markdown in the repository (including the source files of both books). The version is pinned to `0.22.1`.

### cross-build

Confirms that cross-compilation passes on all target platforms. The matrix takes `goos` as `linux`, `darwin`, `windows` and `goarch` as `amd64`, `arm64`, excluding `windows/arm64`, for five combinations in total. Each combination compiles only the main program and discards the output:

```bash
GOOS=... GOARCH=... go build -o /dev/null ./cmd/cjv/
```

This step only verifies that it compiles; it does not run tests.

### mod-check

Ensures the module declarations are clean. It first runs `go mod tidy`, then uses `git diff --exit-code go.mod go.sum` to check for any changes; a change means tidy was forgotten locally. It then runs `go mod verify` to check the integrity of the dependencies.

### vuln-check

`go install`s the latest `govulncheck`, then runs `govulncheck ./...` to scan for known vulnerabilities.

### web-build

The build check for the landing page. It uses `pnpm/action-setup` to install pnpm, Node 22 with the pnpm cache enabled, and the cache key tied to `web/pnpm-lock.yaml`. Then, in the `web` directory:

```bash
pnpm install --frozen-lockfile
pnpm build
```

`--frozen-lockfile` ensures the lockfile stays consistent with `package.json` and that dependencies are not silently changed in CI. For the landing page itself, see [Landing Page](web.md).

### web-test

Runs the unit and component tests for the landing page, with `timeout-minutes: 15`. After installing dependencies, it first resolves Playwright's version number and uses it as the cache key to cache `~/.cache/ms-playwright`, then runs `playwright install --with-deps chromium`. The tests run in Chromium:

```bash
pnpm exec vitest run --exclude src/App.platform.integration.test.tsx
```

This explicitly excludes `src/App.platform.integration.test.tsx`, because that file is a cross-platform, cross-browser integration test, handled instead by `web-browser-integration` below on a matrix of real runners.

### web-coverage

Almost the same setup steps as `web-test` (Playwright cache, install Chromium), finally running `pnpm coverage`, that is `vitest run --coverage`, to generate coverage. Also `timeout-minutes: 15`.

### web-browser-integration

The job is named `Web integration (...)`, and it is the end-to-end verification of the platform/architecture detection logic in the landing page. The landing page has to decide which installer to recommend, or mark the platform as unsupported, based on the operating system and CPU architecture reported by the browser, and this job is what checks that against real machines.

The matrix has two dimensions: `browser` takes `chromium`, `firefox`, `webkit`; `runner` is a set of objects carrying expected values, covering Windows arm64, macOS intel/arm, Windows amd64, Linux amd64/arm64, and Linux runners that masquerade as iOS and Android. Each runner object carries a set of `expected_*` fields, such as `expected_state` (`ready` or `unsupported`), `expected_label`, `expected_binary_goos`/`expected_binary_goarch`. These expected values are passed to the tests through `VITE_EXPECTED_*` environment variables:

```yaml
env:
  VITEST_BROWSER: ${{ matrix.browser }}
  VITE_EXPECTED_STATE: ${{ matrix.runner.expected_state }}
  VITE_EXPECTED_LABEL: ${{ matrix.runner.expected_label }}
  # ...the remaining VITE_EXPECTED_* follow the same pattern
run: pnpm test:integration
```

`pnpm test:integration` just runs `src/App.platform.integration.test.tsx` on its own. With `fail-fast: false`, one failing combination does not take down the others, and `timeout-minutes: 30`. On Linux the browser is installed with `--with-deps`; on other platforms it is installed without.

 >
 > For why browsers cannot reliably read the CPU architecture of a macOS machine, and how the landing page works around that limitation, see the [installation notes](https://cjv.zxilly.dev/book/user-guide/zh-CN/) in the user guide.

### install-script-integration

The job is named `Installer (...)` and verifies that the two install scripts, `install.sh` and `install.ps1`, work correctly across a range of shells. The matrix is expanded by `label`: the Unix side covers `sh`/`bash`/`zsh`/`fish` on Linux, `sh` on Linux arm64, and `bash`/`zsh`/`fish` on macOS arm and intel; the Windows side covers the old `WindowsPowerShell` and the new `pwsh`. Missing shells are installed temporarily with `apt-get` on Linux and `brew` on macOS.

The test entry points are the two scripts in the repository:

```bash
# kind == unix
sh tests/install-scripts/install-sh.sh
# kind == powershell
./tests/install-scripts/install-ps1.ps1
```

The `shell_name`, `test_mode`, and `default_toolchain` from the matrix are passed in through `CJV_INSTALL_TEST_*` environment variables, so the same script is reused across different shells and modes. `fail-fast: false`, `timeout-minutes: 30`.

### go-integration

Runs Go's integration tests. The runner matrix is the same as `go-test` (five platforms, all with `-race`). They are kept separate from the unit tests because they are fenced off with the `integration` build tag:

```bash
go test -race -v -tags integration -count=1 ./tests/integration/
```

The tests under `tests/integration/` actually launch cjv and operate on the real file system and PATH. For details on how the tests are layered, see [Testing](testing.md).

## pages.yml

Deploys the landing page and both books together to GitHub Pages. There are three triggers: push to `master`, `workflow_call` (invoked by `release.yml`), and `workflow_dispatch` (manual). The concurrency group is `pages` with `cancel-in-progress: false`, so a deployment is not interrupted by a new commit. It requires the `pages: write` and `id-token: write` permissions.

There is a single `deploy` job in the `github-pages` environment, which does the following in order:

1. Build the landing page: `pnpm install --frozen-lockfile` followed by `pnpm build`, with the output in `web/dist`.
1. Download the latest release archives: use `gh release download` to fetch the release artifacts matching the two patterns `cjv_*` and `cjv-mirror_*`.
1. Extract the init binaries: run `scripts/extract-init-binaries.sh` to unpack the `cjv-init` binaries for each platform and variant into `web/dist/dl`, so the landing page can distribute them directly.
1. Install mdBook (`MDBOOK_VERSION: 0.5.3`, downloaded directly as a prebuilt binary).
1. Build both books, each producing a Chinese and an English version, all landing under `web/dist/book/<book>/<lang>/`:

```bash
# docs/user-guide
MDBOOK_OUTPUT__HTML__SITE_URL=/book/user-guide/zh-CN/ mdbook build -d ../../web/dist/book/user-guide/zh-CN
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/user-guide/en/ mdbook build -d ../../web/dist/book/user-guide/en
# docs/dev-guide
MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/zh-CN/ mdbook build -d ../../web/dist/book/dev-guide/zh-CN
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/en/ mdbook build -d ../../web/dist/book/dev-guide/en
```

The books must be built after `pnpm build`, because `pnpm build` clears out `web/dist`. Finally the entire `web/dist` is uploaded as the Pages artifact and deployed. For the books' multilingual mechanism, see [Documentation site](documentation.md).

## release.yml

Triggered when a `v*` tag is pushed, to make an official release. The first step, `Ensure stable tag`, uses a regex to restrict the tag to a stable version like `vX.Y.Z`; a prerelease tag with a suffix exits immediately. The permission is `contents: write`, used to create the GitHub Release.

The `release` job, in order:

1. Enables TCP BBR with `Zxilly/actions-bbr` to improve upload bandwidth.
1. `actions/checkout` with `fetch-depth: 0` fetches the full history, since GoReleaser needs the complete tag history to generate the changelog.
1. Pushes a mirror of the repository to GitCode (`git push gitcode`), providing a mirror source for users in mainland China.
1. Runs GoReleaser (`args: release --clean`); its configuration lives in `.goreleaser.yml` at the repository root, and it handles cross-compilation, packaging, checksum generation, and creating the GitHub Release.
1. Uses `Zxilly/upload-gitcode-release` to upload the mirror-variant artifacts (`dist/cjv-mirror_*` and `dist/checksums.txt`) separately to the GitCode release.

After `release` there is a `pages` job with `needs: release` that reuses the Pages workflow above via `uses: ./.github/workflows/pages.yml`, redeploying the landing page once the release is complete so that it points to the latest release artifacts. For the full release process, see [Release process](releasing.md).

## smoke.yml

A scheduled smoke test that confirms the real component download path has not broken. It is triggered by cron (daily at 08:00 UTC) plus manual `workflow_dispatch`, with the concurrency group grouped by ref and `cancel-in-progress: true`.

There is a single `real-component-downloads` job that runs on `ubuntu-latest` and `ubuntu-24.04-arm`, with `fail-fast: false`. After enabling BBR, it runs the real download tests using the `smoke` build tag:

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

The environment variable `CJV_DOWNLOAD_TIMEOUT: "900"` caps a single download at 900 seconds, and the whole job has a 45-minute timeout. This test pulls real LTS/STS components, so it runs as a scheduled job rather than a PR gate. That keeps upstream flakiness out of day-to-day development and avoids burning a lot of bandwidth on every PR.
