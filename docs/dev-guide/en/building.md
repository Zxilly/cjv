# Build from source

`go.mod` declares Go 1.26.0 as the minimum version and `toolchain go1.27.3` as the preferred toolchain. With the default `GOTOOLCHAIN=auto`, Go selects or downloads that toolchain when needed. The Go CLI builds independently of the `web/` frontend. Generated platform catalogs and embedded locale resources are committed to the repository.

```bash
go build ./...
go build -o cjv ./cmd/cjv
```

On Windows, run `go build -o cjv.exe ./cmd/cjv`. The first command verifies that every package in the workspace compiles cleanly; the second produces the standalone `cjv` executable. If you want to install your current working tree directly into `GOBIN`, run `go install ./cmd/cjv`.

When testing changes to `cjv` interactively, you can isolate your development build from any existing `cjv` installation on your machine by pointing `CJV_HOME` and `CJV_CONFIG_FILE` at a temporary directory:

```bash
mkdir -p /tmp/cjv-dev
CJV_HOME=/tmp/cjv-dev CJV_CONFIG_FILE=/tmp/cjv-dev/settings.toml ./cjv init --no-modify-path -y
```

## Version information and mirror builds

Ordinary local builds report their version as `dev` and automatically skip self-update version comparisons. For official releases, GoReleaser injects `main.version` and `main.updateURL` via linker flags (`-ldflags`). You can set an explicit version string locally or build the GitCode mirror variant with `-tags=mirror`:

```bash
go build -ldflags "-X main.version=0.0.0-local" -o cjv ./cmd/cjv
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

The `mirror` build tag selects the GitCode default manifest URL and self-update backend in `internal/config/manifest_*.go` and `internal/selfupdate/update_*.go`. Whenever you modify shared interfaces in `config` or `selfupdate`, make sure to validate both the default build and the `-tags=mirror` build.

## Cross builds and generated files

You can cross-compile `cjv` for other operating systems and architectures using standard Go environment variables. For example, in Bash:

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

The authoritative catalog of release platforms lives in `internal/target/catalog.go`:

- **Linux**: `amd64` and `arm64`
- **macOS (`darwin`)**: `amd64` and `arm64`
- **Windows**: `amd64`
- **OpenHarmony (`openharmony`)**: `amd64` and `arm64`

Whenever you update `internal/target/catalog.go`, regenerate the derived files across the repository:

```bash
go generate ./...
```

After running `go generate ./...`, review the generated frontend platform list in `web/src/generated/platforms.ts`, and check `.goreleaser.yml` and the CI workflow matrices to keep all platform definitions in sync. See [Testing and checks](testing.md) for how to validate your build.

## OpenHarmony

Building `cjv` for OpenHarmony (`GOOS=openharmony`) uses the Linux/amd64 toolchain from [Go-HMOS `go1.27.2-hmos.5`](https://github.com/ZxillyFork/go-hmos-build/releases/tag/go1.27.2-hmos.5) with `CGO_ENABLED=0`. Keep `GOTOOLCHAIN=local` for this build so the preferred-toolchain directive does not switch the custom Go-HMOS toolchain to standard Go.

From the repository root on a Linux x86_64 host, set the tool paths, build the host code-signing helper with your host Go toolchain, and then compile and sign the OpenHarmony executable:

```bash
export GOENV=off GOTOOLCHAIN=local
export HMOS_GO=/absolute/path/to/go-hmos/bin/go
export HMOS_SIGN=/tmp/cjv-hmos-sign
go build -tags=hmos_sign -o "$HMOS_SIGN" ./scripts/harmonyos
GOOS=openharmony GOARCH=arm64 CGO_ENABLED=0 "$HMOS_GO" build -o cjv ./cmd/cjv
"$HMOS_SIGN" cjv cjv
```

Use `GOARCH=amd64` when targeting x86_64 OpenHarmony, and add `-tags=mirror` for the mirror variant.

The host signer (`scripts/harmonyos`) reuses `internal/ohos/selfsign`: it validates the ELF structure and Go build metadata, embeds the `.codesign` section, and verifies the resulting signature. You can also pass `--verify FILE...` to verify existing binaries.

During official [releases](releasing.md), `.goreleaser.yml` builds both default and mirror variants and signs OpenHarmony binaries before archiving and generating `checksums.txt`. When `cjv` itself runs on an OpenHarmony host and downloads a Cangjie SDK, downloaded native executables are self-signed automatically through the platform hook in `internal/dist`.
