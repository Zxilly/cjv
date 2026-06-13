# Command Reference

This chapter lists cjv's commands one by one, giving their usage, arguments, flags, and copy-pasteable examples. A short description of each command also appears in `cjv <command> --help`.

## Global Conventions

Almost all commands accept the global flag `--json`, which outputs the result as a stable JSON structure on standard output for scripts to consume. `cjv run`, `cjv exec`, and `cjv init` do not support JSON output, and passing `--json` to them is an error.

Commands that do not specify a toolchain explicitly resolve the active toolchain by a uniform priority order: the `CJV_TOOLCHAIN` environment variable, directory overrides, the `cangjie-sdk.toml` toolchain file, and the default toolchain, taking the first one that applies in that order. See [Targets and overrides](concepts/targets-overrides.md).

The standard channel names are `lts`, `sts`, and `nightly`, and can also be written as a specific version (such as `lts-1.0.0`). Custom toolchains linked with `cjv toolchain link` use any custom name, but it must not conflict with a reserved channel name. `cjv exec` and `cjv envsetup` also support a `+name` prefix to select a toolchain temporarily, overriding the default resolution.

A proxied or executed subcommand exits with its original exit code; this applies to `cjv run` and `cjv exec`.

---

## Installing and Uninstalling

### `cjv install`

Installs a Cangjie SDK toolchain, optionally with cross-compilation targets and components.

```text
cjv install <toolchain> [-t target]... [-c component]... [--force]
```

Arguments:

- `<toolchain>` (required): the toolchain to install, such as `lts`, `sts`, `nightly`, or a specific version. It cannot install a custom toolchain; use `cjv toolchain link` for that.

Flags:

|Flags|Description|
|-----|-----------|
|`-t`, `--target <suffix>`|Cross-compilation target suffixes to additionally install (repeatable or comma-separated), such as `ohos`, `android`, `ohos-arm32`|
|`-c`, `--component <name>`|Components to additionally install (repeatable or comma-separated), such as `stdx`, `docs`, `stdx-docs`|
|`--force`|Force a re-download and reinstall, even if already installed|

Examples:

```bash
# Install the latest LTS toolchain
cjv install lts

# Install a specific version
cjv install lts-1.0.0

# Install the host STS SDK, plus two extra cross-compilation targets
cjv install sts -t ohos -t android
cjv install sts --target ohos,android

# Install components along with the toolchain
cjv install nightly -c stdx,docs

# Force a reinstall
cjv install lts --force
```

For target, give only the target suffix, not the full platform key (such as `linux-x64-ohos`). A cross-compilation target is an add-on to the host toolchain and does not change the active toolchain. See [Cross-compilation](cross-compilation.md) and [Components](concepts/components.md).

### `cjv uninstall`

Uninstalls a toolchain, also cleaning up its stdx and offline documentation.

```text
cjv uninstall <toolchain> [-y]
```

Arguments:

- `<toolchain>` (required): the name of the toolchain to uninstall.

Flags:

|Flags|Description|
|-----|-----------|
|`-y`, `--yes`|Skip the confirmation prompt|

Uninstalling prompts for confirmation in an interactive terminal; in a non-interactive terminal, in `--json` mode, or with `-y`, it proceeds directly. If the uninstalled toolchain is the default, cjv points the default at another installed host toolchain, and directory overrides pointing at it are cleared. Uninstalling also removes `<CJV_HOME>/stdx/<tc>/` and `<CJV_HOME>/docs/<tc>/`.

```bash
cjv uninstall sts
cjv uninstall lts-1.0.0 -y
```

 >
 > `cjv toolchain uninstall <name>` is equivalent to this command and behaves identically.

### `cjv update`

Updates a specified toolchain or all installed toolchains to the latest version of their respective channels.

```text
cjv update [toolchain] [--no-self-update]
```

Arguments:

- `[toolchain]` (optional): update only the specified toolchain. When omitted, all installed toolchains are updated.

Flags:

|Flags|Description|
|-----|-----------|
|`--no-self-update`|Skip the cjv self-update check|

