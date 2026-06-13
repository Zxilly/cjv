# Configuration

cjv keeps its persisted settings in the TOML file `~/.cjv/settings.toml`. In daily use you do not need to edit it by hand; modify it with the `cjv set` subcommands, which validate values, write the file back, and print a confirmation.

This chapter covers all `cjv set` subcommands, the corresponding fields in `settings.toml`, and the overall layout of the `~/.cjv/` directory. Temporary overrides made at runtime via environment variables (such as `CJV_HOME` and `CJV_GITCODE_API_KEY`) are covered in [Environment Variables](environment-variables.md).

## settings.toml

The settings file always lives at `<user home>/.cjv/settings.toml`, where `<user home>` is the operating system's user home directory (such as `~`), not `CJV_HOME`.

This is intentional. The `home` path itself can be written into the file as a setting (see [`cjv set home`](#cjv-set-home)), and if the settings file followed `CJV_HOME`, it would create a chicken-and-egg dependency. So even if you move the data directory elsewhere, `settings.toml` stays under `~/.cjv/` in the user home directory.

A typical `settings.toml` looks roughly like this:

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true
gitcode_api_key = "your-token-here"

[overrides]
"/home/me/project-a" = "sts"
```

Field overview:

|Field|Type|Corresponding command|Description|
|-----|----|---------------------|-----------|
|`version`|int|(maintained automatically)|Settings file format version, written and migrated automatically by cjv|
|`default_toolchain`|string|`cjv default <toolchain>`|Default toolchain; see [Toolchains](concepts/toolchains.md)|
|`auto_self_update`|string|`cjv set auto-self-update`|Self-update behavior during `cjv update`: `enable` / `disable` / `check`|
|`auto_install`|bool|`cjv set auto-install`|Whether to automatically install missing toolchains in proxy mode|
|`home`|string|`cjv set home`|Persisted `CJV_HOME` data directory path|
|`default_host`|string|`cjv set default-host`|Default host platform identity (`goos-goarch` form)|
|`gitcode_api_key`|string|`cjv set gitcode-api-key`|GitCode API access token, required for nightly builds|
|`overrides`|table|`cjv override`|Directory-to-toolchain override mapping; see [Targets and Overrides](concepts/targets-overrides.md)|

 >
 > Unrecognized keys in the file (for example, typos) are reported with a warn-level log message but do not prevent cjv from starting. Setting `version` to a value higher than the current binary supports does cause an error.

## cjv set

`cjv set` modifies a single setting in `settings.toml`. It writes to disk only when the new value differs from the current one, and prints `Setting '<key>' updated to '<value>'`.

### cjv set auto-self-update

Controls whether `cjv update` also self-updates the cjv binary after updating toolchains.

```bash
# Automatically download and install the latest version of cjv
cjv set auto-self-update enable

# Fully disable self-update (not even a prompt is printed)
cjv set auto-self-update disable

# Default: only notify when a new version is available, do not install automatically
cjv set auto-self-update check
```

The three values mean the following. `enable` automatically upgrades cjv to the latest version after `cjv update` finishes, and refreshes the proxy symlinks. `disable` skips the self-update logic entirely. `check` (the default) does not upgrade automatically; it only prints the current cjv version when the update finishes, leaving it to you to decide whether to run `cjv self update`.

Regardless of this setting, you can upgrade cjv manually at any time with `cjv self update`.

### cjv set auto-install

Controls whether, in [proxy mode](concepts/proxies.md), a resolved toolchain that is not yet installed is installed automatically. Enabled by default (`true`).

```bash
# Default: when you invoke cjc / cjpm directly, a missing toolchain is installed automatically
cjv set auto-install true

# Disabled: error out when a toolchain is missing, instead of installing it automatically
cjv set auto-install false
```

When enabled, running `cjc`, `cjpm`, or other SDK tools directly will install the resolved toolchain first if it is not present, then proxy the call. The [components](concepts/components.md) and [targets](cross-compilation.md) declared in `cangjie-sdk.toml` apply the same way: with `auto-install` enabled, proxy execution fills in the missing components and target SDKs as needed.

### cjv set gitcode-api-key

Sets the GitCode API access token. Querying and downloading nightly toolchains and their components requires it; LTS and STS do not.

```bash
cjv set gitcode-api-key <your-gitcode-api-key>
```

For security, the command masks the token as `********` when echoing it, so it is not leaked in terminal scrollback, CI logs, or screen sharing. The token itself is stored in plaintext in `settings.toml`.

You can also provide the token temporarily with the `CJV_GITCODE_API_KEY` environment variable. It takes priority over the persisted value in `settings.toml`, and is not written back to the file, which suits injecting the credential in CI or deployment environments without persisting it. See [Environment variables](environment-variables.md).

### cjv set home

Persists the `CJV_HOME` data directory path to `settings.toml`. A relative path passed in is converted to an absolute path before being stored.

```bash
# Persist the data directory to a specific location
cjv set home /opt/cjv-data

# Pass an empty string to clear this override and restore the default ~/.cjv
cjv set home ""
```

The `CJV_HOME` environment variable always takes priority over this setting. Even if `home` is persisted in `settings.toml`, the `CJV_HOME` environment variable still wins when set in the shell. The `settings.toml` file itself is unaffected and always stays in `~/.cjv/`.

### cjv set default-host

Sets the default host platform identifier (in `goos-goarch` form, such as `linux-amd64`). You generally do not need to set this manually, as cjv detects the current host platform automatically; it is only needed when automatic detection does not match your expectations and you need to specify it explicitly.

```bash
cjv set default-host linux-amd64
```

The value must be a valid platform identifier that cjv recognizes, otherwise the command will error out.

## ~/.cjv Directory Structure

All of cjv's data is kept under `CJV_HOME` (`~/.cjv` by default), with each subdirectory holding a decoupled responsibility:

```text
~/.cjv/
  bin/            # proxy symlinks and the cjv binary
  toolchains/     # installed SDK toolchains (the SDK proper only)
    <tc>/
      .cjv/components/         # component manifests maintained by cjv
  stdx/           # stdx component (split per toolchain; paths exposed via CANGJIE_STDX_PATH_*)
    <tc>/
      dynamic/
      static/
  docs/           # offline docs (decoupled from toolchains; docs and stdx-docs each own a subdir)
    <tc>/
      main/                    # docs component (dev-guide / libs/std / tools entry)
      stdx/                    # stdx-docs component (libs_stdx entry)
  downloads/      # download staging (cleared after a successful install; only for resume)
  settings.toml   # user settings
```

`bin/` holds the cjv binary itself, along with the proxy symlinks for SDK tools such as `cjc` and `cjpm`, created by cjv when a toolchain is installed. Once this directory is added to `PATH`, calling `cjc` directly is transparently proxied to the active toolchain. See [Proxies](concepts/proxies.md).

`toolchains/<tc>/` is the SDK itself for each installed toolchain. The subdirectory `.cjv/components/` holds the manifests of the components installed for that toolchain, maintained by cjv.

`stdx/<tc>/` holds the `stdx` component per toolchain, split into `dynamic/` and `static/`. During proxying or in the runtime environment, `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` are injected automatically to point at these two directories. See [Components](concepts/components.md).

`docs/<tc>/` is offline documentation, decoupled from the toolchain directory. `main/` holds the `docs` component (dev-guide, libs/std, tools), and `stdx/` holds the `stdx-docs` component. Open them in a browser with `cjv doc`.

`downloads/` is the download staging area, cleared after a successful install and kept only when an install is interrupted, to allow recovery. `settings.toml` is the user settings file described in this chapter.

 >
 > `cjv toolchain uninstall <tc>` also cleans up `stdx/<tc>/` and `docs/<tc>/`, leaving no orphaned component data behind.

If a custom `CJV_HOME` is set or persisted, the `bin/`, `toolchains/`, `stdx/`, `docs/`, and `downloads/` directories above all fall under the new path; only `settings.toml` always stays in the user's home directory at `~/.cjv/` (see [settings.toml](#settingstoml) above).
