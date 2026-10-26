# Components and offline documentation

Components are optional packages that belong to a specific [toolchain](toolchains.md), such as the `stdx` extension library and offline HTML documentation. When you uninstall a toolchain, cjv also removes its components. This chapter explains how to install components, link local `stdx` builds, and open offline documentation.

## Available components

cjv stores components under `<CJV_HOME>` by toolchain name (`<tc>`):

| Component | Contents | Path relative to `CJV_HOME` |
| --- | --- | --- |
| `stdx` | Dynamic and static extension libraries | `stdx/<tc>/{dynamic,static}` |
| `docs` | Language, standard library, and tool documentation | `docs/<tc>/main/` |
| `stdx-docs` | Extension library documentation | `docs/<tc>/stdx/` |

When a toolchain has `stdx`, proxy commands, `cjv exec`, and `cjv envsetup` set `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` automatically. Documentation components do not change the runtime environment; read [Run commands and set the environment](../runtime-environment.md) for environment details.

## Install, list, and remove components

### Manage components from the CLI

You can install components during toolchain installation or add them later with `cjv component`:

```bash
cjv install nightly -c stdx,docs
cjv component add stdx docs --toolchain lts
cjv component list --toolchain lts
cjv component list --installed -q
cjv component remove stdx-docs
```

- If you omit `--toolchain`, cjv uses the active toolchain.
- Pass `--target <suffix>` to manage components for an installed [cross-compilation](../cross-compilation.md) target SDK on that host.
- You can pass component names as separate arguments or as a comma-separated list.
- `cjv component add` skips components that are already installed. Pass `--force` to reinstall them.

Available component archives depend on the channel, version, and platform in your distribution manifest. You can also declare needed components in [cangjie-sdk.toml](../toolchain-file.md) so cjv installs them automatically when a project tool runs.

### Component behavior during updates

When you run `cjv update` on a tracked channel, cjv updates the host SDK and its installed components together. Components that belong to a pinned version such as `lts-1.0.5` stay unchanged. If you linked a local `stdx` directory, the update keeps your local link.

## Link a local `stdx` directory

### Link `stdx` to a managed toolchain

If you build `stdx` locally or receive a prebuilt directory, link it to a toolchain with `cjv component link`:

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

The source directory must contain both `dynamic/` and `static/` subdirectories. Pass `--force` to replace an existing `stdx` installation on that toolchain.

When you remove a linked component with `cjv component remove stdx`, cjv deletes its internal link and keeps your source directory intact. The `docs` and `stdx-docs` components do not support local directory links.

### Manage components on custom SDKs

Component commands can only edit an SDK that lives in a directory managed by cjv. If you registered an external SDK directory with `cjv toolchain link <name> /path/to/sdk`, the owner of that directory maintains its components and `cjv component` commands cannot change it.

If you want cjv to manage components for a custom SDK, install the SDK from a local archive or URL first:

```bash
cjv toolchain link my-sdk ./cangjie-sdk.zip
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

Custom toolchains have no official download source for `cjv component add`, but archive installations can use bundled `stdx` or link a local `stdx` directory. Read [Custom SDKs](../install-from-url.md) for archive installation rules.

## Open offline documentation

After you install `docs` or `stdx-docs`, use `cjv doc` (or its alias `cjv docs`) to open the local HTML manual in your browser:

```bash
cjv doc
cjv doc std
cjv doc dev-guide
cjv doc tools
cjv doc stdx --toolchain nightly
cjv doc --path
```

| Topic argument | Documentation opened | Needed component |
| --- | --- | --- |
| *(omitted)* | Main home page when `docs` is present; otherwise `stdx` home page | `docs` or `stdx-docs` |
| `std` | Standard library reference | `docs` |
| `dev-guide` or `book` | Cangjie developer guide | `docs` |
| `tools` | SDK tools manual | `docs` |
| `stdx` | Extension library reference | `stdx-docs` |

Pass `--path` or `--json` to print the local file path without opening a browser. If the needed component is missing, cjv prints the command to install it. Read [Command reference](../command-reference.md#offline-documentation) for all options.
