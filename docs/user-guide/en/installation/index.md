# Install cjv

cjv is a standalone executable and does not require an existing Cangjie SDK. After installation, cjv and proxies for SDK tools such as `cjc` and `cjpm` reside in `<CJV_HOME>/bin`, which defaults to `~/.cjv/bin` on Unix and `%USERPROFILE%\.cjv\bin` on Windows.

## Installing via the installer scripts

Installer scripts are available for Linux, macOS, Windows, and HarmonyOS HiShell.

On Linux, macOS, or HarmonyOS HiShell:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

On Windows PowerShell:

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

The script detects your operating system and CPU architecture, downloads and verifies the `cjv` binary against its SHA-256 checksum, and then runs `cjv init`. In an interactive terminal, `cjv init` launches a setup wizard; when standard input is not a terminal, it proceeds with sensible defaults.

### Passing arguments for automated or non-interactive installs

Both installer scripts accept options that allow you to customize the installation without interactive prompts. For example, in a CI pipeline where you want to install `cjv` first and let the project's `cangjie-sdk.toml` select the SDK later, pass `-y` and `--default-toolchain none`:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Yes -DefaultToolchain none
```

The Unix `install.sh` script handles `--mirror` and `--version` itself and forwards all remaining arguments directly to `cjv init`. The PowerShell `install.ps1` script exposes matching parameters: `-Mirror`, `-Version`, `-Yes`, `-DefaultToolchain`, and `-NoModifyPath`. See the [init section of the Command reference](../command-reference.md#init) for the full list of `cjv init` flags.

### Pinning the `cjv` version

If you want repeatable automated builds, you can pin the exact `cjv` release tag (including prereleases) that the installer downloads:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --version v0.4.1 -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Version v0.4.1 -Yes -DefaultToolchain none
```

Replace `v0.4.1` with the release tag you need. Both scripts also honor the `CJV_VERSION` [environment variable](../environment-variables.md), though an explicit `--version` or `-Version` flag on the command line takes precedence. When neither is specified, the installer fetches the latest stable release.

`--version` / `CJV_VERSION` selects the version of the `cjv` manager itself, whereas `--default-toolchain` / `-DefaultToolchain` independently selects which Cangjie SDK toolchain to install. Explicit download root overrides (`CJV_UPDATE_ROOT`, `CJV_GITHUB_ROOT`, and `CJV_GITCODE_ROOT`) take precedence over version selection—if you point `cjv` at an internal artifact server, point the URL directly at the desired version's release directory.

### How `cjv init` selects an initial SDK

By default, `cjv init` queries the upstream manifest for channels that publish builds for your current host platform, and picks the first available channel in the order `lts`, `sts`, and `nightly`. You can override this choice with `--default-toolchain <name>`, or skip installing an SDK altogether with `--default-toolchain none`.

If downloading or extracting the initial SDK fails—for example, due to a transient network error—`cjv init` exits with a nonzero status code while keeping the `cjv` binary itself installed. The error output includes a ready-to-run retry command (using the absolute path to `cjv` so it works before you reload your shell) or you can inspect available releases with `cjv toolchain list-remote`.

### Using the GitCode mirror

If GitHub access is slow or unreliable from your network, pass `--mirror` (or `-Mirror` in PowerShell) to install the mirror build of `cjv`:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

The mirror build uses GitCode by default for both the Cangjie version manifests and `cjv self update`. If your organization hosts an internal mirror, see [Enterprise and offline deployment](../enterprise/index.md).

## Manual installation from release archives

If you prefer not to pipe a script into your shell, you can download a release archive directly from [GitHub Releases](https://github.com/Zxilly/cjv/releases), extract `cjv` (or `cjv.exe` on Windows) into a directory on your `PATH`, and run `cjv init`:

| Platform | Release archive |
| --- | --- |
| Linux x86_64 | `cjv_linux_amd64.tar.gz` |
| Linux ARM64 | `cjv_linux_arm64.tar.gz` |
| macOS Apple Silicon | `cjv_darwin_arm64.tar.gz` |
| macOS Intel | `cjv_darwin_amd64.tar.gz` |
| Windows x86_64 | `cjv_windows_amd64.zip` |
| HarmonyOS x86_64 | `cjv_openharmony_amd64.tar.gz` |
| HarmonyOS ARM64 | `cjv_openharmony_arm64.tar.gz` |

[GitCode Releases](https://gitcode.com/Zxilly/cjv/releases) publishes equivalent mirror archives prefixed with `cjv-mirror_`. In addition, the download page at <https://cjv.zxilly.com> provides standalone `cjv-init` (and `cjv-init.exe`) binaries that launch the setup wizard directly when executed.

## Installing from source with Go

If you have a Go toolchain installed that satisfies the repository's `go.mod` requirement, you can build and install `cjv` directly from source:

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
cjv init
```

`go install` places the compiled binary in `GOBIN` (or `$(go env GOPATH)/bin` if `GOBIN` is unset). Make sure that directory is in your `PATH`, or let `cjv init` copy the managed binary into `<CJV_HOME>/bin`. For custom build tags and cross-compiling `cjv`, see [Build from source](https://cjv.zxilly.com/book/dev-guide/en/building.html) in the Developer Guide.

## `PATH` configuration and verification

All SDK proxy binaries (`cjc`, `cjpm`, etc.) live in `<CJV_HOME>/bin`, which defaults to `~/.cjv/bin`. Running `cjv init` adds this directory to your `PATH` automatically—updating the user `PATH` in the registry on Windows, or appending to your shell profile on Unix. Even if you skip `cjv init` and run `cjv install` directly, the first toolchain installation will also ensure `<CJV_HOME>/bin` is added to your `PATH`.

Once installation completes, open a _new_ terminal window so your shell reloads `PATH`, and verify that `cjv` is available:

```bash
cjv --version
```

If you prefer to manage your shell profiles manually, pass `--no-modify-path` to `cjv init` (or `-NoModifyPath` to `install.ps1`), or set the environment variable `CJV_NO_PATH_SETUP=1`.

Next, head over to the [Quick start](../basic-usage.md) chapter to select a toolchain and compile your first project.
