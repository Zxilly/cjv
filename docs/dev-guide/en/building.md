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

cjv does not require cgo for builds. Use Go's platform variables to cross-compile. In Bash:

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

The release platform catalog is in `internal/target/catalog.go`: amd64/arm64 for Linux and macOS, and amd64 for Windows. After changing the catalog, run:

```bash
go generate ./...
```

Review the generated frontend platforms, then check release configuration and the CI matrix for consistency. See [testing and checks](testing.md) to validate behavior after building.
