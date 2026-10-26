# Releases

Pushing a stable semantic version tag in the format `vX.Y.Z` triggers `.github/workflows/release.yml`. Any other `v*` tag starts the workflow but fails the tag format validation step immediately. This chapter describes the release pipeline, artifact naming, OpenHarmony code signing, website installer extraction, and local release rehearsals.

## Automated release sequence

When you push a `vX.Y.Z` tag, `.github/workflows/release.yml` executes four stages in order:

1. **Validate and mirror the tag**: checks the `vX.Y.Z` tag format and pushes the release commit and tag to GitCode.
2. **Build and publish on GitHub**: runs GoReleaser (`.goreleaser.yml`) to compile the default (`cjv`) and mirror (`cjv-mirror`) binaries for all platforms, signs the OpenHarmony binaries, and publishes the archives and `checksums.txt` to GitHub Releases.
3. **Synchronize GitCode Release assets**: triggers `sync-gitcode-release.yml` to upload the release archives and `checksums.txt` to GitCode Releases.
4. **Deploy the website and installers**: after both GitHub and GitCode releases succeed, triggers `.github/workflows/pages.yml` with the release tag to update `/dl/` binaries, the [landing page](web.md), and the [documentation site](documentation.md).

`.github/workflows/mirror.yml` mirrors ordinary branch and tag pushes. If step 3 fails because of a transient network error, re-run `sync-gitcode-release.yml` manually via `workflow_dispatch`.

## Release artifacts

`.goreleaser.yml` builds two variants for every supported platform (`linux`, `darwin`, and `openharmony` on `amd64` and `arm64`, plus `windows` on `amd64`):

- **Default variant (`cjv`)**: uses GitHub for the default manifest and `cjv self update`.
- **Mirror variant (`cjv-mirror`)**: compiled with `-tags=mirror` to use GitCode for the default manifest and `cjv self update` (while still honoring explicit `https://` URLs inside manifests).

| Artifact type | File name pattern |
| --- | --- |
| Default platform archive | `cjv_<os>_<arch>.tar.gz` (`.zip` on Windows) |
| Mirror platform archive | `cjv-mirror_<os>_<arch>.tar.gz` (`.zip` on Windows) |
| SHA-256 checksum manifest | `checksums.txt` (covers all 14 archives) |
| Binary inside archive | `cjv` or `cjv-mirror` (`.exe` on Windows) |

GoReleaser injects `main.version` and `main.updateURL` via `-ldflags`. If you ever change archive names or add a platform, update `.goreleaser.yml`, `internal/target/catalog.go` (`go generate ./...`), `web/public/install.sh`, `web/public/install.ps1`, `scripts/extract-init-binaries.sh`, and the CI workflows together.

## OpenHarmony builds and signing

During the GoReleaser run, `.github/actions/setup-hmos` installs the [Go-HMOS `go1.27.2-hmos.5`](building.md#openharmony) toolchain (`HMOS_GO`) and builds the host code-signing binary (`HMOS_SIGN`). GoReleaser compiles `openharmony_amd64` and `openharmony_arm64` with `HMOS_GO` and runs `HMOS_SIGN` on each binary before packaging the `.tar.gz` archives and generating `checksums.txt`.

The release workflows upload the signed OpenHarmony archives to both GitHub and GitCode Releases and extract them into the website's `/dl/` directory.

## Website installers

When `.github/workflows/pages.yml` runs, it resolves the target release tag once (using the tag passed by `release.yml`, the manual `release_tag` input, or the latest stable release). Next, `scripts/download-release-archives.sh` downloads all 14 archives and `checksums.txt` from that release tag and calls `scripts/extract-init-binaries.sh`:

```bash
ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

`extract-init-binaries.sh` performs these steps:

1. Checks the SHA-256 hash of all 14 archives against `checksums.txt`. Any missing, duplicate, malformed, or mismatched entry aborts the build before extraction.
2. Extracts each binary and renames it to `cjv-init` (`cjv-init.exe` on Windows) under `web/dist/dl/{official,mirror}/{os}_{arch}/`.

When a user downloads and runs `cjv-init` (including browser-renamed copies such as `cjv-init (1).exe`), `cmd/cjv/main.go` detects the `cjv-init` prefix and launches the interactive `cjv init` wizard automatically. Meanwhile, `install.sh` and `install.ps1` download the `.tar.gz` or `.zip` release archives directly and check them against `checksums.txt`. Both installation methods therefore install the exact same release.

## Local rehearsal and post-release verification

Before pushing a release tag, run the [test and check suite](testing.md) and test the GoReleaser configuration locally without publishing:

```bash
goreleaser release --snapshot --clean
```

> **Note:** To include the OpenHarmony targets in a local GoReleaser snapshot, set `HMOS_GO` and `HMOS_SIGN` first as shown in [Build from source](building.md#openharmony).

GoReleaser writes the snapshot archives and `checksums.txt` into `dist/`. After you push a real `vX.Y.Z` tag and the workflows finish, check the release end to end:

- [ ] Both GitHub Releases and GitCode Releases list the new `vX.Y.Z` tag with all 14 archives and `checksums.txt`.
- [ ] `node deployment/verify-site.mjs https://cjv.zxilly.dev/ https://cjv.zxilly.com/` passes for the landing page, bilingual books, installer scripts, and `/dl/` binaries.
- [ ] Running `install.sh` and `install.ps1` installs the new `vX.Y.Z` version cleanly.
