# Install cjv

cjv is a single executable and does not require an existing Cangjie SDK.

## Installer scripts

Linux / macOS / HarmonyOS HiShell:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

The script detects the platform, downloads and verifies cjv, then runs `cjv init`. Interactive terminals show a wizard; non-interactive input uses defaults. For automation, skip confirmation explicitly or install cjv without an SDK:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Yes -DefaultToolchain none
```

The Unix script handles `--mirror` and `--version`, then forwards the remaining arguments to `cjv init`. The PowerShell script provides `-Mirror`, `-Version`, `-Yes`, `-DefaultToolchain`, and `-NoModifyPath`.

### Pin the cjv version

Automated installations can select an exact cjv release tag, including prereleases:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --version v0.4.1 -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Version v0.4.1 -Yes -DefaultToolchain none
```

Replace the example tag with the required version. Both scripts also read `CJV_VERSION`; the command-line version takes precedence. Without either, they download the latest stable release. This option selects cjv itself; `--default-toolchain` / `-DefaultToolchain` independently selects the Cangjie SDK. Explicit `CJV_UPDATE_ROOT`, `CJV_GITHUB_ROOT`, and `CJV_GITCODE_ROOT` download URLs take precedence over the version option. For an internal source, point the URL directly at the required version's artifact directory.

### SDK selection

By default, `cjv init` queries channels for the current platform and selects the first available channel in the order `lts`, `sts`, then `nightly`. Use `--default-toolchain <name>` to select a specific toolchain, or `--default-toolchain none` to install cjv alone.

If SDK installation fails, the command returns a nonzero exit code and keeps the installed cjv. Follow the retry command in the error message or check available versions with `cjv toolchain list-remote`.

### GitCode mirror

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

The mirror build uses GitCode for its default version manifest and cjv updates. See [enterprise deployment](../enterprise/index.md) for internal artifact servers.

## Manual download

Download an archive from [GitHub Releases](https://github.com/Zxilly/cjv/releases), extract `cjv` or `cjv.exe` to a directory on `PATH`, and run `cjv init`.

| Platform | Archive |
| --- | --- |
| Linux x86_64 | `cjv_linux_amd64.tar.gz` |
| Linux ARM64 | `cjv_linux_arm64.tar.gz` |
| macOS Apple Silicon | `cjv_darwin_arm64.tar.gz` |
| macOS Intel | `cjv_darwin_amd64.tar.gz` |
| Windows x86_64 | `cjv_windows_amd64.zip` |
| HarmonyOS x86_64 | `cjv_openharmony_amd64.tar.gz` |
| HarmonyOS ARM64 | `cjv_openharmony_arm64.tar.gz` |

[GitCode Releases](https://gitcode.com/Zxilly/cjv/releases) provides equivalent archives prefixed with `cjv-mirror_`. The download page also offers `cjv-init` executables that start the installation wizard directly.

## Install from source

With a Go version satisfying the repository's `go.mod` requirement:

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
cjv init
```

Go installs the executable to `GOBIN`, or `$(go env GOPATH)/bin` when unset. Add that directory to `PATH`.

## PATH and verification

SDK proxy commands live in `<CJV_HOME>/bin`, which defaults to `~/.cjv/bin`. `init` configures PATH. If you skip initialization and install an SDK directly, the first toolchain installation also attempts PATH setup. Windows updates the user PATH; Unix updates shell configuration. Open a new terminal and check:

```bash
cjv --version
```

To manage PATH yourself, pass `--no-modify-path` to `init` or `-NoModifyPath` to the PowerShell installer. `CJV_NO_PATH_SETUP=1` also skips automatic setup. Continue with the [quick start](../basic-usage.md) to install a toolchain.
