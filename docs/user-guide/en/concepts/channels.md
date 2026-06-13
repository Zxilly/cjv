# Channels

A channel is cjv's name for a Cangjie SDK release stream. Each channel represents a continuously updated release line. When you install a channel, cjv resolves the channel's current latest version and installs it.

cjv supports three built-in channels:

|Channels|Meaning|Download source|Additional requirement|
|--------|-------|---------------|----------------------|
|`lts`|Long-term support release|Official version manifest|None|
|`sts`|Short-term support release|Official version manifest|None|
|`nightly`|Daily build (preview)|GitCode `nightly_build`|Requires `CJV_GITCODE_API_KEY`|

Channel names are case-insensitive; `LTS`, `Lts`, and `lts` are equivalent.

## Which channel to choose

`lts` is the long-term support release. It is relatively stable, iterates slowly, and suits production builds and projects sensitive to compatibility. If you are not sure which one to use, start with `lts`.

`sts` is the short-term support release. It updates faster than LTS and gives you new features sooner, but has a shorter maintenance window. It suits cases where you want to follow language evolution without using daily builds.

`nightly` is the daily build. It contains the latest but not yet stabilized changes, which may change or regress at any time. It suits trying things out, reproducing upstream behavior, or filing bugs against the SDK itself.

```bash
# Install the latest version of a channel
cjv install lts
cjv install sts
cjv install nightly
```

## Channels and version names

Passing a channel name directly to `cjv install` is equivalent to installing the channel's latest version. cjv first resolves the concrete version number, then stores it on disk as `<channel>-<version>`. For example, installing `lts` may produce an installed toolchain named `lts-1.0.5`.

You can also pin the version and skip resolving the latest:

```bash
# Install a specific version of a specific channel
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23

# Give only the bare version number; cjv searches LTS / STS to find which channel the version belongs to
cjv install 1.0.5
```

A bare version number (such as `1.0.5`) is looked up only in the LTS / STS version manifest; it belongs to whichever channel it matches, and nightly does not take part in bare version matching. For the full rules on toolchain naming, see [Toolchains](toolchains.md).

## Differences in download sources

All three channels install the Cangjie SDK; they differ in where the build artifacts are fetched from.

### LTS / STS: the official version manifest

The available versions, download URLs, and checksums for LTS and STS come from an officially maintained JSON version manifest. cjv has a default manifest URL built in; at install time it first fetches the manifest, then uses it to download the SDK archive for the corresponding platform and verify its SHA-256.

The manifest address can be overridden in `~/.cjv/settings.toml` through `manifest_url` (for example to switch to a mirror); leaving it empty restores the built-in default. See [Configuration](../configuration.md).

### nightly: the GitCode daily build repository

nightly does not go through the version manifest. cjv queries the latest release of the `Cangjie/nightly_build` repository via GitCode's release API, resolves the SDK version, and then downloads the build artifact for the corresponding platform from that repository's release assets.

GitCode's release API requires authentication, so installing or checking nightly requires a configured GitCode API access token. When it is not configured, the relevant commands fail with a prompt:

```text
GitCode API key is required to query nightly versions. Set it with: cjv set gitcode-api-key <your-token>
```

There are two ways to configure the token; the environment variable takes precedence over the persisted setting:

```bash
# Option 1: write it to settings (persisted in ~/.cjv/settings.toml)
cjv set gitcode-api-key <your-token>

# Option 2: provide it via an environment variable (higher precedence, suited to CI)
export CJV_GITCODE_API_KEY=<your-token>
```

For the full description of `CJV_GITCODE_API_KEY` see [Environment variables](../environment-variables.md), and for `cjv set` see [Configuration](../configuration.md).

## Channels and component sources

The download sources for components are likewise distinguished by channel. stdx, docs, and stdx-docs come from different release repositories under LTS / STS versus nightly:

|Components|LTS / STS source|nightly source|
|----------|----------------|--------------|
|`stdx`|`cangjie_stdx` release|`nightly_build` release|
|`docs`|`cangjie-docs-bundle` release|`nightly_build` release|
|`stdx-docs`|`cangjie_stdx` release|`nightly_build` release|

All components of a nightly toolchain are pulled from the `nightly_build` repository, so installing nightly components also depends on the GitCode API token configured earlier. For an explanation of the component mechanism itself, see [Components](components.md).

```bash
# Install components together with nightly (the components also come from the nightly_build source)
cjv install nightly -c stdx,docs
```

## Specifying a channel in the toolchain file

A project can declare its required channel in the `channel` field of `cangjie-sdk.toml`, so collaborators automatically use the same channel after getting the code:

```toml
[toolchain]
channel = "lts"
```

`channel` can be either a channel name (`lts` / `sts` / `nightly`) or a versioned toolchain name (such as `lts-1.0.5`). For the complete field semantics, see [Toolchain file](../toolchain-file.md).

## Checking for updates

`cjv check` queries whether updates are available for installed channel toolchains. The nightly check also calls the GitCode API, so if you have a nightly toolchain installed but no token configured, this step reports an error; the LTS / STS checks are unaffected.

```bash
cjv check
```