When given a channel name (such as `lts`), updates the currently installed version of that channel to the latest version. When given a specific version, this is equivalent to installing that version, and is skipped if already installed. Custom (linked) toolchains cannot be updated and are skipped or reported as an error. After updating to a new version, a default toolchain or directory override that pointed at the old version is repointed at the new version, and the old directory is deleted. After the update finishes, whether cjv updates itself depends on the `auto-self-update` setting, which can be disabled with `--no-self-update`.

```bash
# Update all toolchains
cjv update

# Update only LTS
cjv update lts

# Update but do not trigger the cjv self-update
cjv update --no-self-update
```

### `cjv check`

Checks whether updates are available for installed toolchains, but does not perform the install.

```text
cjv check
```

Lists installed toolchains one by one: `current → latest` if an update is available, `✓` if already up to date, and shows cjv's own version at the end. `--json` mode outputs structured results with fields such as `update_available` and `latest`.

```bash
cjv check
cjv check --json
```

---

## Inspecting and Running

### `cjv show`

Shows the active toolchain, the default host platform, and the list of installed toolchains.

```text
cjv show
cjv show active
cjv show installed
cjv show home
```

Subcommands:

|Subcommand|Description|
|----------|-----------|
|`cjv show`|Show the active toolchain + default host + installed list (including the components installed for each toolchain)|
|`cjv show active`|Show only the currently active toolchain and its source|
|`cjv show installed`|List only the installed toolchains|
|`cjv show home`|Show the `CJV_HOME` path and its source|

```bash
cjv show
cjv show active
cjv show home
```

### `cjv run`

Runs a command using the specified toolchain, without affecting the current shell.

```text
cjv run [--install] <toolchain> <command> [args...]
```

Arguments:

- `<toolchain>` (required): the toolchain to run the command with.
- `<command>` (required): the command to run; it can be a tool bundled with the toolchain (such as `cjc` or `cjpm`), or any command on the PATH within that toolchain's environment.
- `[args...]`: arguments passed to the command.

Flags:

|Flags|Description|
|-----|-----------|
|`--install`|Automatically install the target toolchain before running if it is not installed|

The command runs in that toolchain's runtime environment; cjv injects the correct PATH and library paths and applies the environment of installed components (such as `CANGJIE_STDX_PATH_*`). This command does not support `--json`.

```bash
# Check the cjc version with the sts toolchain
cjv run sts cjc --version

# Install the toolchain first if it is not installed, then run
cjv run --install nightly cjpm build
```

### `cjv exec`

Run an arbitrary command in the Cangjie runtime environment, making it easy to run compiled artifacts directly.

```text
cjv exec [+toolchain] <command> [args...]
```

Arguments:

- `[+toolchain]` (optional): use a `+name` prefix to select a toolchain for this invocation; when omitted, the active toolchain is resolved by the standard priority order.
- `<command>` (required): the command to execute.
- `[args...]`: arguments passed to the command.

Binaries compiled by Cangjie dynamically link against the runtime libraries and need the correct library search paths. `cjv exec` runs the command in an environment with the runtime library paths injected, without affecting the current shell. This command does not support `--json`.

```bash
# Run a compiled artifact in the active toolchain's runtime environment
cjv exec ./my_binary arg1 arg2

# Select a toolchain
cjv exec +nightly ./my_binary

# Everything after "--" is passed through as-is, allowing you to run command names starting with "+"
cjv exec -- +weird-command
```

See [Runtime environment](runtime-environment.md) for details.

### `cjv envsetup`

Output shell commands that configure the Cangjie runtime environment, for the current shell session to `eval`.

```text
cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]
```

Arguments and flags:

|Argument / flag|Description|
|---------------|-----------|
|`[+toolchain]`|Select a toolchain for this invocation with `+name`|
|`--shell=TYPE`|Manually specify the shell type: `bash`, `fish`, `powershell`, `cmd`; auto-detected when omitted|
|`--target=SUFFIX`|Output the runtime environment of an installed target SDK (standalone SDK model), e.g. `--target=ohos`|

