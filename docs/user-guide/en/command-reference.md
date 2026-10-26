# Command reference

Run `cjv <command> --help` to see the built-in help for your installed version. Read [Toolchains and versions](concepts/toolchains.md) for version name syntax, and read [Project toolchains](toolchain-file.md) for the toolchain selection order.

## Global options

Pass these options with any supporting subcommand:

| Option | Effect |
| --- | --- |
| `--json` | Write structured JSON results to `stdout`; not supported by `run`, `exec`, or `init` |
| `--quiet`, `-q` | Hide progress output; `component list -q` prints component names in a single column |
| `--verbose` | Enable `debug` logs on `stderr` when `CJV_LOG` is not set; mutually exclusive with `--quiet` |
| `--offline` | Disable cjv network requests and read from local installations and [cache](configuration.md#reusable-offline-cache) only |
| `--config <path>` | Select the user `settings.toml` file; system fallback settings and `cangjie-sdk.toml` still apply |
| `--no-auto-install` | Disable automatic installation of missing project requirements for this invocation |
| `--help`, `-h` | Show help for a command |
| `--version`, `-v` | Show the `cjv` binary version |
| `+name` | Select an SDK before any subcommand that supports toolchain selection |

An explicit `--toolchain` flag takes precedence over a leading global `+name` selector. Both `exec` and `envsetup` also accept `+name` after the subcommand name. The `run` and `exec` commands pass all child arguments and exit codes through unchanged.

## Install, update, and uninstall

### install

```text
cjv install [toolchain]... [-t target]... [-c component]... [--force | --no-update] [--allow-downgrade] [--reinstall]
```

If you omit `[toolchain]`, `install` installs the active toolchain selection. If you pass more than one toolchain name, cjv processes each toolchain in order. Both `-t` (`--target`) and `-c` (`--component`) accept repeated flags or comma-separated lists:

```bash
cjv install lts sts
cjv install nightly -c stdx,docs
```

### Installation and update policies

| Policy or flag | Behavior |
| --- | --- |
| Default policy | Keeps selected components and cross targets; blocks updates when needed archives are missing, while `nightly` searches published history for a compatible build |
| `--force` | Skips and removes unavailable optional components or targets during install or update; does not reinstall an unchanged SDK |
| `install --no-update` | Keeps the installed host release and adds requested targets or components only; mutually exclusive with `--force` |
| `install --reinstall` | Reinstalls the existing host release and keeps selected components and targets to repair a damaged SDK |
| `--allow-downgrade` | Lets `nightly` install or update select an older compatible nightly release |

When you do not request new components or targets, `cjv install --no-update` can finish offline.

### update

```text
cjv update [toolchain]... [--force] [--allow-downgrade] [--no-self-update] [--dry-run]
```

- If you pass toolchain names, `update` updates those toolchains only.
- If you omit names, `update` uses the global `+name` selector if present; otherwise it updates all installed tracked channels (`lts`, `sts`, `nightly`).
- Pinned versions such as `lts-1.0.5` stay unchanged, and [custom toolchains](install-from-url.md) have no official update source.
- Only a full `cjv update` (with no names and no `+name`) checks or installs cjv binary updates according to `auto_self_update`. Pass `--no-self-update` to skip that check.

When updating a tracked channel, cjv prepares the host SDK, tracked cross targets, and components in a staging directory before replacing the installation. If any step fails, cjv keeps your existing installation intact.

```bash
cjv update lts
cjv update --no-self-update
```

Pass `--dry-run` to calculate the update plan and report the latest release, the compatible release, and any blockers without downloading archives or changing `cache/`. Combine `--dry-run` with `--json` for CI scripts.

### uninstall

```text
cjv uninstall <toolchain>... [-y]
```

The `uninstall` command (or `cjv toolchain uninstall`) deletes the named SDK, its attached components, its offline documentation, and any directory overrides that point to it. If you uninstall the default toolchain, cjv selects another installed host SDK as the default or clears the default setting. Interactive terminals prompt for confirmation unless you pass `-y` (`--yes`) or `--json`.

### check

```text
cjv check
```

The `check` command queries upstream manifests for available SDK and cjv updates without installing them. It applies the same component and target compatibility rules as `cjv update`. Local `dev` builds skip the cjv binary version comparison.

### Update history and rollback

```text
cjv toolchain history [toolchain]
cjv toolchain history list [toolchain]
cjv toolchain history prune [toolchain] [--keep 2]
cjv toolchain rollback <toolchain> [snapshot]
```

cjv saves the previous host SDK, cross targets, and components as a snapshot group under `<CJV_HOME>/history/` whenever a tracked channel updates. By default, cjv keeps the 2 most recent snapshots per channel:

- Omit `[toolchain]` in `history list` or `history prune` to process all channels. Pass `--keep 0` to delete all snapshots while keeping active installations.
- `cjv toolchain rollback <toolchain>` restores the newest snapshot by default, checks its files, and saves the replaced group so you can undo the rollback. Read [Automation and JSON interfaces](automation.md#update-history-and-rollback) for details.

### doctor and cache

```text
cjv doctor
cjv cache dir
cjv cache list
cjv cache clean
cjv cache prune [--max-age 720h]
```

- `cjv doctor` does a read-only check of settings, proxy binaries, and SDK metadata, and exits with a nonzero code when it finds a problem.
- `cjv cache` lists or cleans cached manifests and archives with checked SHA-256 hashes under `<CJV_HOME>/cache/`. Cleaning the cache never deletes installed SDKs. Read [Configuration](configuration.md#reusable-offline-cache) and [Automation and JSON interfaces](automation.md).

## Inspect toolchains

| Command | Output |
| --- | --- |
| `cjv show` | Active toolchain, default host platform, and all installed toolchains |
| `cjv show active` | Active toolchain name and the selection rule that chose it |
| `cjv show installed`, `cjv toolchain list` | Installed host toolchains and attached cross targets |
| `cjv show home` | Effective `CJV_HOME` data directory and its configuration source |
| `cjv which [command] [--toolchain <tc>]` | Full path to an SDK tool binary, or the SDK root directory when you omit `[command]` |

### toolchain list-remote

```text
cjv toolchain list-remote [--channel all|lts|sts|nightly] [-t suffix] [--all-platforms] [--limit N]
```

Lists versions available in the distribution manifest. By default, it queries all channels for your current host platform. Pass `--all-platforms` to group output by platform, `-t` (`--target`) to filter by cross-target suffix, or `--limit 0` to show all historical versions without a limit.

## Run commands and set the environment

Read [Run commands and set the environment](runtime-environment.md) for usage examples across shells.

### run

```text
cjv run [--install] <toolchain> <command> [args...]
```

Runs `<command>` with an explicit `<toolchain>`. cjv searches the SDK tool directory first, and then searches the `PATH` of that environment. Place `--install` before `<toolchain>` to install the toolchain automatically if it is missing:

```bash
cjv run --install sts cjc --version
```

### exec

```text
cjv exec [+toolchain] <command> [args...]
```

Runs `<command>` with the environment and dynamic library paths of the active toolchain (or `+toolchain`). Use `exec` to run compiled Cangjie binaries that link against SDK or `stdx` runtime libraries:

```bash
cjv exec ./my_binary
cjv exec +nightly ./my_binary
```

If `<command>` starts with `+`, pass `cjv exec -- +command` so cjv does not parse it as a toolchain selector.

### envsetup

```text
cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]
```

Prints a shell script that exports SDK variables (`CANGJIE_HOME`, `PATH`, library paths, and `stdx` paths) into your current terminal. Supported `--shell` values are `bash`, `fish`, `powershell`, and `cmd`. Pass `--target` to load an installed cross SDK environment, or pass `--json` to get structured JSON data. Evaluate the printed script in your shell for the changes to take effect.

## Custom toolchains

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

Registers a local unpacked SDK directory as a reference, or extracts a local archive (`.zip`, `.tar.gz`) or HTTP/HTTPS URL into a managed directory. The `--sha256`, `--force`, and `--no-stdx` flags apply to archives and URLs only. Read [Custom SDKs](install-from-url.md) for archive layouts.

## Cross-compilation targets

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
```

Manages cross SDKs attached to a host toolchain. `target add` installs targets matching the host's current release without upgrading the host. `target remove` deletes the cross SDK and its target components while keeping the host SDK. `target list --installed` works offline. Read [Cross-compilation](cross-compilation.md).

## Components

```text
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--force]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--force]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [-q]
```

Manages `stdx`, `docs`, and `stdx-docs` packages for a host toolchain or an installed `--target <suffix>` cross SDK:

- `component add --force` reinstalls an installed component.
- `component link` links a local `stdx` directory that contains `dynamic/` and `static/` (pass `--force` to replace an existing `stdx`). `docs` and `stdx-docs` do not support local directory links.
- `cjv component` cannot edit external SDK directories linked with `cjv toolchain link`. Read [Components and offline documentation](concepts/components.md).

## Offline documentation

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

Opens offline HTML documentation in your browser (or prints the local file path when you pass `--path` or `--json`). Supported topics are `std`, `dev-guide` (alias `book`), `tools`, and `stdx`. Omit `[topic]` to open the documentation home page. `cjv docs` is a command alias. Install the `docs` or `stdx-docs` component first.

## Selection and settings

### pin

```text
cjv pin [toolchain] [--check]
```

Writes the resolved concrete release name to `cangjie-sdk.toml` in the current directory while keeping existing `components` and `targets`. Pass `--check` to make sure that `cangjie-sdk.toml` pins a concrete version without changing the file. Read [Project toolchains](toolchain-file.md) and [Automation and JSON interfaces](automation.md#pin-a-project-version).

### default

```text
cjv default [toolchain]
```

Prints the global default toolchain when called without arguments, or clears it when called with `none`. If you pass an official toolchain that is not installed yet, `default` installs it first and keeps your old default if installation fails. A cross SDK cannot be the default.

### override

```text
cjv override set <toolchain> [--path <dir>]
cjv override unset [--path <dir>] [--nonexistent]
cjv override list
```

Sets, removes, or lists local directory overrides in `settings.toml`. By default, `set` and `unset` apply to the current working directory. Pass `--nonexistent` to `unset` to remove entries for directories that no longer exist on disk. Read [Project toolchains](toolchain-file.md#local-directory-overrides).

### set

```text
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set home <path>
```

Updates individual fields in the active user `settings.toml` file (default `~/.cjv/settings.toml`). Read [Configuration](configuration.md) for field defaults and inheritance.

## Initialization and self-management

### init

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
```

Installs the `cjv` binary and proxy entry points into `<CJV_HOME>/bin` and configures `PATH`. By default, `init` queries available channels for your host platform and selects the first available channel in the order `lts`, `sts`, then `nightly`. The interactive wizard lists only channels available for your platform; if `init` cannot confirm any channel, it exits with an error.

| Option | Effect |
| --- | --- |
| `--default-toolchain <name>` | Select a default toolchain explicitly |
| `--default-toolchain none` | Set up cjv without installing an SDK |
| `-c`, `--component` | Install components (`stdx`, `docs`, `stdx-docs`) with the default SDK |
| `-y`, `--yes` | Run non-interactively with default or specified options |
| `--no-modify-path` | Do not edit shell profiles or the Windows user `PATH` |

If SDK installation fails during `init`, the command exits nonzero, keeps the installed `cjv` binary, and prints retry and `list-remote` commands using the absolute binary path. Read [Install cjv](installation/index.md) for installer script flags.

### self update and self uninstall

```text
cjv self update
cjv self uninstall [-y]
```

- `cjv self update` downloads and installs the latest `cjv` release.
- `cjv self uninstall` deletes `<CJV_HOME>`, all installed toolchains, components, and caches, and removes `<CJV_HOME>/bin` from `PATH`. Pass `-y` in non-interactive or `--json` mode.

> **CAUTION:** `cjv self uninstall` permanently deletes all SDKs, components, and history snapshots stored inside `<CJV_HOME>`.

## Shell completion

```text
cjv completion <bash|zsh|fish|powershell>
```

Prints the tab-completion script for the named shell. Run `cjv completion <shell> --help` for instructions on how to load the script in your shell profile.
