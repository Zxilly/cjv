# Releases

Pushing a stable `vX.Y.Z` tag triggers `.github/workflows/release.yml`. Other `v*` tags start the workflow but fail its format check.

## Release order

1. Validate the tag and push the release commit and tag to GitCode.
2. Build default and mirror variants with GoReleaser and upload the GitHub Release.
3. Call `sync-gitcode-release.yml` to synchronize GitCode release assets.
4. After both releases finish, call Pages to update installers, the landing page, and documentation.

`mirror.yml` handles ordinary branch and tag synchronization. Failed asset synchronization can be retried through the manual entry point in `sync-gitcode-release.yml`; see the workflow for inputs and credentials.

## Artifacts

`.goreleaser.yml` defines `cjv` and `cjv-mirror`, the latter built with `-tags=mirror`. Each ships amd64 and arm64 on Linux/macOS, and amd64 on Windows.

| Item | Name |
| --- | --- |
| Default archive | `cjv_<os>_<arch>.tar.gz`, or `.zip` on Windows |
| Mirror archive | `cjv-mirror_<os>_<arch>.tar.gz`, or `.zip` on Windows |
| Checksums | `checksums.txt` |
| Executable | `cjv` or `cjv-mirror`, with `.exe` on Windows |

Releases inject the version and update URL with ldflags. Mirror switches the default manifest and self-update backend; absolute artifact URLs in manifests are still used as written.

When changing archive names or platforms, check the installers, `scripts/extract-init-binaries.sh`, generated frontend platform data, and CI matrix together.

## Website installers

Pages downloads the latest release archives and runs:

```bash
ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

The script extracts and renames executables to `cjv-init[.exe]` under `dl/{official,mirror}/{os}_{arch}/`. The program recognizes that name and enters `init`. Prefix matching also accepts suffixes added by browsers to duplicate downloads.

`install.sh` and `install.ps1` download release archives directly and read `checksums.txt`. Website executables and script installers therefore depend on the same release but download different files.

## Rehearsal and verification

```bash
goreleaser release --snapshot --clean
```

Snapshot builds write artifacts to `dist/` without publishing a release. Run relevant [tests and checks](testing.md) and inspect archive contents and checksums before release. After publication, verify GitHub and GitCode tags and assets, Pages download paths, and that installers download the intended version.
