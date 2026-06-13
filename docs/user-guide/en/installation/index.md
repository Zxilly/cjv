# Installing cjv

This chapter covers how to install cjv itself. cjv is a single-file executable, and installing it does not require any Cangjie SDK to be prepared in advance. Toolchains are downloaded by cjv on demand after installation (see [Basic usage](../basic-usage.md)).

There are three ways to install:

- [One-line install script](#one-line-install-script) (`install.sh` / `install.ps1`, recommended): automatically downloads the binary for your platform and performs initialization.
- [Download a prebuilt binary](#download-a-prebuilt-binary): manually fetch the binary from GitHub Releases and place it on your `PATH`.
- [Build from source](#build-from-source): build it yourself with `go install`.

If your network cannot reach GitHub, use the mirror variant of each method described here to download from [GitCode](https://gitcode.com/Zxilly/cjv).

## One-line install script

The script detects your platform, downloads the matching binary, verifies its SHA-256, and then runs `cjv init` to perform initialization (including adding cjv's `bin` directory to `PATH`).

### Linux / macOS

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh
```

When the script cannot detect a terminal (for example when run through a pipe), it installs silently in non-interactive mode. If a controlling terminal is detected, `cjv init` enters an interactive wizard that asks about the install directory, whether to modify `PATH`, and so on. Forward extra arguments to `cjv init` to skip the interaction, for example use `-y` to accept the defaults:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- -y
```

All arguments after `-s --` (except `--mirror`) are passed verbatim to `cjv init`, so `init` options such as `--no-modify-path` and `--default-toolchain none` can all be used here.

### Windows (PowerShell)

```powershell
irm https://cjv.zxilly.dev/install.ps1 | iex
```

`install.ps1` accepts parameters such as `-Yes` (skip confirmation), `-DefaultToolchain <name>` (the toolchain to install by default, with `none` meaning install nothing), and `-NoModifyPath` (do not modify `PATH`). When running through a pipe, pass parameters using the script block form:

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Yes
```

 >
 > There is no native build for Windows ARM64, so the script automatically falls back to the amd64 build, which runs under the system's x64 emulation layer.

### Mirror (GitCode)

When access to GitHub is poor, use the mirror to download the `cjv-mirror` archive from GitCode:

```bash
# Linux / macOS: add the --mirror flag
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror
```

```powershell
# Windows: set the CJV_MIRROR environment variable
$env:CJV_MIRROR = "1"; irm https://cjv.zxilly.dev/install.ps1 | iex
```

The mirror and the default source install the same cjv; the only difference is the download source and, subsequently, the update source for `cjv self update`. The mirror variant can be freely combined with the other parameters above, for example `curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror -y`.

## Download a prebuilt binary

Go to the [Releases](https://github.com/Zxilly/cjv/releases) page, download the archive matching your platform and extract it, then place the resulting `cjv` (`cjv.exe` on Windows) into any directory on your `PATH`.

Archives are named `cjv_<goos>_<goarch>`, with the extension being `.zip` on Windows and `.tar.gz` on other platforms:

|Platform|Archive|
|--------|-------|
|Linux x86_64|`cjv_linux_amd64.tar.gz`|
|Linux ARM64|`cjv_linux_arm64.tar.gz`|
|macOS Apple Silicon|`cjv_darwin_arm64.tar.gz`|
|macOS Intel|`cjv_darwin_amd64.tar.gz`|
|Windows x86_64|`cjv_windows_amd64.zip`|

Mirror users can download the corresponding `cjv-mirror_<goos>_<goarch>` archive from [GitCode Releases](https://gitcode.com/Zxilly/cjv/releases).

A manually installed binary is not initialized automatically. The first time you install a toolchain, cjv fills in the `PATH` configuration; see [PATH configuration on first install](#path-configuration-on-first-install) below.

## Build from source

This requires [Go](https://go.dev/) to be installed locally (see the repository's `go.mod` for the version requirement):

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

The binary is installed to `$(go env GOBIN)` (defaulting to `$(go env GOPATH)/bin`); make sure that directory is on your `PATH`. On networks in mainland China you can add a proxy:

```bash
GOPROXY=https://goproxy.cn,direct go install github.com/Zxilly/cjv/cmd/cjv@latest
```

As with downloading the binary manually, a source install is not initialized automatically; `PATH` is configured the first time you install a toolchain.

## PATH configuration on first install

cjv places its binary, along with proxy symlinks pointing to each toolchain (see [Proxies](../concepts/proxies.md)), under its own `bin` directory (`~/.cjv/bin` by default). For commands such as `cjc` and `cjpm` to be directly available in the terminal, this directory must be on your `PATH`.

When installed through the install script, `cjv init` completes the `PATH` configuration immediately. When installed through `go install` or by manually downloading the binary, cjv adds the `bin` directory to `PATH` only the first time you install a toolchain (such as `cjv install lts`).

How this is written depends on the platform. On Windows it is written to the user-level registry `PATH`; on Linux and macOS it is appended to a shell config file (such as `~/.profile`, `~/.bashrc`, `~/.zshenv`, and the fish config). Either way, the configuration takes effect only in a newly opened terminal; the current session needs a restart or a manual `source` before it recognizes the new `PATH`.

### Skipping automatic configuration

Set the environment variable `CJV_NO_PATH_SETUP` to `1` to skip this `PATH` modification, which suits CI and other scenarios where you do not want to change the user environment:

```bash
CJV_NO_PATH_SETUP=1 cjv install lts
```

When using the install script, the equivalent is to tell the underlying `cjv init` not to change `PATH`: on Linux / macOS pass `--no-modify-path`, on Windows use `-NoModifyPath`. In that case you need to add the `bin` directory to `PATH` yourself. `cjv init` also prints the path of an `env` script you can `source` (`~/.cjv/env` on Linux / macOS, `~/.cjv/env.ps1` and `~/.cjv/env.bat` on Windows).

For full documentation of environment variables such as `CJV_NO_PATH_SETUP`, see [Environment Variables](../environment-variables.md).

## Verifying the installation

```bash
cjv --version
```

If it prints a version number, the installation succeeded. Next you can install your first toolchain; continue with [Basic usage](../basic-usage.md).

 >
 > Update cjv itself with `cjv self update`, and uninstall it with `cjv self uninstall` (which also removes all installed toolchains). See the [Command Reference](../command-reference.md) for details.
