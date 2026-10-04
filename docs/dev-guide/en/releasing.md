# Release Process

Releases of cjv are driven by git tags. When you tag a commit with a stable version and push it, `.github/workflows/release.yml` uses goreleaser to build artifacts for every platform, publish them to GitHub Releases, mirror the repository and artifacts to GitCode, and finally redeploy the landing page and the documentation site. This chapter explains what each step in that pipeline does, and how the `cjv-init` binary travels from the release artifacts to the download button on the landing page.

## Version numbers and the tag convention

A release only recognizes a stable tag of the form `vX.Y.Z`. The first step of `release.yml` validates the tag name and fails outright if it does not match:

```yaml
- name: Ensure stable tag
  run: |
    if ! [[ "${GITHUB_REF_NAME}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "Only stable tags trigger a release"
      exit 1
    fi
```

In other words, `v1.2.3` is released, while `v1.2.3-rc1`, `v1.2`, and `1.2.3` (missing the `v` prefix) are all rejected. The workflow trigger condition is `tags: ["v*"]`, which is broader than this regex, so a pre-release tag still starts the workflow; it just stops at the first step.

The version number is injected into the binary through ldflags. goreleaser adds it to both builds:

```yaml
ldflags:
  - -s -w
  - -X main.version={{.Version}}
  - -X main.commit={{.Commit}}
  - -X main.date={{.Date}}
```

Here `{{.Version}}` is the tag with the `v` prefix stripped (for example, tag `v1.2.3` maps to `main.version=1.2.3`). In `cmd/cjv/main.go`, `version` defaults to `"dev"`, which is the value you get when you `go build` directly from source; only going through goreleaser replaces it with the real version. `main.commit` and `main.date` currently have no corresponding package-level variables in `cmd/cjv`, so those two `-X` flags are no-ops. They are kept as preparation for wiring up version information later and have no effect on the artifacts for now.

## goreleaser configuration

`.goreleaser.yml` (`version: 2`) defines two parallel sets of artifacts: the official variant and the mirror variant.

The `main` of both builds is `./cmd/cjv`, the source is identical, and the only difference is the compile parameters:

```yaml
builds:
  - id: cjv
    binary: cjv
    ldflags:
      - -X main.updateURL=https://github.com/Zxilly/cjv/releases
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64

  - id: cjv-mirror
    binary: cjv-mirror
    flags:
      - -tags=mirror
    ldflags:
      - -X main.updateURL=https://gitcode.com/Zxilly/cjv/releases
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64
```

`cjv-mirror` adds `-tags=mirror`. This build tag switches two compile-time constants: `internal/config/manifest_mirror.go` changes the default SDK manifest address to the GitCode mirror, and `internal/selfupdate/update_mirror.go` makes self-update go through GitCode. Together with `main.updateURL` pointing at GitCode, the mirror variant never touches GitHub across the whole chain, from download and install to fetching the manifest to self-update, which serves environments where GitHub access is poor. The default implementation for the official variant (`!mirror`) lives in `manifest_default.go` and points at GitHub.

The platform matrix is three OSes (`linux`, `darwin`, `windows`) times two architectures (`amd64`, `arm64`), with `windows/arm64` excluded via `ignore`, giving 5 targets per set and 10 binaries across both sets.

### Artifact naming

```yaml
archives:
  - id: cjv
    ids: [cjv]
    name_template: "cjv_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
  - id: cjv-mirror
    ids: [cjv-mirror]
    name_template: "cjv-mirror_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
```

The archive names are `cjv_<os>_<arch>` and `cjv-mirror_<os>_<arch>`. Windows is packaged as `.zip`, and every other platform uses the default `.tar.gz`. So the final file names look like `cjv_linux_amd64.tar.gz`, `cjv_windows_amd64.zip`, and `cjv-mirror_darwin_arm64.tar.gz`. The name of the executable inside the archive is set by the `binary` field: `cjv` for the official variant (`cjv.exe` on Windows) and `cjv-mirror` for the mirror variant (`cjv-mirror.exe`).

Checksums are emitted to a separate file:

```yaml
checksum:
  name_template: "checksums.txt"
```

`checksums.txt` covers every archive. After downloading an archive, `install.sh` fetches this file to verify the sha256; if the file is missing or the machine has no sha256 tool, it degrades to a warning. Only the case where "a checksum is available but does not match" is treated as a failure and aborts the install.

## release workflow

`.github/workflows/release.yml` triggers when a `v*` tag is pushed. Official releases accept only stable `vX.Y.Z` tags, and CI drives the complete flow automatically:

1. The `release` job validates the tag, enables TCP BBR, checks out the full history, and sets up Go.
2. It pushes the release tag to GitCode, transferring the referenced commit without moving GitCode's `master` to an older release commit.
3. GoReleaser publishes the GitHub Release with all 10 official and mirror archives plus `checksums.txt`.
4. The `gitcode` job automatically calls `sync-gitcode-release.yml` through `workflow_call`, passing the tag and GitCode secret explicitly. It checks that the GitHub Release is published and has all six required assets, then ensures that the GitCode tag exists.
5. Each asset has its own upload job with BBR and a 15-minute timeout, with at most two concurrent uploads. The five mirror archives are verified against GitHub's `checksums.txt` before upload. Every published GitCode asset is downloaded again and compared byte for byte with its GitHub original.
6. The `pages` job waits for both `release` and `gitcode` to succeed before reusing `pages.yml` to deploy the site.

