# Install cjv

cjv is a single executable and does not require an existing Cangjie SDK.

## Installer scripts

Linux / macOS:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://cjv.zxilly.dev/install.ps1 | iex
```

The script detects the platform, downloads and verifies cjv, then runs `cjv init`. Interactive terminals show a wizard; non-interactive input uses defaults. For automation, skip confirmation explicitly or install cjv without an SDK:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Yes -DefaultToolchain none
```

The Unix script forwards all arguments except `--mirror` to `cjv init`. The PowerShell script provides corresponding options such as `-Yes`, `-DefaultToolchain`, and `-NoModifyPath`.

### GitCode mirror

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Mirror
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
