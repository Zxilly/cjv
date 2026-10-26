# Toolchains and versions

A toolchain is a Cangjie SDK that contains the compiler, build tools, and runtime libraries. cjv stores installed SDKs under `<CJV_HOME>/toolchains/`. It stores [components](components.md) and offline documentation in separate directories. This chapter explains channel names, version selectors, host platforms, and custom toolchains.

## Channel and version names

Use these name formats to select a release channel, a specific version, or a custom SDK:

| Name format | Meaning |
| --- | --- |
| `lts` | Track the long-term support channel |
| `sts` | Track the short-term support channel |
| `nightly` | Track development builds from the published manifest |
| `lts-1.0.5` | Pin a specific release in a channel |
| `1.0.5` | Omit the channel and let cjv find the matching release |
| `sts-1.2`, `1.2` | Select the latest stable patch in that minor version |
| `nightly-2026-10-04` | Select the latest published nightly build for that date |
| `my-sdk` | Select a custom toolchain created with `cjv toolchain link` |

The version numbers and dates in the table show the syntax only. Run these commands to query the versions in your distribution source:

```bash
cjv toolchain list-remote --channel lts
cjv toolchain list-remote --channel nightly --limit 10
```

Channel names are not case-sensitive. A custom toolchain name must not match an official name, contain path separators, or start with `+`. It also cannot be empty, `.`, or `..`.

## Tracked channels and pinned versions

### Separate installations

A tracked channel and a pinned version are separate installations:

```bash
cjv install lts
cjv install lts-1.0.5
```

When you run `cjv update lts`, cjv updates the tracked `lts` installation and leaves `lts-1.0.5` unchanged. cjv also manages the components of each installation separately. When you run `cjc`, cjv uses the installed release and does not upgrade it until you run `cjv install` or `cjv update`.

### Version resolution

Minor-version selectors such as `sts-1.2` and date selectors such as `nightly-2026-10-04` resolve to a concrete version during installation. If your team must use the exact same release, write the full version name in [cangjie-sdk.toml](../toolchain-file.md) or run `cjv pin`.

When you install or update `nightly`, cjv checks the components and cross targets that you need. It searches the published history for a compatible build. By default, cjv does not downgrade an installed nightly toolchain. Pass `--allow-downgrade` if you want to allow an older compatible nightly build. Read [Command reference](../command-reference.md#installation-and-update-policies) for all update policies.

## Host platforms and cross targets

You can append a host platform suffix to a toolchain name, such as `sts-linux-x64` or `sts-1.2.0-darwin-arm64`. cjv installs each host platform separately. If you specify the default host platform explicitly, it refers to the same installation as the name without a platform suffix.

Installing an SDK for another operating system does not make its binaries run on your machine. To compile code for another target platform on your current host, attach a cross SDK to your host toolchain. A cross SDK cannot be the default or active toolchain on its own. Read [Cross-compilation](../cross-compilation.md) for target management.

## Custom SDKs

You can register a local SDK directory as a named custom toolchain:

```bash
cjv toolchain link my-sdk /path/to/local/sdk
cjv default my-sdk
```

A directory link points directly to an SDK directory that you maintain. When you run `cjv uninstall my-sdk`, cjv removes the link and keeps your source directory intact.

If you want cjv to manage a copy of the SDK or attach local `stdx` libraries, install the SDK from a local archive or URL instead. The `cjv update` command does not update custom toolchains, and `cjv component add` cannot download official components for them. Read [Custom SDKs](../install-from-url.md) for archive and directory rules.

## Inspect local toolchains

Run `cjv toolchain list` to see all installed toolchains. Run `cjv show active` to see the active toolchain and the rule that selected it. Read [Project toolchains](../toolchain-file.md) next to learn how cjv selects a toolchain for each directory.
