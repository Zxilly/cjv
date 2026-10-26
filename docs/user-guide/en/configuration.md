# Configuration

cjv saves your user settings in `~/.cjv/settings.toml` by default. You can select another user settings file with `--config <path>` or `CJV_CONFIG_FILE`, where `--config` takes precedence. Setting `CJV_HOME` or `home` changes the data directory only and does not move `settings.toml`. This chapter describes all settings fields, inheritance rules, data directories, download caches, and file deduplication.

## Settings file and fields

An example `settings.toml` file looks like this:

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true

[overrides]
"/home/me/project-a" = "sts"
```

The table below lists all supported fields in `settings.toml`:

| Field | Default or meaning | How to change it |
| --- | --- | --- |
| `version` | Format version managed by cjv | Written and migrated automatically by cjv |
| `default_toolchain` | Global default toolchain name | Run `cjv default <name>`, or `cjv default none` to clear it |
| `auto_self_update` | `check` (print update notice); also accepts `enable` or `disable` | Run `cjv set auto-self-update <value>` |
| `auto_install` | `true` (install missing project requirements automatically) | Run `cjv set auto-install <true\|false>` |
| `home` | Data directory path; defaults to `~/.cjv` | Run `cjv set home <path>` |
| `default_host` | Default host platform; detected automatically | Run `cjv set default-host <goos-goarch>` |
| `manifest_url` | URL for the LTS and STS manifest; nightly uses `nightly.json` in the same directory | Edit `settings.toml` directly |
| `dist_server` | Base URL that serves both `versions.json` and `nightly.json` | Edit `settings.toml` directly |
| `link_mode` | File deduplication mode: `hardlink` or `copy`; defaults to `hardlink` | Edit `settings.toml` directly |
| `overrides` | Table that maps directory paths to toolchain names | Run `cjv override set` or `cjv override unset` |

### Automatic self-updates

The `auto_self_update` setting applies only when you run a full `cjv update` without toolchain names or a global `+name` selector:

- `enable` downloads and installs new cjv releases automatically.
- `check` checks for a new cjv release and prints a message without installing it.
- `disable` skips the cjv release check during `cjv update`.

Local development builds of cjv skip release checks automatically. You can always run `cjv self update` to update cjv manually.

### Data directory and host platform

When you run `cjv set home <path>`, cjv converts a relative path into an absolute path before saving it. Run `cjv set home ""` to reset the setting to the default `~/.cjv` directory.

The `default_host` field uses Go platform tuples such as `linux-amd64` or `darwin-arm64`. Note that this format differs from the `linux-x64` spelling used in [toolchain names](concepts/toolchains.md#host-platforms-and-cross-targets).

## Inheritance and precedence

### User settings vs. system fallback settings

User settings override [system fallback settings](enterprise/index.md#system-fallback-settings) field by field. Any field that you do not set in your user file continues to inherit the system fallback value. Explicit `false` values and empty strings `""` count as user values and override the fallback file.

When you run `cjv set`, cjv writes only the field that you changed and does not copy other inherited values into your user file. Delete a field from your user `settings.toml` to restore inheritance from the system fallback file.

If `settings.toml` contains unknown keys, cjv prints a warning and reads the valid keys. If the file has invalid TOML syntax or an unsupported `version` number, cjv stops with an error.

### Precedence for paths, sources, and flags

cjv resolves data directories, distribution servers, and temporary overrides in this order:

- **Data directory**: `CJV_HOME` environment variable, then `home` in merged settings, then `~/.cjv`.
- **Distribution server**: `CJV_DIST_SERVER` environment variable, then `dist_server` in merged settings, then `manifest_url`. Read [Internal distribution](enterprise/distribution-server.md) for server layout.
- **Settings file selection**: `--config <path>` replaces the user settings file only. System fallback settings, [project toolchain files](toolchain-file.md), and [environment variables](environment-variables.md) still apply.
- **Automatic installation**: `--no-auto-install` disables automatic installation for one CLI call, and `CJV_AUTO_INSTALL=0` or `1` overrides `auto_install` for the process tree. Neither option writes to `settings.toml`.

## Data directory layout

cjv organizes installed SDKs, components, caches, and snapshots under `<CJV_HOME>`:

```text
<CJV_HOME>/
  bin/              # cjv binary and SDK proxy commands (cjc, cjpm, ...)
  toolchains/<tc>/  # Installed host SDKs and cross target SDKs
  stdx/<tc>/        # Installed stdx dynamic/ and static/ libraries
  docs/<tc>/main/   # Offline main documentation
  docs/<tc>/stdx/   # Offline stdx documentation
  downloads/        # Temporary staging and resumable partial downloads
  cache/            # Cached manifests and checksum-verified archives
  history/<tc>/     # Complete toolchain group snapshots for rollback
```

Your `settings.toml` file lives inside `<CJV_HOME>` only when `<CJV_HOME>` uses the default `~/.cjv` path. Run `cjv show home` to check the active data directory and the rule that selected it.

## Download staging and reusable cache

### Resumable downloads

When cjv downloads an archive that has a SHA-256 hash, it keeps the partial file in `downloads/` if the network connection drops. The next command resumes the download and checks the SHA-256 hash of the complete archive.

When an installation finishes, cjv deletes the staging files for that operation. A full `cjv update` that succeeds across all toolchains also cleans up any leftover files in `downloads/`.

### Reusable offline cache

After an installation with a valid SHA-256 hash succeeds, cjv stores the manifest and the checked archive in `cache/`. cjv indexes cached archives by source URL and SHA-256 digest, and checks the digest again before every reuse. Archives that have no SHA-256 checksum are never saved in `cache/`.

When you pass `--offline` or set `CJV_OFFLINE=1`, cjv makes no network requests and reads from `cache/` only. Cached manifests contain the data from the last online fetch and might not list newer upstream releases. If a needed manifest or archive is missing from `cache/`, the offline command fails with an error.

Use these commands to check or clean the reusable cache:

```bash
cjv cache dir
cjv cache list
cjv cache prune --max-age 720h
cjv cache clean
```

Cleaning `cache/` does not delete any installed SDKs. Read [Automation and JSON interfaces](automation.md#ci-isolation-and-offline-caches) for CI caching workflows.

## SDK file reuse

By default, `link_mode = "hardlink"` reuses identical regular files across SDK installations that share the same release, platform, and SHA-256 digest. cjv keeps installation metadata and components separate. If the filesystem does not support hard links, cjv keeps independent file copies automatically.

> **CAUTION:** Hard-linked files share the same underlying disk blocks. Do not edit files inside `<CJV_HOME>/toolchains/` in place, because changes will affect other hard-linked SDK installations.

If you need to change SDK files in place for experiments, set `link_mode = "copy"` in `settings.toml` before installing the SDK. Changing `link_mode` applies to new installations only and does not split existing hard links. When `cjv update` runs, it publishes a new directory tree instead of editing shared files in place.
