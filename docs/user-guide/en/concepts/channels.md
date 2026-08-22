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

Passing a channel to `cjv install` installs its latest version and stores it as `<channel>-<version>`. You can also pin a concrete version:

```bash
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23
cjv install nightly-1.1.0-alpha.20260306010001

# A bare version is matched to its LTS / STS channel
cjv install 1.0.5
```

Concrete nightly versions use a channel-qualified name. See [Toolchains](toolchains.md) for the complete naming rules.

## Manifest distribution model

All three channels share one JSON version manifest. It records available versions, platform SDK URLs, component URLs, and checksums; cjv resolves the manifest before downloading its selected artifact.

The default manifest is maintained by [`cangjie-version-manifest`](https://github.com/Zxilly/cangjie-version-manifest). Formal release updates are merged through reviewed pull requests, while a scheduled job collects GitCode `Cangjie/nightly_build` Releases and commits nightly updates directly. Clients read the resulting static manifest.

Override the manifest address with `manifest_url` in `~/.cjv/settings.toml`. With `dist_server` or `CJV_DIST_SERVER`, cjv reads `<dist_server>/versions.json`, which suits centrally hosted enterprise distribution. See [Configuration](../configuration.md) and [Internal distribution source](../enterprise/distribution-server.md) for precedence and the deployment contract.

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

`cjv check` queries the manifest and compares installed channel toolchains with each channel's latest version:

```bash
cjv check
```
