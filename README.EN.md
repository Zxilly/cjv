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
| `cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]` | Print shell commands to configure Cangjie runtime environment |
| `cjv which <command>`                               | Show the path of an SDK tool for the active toolchain   |
| `cjv check`                                         | Check for available updates without installing          |
| `cjv override set <toolchain>`                      | Set a toolchain override for the current directory      |
| `cjv override unset`                                | Remove the toolchain override for the current directory |
| `cjv override list`                                 | List all directory overrides                            |
| `cjv toolchain list`                                | List installed toolchains                               |
| `cjv toolchain link <name> <path>`                  | Link a custom toolchain to a local directory            |
| `cjv toolchain uninstall <name>`                    | Uninstall a toolchain                                   |
| `cjv component add <name>... [--toolchain <tc>]`    | Install a component (e.g. stdx) onto a toolchain        |
| `cjv component link stdx <path> [--toolchain <tc>]` | Link a local stdx directory onto a toolchain (custom OK)|
| `cjv component remove <name>... [--toolchain <tc>]` | Remove a component from a toolchain                     |
| `cjv component list [--toolchain <tc>] [--installed]` | List installed and available components               |
| `cjv doc [--path] [--toolchain <tc>] [topic]`       | Open the toolchain's offline documentation in your browser |
| `cjv set auto-self-update <enable\|disable\|check>` | Set auto-self-update behavior                           |
| `cjv set auto-install <true\|false>`                | Set auto-install for missing toolchains in proxy mode   |
| `cjv set gitcode-api-key <key>`                     | Set GitCode API access token (required for nightly builds) |
| `cjv self update`                                   | Update cjv to the latest version                        |
| `cjv self uninstall`                                | Uninstall cjv and all installed toolchains              |

## Toolchain Resolution

cjv resolves the active toolchain in the following order (highest priority first):

1. `CJV_TOOLCHAIN` environment variable
2. Directory override (set via `cjv override set`)
3. Toolchain file (`cangjie-sdk.toml` in the current or parent directories)
4. Default toolchain (set via `cjv default`)

## Toolchain File `cangjie-sdk.toml`

cjv walks up from the current directory and uses the first `cangjie-sdk.toml` it finds as the project's toolchain declaration. All fields live under the `[toolchain]` table:

```toml
[toolchain]
channel = "lts"                          # required; toolchain name (e.g. lts / sts / nightly / a specific version)
components = ["stdx", "docs"]            # optional; components to install alongside the toolchain
targets = ["ohos", "android"]            # optional; additive cross-compilation target suffixes
```

| Field        | Type     | Description                                                                              |
| ------------ | -------- | ---------------------------------------------------------------------------------------- |
| `channel`    | string   | Toolchain name. An empty file is equivalent to no declaration and falls through to the next resolution step. |
| `components` | string[] | When `auto_install` is enabled, missing components are installed transparently during proxy execution. |
| `targets`    | string[] | Target suffixes only (e.g. `ohos`, `android`, `ohos-arm32`); do not use full platform keys. |

Unrecognized keys (such as a typo `[toolchian]` or `channal = "lts"`) are reported as warn-level log messages but do not block parsing. The semantics of `targets` and `components` are detailed in the sections below.

## Cross-Compilation SDKs

Target SDKs are additive installs on top of the host toolchain and do not change the active toolchain. Proxy execution of `cjc` and `cjpm` still uses the host SDK.

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

## Components

cjv ships a component system to manage SDK extras that release alongside each toolchain. Currently supported:

- `stdx` — Cangjie extension libraries. Installs into `<CJV_HOME>/stdx/<tc>/{dynamic,static}` and exposes `CANGJIE_STDX_PATH_DYNAMIC` / `CANGJIE_STDX_PATH_STATIC` to proxied tools. Downloads from [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases) on LTS / STS and from [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) on nightly.
- `docs` — Cangjie main offline documentation (dev-guide, libs/std, tools). LTS / STS download from the [`cangjie-docs-bundle`](https://github.com/Zxilly/cangjie-docs-bundle/releases) GitHub release; nightly downloads from [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases).
- `stdx-docs` — Cangjie extension library offline docs. LTS / STS download from [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases); nightly downloads from [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases).

```bash
# Install components alongside the toolchain
cjv install nightly -c stdx,docs

# Or manage them independently
cjv component add stdx --toolchain lts
cjv component remove stdx-docs
cjv component list --toolchain nightly
```

### Linking a local stdx

`cjv component add stdx` does not work for custom toolchains created via `cjv toolchain link` because there is no matching release asset. Use `cjv component link stdx <path>` to point cjv at a local stdx directory instead:

```bash
# Link a locally built / fetched SDK first
cjv toolchain link mysdk /path/to/local/sdk

# Then point a local stdx directory at it
cjv component link stdx /path/to/local/stdx --toolchain mysdk

# Standard channels can also use link instead of download (offline boxes, stdx debugging, etc.)
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

`<path>` must contain `dynamic/` and `static/` subdirectories (the standard stdx layout). cjv creates two symlinks under `<CJV_HOME>/stdx/<tc>/` (falling back to directory junctions on Windows when symlinks need elevation), and the `CANGJIE_STDX_PATH_*` env vars are injected the same way. Both `cjv component remove stdx` and `cjv toolchain uninstall` only remove the symlinks — your original directory is never touched.

`cangjie-sdk.toml` recognises a `components` field; when `auto_install` is enabled, missing components are installed transparently during proxy execution:

```toml
[toolchain]
channel = "nightly"
components = ["stdx", "docs"]
```

`cjv doc` opens the local HTML in your browser (defaults to `index.html`; accepts topics like `stdx`, `std`, `dev-guide`, `book`, `tools`). Use `--path` to print the resolved path without launching a browser. If the docs (or stdx-docs) component is not installed for the active toolchain, `cjv doc` prints a hint pointing at `cjv component add`.

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

For cross-compilation, use `--target=SUFFIX` (e.g. `ohos`) to emit the environment of an installed target SDK (standalone-SDK model: `CANGJIE_HOME` points at the target SDK directory, with PATH/library paths taken from it). Install the target SDK first with `cjv install <toolchain> --target <suffix>`.

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
| `CANGJIE_STDX_PATH_DYNAMIC` | Auto-injected by cjv to `<CJV_HOME>/stdx/<tc>/dynamic` (only when stdx is installed) |
| `CANGJIE_STDX_PATH_STATIC`  | Auto-injected by cjv to `<CJV_HOME>/stdx/<tc>/static` (only when stdx is installed)  |

## Directory Structure

```
~/.cjv/
  bin/            # Proxy symlinks and the cjv binary
  toolchains/     # Installed SDK toolchains (SDK files only)
    <tc>/
      .cjv/components/         # cjv-managed component manifests
  stdx/           # stdx component, per toolchain (path exposed via CANGJIE_STDX_PATH_*)
    <tc>/
      dynamic/
      static/
  docs/           # Offline docs, decoupled from the toolchain tree
    <tc>/
      main/                    # docs component (dev-guide, libs/std, tools)
      stdx/                    # stdx-docs component (libs_stdx)
  downloads/      # Transient staging area (drained on successful install; only retained for crash recovery)
  settings.toml   # User settings
```

`cjv toolchain uninstall <tc>` cleans up `stdx/<tc>/` and `docs/<tc>/` along with the toolchain directory.

## Configuration

Settings are stored in `~/.cjv/settings.toml` and can be modified via `cjv set` commands.

## License

Apache-2.0. See [LICENSE](LICENSE) for details.