`envsetup` uses the same toolchain resolution priority as proxy mode. The target SDK referenced by `--target` must first be installed via `cjv install <toolchain> --target <suffix>`. The `--json` mode outputs a structured description of the environment (variables, PATH prepend/append, library path keys) instead of printing a shell script.

```bash
# Bash / Zsh
eval "$(cjv envsetup)"

# Fish
cjv envsetup | source

# PowerShell
cjv envsetup | Invoke-Expression

# Select a toolchain and force bash format
cjv envsetup +nightly --shell=bash

# Output the environment of the installed ohos target SDK
cjv envsetup --target=ohos
```

### `cjv which`

Show the path of an SDK tool in the active toolchain; with no argument, print the toolchain root directory.

```text
cjv which [command]
```

Arguments:

- `[command]` (optional): the tool name to query, such as `cjc` or `cjpm`. When omitted, print the active toolchain root directory.

```bash
# Print the active toolchain root directory
cjv which

# Print the absolute path of cjc
cjv which cjc
```

`cjv which` uses the same tool resolution logic as `cjv run`: besides the fixed proxied tools, it can also resolve binaries under `bin/` and `tools/bin/`.

### `cjv doc`

Open the current toolchain's offline documentation in the browser.

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

Arguments:

- `[topic]` (optional): the subpage topic to jump to, such as `stdx`, `std`, `dev-guide`, `book`, `tools`. When omitted, open the root `index.html`.

Flags:

|Flags|Description|
|-----|-----------|
|`--path`|Print only the documentation path or URL, without opening the browser|
|`--toolchain <tc>`|Specify the toolchain whose documentation to open (defaults to the current active toolchain)|

If the target toolchain has not yet installed `docs` / `stdx-docs`, you are prompted to install it first with `cjv component add`. The `--json` mode likewise returns only the path without launching the browser. Command alias: `cjv docs`.

```bash
cjv doc
cjv doc std
cjv doc --path
cjv doc stdx --toolchain nightly
```

---

## Toolchain management

### `cjv toolchain list`

List installed toolchains (equivalent to `cjv show installed`).

```text
cjv toolchain list
```

### `cjv toolchain link`

Link a custom toolchain to a local directory (reference), or extract a local archive / URL and install it as a cjv-owned toolchain (materialize).

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

Arguments:

- `<name>` (required): the custom toolchain name. It must be a custom name, must not conflict with the reserved channel names `lts`, `sts`, or `nightly`, and must not contain path separators, a `+` prefix, or otherwise be an invalid name.
- `<path|url>` (required): a local directory, a local archive file (`.zip` / `.tar.gz`), or an `http(s)://` URL. The command first checks whether the argument matches `^https?://`; otherwise it treats the path as local — a regular file is materialized, a directory is referenced.

Two behaviors:

|Aspect|Reference mode (local directory)|Materialize mode (local archive / URL)|
|------|--------------------------------|--------------------------------------|
|`<path>` form|Local directory|Local archive `sdk.zip`, or `https://...`|
|`toolchains/<name>` contents|Symlink / junction (falls back to junction on Windows)|The real directory materialized after extraction|
|Data ownership|Not owned by cjv, only referenced|Owned by cjv|
|Uninstall behavior|Only the link is deleted; the original directory is preserved|Deletes the entire directory (including stdx)|

In materialize mode a local archive and a URL share the same extraction logic; the only difference is that a URL is downloaded and staged first, while a local archive is read in place and never deleted.

Flags (materialize mode only; apply equally to a local archive and a URL):

|Flags|Description|
|-----|-----------|
|`--sha256 <hash>`|Verify the archive against this SHA-256|
|`--force`|Overwrite an existing toolchain with the same name|
|`--no-stdx`|Skip installing the bundled stdx component|

These three flags only apply to materialize mode; using them with a local directory is an error rather than being silently ignored. Reference mode requires the directory to be a real Cangjie SDK (`bin/cjc` must exist).

