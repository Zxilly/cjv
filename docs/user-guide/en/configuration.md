# Configuration

`cjv` stores your personal user settings in a [TOML](https://toml.io/) file at `~/.cjv/settings.toml` (or `%USERPROFILE%\.cjv\settings.toml` on Windows) by default. You can select an alternate user settings file for a command or session with `--config <path>` or `CJV_CONFIG_FILE`, where the command-line `--config` flag takes precedence. Note that `CJV_HOME` and the `home` setting only relocate the data directory (`toolchains/`, `cache/`, etc.) and do not move `settings.toml`.

A typical `settings.toml` file looks like this:

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true

[overrides]
"/home/me/project-a" = "sts"
```

## Settings

Most day-to-day settings can be updated through `cjv` subcommands (`cjv default`, `cjv set`, `cjv override`), while advanced distribution settings can be edited in `settings.toml` directly:

| Field | Default or purpose | How to change it |
| --- | --- | --- |
| `version` | Settings format version maintained by `cjv` | Written and migrated automatically |
| `default_toolchain` | Global default toolchain | `cjv default <name>`; pass `none` to clear it |
| `auto_self_update` | `check` (report updates); also accepts `enable` or `disable` | `cjv set auto-self-update <value>` |
| `auto_install` | `true` (install missing requirements of the selected toolchain) | `cjv set auto-install <true\|false>` |
| `home` | Data directory; defaults to `~/.cjv` | `cjv set home <path>` |
| `default_host` | Default host platform; detected automatically | `cjv set default-host <goos-goarch>` |
| `manifest_url` | Default LTS/STS manifest URL; `nightly` uses its sibling `nightly.json` | Edit `settings.toml` |
| `dist_server` | Base URL serving both `versions.json` and `nightly.json` | Edit `settings.toml` |
| `link_mode` | `hardlink` or `copy`; defaults to `hardlink` | Edit `settings.toml` |
| `overrides` | Per-directory toolchain mapping table | `cjv override set` / `cjv override unset` |

## Automatic updates

The `auto_self_update` setting controls how `cjv` handles updates to the `cjv` binary itself when you run a full `cjv update` (that is, without specifying individual toolchain names or a global `+name` selector):

- **`check`** *(default)* — Checks whether a newer release of `cjv` is available and prints a notice, without installing it automatically.
- **`enable`** — Automatically downloads and installs any available update to `cjv` during `cjv update`.
- **`disable`** — Skips checking for or installing `cjv` self-updates during `cjv update`.

Local development builds of `cjv` (which report their version as `dev`) automatically skip release checks. Regardless of `auto_self_update`, you can always update `cjv` manually by running `cjv self update`, or suppress the check for a single run with `cjv update --no-self-update`.

## Data directory and host platform

When you run `cjv set home <path>`, `cjv` resolves relative paths to an absolute path before writing `settings.toml`. Running `cjv set home ""` explicitly resets the data directory back to the default `~/.cjv`.

The `default_host` setting uses Go platform tuples such as `linux-amd64` or `darwin-arm64`. Note that this differs slightly from the `linux-x64` spelling used in [toolchain names](concepts/toolchains.md#host-platforms).

## Inheritance and temporary overrides

On managed machines, user settings in `~/.cjv/settings.toml` override [system fallback settings](enterprise/index.md#system-fallback-settings) (`/etc/cjv/settings.toml` on Unix, `C:\ProgramData\cjv\settings.toml` on Windows) field by field. Any field that is not present in your user settings file continues to inherit its value from the system fallback file, whereas explicit `false` values and empty strings `""` in your user file count as explicit user overrides.

When you run `cjv set`, `cjv` writes only the specific field you modified so that other unset fields continue to inherit from the system fallback file. To restore system inheritance for a field, delete that key from your user `settings.toml`.

Here is how `cjv` resolves precedence across environment variables and configuration files:

- **Data directory precedence**: `CJV_HOME` environment variable, then `home` in merged settings, then `~/.cjv`.
- **Distribution source precedence**: `CJV_DIST_SERVER` environment variable, then `dist_server` in merged settings, then `manifest_url`. Environment variable overrides are temporary and are never persisted to disk when you run `cjv set`.
- **Custom config file (`--config`)**: Passing `--config <path>` (or setting `CJV_CONFIG_FILE`) replaces only the user settings file; system fallback settings, [project toolchain files](toolchain-file.md), and [environment variables](environment-variables.md) still apply.
- **Per-invocation auto-install overrides**: Passing `--no-auto-install` disables automatic installation for that invocation, and `CJV_AUTO_INSTALL=0` or `1` temporarily overrides `auto_install` for the current process tree without modifying `settings.toml`.

Unrecognized keys in `settings.toml` produce a warning on `stderr`, while invalid TOML syntax or an unsupported future `version` number causes `cjv` to exit with an error.

## Data layout

`cjv` organizes all installed toolchains, components, caches, and rollback snapshots under `<CJV_HOME>`:

```text
<CJV_HOME>/
  bin/              # cjv binary and SDK proxy entry points (cjc, cjpm, ...)
  toolchains/<tc>/  # Installed host SDKs and cross-compilation target SDKs
  stdx/<tc>/        # Installed stdx dynamic/ and static/ libraries
  docs/<tc>/main/   # Offline main documentation
  docs/<tc>/stdx/   # Offline extension library (stdx) documentation
  downloads/        # Installation staging and resumable partial downloads
  cache/            # Cached manifests and SHA-256-verified archives
  history/<tc>/     # Complete toolchain group snapshots from channel updates
```

Your `settings.toml` file lives inside `<CJV_HOME>` only when `<CJV_HOME>` uses the default `~/.cjv` path. You can run `cjv show home` at any time to see the effective data directory and which configuration source selected it.

### Download cache

When `cjv` downloads an archive that has an expected SHA-256 checksum, partial downloads are kept in `downloads/` if a network failure or interruption occurs so the next command can resume where it left off and verify the complete file. Successful operations clean up their own staging files, and a full `cjv update` across all toolchains that succeeds completely also cleans up any leftover files in `downloads/`.

Separately, `<CJV_HOME>/cache/` retains downloaded version manifests and SHA-256-verified archives after installation succeeds so they can be reused later. Cached archives are keyed by source URL and SHA-256 digest and are re-verified every time they are reused (archives without a checksum are never stored in the reusable cache).

When you pass `--offline` or set `CJV_OFFLINE=1`, `cjv` makes no network requests and relies exclusively on local installations and `cache/`. Keep in mind that cached manifests reflect the last successful online fetch and may not include newer upstream releases; if a needed manifest or archive is absent from `cache/`, the offline command fails with an error.

You can manage the reusable cache with the `cjv cache` subcommands:

- `cjv cache dir` — Print the path to the cache directory.
- `cjv cache list` — Inspect cached manifests and archives.
- `cjv cache prune --max-age 720h` — Remove cached entries older than the given duration.
- `cjv cache clean` — Remove all cached manifests and archives.

Cleaning the cache never deletes your installed SDKs. See [Automation and JSON interfaces](automation.md#configuration-and-network-access) for CI caching workflows.

## SDK file reuse

By default, `link_mode = "hardlink"` deduplicates identical regular files across SDK installations that share the same release, platform, and SHA-256 digest. Installation metadata and components remain separate, and if hard links are not supported by the underlying filesystem, `cjv` automatically falls back to keeping independent file copies.

Because hard-linked files share the same underlying disk content, manually editing a file in place inside `<CJV_HOME>/toolchains/` can affect other hard-linked SDK installations. If you need completely independent file copies so you can modify SDK files in place, set `link_mode = "copy"` in `settings.toml` (this setting applies to future installations and does not split existing hard links). When `cjv update` updates a toolchain, it always publishes a replacement directory tree rather than modifying shared files in place.
