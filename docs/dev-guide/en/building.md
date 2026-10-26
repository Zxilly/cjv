# Build from source

You can compile the `cjv` CLI with the Go version in `go.mod`. The Go CLI compiles without building the `web/` frontend first, because the repository already commits generated platform tables and locale files. This chapter covers standard builds, version injection, mirror builds, cross-compilation, and OpenHarmony signing.

## Build the standard CLI

Run these commands from the repository root directory:

```bash
go build ./...
go build -o cjv ./cmd/cjv
```

- On Windows, use `go build -o cjv.exe ./cmd/cjv`.
- `go build ./...` checks that every package in the workspace compiles.
- `go build -o cjv ./cmd/cjv` produces the standalone executable.
- Run `go install ./cmd/cjv` if you want to install the binary from your working tree into `GOBIN`.

## Version strings and mirror builds

By default, a local build reports `dev` as its version and skips self-update version comparisons. During official releases, GoReleaser injects `main.version` and `main.updateURL` with `-ldflags`. You can set a version string or build the GitCode mirror variant locally:

```bash
go build -ldflags "-X main.version=0.0.0-local" -o cjv ./cmd/cjv
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

The `mirror` build tag selects the GitCode default manifest URL and self-update backend in `internal/config/manifest_*.go` and `internal/selfupdate/update_*.go`. Whenever you change shared interfaces in `config` or `selfupdate`, compile and test both the default build and the `-tags=mirror` build.

## Cross-compilation and generated platform catalogs

Use standard Go environment variables (`GOOS` and `GOARCH`) to cross-compile `cjv`. For example, in Bash:

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

The supported release platforms live in `internal/target/catalog.go`:

- **Linux**: `amd64` and `arm64`
- **macOS (`darwin`)**: `amd64` and `arm64`
- **Windows**: `amd64`
- **OpenHarmony (`openharmony`)**: `amd64` and `arm64`

If you change `internal/target/catalog.go`, regenerate the derived files:

```bash
go generate ./...
```

After running `go generate ./...`, check the updated frontend catalog in `web/src/generated/platforms.ts`, `.goreleaser.yml`, and the CI workflows to keep all platform lists synchronized. Read [Tests and checks](testing.md) for verification commands.

## OpenHarmony

To build `cjv` for OpenHarmony (`GOOS=openharmony`), use the Linux `amd64` toolchain from [Go-HMOS go1.27.2-hmos.5](https://github.com/ZxillyFork/go-hmos-build/releases/tag/go1.27.2-hmos.5) with `CGO_ENABLED=0`.

From the repository root on a Linux x86_64 host, build the host code-signing tool and then compile and sign the OpenHarmony binary:

```bash
export GOENV=off GOTOOLCHAIN=local
export HMOS_GO=/absolute/path/to/go-hmos/bin/go
export HMOS_SIGN=/tmp/cjv-hmos-sign
go build -tags=hmos_sign -o "$HMOS_SIGN" ./scripts/harmonyos
GOOS=openharmony GOARCH=arm64 CGO_ENABLED=0 "$HMOS_GO" build -o cjv ./cmd/cjv
"$HMOS_SIGN" cjv cjv
```

- Set `GOARCH=amd64` to build the x86_64 OpenHarmony binary, and add `-tags=mirror` to build the mirror variant.
- The host signer (`scripts/harmonyos`) reuses `internal/ohos/selfsign` to check the ELF layout and Go build metadata, embed the `.codesign` section, and check the signature. Pass `--verify FILE...` to check existing binaries.
- During official [releases](releasing.md), `.goreleaser.yml` signs OpenHarmony binaries before creating archives and checksums. When `cjv` runs on an OpenHarmony host and installs a Cangjie SDK, the platform hook in `internal/dist` self-signs downloaded SDK executables automatically.
