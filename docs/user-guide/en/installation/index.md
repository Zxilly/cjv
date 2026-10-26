# Install cjv

cjv is a standalone executable. You do not need an existing Cangjie SDK before you install cjv. After you finish this chapter, continue to [Quick start](../basic-usage.md) to manage your toolchains.

## Run the installer script

### Standard installation

On Linux, macOS, or HarmonyOS HiShell, run this command:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

On Windows PowerShell, run this command:

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

The script detects your platform, downloads cjv, checks the checksum, and runs `cjv init`. Interactive terminals show the setup wizard, and non-interactive terminals use default settings.

### Non-interactive installation

In CI or automated scripts, skip confirmation prompts or install cjv without an SDK:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Yes -DefaultToolchain none
```

The Unix script reads `--mirror` and `--version`, and passes all other flags to `cjv init`. The PowerShell script accepts `-Mirror`, `-Version`, `-Yes`, `-DefaultToolchain`, and `-NoModifyPath`. Read [Command reference](../command-reference.md#init) for all `cjv init` flags.

### Pin the cjv version

In automated setups, you can select an exact cjv release tag, including prereleases:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --version v0.4.1 -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Version v0.4.1 -Yes -DefaultToolchain none
```

Replace `v0.4.1` with the tag that you need. Both scripts also read the `CJV_VERSION` environment variable, but the command-line flag takes precedence. If you do not set a version, the script downloads the latest stable release.

This version flag selects the cjv binary itself. The `--default-toolchain` or `-DefaultToolchain` flag selects the Cangjie SDK independently.

Explicit `CJV_UPDATE_ROOT`, `CJV_GITHUB_ROOT`, and `CJV_GITCODE_ROOT` download URLs take precedence over the version option. If you use an internal server, point the URL directly to the artifact directory of the target version. Read [Environment variables](../environment-variables.md) for details on these variables.

### SDK selection during initialization

By default, `cjv init` queries the channels for your platform. It selects the first available channel in this sequence: `lts`, `sts`, and `nightly`. Pass `--default-toolchain <name>` to select a specific toolchain, or pass `--default-toolchain none` to install cjv alone.

If the SDK download fails, `cjv init` returns a nonzero exit code and keeps the installed cjv binary. Run the retry command from the error message, or run `cjv toolchain list-remote` to see available releases.

### Use the GitCode mirror

If GitHub access is slow or blocked, install the mirror build from GitCode:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

The mirror build uses GitCode for the default version manifest and for `cjv self update`. If your organization hosts its own mirror, read [Enterprise and offline deployment](../enterprise/index.md).

## Download an archive manually

If you prefer not to run a script, install cjv from a release archive:

1. Download the archive for your platform from [GitHub Releases](https://github.com/Zxilly/cjv/releases).
2. Extract `cjv` or `cjv.exe` into a directory on your `PATH`.
3. Run `cjv init` to create the proxy binaries and configure your environment.

| Platform | Archive name |
| --- | --- |
| Linux x86_64 | `cjv_linux_amd64.tar.gz` |
| Linux ARM64 | `cjv_linux_arm64.tar.gz` |
| macOS Apple Silicon | `cjv_darwin_arm64.tar.gz` |
| macOS Intel | `cjv_darwin_amd64.tar.gz` |
| Windows x86_64 | `cjv_windows_amd64.zip` |
| HarmonyOS x86_64 | `cjv_openharmony_amd64.tar.gz` |
| HarmonyOS ARM64 | `cjv_openharmony_arm64.tar.gz` |

[GitCode Releases](https://gitcode.com/Zxilly/cjv/releases) publishes matching archives with the `cjv-mirror_` prefix. The [landing page](https://cjv.zxilly.com) also offers standalone `cjv-init` binaries that start the wizard directly.

## Build and install from source

If you have the Go version listed in `go.mod`, you can install cjv from source:

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
cjv init
```

Go writes the binary to `GOBIN`, or to `$(go env GOPATH)/bin` when `GOBIN` is not set. Add that directory to your `PATH`. Read [Build from source](https://cjv.zxilly.com/book/dev-guide/en/building.html) in the developer guide for custom and cross-platform builds.

## Configure PATH and check the setup

cjv puts SDK proxy binaries in `<CJV_HOME>/bin`, which defaults to `~/.cjv/bin`. The `cjv init` command adds this directory to your `PATH`. If you skip `cjv init` and install an SDK directly, the first toolchain installation also tries to configure `PATH`.

On Windows, cjv updates the user `PATH` in the registry. On Unix systems, cjv updates your shell profile. Open a new terminal and check the installation:

```bash
cjv --version
```

If you want to manage `PATH` yourself, pass `--no-modify-path` to `cjv init` or `-NoModifyPath` to `install.ps1`. You can also set `CJV_NO_PATH_SETUP=1` in your environment. Next, read [Quick start](../basic-usage.md) to install and select a Cangjie SDK.
