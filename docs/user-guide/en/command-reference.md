# Command reference

The `cjv` command line includes built-in help for every subcommand; run `cjv help` or `cjv <command> --help` to inspect the options supported by your installed version. See [Toolchains and versions](concepts/toolchains.md) for toolchain naming syntax and [Project toolchains](toolchain-file.md) for the toolchain selection order.

## Global options

| Option | Effect |
| --- | --- |
| `--json` | Write structured JSON results to `stdout`; unavailable for `run`, `exec`, and `init` |
| `--quiet`, `-q` | Hide progress output; `component list -q` prints names in a single column |
| `--verbose` | Enable debug logs on `stderr` when `CJV_LOG` is unset; mutually exclusive with `--quiet` |
| `--offline` | Disable `cjv` network access and use local installations and [cached files](configuration.md#download-cache) only |
| `--config <path>` | Select the user `settings.toml` file; system fallback settings and project selection still apply |
| `--no-auto-install` | Disable automatic installation of missing project requirements for this invocation |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the `cjv` version |
| `+name` | Select an SDK before any subcommand that supports toolchain selection |

An explicit `--toolchain` flag on a subcommand takes precedence over a leading global `+name` selector. Both `exec` and `envsetup` also accept `+name` after the subcommand name. `run` and `exec` forward all child arguments and preserve child exit codes.

## Install, update, and uninstall

### install

```text
cjv install [toolchain]... [-t target]... [-c component]... [--force | --no-update] [--allow-downgrade] [--reinstall]
```

When called without a toolchain name, `cjv install` installs the currently active selection. When multiple toolchain names are given, each one is processed in order. Both `-t` (`--target`) and `-c` (`--component`) accept repeated flags or comma-separated lists:

```bash
cjv install lts sts
cjv install nightly -c stdx,docs
```

### Installation and update policies

| Policy | Behavior |
| --- | --- |
| Default | Preserve required components and cross targets; missing artifacts block updates, while `nightly` searches published history for a compatible release |
| `--force` | Allow unavailable optional components or targets to be skipped and removed; do not reinstall an unchanged SDK |
| `install --no-update` | Keep the installed host release and add requested targets or components only; mutually exclusive with `--force` |
| `install --reinstall` | Reinstall the existing release and retain selected components and targets to repair a damaged SDK |
| `--allow-downgrade` | Allow `nightly` installation or update to select an older compatible nightly build |

When no new components or targets are requested, `cjv install --no-update` can complete entirely offline.

### update

```text
cjv update [toolchain]... [--force] [--allow-downgrade] [--no-self-update] [--dry-run]
```

`cjv update` updates the named toolchains. When called without names, it uses the global `+name` selector if present, or otherwise updates all installed tracked channels (`lts`, `sts`, `nightly`). Pinned concrete versions remain unchanged, and custom toolchains have no official update source. Only a full `cjv update` (without names or `+name`) checks for or installs `cjv` self-updates according to [`auto_self_update`](configuration.md#automatic-updates); pass `--no-self-update` to skip that check.

Channel updates stage the host SDK, tracked cross targets, and components before replacing the active installation atomically. Any failure preserves or restores the previous installation.

```bash
cjv update lts
cjv update --no-self-update
```

The first command updates `lts` only; the second updates all tracked channels while skipping the `cjv` self-update check.

Passing `--dry-run` resolves the update plan and reports the newest release, the compatible release, and any blockers without downloading archives, modifying installations, or writing to `cache/`. Combine `--dry-run` with `--json` for CI checks.

### uninstall

```text
cjv uninstall <toolchain>... [-y]
```

`cjv uninstall` (or its alias `cjv toolchain uninstall`) removes the specified SDK along with its attached components, offline documentation, and any directory overrides pointing to it. If you uninstall the current default toolchain, `cjv` attempts to select another installed host toolchain as the default, or clears the default if none remain. Interactive terminals prompt for confirmation; `-y` (`--yes`), non-interactive input, and `--json` mode proceed without prompting.

### check

```text
cjv check
```

`cjv check` queries upstream manifests for available SDK and `cjv` updates without installing them, applying the same component and target compatibility rules as `cjv update`. Local development builds (`dev`) skip comparing their own binary version.

### Update history and rollback

```text
cjv toolchain history [toolchain]
cjv toolchain history list [toolchain]
cjv toolchain history prune [toolchain] [--keep 2]
cjv toolchain rollback <toolchain> [snapshot]
```

Whenever a tracked channel updates to a new release or digest, `cjv` retains the previous host SDK, its tracked targets, and its components together as a snapshot group under `<CJV_HOME>/history/`, keeping the **2** most recent snapshots by default. Listing or pruning without a toolchain name operates on all history groups; `--keep 0` removes all snapshots while keeping current installations intact. `cjv toolchain rollback` selects the newest snapshot by default, verifies its contents on disk, and saves the replaced group so the rollback can itself be undone. See [Update history and rollback](automation.md#update-history-and-rollback).

### doctor and cache

```text
cjv doctor
cjv cache dir
cjv cache list
cjv cache clean
cjv cache prune [--max-age 720h]
```

`cjv doctor` performs a read-only check of your configuration, proxy command entry points, and SDK metadata, exiting with a nonzero status code when it finds a problem. The `cjv cache` commands inspect and clean SHA-256-verified archives and manifests in `<CJV_HOME>/cache/`; cleaning the cache never removes installed SDKs. See [Automation and JSON interfaces](automation.md) for JSON schemas and offline workflows.

## Inspect toolchains

| Command | Purpose |
| --- | --- |
| `cjv show` | Show the active toolchain, default host platform, and all installed toolchains |
| `cjv show active` | Show the active toolchain and the selection rule that chose it |
| `cjv show installed`, `cjv toolchain list` | List installed toolchains and attached cross targets |
| `cjv show home` | Show the effective `CJV_HOME` data directory and its configuration source |
| `cjv which [command] [--toolchain <tc>]` | Print the resolved path to an SDK tool binary, or the SDK root directory when `[command]` is omitted |

### toolchain list-remote

```text
cjv toolchain list-remote [--channel all|lts|sts|nightly] [-t suffix] [--all-platforms] [--limit N]
```

Lists releases available in the remote distribution manifests. By default, it queries all channels for your current host platform. `--all-platforms` groups results across every platform, `--limit 0` removes the per-group version limit, and `-t` (`--target`) filters releases by cross-target suffix.

## Run commands and set the environment

### run

```text
cjv run [--install] <toolchain> <command> [args...]
```

Runs a command with an explicitly named toolchain. `cjv` searches that SDK's tool directories first, and then searches that environment's `PATH`. Place `--install` before the `<toolchain>` argument to allow a missing toolchain to be installed automatically:

```bash
cjv run --install sts cjc --version
```

### exec

```text
cjv exec [+toolchain] <command> [args...]
```

Runs a command with the active toolchain's runtime environment (or selects a specific toolchain with `+name`). This is the recommended way to run compiled Cangjie binaries that dynamically link against SDK or `stdx` runtime libraries:

```bash
cjv exec ./my_binary
cjv exec +nightly ./my_binary
```

Use `cjv exec -- +command` if the executable name itself begins with `+`. Both `run` and `exec` forward child arguments and preserve exit codes.

### envsetup

```text
cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]
```

Prints a shell script that exports the SDK environment into your current shell session. Supported `--shell` formats are `bash`, `fish`, `powershell`, and `cmd`. Passing `--target` loads the environment of an installed cross SDK, and `--json` returns structured environment data.

Remember to evaluate the printed script in your shell (`eval "$(cjv envsetup)"` or `cjv envsetup | Invoke-Expression`) for the changes to take effect. See [Running commands and setting the environment](runtime-environment.md) for shell examples and how to return to per-project proxy selection.

## Custom toolchains

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

Passing an unpacked directory creates a lightweight reference; passing a local `.zip` / `.tar.gz` archive or HTTP(S) URL extracts it into a `cjv`-managed installation. The `--sha256`, `--force`, and `--no-stdx` flags apply only to archives and URLs: they verify the expected SHA-256 digest, replace an existing custom toolchain of the same name, and skip bundled `stdx` installation respectively. See [Custom SDKs](install-from-url.md) for supported archive layouts and platform rules.

## Cross-compilation targets

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
```

Manages cross-compilation target SDKs attached to a host toolchain. `target add` installs target SDKs matching the host's current release without upgrading the host; `target remove` removes the cross SDK and its target components while keeping the host installation. `target list --installed` works offline. See [Cross-compilation](cross-compilation.md).

## Components

```text
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--force]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--force]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [-q]
```

Manages optional components (`stdx`, `docs`, and `stdx-docs`), which can be passed as multiple arguments or comma-separated lists. Passing `--target <suffix>` targets an already-installed cross SDK on the selected host without installing the target itself.

`component add --force` reinstalls an installed component, and `component link --force` replaces an existing component link. Only `stdx` supports local directory links (`component link`), and the source directory must contain both `dynamic/` and `static/`. External SDK directories registered via `cjv toolchain link` cannot be modified through component commands. See [Components and offline documentation](concepts/components.md).

## Offline documentation

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

Opens local HTML documentation in your browser. Supported topics are `std`, `dev-guide` (alias `book`), `tools`, and `stdx`; omitting the topic opens the documentation home page. `--path` and `--json` print the resolved filesystem path without opening a browser. `cjv docs` is a command alias. Make sure the corresponding `docs` or `stdx-docs` component is installed first.

## Selection and settings

### pin

```text
cjv pin [toolchain] [--check]
```

Writes `cangjie-sdk.toml` in the current directory with a resolved concrete release name while preserving existing `components` and `targets`. `--check` verifies that the project file pins a concrete release without writing to disk. See [Automation and JSON interfaces](automation.md#pin-a-project-version).

### default

```text
cjv default [toolchain]
```

Displays the current global default toolchain when called without arguments, or clears it when passed `none`. Passing an official toolchain that is not yet installed downloads and installs it first, preserving your previous default if installation fails. A cross-compilation target SDK cannot be set as the default.

### override

```text
cjv override set <toolchain> [--path <dir>]
cjv override unset [--path <dir>] [--nonexistent]
cjv override list
```

Manages per-directory toolchain overrides, defaulting to the current working directory when `--path` is omitted. `--nonexistent` cleans up overrides for directories that no longer exist on disk. Overrides are stored in the active user settings file (default `~/.cjv/settings.toml`) regardless of `CJV_HOME`. See [Project toolchains](toolchain-file.md#local-directory-overrides) and [Configuration](configuration.md).

### set

```text
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set home <path>
```

Updates individual settings in the active user settings file (default `~/.cjv/settings.toml`). See [Configuration](configuration.md) for defaults, system fallback inheritance, and environment variable overrides.

## Initialization and self-management

### init

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
```

Installs `cjv` and SDK proxy entry points into `<CJV_HOME>/bin` and configures `PATH`. By default, `init` queries the channels available for your current platform and selects the first available channel in the order `lts`, `sts`, then `nightly`. The interactive wizard lists only channels available for your platform; if no channel can be confirmed, initialization exits with an error.

| Option | Effect |
| --- | --- |
| `--default-toolchain <name>` | Select a default toolchain explicitly |
| `--default-toolchain none` | Install `cjv` without installing an SDK |
| `-c`, `--component` | Install components (`stdx`, `docs`, `stdx-docs`) alongside the initial SDK |
| `-y`, `--yes` | Skip interactive prompts and accept defaults or explicit flags |
| `--no-modify-path` | Leave shell profiles and `PATH` unchanged |

When standard input is not a terminal, `init` runs in non-interactive mode automatically. If installing the initial SDK fails, `init` exits with a nonzero status code while keeping the installed `cjv` binary, and prints ready-to-run retry and `list-remote` commands using the binary's absolute path (so they work before `PATH` is reloaded in a new shell).

### self update / self uninstall

```text
cjv self update
cjv self uninstall [-y]
```

`cjv self update` checks for and installs the newest release of `cjv` itself. `cjv self uninstall` removes the `<CJV_HOME>` data directory, all installed toolchains, components, caches, and snapshots, and cleans up `PATH` (`--json` mode requires `-y`).

## Shell completion

```text
cjv completion <bash|zsh|fish|powershell>
```

Prints a tab-completion script for the specified shell. Run `cjv completion <shell> --help` for instructions on loading it in your shell profile.
