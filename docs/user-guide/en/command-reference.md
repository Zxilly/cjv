# Command reference

The `cjv` command line includes built-in help for every subcommand; run `cjv help` or `cjv <command> --help` to inspect the options supported by your installed version. See [Toolchains and versions](concepts/toolchains.md) for toolchain naming syntax and [Project toolchains](toolchain-file.md) for the toolchain selection order.

## Global options

| Option | Effect |
| --- | --- |
| `--json` | Write structured JSON to stdout; unsupported by `run`, `exec`, `init`, and `completion` |
| `--quiet`, `-q` | Hide progress in every argument position; use `component list --names-only` for one name per line |
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
cjv install [toolchain]... [-t target]... [-c component]... [--allow-missing] [--allow-downgrade] [--reinstall]
```

`install` ensures the selected toolchain exists, preserving an installed release and adding requested targets or components. Use `update` to advance an installed channel. With no name it uses the active selection; multiple names are processed in order. `-t` and `-c` accept repeated flags or comma-separated lists:

```bash
cjv install lts sts
cjv install nightly -c stdx,docs
```

### Installation and update policies

| Policy | Behavior |
| --- | --- |
| Default | Preserve required components and cross targets; missing artifacts block updates, while `nightly` searches published history for a compatible release |
| `--allow-missing` | Allow unavailable optional components or targets to be skipped and removed, without reinstalling unchanged SDKs |
| Default `install` | Keep the installed host release and add requested targets or components only |
| `install --reinstall` | Reinstall the existing release and retain selected components and targets to repair a damaged SDK |
| `--allow-downgrade` | Allow `nightly` installation or update to select an older compatible nightly build |

With no new components or targets, `install` for an installed toolchain can complete entirely offline. `--force` remains a deprecated compatibility flag for `--allow-missing`; `install --no-update` is retained because preserving the installed release is now the default. `--allow-downgrade` affects release resolution, without making ordinary `install` advance or downgrade an installed channel.

### update

```text
cjv update [toolchain]... [--allow-missing] [--allow-downgrade] [--no-self-update] [--dry-run]
```

`cjv update` updates the named toolchains. When called without names, it uses the global `+name` selector if present, or otherwise updates all installed tracked channels (`lts`, `sts`, `nightly`). Pinned concrete versions remain unchanged, and custom toolchains have no official update source. Only a full `cjv update` (without names or `+name`) checks for or installs `cjv` self-updates according to [`auto_self_update`](configuration.md#automatic-updates); pass `--no-self-update` to skip that check.

Channel updates stage the host SDK, tracked cross targets, and components before replacing the active installation atomically. Any failure preserves or restores the previous installation.

```bash
cjv update lts
cjv update --no-self-update
```

The first command updates `lts` only; the second updates all tracked channels while skipping the `cjv` self-update check.

`--dry-run` reports the newest release, the compatible release, and any blockers. A full update also reports the planned cjv self-check or self-update according to current policy. It never downloads archives, modifies installations, or writes caches. Use `--json` in CI. Explicitly naming a missing toolchain in `update` still installs it.

### uninstall

```text
cjv uninstall <toolchain>... [-y]
```

`uninstall` removes the SDK, owned cross SDKs, components, documentation, and directory overrides. Removing the default selects another installed host or clears the default. Interactive terminals prompt; non-interactive input or `--json` requires explicit `-y` (`--yes`). Output format does not authorize deletion. `toolchain uninstall` remains a hidden compatibility command.

### check

```text
cjv check
```

`check` queries SDK and cjv updates without installation or cache writes, using the same compatibility rules as `update`. A leading `+name` limits SDK inspection to that toolchain. Development builds skip their own version comparison. `doctor` checks local health; `pin --check` validates a concrete project selection, without checking for updates.

### Update history and rollback

```text
cjv toolchain history list [toolchain]
cjv toolchain history prune [toolchain] [--keep 2]
cjv toolchain rollback <toolchain> [snapshot]
```

Tracked updates save hosts, targets, and components as a group, retaining **2** snapshots. Listing/pruning without a name uses a leading `+name`, or all groups if no selector is provided. Omitting `list` remains a shorthand. `--keep 0` clears snapshots without removing installations. Rollback verifies the chosen snapshot and saves the current group for undo. See [Update history and rollback](automation.md#update-history-and-rollback).

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
| `cjv`, `cjv show` | Show the same active toolchain, default host, and installation report, with the same JSON structure |
| `cjv show active` | Show the active toolchain and the selection rule that chose it |
| `cjv toolchain list` | List hosts and cross targets, including actual releases and tracking, pinned, or custom kinds |
| `cjv show home` | Show the effective `CJV_HOME` data directory and its configuration source |
| `cjv which [command] [--toolchain <tc>]` | Print the resolved path to an SDK tool binary, or the SDK root directory when `[command]` is omitted |

Inspection commands do not install missing SDKs or components. `show installed` remains a hidden compatibility command for `toolchain list`.

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

Run any command using an explicitly named toolchain. Like `exec`, this prepares SDK paths and runtime libraries, installing missing requirements according to `auto_install`. `--install` explicitly enables installation, precedes the toolchain name, and conflicts with `--no-auto-install`:

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

Evaluate the printed script in the current shell to apply it. `envsetup` reads installed SDKs and validates project requirements; missing requirements produce explicit installation hints, without automatic installation. See [Running commands and setting the environment](runtime-environment.md) for shell examples and project-based proxy selection.

## Custom toolchains

```text
cjv toolchain link <name> <directory>
cjv toolchain import <name> <archive|url> [--sha256 <hash>] [--replace] [--no-stdx]
```

`link` references an existing directory whose owner maintains its contents. `import` extracts a local archive or HTTP(S) URL into a managed installation. Its `--sha256`, `--replace`, and `--no-stdx` verify a digest, replace the named installation, and skip bundled stdx. Legacy archive/URL `link` forms remain compatible with a migration note and their old `--force` replacement flag. See [Custom SDKs](install-from-url.md).

## Cross-compilation targets

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
```

Manages cross-compilation target SDKs attached to a host toolchain. `target add` installs target SDKs matching the host's current release without upgrading the host; `target remove` removes the cross SDK and its target components while keeping the host installation. `target list --installed` works offline. See [Cross-compilation](cross-compilation.md).

## Components

```text
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--reinstall]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--replace]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [--names-only]
```

Manages optional components (`stdx`, `docs`, and `stdx-docs`), which can be passed as multiple arguments or comma-separated lists. Passing `--target <suffix>` targets an already-installed cross SDK on the selected host without installing the target itself.

`component add --reinstall` reinstalls a component; `component link --replace` replaces one. Old `--force` flags remain hidden compatibility options. `list --names-only` prints one name per line; `-q` always hides progress only. Only stdx supports directory links, with both `dynamic/` and `static/` present. External SDK directory links cannot be modified through component commands. See [Components and offline documentation](concepts/components.md).

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

With no argument, display the global default; `none` clears it. Selecting a missing official SDK normally installs it first; `--no-auto-install` requires it to exist already. Failure leaves the previous selection intact. Cross SDKs cannot be defaults.

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

`self update` checks and updates cjv. `self uninstall` removes `<CJV_HOME>`, all toolchains, components, caches, snapshots, and PATH entries. Non-interactive input or `--json` requires explicit `-y`, matching SDK uninstall.

## Shell completion

```text
cjv completion <bash|zsh|fish|powershell>
```

Prints a tab-completion script for the specified shell. Run `cjv completion <shell> --help` for instructions on loading it in your shell profile.
