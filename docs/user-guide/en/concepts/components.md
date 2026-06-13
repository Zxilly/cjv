# Components

A component is an extension resource released together with the Cangjie SDK but managed separately from the SDK itself. After installing a [toolchain](toolchains.md), you can attach components to it as needed, and remove them individually without affecting the SDK itself.

cjv currently supports three kinds of components:

|Components|Content|Install location (relative to `CJV_HOME`)|
|----------|-------|-----------------------------------------|
|`stdx`|Cangjie extension libraries (the dynamic/static library files of the Cangjie extension libraries)|`stdx/<tc>/{dynamic,static}`|
|`docs`|Offline documentation for the Cangjie core (dev-guide, libs/std, tools)|`docs/<tc>/main/`|
|`stdx-docs`|Offline documentation for the Cangjie extension libraries|`docs/<tc>/stdx/`|

Here `<tc>` is the toolchain name (such as `lts-1.0.5`). Components are stored split by toolchain, with each toolchain having its own independent set of components. When a toolchain is uninstalled, its `stdx/<tc>/` and `docs/<tc>/` are cleaned up along with it.

The download source for components depends on the [channel](channels.md):

- `stdx`: for LTS / STS, downloaded from [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases); for nightly, downloaded from [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases).
- `docs`: for LTS / STS, downloaded from the GitHub release at [`cangjie-docs-bundle`](https://github.com/Zxilly/cangjie-docs-bundle/releases); for nightly, downloaded from [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases).
- `stdx-docs`: for LTS / STS, downloaded from [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases); for nightly, downloaded from [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases).

## Automatically injected environment variables

`stdx` is the only component that contributes variables to the runtime environment. Once a toolchain has `stdx` installed, cjv automatically injects two environment variables during [proxy execution](proxies.md) and in `cjv exec` / `cjv envsetup`, with no manual setup needed:

|Environment Variables|Points to|
|---------------------|---------|
|`CANGJIE_STDX_PATH_DYNAMIC`|`<CJV_HOME>/stdx/<tc>/dynamic`|
|`CANGJIE_STDX_PATH_STATIC`|`<CJV_HOME>/stdx/<tc>/static`|

These two variables appear only when the corresponding toolchain actually has `stdx` installed. `docs` and `stdx-docs` are pure documentation data and contribute no runtime environment variables. For a full explanation of the runtime environment, see [The runtime environment](../runtime-environment.md).

## Installing and uninstalling components

The most direct way is to install components alongside the toolchain using `-c` / `--component`:

```bash
# Install the nightly toolchain, along with stdx and docs
cjv install nightly -c stdx,docs
```

Component names can be comma-separated, and the flag can also be passed repeatedly (e.g. `-c stdx -c docs`).

After a toolchain is installed, you can also manage its components individually. The subcommands of `cjv component` act on the currently active toolchain by default; use `--toolchain <tc>` to specify another toolchain:

```bash
# Add stdx to the lts toolchain
cjv component add stdx --toolchain lts

# Add several at once
cjv component add stdx docs

# Uninstall a component (remove can also be written as rm / uninstall / delete)
cjv component remove stdx-docs
```

`cjv component add` skips a component that is already installed. To force a re-download and reinstall, add `--force`.

## Viewing components

`cjv component list` shows the installed and available status of components for the current toolchain:

```bash
# List all components of a toolchain along with their status
cjv component list --toolchain nightly

# Show only the installed components
cjv component list --installed
```

Whether a component is available depends on the channel. All three component types are supported on LTS, STS, and nightly, but a custom toolchain has no corresponding release asset, so installing via `cjv component add` fails (see the next section).

## Linking a local stdx

For a custom toolchain linked through `cjv toolchain link`, `cjv component add stdx` cannot work, because a custom toolchain has no release asset to download. In this case, use `cjv component link stdx <path>` instead to attach a local stdx directory to the toolchain:

```bash
# First link a locally built/obtained SDK
cjv toolchain link mysdk /path/to/local/sdk

# Then link the local stdx to this toolchain
cjv component link stdx /path/to/local/stdx --toolchain mysdk
```

A standard channel (such as `lts`) can also use `link` in place of downloading, which suits offline environments or debugging a self-built stdx. The toolchain may already have a downloaded stdx installed, in which case you need `--force` to overwrite it:

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

`<path>` must be a directory containing the two subdirectories `dynamic/` and `static/`, that is, the standard stdx layout after extraction. When linking, cjv creates a symlink for each of these two subdirectories under `<CJV_HOME>/stdx/<tc>/` (on Windows, if a symlink requires elevation, it falls back to a directory junction). `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` are still injected as usual, pointing to these links.

Linking is safe. `cjv component remove stdx` and `cjv toolchain uninstall` only delete the symlinks cjv created, and do not follow the links to delete the data in the original directory.

 >
 > `link` currently works only for `stdx`; `docs` and `stdx-docs` do not support linking and can only be downloaded and installed.

## Declaring components in the toolchain file

The `components` field in the [toolchain file](../toolchain-file.md) `cangjie-sdk.toml` is also recognized. With `auto_install` enabled, [proxy execution](proxies.md) fills in the components missing for the current project on demand:

```toml
[toolchain]
channel = "nightly"
components = ["stdx", "docs"]
```

When a team member enters the project directory and runs `cjc` or `cjpm`, cjv automatically installs the declared components before proxying.

## Opening offline documentation

Once `docs` or `stdx-docs` is installed, `cjv doc` opens the current toolchain's local HTML documentation in the browser:

```bash
# Open the documentation home page
cjv doc

# Jump to a specific topic
cjv doc stdx        # Extension library docs (from stdx-docs)
cjv doc std         # Standard library
cjv doc dev-guide   # Development guide (book also works)
cjv doc tools       # Tools documentation
```

Common options:

- `--toolchain <tc>`: open the documentation of the specified toolchain (defaults to the active toolchain).
- `--path`: print only the resolved file path without launching the browser, which is handy for use in scripts or to confirm where the docs are located.

```bash
# Print the path only, without opening the browser
cjv doc --path

# View the tools documentation path of the nightly toolchain
cjv doc tools --toolchain nightly --path
```

Without a topic, `cjv doc` opens the documentation entry point (the `docs` home page first, then `stdx-docs`). If the corresponding toolchain does not yet have `docs` / `stdx-docs` installed, the command prompts you to install the component first with `cjv component add`.