```bash
# Reference mode: only create a link, the original directory is preserved
cjv toolchain link mysdk /path/to/local/sdk

# Materialize mode (local archive): extract into a cjv-owned real directory, the source file is kept
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip

# Materialize mode (URL): download, extract, and materialize into a cjv-owned real directory
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip

# Materialize mode + verification + overwrite same name + skip bundled stdx
cjv toolchain link mysdk https://example.com/sdk.zip \
  --sha256 <hash> --force --no-stdx
```

For the full semantics of materialize mode (timing of name validation, bundled stdx, cross-system limitations, etc.), see [Installing a toolchain from a URL or archive](install-from-url.md). For linking a local stdx, see [Components](concepts/components.md).

### `cjv toolchain uninstall`

Uninstall a toolchain (equivalent to `cjv uninstall`).

```text
cjv toolchain uninstall <name> [-y]
```

|Flags|Description|
|-----|-----------|
|`-y`, `--yes`|Skip the confirmation prompt|

---

## Component management

The subcommands of `cjv component` all support the persistent flag `--toolchain <tc>` to specify the target toolchain; when omitted, the current active toolchain is used.

### `cjv component add`

Install one or more components for a toolchain (such as `stdx`, `docs`, `stdx-docs`).

```text
cjv component add <name>... [--toolchain <tc>] [--force]
```

|Flags|Description|
|-----|-----------|
|`--toolchain <tc>`|The target toolchain (defaults to the current active toolchain)|
|`--force`|Force a re-download and reinstall, even if already installed|

`<name>` may be repeated or comma-separated. A custom toolchain linked with `cjv toolchain link` has no corresponding release asset, so `component add` is not available for it; use `cjv component link` instead.

```bash
cjv component add stdx --toolchain lts
cjv component add stdx,docs
cjv component add stdx --force
```

### `cjv component link`

Link a local component directory to a toolchain instead of installing via download. Currently applies to `stdx`.

```text
cjv component link <name> <path> [--toolchain <tc>] [--force]
```

|Flags|Description|
|-----|-----------|
|`--toolchain <tc>`|The target toolchain (defaults to the current active toolchain)|
|`--force`|Replace an existing component installation (whether it was obtained via link or download)|

`<path>` must be a directory with the extracted stdx layout, containing the two subdirectories `dynamic/` and `static/`. After linking, cjv creates symlinks under `<CJV_HOME>/stdx/<tc>/` (falling back to a junction on Windows), and `CANGJIE_STDX_PATH_DYNAMIC` / `CANGJIE_STDX_PATH_STATIC` are injected as usual. `cjv component remove` and uninstalling a toolchain only delete the symlinks and do not touch the original data.

```bash
# A custom toolchain has no release assets, so use link to attach a local stdx
cjv toolchain link mysdk /path/to/local/sdk
cjv component link stdx /path/to/local/stdx --toolchain mysdk

# Standard channels can also use link instead of download (offline / debugging a self-compiled stdx)
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

### `cjv component remove`

Uninstall one or more components from a toolchain.

```text
cjv component remove <name>... [--toolchain <tc>]
```

`<name>` may be repeated or comma-separated. Aliases: `uninstall`, `rm`, `delete`, `del`.

```bash
cjv component remove stdx-docs
cjv component remove stdx,docs --toolchain nightly
```

### `cjv component list`

List the installed and installable status of components.

```text
cjv component list [--toolchain <tc>] [--installed] [-q]
```

|Flags|Description|
|-----|-----------|
|`--toolchain <tc>`|The target toolchain (defaults to the current active toolchain)|
|`--installed`|List only installed components|
|`-q`, `--quiet`|Output in a single column (print only names, convenient for scripts)|

```bash
cjv component list
cjv component list --toolchain nightly
cjv component list --installed -q
```

See [Components](concepts/components.md) for details.

---

## Default toolchain and overrides

### `cjv default`

Set or show the default toolchain.

```text
cjv default [toolchain]
```

Arguments:

- `[toolchain]` (optional): the toolchain to set as the default. When omitted, show the current default. Pass `none` to clear the default setting.

A cross-compilation target variant (such as `lts-1.0.0-ohos`) cannot be set as the active or default toolchain; use the host toolchain and configure it through targets. Setting a toolchain that is not yet installed gives a warn but is not blocked.

```bash
# Show the current default
cjv default

