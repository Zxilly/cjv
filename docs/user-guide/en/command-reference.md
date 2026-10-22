# Command reference

Use `cjv <command> --help` for the installed version's help. See [toolchains and versions](concepts/toolchains.md) for names and [project toolchains](toolchain-file.md) for selection order.

## Global options

| Option | Effect |
| --- | --- |
| `--json` | Write structured results to stdout; unavailable for `run`, `exec`, and `init` |
| `--quiet`, `-q` | Hide progress; `component list -q` prints names in one column |
| `--verbose` | Enable debug logs when `CJV_LOG` is unset; mutually exclusive with global quiet |
| `--offline` | Disable cjv network access and use local installations and cached files |
| `--config <path>` | Select user settings; system fallback and project selection still apply |
| `--no-auto-install` | Disable automatic installation for this invocation |
| `--help`, `-h` | Show help |
| `--version`, `-v` | Show the cjv version |
| `+name` | Select an SDK before a subcommand that supports toolchain selection |

Explicit `--toolchain` takes precedence over global `+name`. `exec` and `envsetup` also accept `+name` after the subcommand. `run` and `exec` forward child arguments and preserve exit codes.

## Install, update, and uninstall

### install

```text
cjv install [toolchain]... [-t target]... [-c component]... [--force | --no-update] [--allow-downgrade] [--reinstall]
```

Without a name, `install` installs the current selection. Multiple names are processed individually. `-t/--target` and `-c/--component` accept repeated or comma-separated values.

Install several channels, or include components during installation:

```bash
cjv install lts sts
cjv install nightly -c stdx,docs
```

### Installation and update policies

| Policy | Behavior |
| --- | --- |
| Default | Preserve required components and targets; missing artifacts block updates, while nightly can search compatible history |
| `--force` | Allow unavailable optional components or targets to be skipped and removed; do not reinstall unchanged SDKs |
| `install --no-update` | Keep the installed release and add targets or components; mutually exclusive with `--force` |
| `install --reinstall` | Reinstall the existing release and retain components and targets to repair a damaged SDK |
| `--allow-downgrade` | Allow an older compatible nightly |

`install --no-update` can complete offline when no additional components or targets are requested.

### update

```text
cjv update [toolchain]... [--force] [--allow-downgrade] [--no-self-update] [--dry-run]
```

`update` processes the named toolchains. Without names, it uses global `+name` if present, otherwise all tracked channels. Concrete versions remain pinned; custom toolchains have no official update source. Only a full update checks or updates cjv according to `auto_self_update`; `--no-self-update` skips this.

Channel updates prepare the host SDK, tracked targets, and components before replacing the installation. Failures preserve or restore the previous installation. Pinned versions remain separate.

```bash
cjv update lts
cjv update --no-self-update
```

The first command updates LTS only. The second updates all tracked channels and skips the check for cjv updates.

`--dry-run` resolves the update plan and reports the newest release, the compatible release, and blockers without installing or writing caches. Add `--json` for CI use.

### uninstall

```text
cjv uninstall <toolchain>... [-y]
```

`uninstall` also removes components, documentation, and related directory overrides. Removing the default toolchain tries to select another installed host. Interactive terminals ask for confirmation; `-y/--yes`, non-interactive input, and JSON mode proceed directly. `toolchain uninstall` is equivalent.

### check

```text
cjv check
```

`check` only queries SDK and cjv updates. SDK results use the same component and target compatibility rules as installation. Development builds skip their own version comparison.

### Update history and rollback

```text
cjv toolchain history [toolchain]
cjv toolchain history list [toolchain]
cjv toolchain history prune [toolchain] [--keep 2]
cjv toolchain rollback <toolchain> [snapshot]
```

