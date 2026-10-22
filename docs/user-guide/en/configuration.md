# Configuration

User settings default to `~/.cjv/settings.toml`. Select another file with `--config <path>` or `CJV_CONFIG_FILE`; the command-line option takes precedence. `CJV_HOME` and `home` only change the data directory.

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true

[overrides]
"/home/me/project-a" = "sts"
```

## Settings

| Field | Default or purpose | How to change it |
| --- | --- | --- |
| `version` | Settings format version maintained by cjv | Written and migrated automatically |
| `default_toolchain` | Default toolchain | `cjv default <name>`; `none` clears it |
| `auto_self_update` | `check`: report updates; also accepts `enable`, `disable` | `cjv set auto-self-update <value>` |
| `auto_install` | `true`: install missing requirements of the selected toolchain | `cjv set auto-install <true\|false>` |
| `home` | Data directory; defaults to `~/.cjv` | `cjv set home <path>` |
| `default_host` | Default host platform; detected automatically | `cjv set default-host <goos-goarch>` |
| `manifest_url` | Default LTS/STS manifest; nightly uses its sibling `nightly.json` | Edit settings |
| `dist_server` | Root serving both `versions.json` and `nightly.json` | Edit settings |
| `link_mode` | `hardlink` or `copy`; defaults to `hardlink` | Edit settings |
| `overrides` | Directory-to-toolchain mapping | `cjv override` |

## Automatic updates

`auto_self_update` applies only to a full `cjv update` without names or a global `+name`. `enable` updates cjv, `check` reports available updates, and `disable` skips the check. Development builds skip release checks. `cjv self update` remains available for manual use.

## Data directory and host platform

`cjv set home` converts relative paths to absolute paths. `cjv set home ""` explicitly selects the default data directory. `default_host` uses Go platform names such as `linux-amd64`, unlike the `linux-x64` spelling in toolchain names.

## Inheritance and temporary overrides

User settings override [system fallback settings](enterprise/index.md) field by field. Unset fields continue to inherit; `false` and empty strings count as explicit values. `cjv set` saves only the selected field without persisting other inherited values. Remove a field to restore inheritance.

Data directory precedence is `CJV_HOME`, then `home`, then `~/.cjv`. Distribution precedence is `CJV_DIST_SERVER`, then `dist_server` in merged settings, then `manifest_url`. Environment overrides are not persisted by `cjv set`.

Unknown keys produce warnings. Invalid settings and unsupported future format versions cause errors.

`--config` replaces only the user settings file. System fallback settings, project toolchain files, and environment variables still apply. `--no-auto-install` disables automatic installation for this invocation; `CJV_AUTO_INSTALL=0` or `1` temporarily overrides `auto_install`. These options do not modify settings and are passed to child processes started through cjv.

## Data layout

```text
<CJV_HOME>/
  bin/              # cjv and SDK command entry points
  toolchains/<tc>/  # SDK
  stdx/<tc>/        # dynamic/ and static/
  docs/<tc>/main/   # Main documentation
  docs/<tc>/stdx/   # Extension library documentation
  downloads/        # Installation staging and resumable partial downloads
  cache/            # Manifests and verified archives
  history/<tc>/    # Complete group snapshots from tracking-channel updates
```

The settings file shares this location only when the default data directory is used. `cjv show home` reports the effective data directory and its source.

### Download cache

Downloads with SHA-256 retain resumable files after failure or interruption. A later download verifies the complete archive. Successful operations clean up their own downloads; a fully successful update of all toolchains also cleans leftover downloads.

`cache/` separately retains manifests and verified archives with SHA-256 after successful installation. Archives are keyed by source URL and digest and verified again on reuse. `--offline` or `CJV_OFFLINE=1` prevents cjv from making network requests; missing metadata or archives cause an error. Cached metadata reflects the last successful fetch and may not include the latest release. Archives without checksums are not retained in the reusable cache.

Use `cjv cache dir` to find the cache, `cjv cache list` to inspect it, `cjv cache clean` to clear it, or `cjv cache prune --max-age 720h` to remove entries older than the given duration. Removing cached files does not remove installed SDKs. See [automation](automation.md) for CI examples.

## SDK file reuse

The default `link_mode = "hardlink"` tries to reuse identical regular files for SDKs with the same release, platform, and SHA-256. Metadata and components remain separate. If hard links are unavailable, the copied files are kept.

Hard links share content, so manually editing an SDK file in place can affect other installations. Use `link_mode = "copy"` for independent copies. This affects future installations and does not split existing hard links. cjv updates publish replacement directories without editing shared files in place.
