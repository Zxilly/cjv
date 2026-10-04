# Channels

A channel is cjv's name for a Cangjie SDK release stream. Each channel represents a continuously updated release line. When you install a channel, cjv resolves its current latest version from the version manifest and installs it.

| Channel | Meaning | Metadata source |
| --- | --- | --- |
| `lts` | Long-term support release | Version manifest |
| `sts` | Short-term support release | Version manifest |
| `nightly` | Daily build (preview) | Version manifest |

Channel names are case-insensitive; `LTS`, `Lts`, and `lts` are equivalent.

## Which channel to choose

`lts` is relatively stable and maintained longer, which suits production builds and compatibility-sensitive projects. `sts` updates faster and provides new features sooner. `nightly` contains the latest changes before stabilization, which suits experimentation, reproducing upstream behavior, and reporting SDK bugs.

```bash
cjv install lts
cjv install sts
cjv install nightly
```

## Channels and version names

Passing a channel to `cjv install` installs its latest version in an independent directory such as `toolchains/sts`. Fixed releases have separate directories such as `toolchains/sts-1.2.0`, with their own component choices. You can also pin a concrete version:

```bash
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23
cjv install nightly-1.1.0-alpha.20260306010001

# A bare version is matched to its LTS / STS channel
cjv install 1.0.5
```

Concrete nightly versions use a channel-qualified name. See [Toolchains](toolchains.md) for the complete naming rules.

## Manifest distribution model

Release channels and nightly use separate static manifests: `versions.json` records LTS/STS, while `nightly.json` records nightly. Both contain available versions, platform SDK URLs, component URLs, and checksums. cjv loads the file selected by the requested channel, so ordinary LTS/STS operations carry no nightly history cost.

The default files are maintained by [`cangjie-version-manifest`](https://github.com/Zxilly/cangjie-version-manifest). Formal release updates modify `versions.json` through reviewed pull requests, while a scheduled job collects GitCode `Cangjie/nightly_build` Releases and commits `nightly.json` directly.

`manifest_url` points to the release-channel file and derives sibling `nightly.json`. With `dist_server` or `CJV_DIST_SERVER`, both files live under the distribution root. See [Configuration](../configuration.md) and [Internal distribution source](../enterprise/distribution-server.md) for precedence and the deployment contract.

## Channels and components

The manifest records the effective URL for stdx, docs, and stdx-docs. The default manifest collects these upstream sources:

| Component | LTS / STS source | nightly source |
| --- | --- | --- |
| `stdx` | `cangjie_stdx` release | `nightly_build` release |
| `docs` | `cangjie-docs-bundle` release | `nightly_build` release |
| `stdx-docs` | `cangjie_stdx` release | `nightly_build` release |

Components for all three channels use the URLs declared by the manifest and may carry SHA-256 checksums. See [Components](components.md) for the component mechanism.

```bash
cjv install nightly -c stdx,docs
```

## Specifying a channel in the toolchain file

A project can declare its channel in `cangjie-sdk.toml` so collaborators use the same toolchain selection:

```toml
[toolchain]
channel = "lts"
```

`channel` accepts either a channel name or a versioned toolchain name. See [Toolchain file](../toolchain-file.md) for the complete field semantics.

## Checking for updates

Installing a channel creates an independent tracking directory such as `toolchains/sts`. Its release record lives in `.cjv/toolchain.toml` inside that directory. Both `cjv install sts` and `cjv update sts` advance it to the latest STS release; `update sts` also installs a missing channel. Plain `cjv update` updates installed tracking channels and their cross-compilation SDKs. Selected components are preserved, and obsolete SDK, stdx and documentation content is removed after a successful update.

An explicit installation such as `cjv install sts-1.2.0` creates a separate fixed directory. Plain `update` does not replace it. Channels and fixed versions own independent SDK, stdx, documentation and component records. `cjv uninstall sts` removes the channel and its tracking cross SDKs, while `cjv uninstall sts-1.2.0` removes only the fixed installation.

The first channel installation records a channel name as the default. Defaults, directory overrides and project files selecting `sts` follow its installed channel release; explicit versions remain fixed. Updates do not rewrite project files.

The first use of the new layout performs an offline migration. All old version directories remain fixed installations. Each channel's previously implicit highest installed release is copied into an independent channel directory, including its documentation, stdx and component records. Cross SDKs migrate at the selected host's exact version; when that target release is missing, existing fixed target installations are retained. Defaults, overrides and project files are unchanged. A completed migration never recreates a deliberately uninstalled channel. Explicitly uninstall old versions when no longer needed. Identical SDK payloads can use hardlinks to reduce disk usage; metadata remains independent. See [configuration](../configuration.md). `cjv check` reports available updates for tracking channels only.

`cjv check` queries the manifest and compares installed channel toolchains with each channel's latest version:

```bash
cjv check
```
