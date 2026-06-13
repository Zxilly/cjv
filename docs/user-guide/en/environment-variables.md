# Environment Variables

Some of cjv's behavior can be adjusted through environment variables. Environment variables are suited for temporary overrides, injecting credentials in CI, and changing default behavior without writing to `settings.toml`.

The table below lists the common user-facing variables. Unless noted otherwise, all variables are read when a command runs, so you can set them temporarily for individual invocations:

```bash
CJV_LOG=debug cjv install lts
```

## Common variables

|Variable|Default|Description|
|--------|-------|-----------|
|`CJV_HOME`|`~/.cjv`|Overrides cjv's home directory (the data root). It must be an absolute path, otherwise cjv exits with an error. This variable takes priority over the `home` setting persisted in `settings.toml`.|
|`CJV_TOOLCHAIN`|None|Force the active toolchain, overriding all other resolution methods (directory override, toolchain file, default toolchain).|
|`CJV_LOG`|`warn`|Log level; one of `debug`, `info`, `warn`, `error`. Unrecognized values are treated as `warn`. Logs are written to standard error (stderr).|
|`CJV_MAX_RETRIES`|`3`|Maximum number of retries after a single download failure. The value must be a non-negative integer; invalid values are ignored and fall back to the default.|
|`CJV_DOWNLOAD_TIMEOUT`|`180`|Timeout for HTTP downloads, in seconds. The value must be a positive integer; invalid values are ignored and fall back to the default.|
|`CJV_GITCODE_API_KEY`|None|GitCode API access token, used to query and download nightly toolchains and components. It takes priority over the token persisted in `settings.toml`, and is not written back to disk.|
|`CJV_NO_PATH_SETUP`|None|Set to `1` to skip the automatic `PATH` configuration on first install (useful for CI environments and integration tests). Any other value (including unset) has no effect.|
|`CANGJIE_STDX_PATH_DYNAMIC`|Injected by cjv|Points to `<CJV_HOME>/stdx/<tc>/dynamic`, injected only when the corresponding toolchain has the `stdx` component installed. You normally do not need to set it manually.|
|`CANGJIE_STDX_PATH_STATIC`|Injected by cjv|Points to `<CJV_HOME>/stdx/<tc>/static`, injected only when the corresponding toolchain has the `stdx` component installed. You normally do not need to set it manually.|

## Details

### `CJV_HOME`

`CJV_HOME` determines the root directory where cjv stores toolchains, components, documentation, the download cache, and the settings file. It defaults to `~/.cjv` under the user's home directory.

It must be an absolute path. A relative path would point to different locations as the working directory changes, so calling cjv from different directories would see different sets of installations. cjv therefore rejects relative paths and exits with an error.

The home directory is resolved in the following order (highest to lowest):

1. The `CJV_HOME` environment variable
1. The `home` value persisted in `settings.toml` (written via `cjv set home <path>`)
1. The default value `~/.cjv`

To persist the home directory instead of setting the environment variable each time, see [Configuration](configuration.md). `cjv show home` prints the home directory currently in effect and its source.

### `CJV_TOOLCHAIN`

`CJV_TOOLCHAIN` sits at the very top of the toolchain resolution priority, overriding directory overrides, the toolchain file (`cangjie-sdk.toml`), and the default toolchain. It is commonly used to temporarily switch toolchains for a single command:

```bash
CJV_TOOLCHAIN=nightly cjc --version
```

For the full resolution order, see [Targets and Overrides](concepts/targets-overrides.md).

### `CJV_LOG`

Setting the log level to `debug` lets you observe the details of downloads, resolution, and proxy execution, which helps with troubleshooting:

```bash
CJV_LOG=debug cjv install lts
```

### `CJV_MAX_RETRIES` and `CJV_DOWNLOAD_TIMEOUT`

These two variables let you tune download behavior when the network is unstable or a mirror is slow:

```bash
# Increase the retry count and extend the timeout (for slow networks)
CJV_MAX_RETRIES=5 CJV_DOWNLOAD_TIMEOUT=600 cjv install sts
```

`CJV_MAX_RETRIES` is the number of retries after a failure, and `CJV_DOWNLOAD_TIMEOUT` is in seconds. Invalid values for either are ignored and fall back to the default.

### `CJV_GITCODE_API_KEY`

Querying and downloading nightly toolchains and their components requires a GitCode API token. Setting this environment variable provides the credential without writing the token into `settings.toml`, which suits CI and deployment scenarios:

```bash
CJV_GITCODE_API_KEY=your_token cjv install nightly
```

This environment variable takes priority over the persisted setting, and is not written back to disk. To save the token persistently, use:

```bash
cjv set gitcode-api-key <key>
```

For more about the nightly channel and GitCode, see [Channels](concepts/channels.md).

### `CJV_NO_PATH_SETUP`

When you first install a toolchain, cjv adds its own `bin` directory to `PATH` so that proxied commands (such as `cjc` and `cjpm`) become immediately available. In CI environments or integration tests, automatically modifying `PATH` is often unnecessary, so you can set this variable to `1` to skip it:

```bash
CJV_NO_PATH_SETUP=1 cjv install lts
```

It is skipped only when the value is exactly `1`; other values have no effect.

### `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC`

These two variables are injected automatically by cjv during proxy execution and runtime environment setup (`cjv exec` / `cjv envsetup`), pointing respectively to the extracted `dynamic` and `static` directories of the `stdx` component. They are injected only when the current toolchain has the `stdx` component installed.

You normally do not need to set them manually; cjv makes sure the Cangjie compiler and build tools can find the extension libraries. For installation and directory layout of the `stdx` component, see [Components](concepts/components.md); for how the runtime environment is set up, see [Runtime environment](runtime-environment.md).

## Advanced and internal variables

The following variables target special scenarios or are used internally by cjv, and ordinary users usually need not be concerned with them.

|Variable|Description|
|--------|-----------|
|`CJV_LANG`|Override the interface language (such as `zh`, `en`, `ja`). When unset, it follows the system locale setting.|
|`CJV_ALLOW_INSECURE_MANIFEST`|When set to `1`, allows fetching the toolchain manifest over plaintext HTTP from non-loopback hosts. HTTPS is required by default, because the manifest carries both download URLs and their checksums. Use this only with trusted internal mirrors; see [Installing a Toolchain from a URL](install-from-url.md).|
|`CJV_FALLBACK_SETTINGS`|Specifies the path to a system-level fallback settings file, used to provide defaults beyond the user settings (such as an enterprise mirror configuration). When unset, the platform default location is used.|
|`CJV_RECURSION_COUNT`|Internal use only. cjv sets this variable during proxy execution to detect and prevent infinite recursive calls. Users should not set it manually.|