# Set it to lts
cjv default lts

# Clear the default
cjv default none
```

### `cjv override set`

Set a toolchain override for a directory. When entering that directory (or any of its subdirectories), cjv uses that toolchain preferentially.

```text
cjv override set <toolchain> [--path <dir>]
```

|Flags|Description|
|-----|-----------|
|`--path <dir>`|Set the override for the specified directory instead of the current directory|

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
```

### `cjv override unset`

Remove a directory's toolchain override.

```text
cjv override unset [--path <dir>] [--nonexistent]
```

|Flags|Description|
|-----|-----------|
|`--path <dir>`|Remove the override for the specified directory instead of the current directory|
|`--nonexistent`|Remove all overrides pointing to directories that no longer exist|

```bash
cjv override unset
cjv override unset --path /path/to/project
cjv override unset --nonexistent
```

### `cjv override list`

List all directory overrides.

```text
cjv override list
```

For the toolchain resolution priority and override semantics, see [Targets and Overrides](concepts/targets-overrides.md).

---

## Configuration

### `cjv set`

Modify cjv settings (stored in `<CJV_HOME>/settings.toml`).

```text
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set gitcode-api-key <key>
cjv set home <path>
```

Subcommands:

|Subcommand|Value|Description|
|----------|-----|-----------|
|`auto-self-update`|`enable` / `disable` / `check`|Set the automatic self-update behavior; `check` only checks without updating|
|`auto-install`|`true` / `false`|Whether to automatically install the resolved toolchain in proxy mode when it is not yet installed|
|`default-host`|`<goos-goarch>`|Set the default host platform identifier (e.g. `linux-amd64`), used to resolve the download platform|
|`gitcode-api-key`|`<key>`|Set the GitCode API access token (required to query and download nightly builds); it is masked when displayed|
|`home`|`<path>`|Persist `CJV_HOME` to settings.toml; pass an empty string to clear this override; the `CJV_HOME` environment variable still takes precedence|

```bash
cjv set auto-self-update check
cjv set auto-install true
cjv set default-host linux-amd64
cjv set gitcode-api-key <your-token>
cjv set home /opt/cjv
```

See [Configuration](configuration.md) and [Environment Variables](environment-variables.md).

---

## Self-management

### `cjv self update`

Update cjv itself to the latest version, and refresh the proxy symlinks and the managed env scripts.

```text
cjv self update
```

```bash
cjv self update
```

### `cjv self uninstall`

Uninstall cjv itself along with all installed toolchains (removes the entire `<CJV_HOME>/` and cleans up the PATH configuration).

```text
cjv self uninstall [-y]
```

|Flags|Description|
|-----|-----------|
|`-y`, `--yes`|Skip the confirmation prompt|

An interactive terminal prompts for confirmation. In `--json` mode, `-y` must be supplied for it to proceed.

```bash
cjv self uninstall
cjv self uninstall -y
```

---

## Installation bootstrap

### `cjv init`

Interactively bootstrap the first-time installation: configure the data directory and PATH, and optionally install a default toolchain and components. Usually invoked by the install script, but can also be run manually.

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
```

|Flags|Description|
|-----|-----------|
|`-y`, `--yes`|Skip the interactive menu and install non-interactively with the default options|
|`--default-toolchain <name>`|The default toolchain to install (defaults to `lts`; use `none` to skip installing a toolchain)|
|`-c`, `--component <name>`|Components to install along with the default toolchain (repeatable or comma-separated)|
|`--no-modify-path`|Do not modify PATH|

When standard input is not a terminal (e.g. a `curl ... | sh` bootstrap), it automatically falls back to a non-interactive installation. This command does not support `--json`.

```bash
cjv init
cjv init -y --default-toolchain lts -c stdx,docs
cjv init -y --default-toolchain none --no-modify-path
```

For installation methods, see [Installing cjv](installation/index.md).