GitHub carries all official and mirror artifacts. GitCode carries the five mirror archives and checksums, with its release body linking to the full GitHub changelog.

The independent `mirror.yml` automatically synchronizes the current `master` and all tags when a `master` commit or any tag is pushed. It always checks out `master`, so synchronizing an older tag does not rewind the mirrored branch to that release commit.

Normal releases require no manual synchronization. `workflow_dispatch` on `sync-gitcode-release.yml` is only for recovery or historical releases; it accepts a stable tag and reuses the same validation and upload flow. Failed asset jobs can be rerun independently without rebuilding binaries.

The release pipeline calls synchronization directly instead of relying on a second GitHub Release event, because releases created using `GITHUB_TOKEN` do not trigger those subsequent workflows. See [GitHub's trigger rules](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow).

## How cjv-init is produced

Clicking "Download" on the landing page does not hand you a separately built installer; it is just the `cjv` binary under a different name. At startup `cmd/cjv/main.go` looks at the name it was invoked as: when the filename starts with `cjv-init` (or `cjv-setup`), it runs itself as the installer, automatically inserting the `init` subcommand ahead of the arguments, and after running in a standalone console window it pauses for Enter so it works when double-clicked from the file explorer.

```go
func isInitInvocation(toolName string) bool {
	return strings.HasPrefix(toolName, "cjv-init") || strings.HasPrefix(toolName, "cjv-setup")
}
```

Prefix matching tolerates copies renamed by the browser, such as `cjv-init(1)` or `cjv-init-2`.

So there is nothing named `cjv-init` in the release artifacts; it is taken from the release archives and named on the fly when the landing page is deployed. These two steps in `pages.yml` handle it:

```yaml
- name: Download latest release archives
  run: |
    mkdir -p archives
    gh release download --repo "${GITHUB_REPOSITORY}" --dir archives \
      --pattern 'cjv_*' --pattern 'cjv-mirror_*'
- name: Extract cjv-init binaries by platform and variant
  run: |
    ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

`scripts/extract-init-binaries.sh` walks the official and mirror archives for the 5 platforms, extracts the executable inside each, and renames and places it by platform and variant into:

```text
web/dist/dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
```

The `cjv` (`cjv.exe`) from the official archive is renamed to `cjv-init` (`cjv-init.exe`) and placed in `official/`, and the `cjv-mirror` from the mirror archive is renamed to the same `cjv-init` and placed in `mirror/`. These files are published together with Pages, and the landing page download links point straight at them, in the form `/dl/official/windows_amd64/cjv-init.exe` or `/dl/mirror/darwin_arm64/cjv-init` (the URLs are assembled in `web/src/hooks/use-platform.ts`). The user downloads one and double-clicks it; the binary recognizes that it is named `cjv-init` and enters the install flow.

There is an ordering assumption here: `pages.yml` uses `gh release download ... releases/latest` to fetch the latest Release. In `release.yml` the `pages` job has `needs: [release, gitcode]`, so the Release is always published before the page is deployed, and what gets downloaded is the version just released.

## The install script takes a different path

The landing page download button serves the single-file `cjv-init`, while `install.sh` (and the PowerShell version) use the full release archives. It pulls the `.tar.gz` for the matching platform straight from `releases/latest/download`:

```sh
CJV_GITHUB_ROOT="${CJV_GITHUB_ROOT:-https://github.com/Zxilly/cjv/releases/latest/download}"
CJV_GITCODE_ROOT="${CJV_GITCODE_ROOT:-https://gitcode.com/Zxilly/cjv/releases/latest/download}"
```

By default it pulls the official `cjv_<platform>_<arch>.tar.gz`; with `--mirror` (or the environment variable `CJV_MIRROR=1`) it switches to `cjv-mirror_<platform>_<arch>.tar.gz` and to the GitCode root. After downloading it verifies against `checksums.txt`, then extracts and runs `cjv init`. Note that the script handles only `.tar.gz`, so `install.sh` serves Linux and macOS; Windows users go through `install.ps1` or download `cjv-init.exe` directly. These names and the `releases/latest/download` paths all depend on goreleaser's archive naming, so when changing the `name_template` in `.goreleaser.yml` you need to check the install scripts and `extract-init-binaries.sh` in step.

## Local dry run

If you do not want to actually publish and just want to see what goreleaser would produce, run this in the repository root:

```bash
goreleaser release --snapshot --clean
```

`--snapshot` does not tag or upload; it generates all archives into `dist/`, which is good for checking artifact naming and whether the build passes. To verify the init landing logic on its own, you can manually drop the archives into a directory and run the extraction script:

```bash
ARCHIVES_DIR=./archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

This is exactly the same script and the same set of environment variables that `pages.yml` uses.