History retains a tracking host, its targets, and components as one group, keeping the two most recent snapshots after updates. Listing or pruning without a toolchain selects all history groups; `--keep 0` clears history. Rollback uses the newest snapshot by default, verifies its contents, and saves the current group before restoring. See [update history and rollback](automation.md#update-history-and-rollback).

### doctor and cache

```text
cjv doctor
cjv cache dir
cjv cache list
cjv cache clean
cjv cache prune [--max-age 720h]
```

`doctor` reads configuration, command entry points, and SDK integrity, and exits nonzero when it finds a problem. Cache commands manage verified archives and manifests; removing cached files does not delete installed SDKs. See [automation](automation.md) for JSON conventions and offline use.

## Inspect toolchains

| Command | Purpose |
| --- | --- |
| `cjv show` | Active toolchain, default host, and installed toolchains |
| `cjv show active` | Active toolchain and its source |
| `cjv show installed`, `cjv toolchain list` | Installed toolchains |
| `cjv show home` | Data directory and its source |
| `cjv which [command] [--toolchain <tc>]` | SDK tool path; SDK root when command is omitted |

### toolchain list-remote

```text
cjv toolchain list-remote [--channel all|lts|sts|nightly] [-t suffix] [--all-platforms] [--limit N]
```

Remote listing defaults to all channels on the current host platform. `--all-platforms` groups results by every platform. `--limit 0` leaves each group's version count unlimited. `-t/--target` filters by cross-target suffix.

## Run commands and set the environment

### run

```text
cjv run [--install] <toolchain> <command> [args...]
```

Run a command with an explicit toolchain. cjv searches SDK tools first, then that environment's PATH. Put `--install` before the toolchain argument to allow a missing toolchain to be installed:

```bash
cjv run --install sts cjc --version
```

### exec

```text
cjv exec [+toolchain] <command> [args...]
```

Run a command with the active toolchain's environment, or select a toolchain with `+name`. This is useful for compiled programs that need Cangjie runtime libraries:

```bash
cjv exec ./my_binary
cjv exec +nightly ./my_binary
```

Use `cjv exec -- +command` to run a command whose name starts with `+`. Both `run` and `exec` forward child arguments and preserve exit codes.

### envsetup

```text
cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]
```

Print a script that configures the current shell. Supported output formats are `bash`, `fish`, `powershell`, and `cmd`. `--target` requires an installed cross SDK. In JSON mode, the command returns environment data.

The output must be evaluated in your shell to take effect. See [runtime environments](runtime-environment.md) for shell-specific examples and how to switch back to project-based toolchain selection.

## Custom toolchains

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

A directory creates a reference; an archive or URL is extracted into a managed installation. The three flags apply only to archives: verify SHA-256, replace an existing name, and skip bundled stdx. See [custom SDKs](install-from-url.md) for layout and platform requirements.

## Cross-compilation targets

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
```

`target add` keeps the host release; `target remove` keeps the host installation. `target list --installed` works offline.

## Components

```text
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--force]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--force]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [-q]
```

Component `--target` selects an existing cross SDK and does not install it.

Components are `stdx`, `docs`, and `stdx-docs`; names may be comma-separated. `component add --force` reinstalls a component, and `component link --force` replaces an existing one. Only stdx supports local links, with `dynamic/` and `static/` required in the source. External SDK directory links cannot be edited through component commands. See [components](concepts/components.md).

## Offline documentation

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

Topics are `std`, `dev-guide` (alias `book`), `tools`, and `stdx`. Omitting the topic opens the documentation home page. `--path` and `--json` return the path without opening a browser. `docs` is a command alias. Install the documentation components first.

## Selection and settings

### pin

```text
cjv pin [toolchain] [--check]
```

Write `cangjie-sdk.toml` in the current directory with a concrete release, preserving existing components and targets. `--check` only verifies that the project file pins a concrete release. See [automation](automation.md#pin-a-project-version).

### default

```text
cjv default [toolchain]
```

`default` displays the selection when no argument is supplied; `none` clears it. Selecting a missing official toolchain installs it first and preserves the old default on failure. A cross SDK cannot be the default.

### override

```text
cjv override set <toolchain> [--path <dir>]
cjv override unset [--path <dir>] [--nonexistent]
cjv override list
```

Overrides apply to the current directory unless a path is specified. `--nonexistent` removes stale directory entries. Settings are saved to the selected user settings file, defaulting to `~/.cjv/settings.toml`, regardless of `CJV_HOME`. See [configuration](configuration.md) for field meanings.

### set

```text
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set home <path>
```

Change the selected user settings file, which defaults to `~/.cjv/settings.toml`. See [configuration](configuration.md) for defaults, inheritance, and environment overrides.

## Initialization and self-management

### init

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
```

`init` installs cjv command entry points and configures PATH. It queries channels available for the current platform and selects the first available channel in the order `lts`, `sts`, then `nightly`. The interactive menu lists only available channels. If no channel can be confirmed, initialization fails.

| Option | Effect |
| --- | --- |
| `--default-toolchain <name>` | Select a toolchain explicitly |
| `--default-toolchain none` | Install cjv without an SDK |
| `-c`, `--component` | Select components |
| `-y`, `--yes` | Skip interaction |
| `--no-modify-path` | Leave PATH unchanged |

Non-terminal input uses non-interactive mode. If toolchain installation fails, the command returns a nonzero exit code and keeps the installed cjv. The error includes commands to retry with the same components and list compatible versions. Both use the installed binary's absolute path, so they work before PATH is reloaded.

### self update / self uninstall

```text
cjv self update
cjv self uninstall [-y]
```

`self update` updates cjv itself. `self uninstall` removes the data directory, toolchains, and components, and cleans up PATH. JSON mode requires `-y`.

## Shell completion

```text
cjv completion <bash|zsh|fish|powershell>
```

Print a completion script for the selected shell. See `cjv completion <shell> --help` for loading instructions.
