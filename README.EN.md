# cjv - Cangjie Version Manager

English | [中文](README.md)

A toolchain manager for the [Cangjie](https://cangjie-lang.cn/) programming language SDK.

cjv manages multiple Cangjie SDK installations, handles version switching, and provides transparent proxy execution of SDK tools.

## Installation

### From source

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

### From release binaries

Download the appropriate binary from the [Releases](https://github.com/Zxilly/cjv/releases) page and place it in your PATH.

## Quick Start

```bash
# Install the latest LTS toolchain
cjv install lts

# Set it as the default
cjv default lts

# Verify installation
cjv show

# Run a command with a specific toolchain
cjv run sts cjc --version
```

## Commands

| Command                                             | Description                                             |
| --------------------------------------------------- | ------------------------------------------------------- |
| `cjv install <toolchain> [-t target]`                | Install a Cangjie SDK toolchain, optionally with cross targets |
| `cjv uninstall <toolchain>`                         | Uninstall a toolchain                                   |
| `cjv update [toolchain]`                            | Update installed toolchains                             |
| `cjv default [toolchain]`                           | Set or show the default toolchain                       |
| `cjv show`                                          | Show active and installed toolchains                    |
| `cjv show active`                                   | Show the active toolchain                               |
| `cjv show installed`                                | List installed toolchains                               |
| `cjv show home`                                     | Show CJV_HOME path                                      |
| `cjv run <toolchain> <command> [args...]`           | Run a command with a specific toolchain                 |
| `cjv exec [+toolchain] <command> [args...]`         | Run a command with Cangjie runtime environment          |
| `cjv envsetup [+toolchain] [--shell=TYPE]`          | Print shell commands to configure Cangjie runtime environment |
| `cjv which <command>`                               | Show the path of an SDK tool for the active toolchain   |
| `cjv check`                                         | Check for available updates without installing          |
| `cjv override set <toolchain>`                      | Set a toolchain override for the current directory      |
| `cjv override unset`                                | Remove the toolchain override for the current directory |
| `cjv override list`                                 | List all directory overrides                            |
| `cjv toolchain list`                                | List installed toolchains                               |
| `cjv toolchain link <name> <path>`                  | Link a custom toolchain to a local directory            |
| `cjv toolchain uninstall <name>`                    | Uninstall a toolchain                                   |
| `cjv set auto-self-update <enable\|disable\|check>` | Set auto-self-update behavior                           |
| `cjv set auto-install <true\|false>`                | Set auto-install for missing toolchains in proxy mode   |
| `cjv set gitcode-api-key <key>`                     | Set GitCode API access token (required for nightly builds) |
| `cjv self update`                                   | Update cjv to the latest version                        |
| `cjv self uninstall`                                | Uninstall cjv and all installed toolchains              |
| `cjv self clean-cache`                              | Clean the download cache                                |

## Toolchain Resolution

cjv resolves the active toolchain in the following order (highest priority first):

1. `CJV_TOOLCHAIN` environment variable
2. Directory override (set via `cjv override set`)
3. Toolchain file (`cangjie-sdk.toml` in the current or parent directories)
4. Default toolchain (set via `cjv default`)

## Cross-Compilation SDKs

cjv follows rustup-style `targets` semantics: target SDKs are additive installs for the host toolchain and do not change the active toolchain. Proxy execution of `cjc` and `cjpm` still uses the host SDK.

```bash
# Install the host STS SDK and the OHOS cross SDK for the current host
cjv install sts -t ohos

# Targets can be repeated or comma-separated
cjv install sts -t ohos -t android
cjv install sts --target ohos,android
```

Projects can also declare additive targets in `cangjie-sdk.toml`. When `auto_install` is enabled, proxy execution ensures missing target SDKs are installed:

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

`targets` accepts suffixes such as `ohos`, `android`, and `ohos-arm32`; do not use full platform keys such as `linux-x64-ohos`.

## Proxy Mode

When SDK tools (e.g., `cjc`, `cjpm`) are invoked directly, cjv transparently proxies the call to the appropriate toolchain. Proxy symlinks are created in the cjv bin directory during installation.

If `auto_install` is enabled in settings and the resolved toolchain is not installed, cjv will automatically install it before proxying.

## Runtime Environment

Cangjie-compiled binaries dynamically link against runtime libraries (e.g., `libcangjie-runtime`) and require the correct library search paths to run. cjv provides two ways to configure the runtime environment:

**One-shot execution**: Use `cjv exec` to run a command with the correct runtime environment without affecting the current shell:

```bash
cjv exec ./my_binary arg1 arg2

# Specify a toolchain
cjv exec +nightly ./my_binary
```

**Configure the current shell session**: Use `cjv envsetup` to output environment configuration scripts, then run compiled binaries directly:

```bash
# Bash/Zsh
eval "$(cjv envsetup)"

# Fish
cjv envsetup | source

# PowerShell
cjv envsetup | Invoke-Expression
```

Both commands use the same toolchain resolution priority as proxy mode and support `+toolchain` syntax to specify a toolchain. `cjv envsetup` auto-detects the current shell type, or you can override it with `--shell=TYPE` (supported: `bash`, `fish`, `powershell`, `cmd`).

## Environment Variables

| Variable                | Description                                                   |
| ----------------------- | ------------------------------------------------------------- |
| `CJV_HOME`              | Override the default home directory (default: `~/.cjv`)       |
| `CJV_TOOLCHAIN`         | Force a specific toolchain, overriding all other resolution   |
| `CJV_LOG`               | Set log verbosity: `debug`, `info`, `warn` (default), `error` |
| `CJV_MAX_RETRIES`       | Max download retry attempts (default: `3`)                    |
| `CJV_DOWNLOAD_TIMEOUT`  | HTTP download timeout in seconds (default: `180`)             |
| `CJV_GITCODE_API_KEY`   | GitCode API access token for querying and downloading nightly toolchains |
| `CJV_NO_PATH_SETUP`     | Set to `1` to skip PATH configuration on first install        |

## Directory Structure

```
~/.cjv/
  bin/            # Proxy symlinks and the cjv binary
  toolchains/     # Installed SDK toolchains
  downloads/      # Downloaded SDK archives (cache)
  settings.toml   # User settings
```

## Configuration

Settings are stored in `~/.cjv/settings.toml` and can be modified via `cjv set` commands.

## License

Apache-2.0. See [LICENSE](LICENSE) for details.

## Credits

cjv's design is inspired by [rustup](https://github.com/rust-lang/rustup)
