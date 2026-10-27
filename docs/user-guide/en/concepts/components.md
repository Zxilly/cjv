# Components and offline documentation

Components are optional extension libraries and offline documentation attached to a [toolchain](toolchains.md). Each toolchain has its own components, which are removed when that toolchain is uninstalled:

| Component | Contents | Location relative to `CJV_HOME` |
| --- | --- | --- |
| `stdx` | Dynamic and static extension libraries | `stdx/<tc>/{dynamic,static}` |
| `docs` | Language guide, standard library, and tool documentation | `docs/<tc>/main/` |
| `stdx-docs` | Extension library (`stdx`) documentation | `docs/<tc>/stdx/` |

When `stdx` is installed on the selected toolchain, proxy commands (`cjc`, `cjpm`), `cjv exec`, and `cjv envsetup` automatically export `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` in the runtime environment. Documentation components (`docs` and `stdx-docs`) are used by `cjv doc` and do not alter runtime environment variables.

## Installing and removing components

Components can be added when you first install a toolchain with the `-c` (`--component`) flag, or managed on an already-installed toolchain with `cjv component`:

```bash
cjv install nightly -c stdx,docs
cjv component add stdx docs --toolchain lts
cjv component list --toolchain lts
cjv component list --installed -q
cjv component remove stdx-docs
```

- When `--toolchain` is omitted, `cjv component` operates on the currently active toolchain.
- Passing `--target <suffix>` selects an installed [cross-compilation](../cross-compilation.md) target SDK belonging to that host toolchain (for example, `cjv component add stdx --toolchain sts --target ohos`).
- Component names can be passed as multiple arguments or as a comma-separated list.
- `cjv component add` skips components that are already installed; pass `--force` if you want to reinstall them from the distribution source.

Not all components are published for every channel, version, or platform. Custom toolchains have no official upstream record in the distribution manifest, so `cjv component add` cannot download components for them. In team repositories, you can also list required components in [`cangjie-sdk.toml`](../toolchain-file.md) so `cjv` installs them automatically.

### How components behave during updates

When you run `cjv update` on a tracked channel (`lts`, `sts`, `nightly`), `cjv` updates the host SDK and all of its selected components together in a single transaction. Components attached to a pinned version (such as `lts-1.0.5`) remain unchanged. If you have linked a local `stdx` directory to a tracked channel, updating the channel preserves your local `stdx` link.

## Linking a local `stdx` directory

If you build `stdx` from source or have a pre-extracted `stdx` directory on disk, you can attach it to a toolchain with `cjv component link`:

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

The directory you point to must contain both `dynamic/` and `static/` subdirectories. Passing `--force` replaces an existing `stdx` component on that toolchain. When you later run `cjv component remove stdx`, `cjv` removes its internal link entry while preserving your local source directory untouched. Note that `docs` and `stdx-docs` do not support local directory links.

### Managing components on custom SDKs

Because `cjv component` commands record component manifests inside a `cjv`-managed toolchain installation, an external SDK directory referenced via `cjv toolchain link <name> /path/to/sdk` cannot be modified by `cjv component`—the owner of that external directory maintains its contents directly.

If you want `cjv` to manage components (such as linking a local `stdx`) for a custom SDK, install the custom SDK from a [local archive or URL](../install-from-url.md) so that `cjv` extracts it into a managed directory first:

```bash
cjv toolchain link my-sdk ./cangjie-sdk.zip
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

## Viewing offline documentation with `cjv doc`

Once you have installed the `docs` or `stdx-docs` component, use `cjv doc` (or its alias `cjv docs`) to open the local HTML documentation in your default web browser:

```bash
cjv doc
cjv doc std
cjv doc dev-guide
cjv doc tools
cjv doc stdx --toolchain nightly
cjv doc --path
```

You can optionally specify a documentation topic:

- **No topic (`cjv doc`)** — Opens the main documentation home page if `docs` is installed, or falls back to the `stdx` documentation home page if only `stdx-docs` is installed.
- **`std`** — Opens the Cangjie standard library API reference (requires `docs`).
- **`dev-guide`** (or **`book`**) — Opens the Cangjie developer guide (requires `docs`).
- **`tools`** — Opens the Cangjie SDK tools manual (requires `docs`).
- **`stdx`** — Opens the extension library documentation (requires `stdx-docs`).

Passing `--path` (or `--json`) prints the resolved local filesystem path to `stdout` without launching a browser. If the required documentation component is not installed yet, `cjv doc` tells you which `cjv component add` command to run.
