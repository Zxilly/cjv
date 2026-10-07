# Toolchains and versions

A toolchain is a Cangjie SDK containing the compiler, build tools, and runtime libraries. cjv stores multiple SDKs under `<CJV_HOME>/toolchains/`. Components and offline documentation are stored separately.

## Channels and names

| Name | Meaning |
| --- | --- |
| `lts` | Track the long-term support channel |
| `sts` | Track the short-term support channel |
| `nightly` | Track development builds available in the published manifest |
| `lts-1.0.5` | Pin a specific version within a channel |
| `1.0.5` | Omit the channel and let cjv find the matching version |
| `sts-1.2`, `1.2` | Select the latest stable patch in that minor version |
| `nightly-2026-10-04` | Select the latest published nightly for that date |
| `my-sdk` | A custom toolchain created with `cjv toolchain link` |

Versions and dates in this table illustrate the syntax. Query your distribution for available versions:

```bash
cjv toolchain list-remote --channel lts
cjv toolchain list-remote --channel nightly --limit 10
```

Channel names are case-insensitive. Custom names must not conflict with official names, contain path separators, start with `+`, or be empty, `.` or `..`.

## Tracked channels and pinned versions

```bash
cjv install lts
cjv install lts-1.0.5
```

These are separate installations. `cjv update lts` updates the tracked channel and leaves `lts-1.0.5` unchanged. Their components are also managed separately. Running `cjc` uses the installed release; upgrading requires `install` or `update`.

Minor-version and date selectors install the resolved concrete version. Put the full version name in [cangjie-sdk.toml](../toolchain-file.md) when a team needs the same release.

Nightly installation considers required components and cross SDKs when choosing a compatible release from published history. It does not downgrade by default; `--allow-downgrade` permits an older compatible nightly. See the [command reference](../command-reference.md) for other installation policies.

## Host platforms

Names such as `sts-linux-x64` and `sts-1.2.0-darwin-arm64` select a host platform. Hosts are installed separately. An explicit name for the default platform identifies the same installation as the name without a platform. Installing another platform's SDK does not make its programs executable on your machine.

[Target management](../cross-compilation.md) attaches cross SDKs to a host toolchain. A cross SDK cannot be the default or active toolchain.

## Custom SDKs

```bash
cjv toolchain link my-sdk /path/to/local/sdk
cjv default my-sdk
```

A directory link refers directly to an SDK you maintain. Uninstalling removes the link and preserves the source directory. Use [archive installation](../install-from-url.md) for a copy managed by cjv or to attach local stdx. Custom toolchains are not updated by `cjv update` and have no official components available through `component add`.

Use `cjv toolchain list` to inspect installations and `cjv show active` to see the current selection and its source. Selection rules are covered in [project toolchains](../toolchain-file.md).
