# Building from Source

cjv is a standard Go module. It has no code generation step and does not depend on any external build system. Once you have the Go toolchain installed and the repository cloned, you can build it directly.

## Prerequisites

The Go version declared in go.mod is the lower bound for building cjv:

```toml
module github.com/Zxilly/cjv

go 1.26.0
```

Use `go version` to confirm your local version is no lower than the one declared here. CI uses `go-version: stable` in `actions/setup-go`, which tracks the latest stable release, so using the current stable version locally is sufficient.

The landing page (`web/`) is a separate frontend project. It is not compiled into the cjv binary and is not needed when building the Go part. The localized copy (`internal/i18n/locales/*.toml`) is embedded via `go:embed` and is already committed to the repository, so no extra generation is needed before building. For landing page development, see [Landing Page](web.md).

## Building All Packages

Compile every package from the repository root to confirm the whole codebase compiles:

```bash
go build ./...
```

This command does not produce an executable; it only checks that each package builds successfully. It is equivalent to the `Build all packages` step in the `go-test` job in CI.

## Building the Binary

The binary entry point is `cmd/cjv`, the only `main` package in the repository. Build an executable for your host platform:

```bash
go build -o cjv ./cmd/cjv
```

On Windows the output name ends in `.exe`:

```bash
go build -o cjv.exe ./cmd/cjv
```

Without `-o`, `go build ./cmd/cjv` produces a file named `cjv` (or `cjv.exe`) in the current directory.

Install directly into `GOBIN` (defaults to `$(go env GOPATH)/bin`):

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

`@latest` fetches the latest tagged release; to install a specific version, replace it with the corresponding tag, and to install the code in your current working tree, run `go install ./cmd/cjv` from within the repository. A binary installed this way is "bare", carrying no version information, for the reason explained in the next section.

## Version information

`cmd/cjv/main.go` has two package-level variables that are injected at link time:

```go
var (
	version   = "dev"
	updateURL string
)
```

A plain `go build` / `go install` does not set them, so `version` stays `dev` and `cjv --version` shows `dev`. Official releases are injected by goreleaser at release time through `-ldflags -X`, with the rules written in `.goreleaser.yml`:

```yaml
ldflags:
  - -s -w
  - -X main.version={{.Version}}
  - -X main.commit={{.Commit}}
  - -X main.date={{.Date}}
  - -X main.updateURL=https://github.com/Zxilly/cjv/releases
```

`updateURL` determines which release page `cjv update` looks at for new versions when self-updating, and defaults to GitHub. Injecting a version number manually in a local build uses the same syntax:

```bash
go build -ldflags "-X main.version=$(git describe --tags)" -o cjv ./cmd/cjv
```

For the full description of the release process, see [Release process](releasing.md).

## The mirror variant

cjv has a mirror build variant for environments where GitHub access is unreliable: it swaps the default data source for a GitCode mirror. The variant is toggled by the `mirror` build tag, and the source distinguishes the paired files with build constraints:

```go
//go:build !mirror
```

```go
//go:build mirror
```

Two places are currently affected by the tag. The first is the SDK manifest address `DefaultManifestURL` in `internal/config/manifest_default.go` (GitHub raw) and `internal/config/manifest_mirror.go` (GitCode raw). The second is the self-update logic in `internal/selfupdate/update_default.go` (using go-selfupdate's GitHub source) and `internal/selfupdate/update_mirror.go` (a custom GitCode source that detects the latest tag by following the `releases/latest` redirect).

Add `-tags=mirror` to build the mirror variant:

```bash
go build -tags=mirror ./...
```

```bash
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

CI has a dedicated `Build mirror variant` step in the `go-test` job that runs `go build -tags=mirror ./...`, and it runs the tests and `go vet` separately for the mirror variant to make sure both sets of build constraints compile and do not regress. goreleaser also ships it as a separate artifact `cjv-mirror`, with `updateURL` pointing at GitCode.

When writing code with build tags, remember: the default build uses the `!mirror` side. IDEs and `go build ./...` do not carry the `mirror` tag by default, so files with the `mirror` tag normally take no part in compilation and are easy to miss during refactoring. After changing one side, run `go build -tags=mirror ./...` to verify that the other side is not broken either.

## Cross-compilation

cjv is pure Go with no cgo dependency, so cross-compilation only requires setting `GOOS` and `GOARCH`:

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

CI's `cross-build` job uses a matrix to cover all release targets, dumping the artifacts to `/dev/null` just to verify they compile:

```bash
GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build -o /dev/null ./cmd/cjv/
```

The matrix takes `goos` as `linux`, `darwin`, `windows` and `goarch` as `amd64`, `arm64`, while excluding `windows/arm64`. These five combinations are exactly the set of platforms cjv actually releases for, matching the `goos` / `goarch` / `ignore` configuration in `.goreleaser.yml`. To confirm locally that a change does not break cross-compilation, just run `go build` against these few combinations.

## Verification

Building is only the first step. Before committing you also need to run the tests, `go vet`, and the linter; for the corresponding commands see [Testing](testing.md) and [Linting and formatting](linting.md). For what CI does and how the individual jobs are organized, see [Continuous integration](ci.md).
