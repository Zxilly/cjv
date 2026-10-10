# Command reference

Use `cjv <command> --help` for the installed version's help. See [toolchains and versions](concepts/toolchains.md) for names and [project toolchains](toolchain-file.md) for selection order.

## Global options

| Option | Effect |
| --- | --- |
| `--json` | Write structured results to stdout; unavailable for `run`, `exec`, and `init` |
| `--quiet`, `-q` | Hide progress; `component list -q` prints names in one column |
| `--verbose` | Enable debug logs when `CJV_LOG` is unset; mutually exclusive with global quiet |
| `--help`, `-h` | Show help |
| `--version`, `-v` | Show the cjv version |
| `+name` | Select an SDK before a subcommand that supports toolchain selection |

Explicit `--toolchain` takes precedence over global `+name`. `exec` and `envsetup` also accept `+name` after the subcommand. `run` and `exec` forward child arguments and preserve exit codes.

## Install, update, and uninstall

```text
cjv install [toolchain]... [-t target]... [-c component]... [--force | --no-update] [--allow-downgrade]
cjv update [toolchain]... [--force] [--allow-downgrade] [--no-self-update]
cjv uninstall <toolchain>... [-y]
cjv check
```

Without a name, `install` installs the current selection. Multiple names are processed individually. `-t/--target` and `-c/--component` accept repeated or comma-separated values.

`update` processes the named toolchains. Without names, it uses global `+name` if present, otherwise all tracked channels. Concrete versions remain pinned; custom toolchains have no official update source. Only a full update checks or updates cjv according to `auto_self_update`; `--no-self-update` skips this.

| Policy | Behavior |
| --- | --- |
| Default | Preserve required components and targets; missing artifacts block updates, while nightly can search compatible history |
| `--force` | Allow unavailable optional components or targets to be skipped and removed; do not reinstall unchanged SDKs |
| `install --no-update` | Keep the installed release and add targets or components; mutually exclusive with `--force` |
| `--allow-downgrade` | Allow an older compatible nightly |

Channel updates prepare the host, tracked targets, and components before publishing them together. Failures preserve or restore the previous installation. Pinned versions remain separate. `--no-update` with no new requirements can complete offline.

`uninstall` also removes components, documentation, and related directory overrides. Removing the default toolchain tries to select another installed host. Interactive terminals ask for confirmation; `-y/--yes`, non-interactive input, and JSON mode proceed directly. `toolchain uninstall` is equivalent.

`check` only queries SDK and cjv updates. Development builds skip their own version comparison.

## Inspect and run

| Command | Purpose |
| --- | --- |
| `cjv show` | Active toolchain, default host, and installed toolchains |
| `cjv show active` | Active toolchain and its source |
| `cjv show installed`, `cjv toolchain list` | Installed toolchains |
| `cjv show home` | Data directory and its source |
| `cjv which [command] [--toolchain <tc>]` | SDK tool path; SDK root when command is omitted |
| `cjv run [--install] <toolchain> <command> [args...]` | Run with an explicit SDK; search SDK tools first, then that environment's PATH |
| `cjv exec [+toolchain] <command> [args...]` | Run with the active or specified toolchain environment |
| `cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]` | Print a shell environment script, or environment data in JSON mode |

Put `run --install` before the toolchain argument to allow installation of a missing toolchain. `exec -- +command` runs a command starting with `+`. `envsetup` supports `bash`, `fish`, `powershell`, and `cmd`; `--target` requires an installed cross SDK. See [runtime environments](runtime-environment.md).

```text
cjv toolchain list-remote [--channel all|lts|sts|nightly] [-t suffix] [--all-platforms] [--limit N]
```

Remote listing defaults to all channels on the current host platform. `--all-platforms` groups results by every platform. `--limit 0` leaves each group's version count unlimited. `-t/--target` filters by cross-target suffix.

## Custom toolchains

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

A directory creates a reference; an archive or URL is extracted into a managed installation. The three flags apply only to archives: verify SHA-256, replace an existing name, and skip bundled stdx. See [custom SDKs](install-from-url.md) for layout and platform requirements.

## Targets and components

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--force]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--force]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [-q]
```

`target add` keeps the host release; `target remove` keeps the host installation. `target list --installed` works offline. Component `--target` selects an existing cross SDK and does not install it.

Components are `stdx`, `docs`, and `stdx-docs`; names may be comma-separated. `component add --force` reinstalls a component, and `component link --force` replaces an existing one. Only stdx supports local links, with `dynamic/` and `static/` required in the source. External SDK directory links cannot be edited through component commands. See [components](concepts/components.md).

## Offline documentation

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

Topics are `std`, `dev-guide` (alias `book`), `tools`, and `stdx`. Omitting the topic opens the documentation home page. `--path` and `--json` return the path without opening a browser. `docs` is a command alias. Install the documentation components first.

## Selection and settings

```text
cjv default [toolchain]
cjv override set <toolchain> [--path <dir>]
cjv override unset [--path <dir>] [--nonexistent]
cjv override list
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set home <path>
```

`default` displays the selection when no argument is supplied; `none` clears it. Selecting a missing official toolchain installs it first and preserves the old default on failure. A cross SDK cannot be the default.

Overrides apply to the current directory unless a path is specified. `--nonexistent` removes stale directory entries. Settings are saved to `~/.cjv/settings.toml`, regardless of `CJV_HOME`. See [configuration](configuration.md) for field meanings.

## Initialization and self-management

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
cjv self update
cjv self uninstall [-y]
```

`init` installs cjv command entry points, sets up PATH, and automatically selects a channel available for the current platform, in order of `lts`, `sts`, then `nightly`. The interactive menu only lists available channels. For example, HarmonyOS currently selects `nightly`. If no channel can be confirmed, initialization fails. A toolchain installation failure also returns a nonzero exit code without reporting success; the installed cjv remains available. The error includes commands to retry with the same components and list compatible versions; both use the installed binary path so they work before PATH is reloaded. An explicit `--default-toolchain <name>` overrides automatic selection. `--default-toolchain none` skips the SDK, `-c/--component` selects components, `-y/--yes` skips interaction, and `--no-modify-path` leaves PATH unchanged. Non-terminal input uses non-interactive mode.

`self update` updates cjv itself. `self uninstall` removes the data directory, toolchains, and components, and cleans up PATH. JSON mode requires `-y`.

## Shell completion

```text
cjv completion <bash|zsh|fish|powershell>
```

Print a completion script for the selected shell. See `cjv completion <shell> --help` for loading instructions.
