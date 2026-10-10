# Build from source

Use the Go version required by `go.mod`. The CLI builds independently of the frontend. Generated platform files and embedded locale resources are committed and ready to compile.

```bash
go build ./...
go build -o cjv ./cmd/cjv
```

On Windows, use `go build -o cjv.exe ./cmd/cjv`. The first command checks all packages; the second writes an executable. Run `go install ./cmd/cjv` to install the current checkout into `GOBIN`.

## Version information and mirror builds

Ordinary builds report `dev`. GoReleaser injects `main.version` and `main.updateURL` for releases. You can set a local version explicitly:

```bash
go build -ldflags "-X main.version=0.0.0-local" -o cjv ./cmd/cjv
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

The `mirror` build tag selects the GitCode default manifest and self-update implementation in `internal/config/manifest_*.go` and `internal/selfupdate/update_*.go`. Validate both variants when changing shared interfaces.

## Cross builds and generated files

Use Go's platform variables to cross-compile. In Bash:

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

The release platform catalog is in `internal/target/catalog.go`: amd64/arm64 for Linux and macOS, amd64 for Windows, and amd64/arm64 for OpenHarmony. After changing the catalog, run:

```bash
go generate ./...
```

Review the generated frontend platforms, then check release configuration and the CI matrix for consistency. See [testing and checks](testing.md) to validate behavior after building.

## OpenHarmony

Use the Linux/amd64 toolchain from [Go-HMOS go1.27.2-hmos.5](https://github.com/ZxillyFork/go-hmos-build/releases/tag/go1.27.2-hmos.5) with `CGO_ENABLED=0`.

From the repository root, set the tool paths and build the signer with host Go:

```bash
export GOENV=off GOTOOLCHAIN=local
export HMOS_GO=/absolute/path/to/go-hmos/bin/go
export HMOS_SIGN=/tmp/cjv-hmos-sign
go build -tags=hmos_sign -o "$HMOS_SIGN" ./scripts/harmonyos
GOOS=openharmony GOARCH=arm64 CGO_ENABLED=0 "$HMOS_GO" build -o cjv ./cmd/cjv
"$HMOS_SIGN" cjv cjv
```

Use `GOARCH=amd64` for x64 and add `-tags=mirror` for the mirror variant.

The host signer reuses `internal/ohos/selfsign`, checks the ELF and Go build information, then adds and verifies `.codesign`. Use `--verify FILE...` to check existing artifacts.

Releases use `.goreleaser.yml` to build default and mirror variants. OpenHarmony binaries are signed before archiving and checksum generation. Downloaded SDK payloads are signed through the platform hook in `internal/dist`.
