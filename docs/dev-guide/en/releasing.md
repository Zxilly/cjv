# Releases

Pushing a stable `vX.Y.Z` tag triggers `.github/workflows/release.yml`. Other `v*` tags also start the workflow, but fail immediately at the tag format validation step.

## Release order

When a `vX.Y.Z` tag is pushed, the release pipeline runs in four ordered stages:

1. Validate the `vX.Y.Z` tag format and push the release commit and tag to GitCode.
2. Build the default (`cjv`) and mirror (`cjv-mirror`) variants across all supported platforms with GoReleaser, and publish the GitHub Release.
3. Call `sync-gitcode-release.yml` to synchronize the release archives and `checksums.txt` to GitCode Releases.
4. Once both GitHub and GitCode releases finish, call `.github/workflows/pages.yml` to update the website's `/dl/` installers, [landing page](web.md), and [documentation site](documentation.md).

Ordinary branch and tag synchronization is handled by `.github/workflows/mirror.yml`. If step 3 fails due to a transient network issue, you can retry asset synchronization manually via the `workflow_dispatch` entry point in `sync-gitcode-release.yml` (see the workflow file for inputs and credentials).

## Artifacts

`.goreleaser.yml` defines two builds—`cjv` and `cjv-mirror` (compiled with `-tags=mirror`)—each shipping `amd64` and `arm64` binaries for Linux, macOS, and OpenHarmony, plus `amd64` for Windows:

| Item | Name |
| --- | --- |
| Default archive | `cjv_<os>_<arch>.tar.gz`, or `.zip` on Windows |
| Mirror archive | `cjv-mirror_<os>_<arch>.tar.gz`, or `.zip` on Windows |
| Checksums | `checksums.txt` |
| Executable inside archive | `cjv` or `cjv-mirror`, with `.exe` on Windows |

Release builds inject `main.version` and `main.updateURL` via `-ldflags`. The `mirror` build switches the default manifest URL and self-update backend to GitCode, while still honoring absolute artifact URLs written in manifests.

Whenever you change archive naming rules or supported platforms, update `.goreleaser.yml`, the installer scripts (`install.sh`, `install.ps1`), `scripts/extract-init-binaries.sh`, the generated frontend platform catalog (`go generate ./...`), and the CI matrix together.

## OpenHarmony builds and signing

During release builds, `.github/actions/setup-hmos` installs the [Go-HMOS `go1.27.2-hmos.5`](building.md#openharmony) toolchain (`HMOS_GO`) and builds the host code-signing tool (`HMOS_SIGN`). GoReleaser compiles `openharmony_arm64` and `openharmony_amd64` with `HMOS_GO` and invokes `HMOS_SIGN` on each binary before packaging the archives and generating `checksums.txt`. The signed OpenHarmony archives are included in both GitCode synchronization and Pages `/dl/` extraction.

Before running a local GoReleaser snapshot, set `HMOS_GO` and `HMOS_SIGN` and build the host signer as described in [Build from source](building.md#openharmony).

## Website installers

The release workflow passes its current tag to Pages; ordinary website deployments resolve the latest stable release tag once before downloading (and manual Pages runs may supply an explicit `release_tag` input). `scripts/download-release-archives.sh` downloads all 14 archives and `checksums.txt` from that single release tag, and then runs:

```bash
ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

`extract-init-binaries.sh` verifies the SHA-256 checksum of all 14 archives against `checksums.txt` first—any missing, duplicate, malformed, or mismatched checksum aborts the script before extraction. Next, it extracts and renames each executable to `cjv-init` (`cjv-init.exe` on Windows) under `web/dist/dl/{official,mirror}/{os}_{arch}/`. When a user runs that downloaded executable, `cmd/cjv/main.go` recognizes the `cjv-init` prefix (including browser-added duplicate suffixes such as `cjv-init (1).exe`) and launches the `cjv init` wizard directly.

Meanwhile, `install.sh` and `install.ps1` download the release archives directly and verify them against `checksums.txt`. As a result, standalone website installers and script installers always come from the exact same release tag while downloading their respective artifact formats.

## Rehearsal and verification

Before tagging a release, run the [test and check suite](testing.md) and rehearse the GoReleaser build locally:

```bash
goreleaser release --snapshot --clean
```

Snapshot builds write all archives and `checksums.txt` to `dist/` without publishing a release. Inspect the generated archive contents and checksums before pushing a tag. After publication finishes, verify that both GitHub and GitCode Releases contain the tag and all assets, check the live Pages download routes (`node deployment/verify-site.mjs https://cjv.zxilly.dev/ https://cjv.zxilly.com/`), and confirm that `install.sh` and `install.ps1` download the intended version.
