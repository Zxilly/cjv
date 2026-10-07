# Components and offline documentation

Components belong to individual toolchains and are removed when their toolchain is uninstalled:

| Component | Contents | Location relative to `CJV_HOME` |
| --- | --- | --- |
| `stdx` | Extension libraries | `stdx/<tc>/{dynamic,static}` |
| `docs` | Language, standard library, and tool documentation | `docs/<tc>/main/` |
| `stdx-docs` | Extension library documentation | `docs/<tc>/stdx/` |

## Install and remove

```bash
cjv install nightly -c stdx,docs
cjv component add stdx docs --toolchain lts
cjv component list --toolchain lts
cjv component list --installed -q
cjv component remove stdx-docs
```

Without `--toolchain`, commands use the current selection. `--target <suffix>` selects an installed cross SDK belonging to that host. Component names may be repeated or comma-separated. `component add` skips installed entries; `--force` reinstalls them.

Available artifacts depend on the channel, release, and platform. Custom toolchains have no official component download source. Projects can also declare components in [cangjie-sdk.toml](../toolchain-file.md) for automatic installation.

Tracked channel updates replace the SDK and selected components together. Components of pinned versions remain unchanged. Locally linked stdx keeps its source.

## Link local stdx

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

The source must contain `dynamic/` and `static/`. `--force` replaces an existing component. Removing a link deletes cjv's entries and preserves the source directory. `docs` and `stdx-docs` do not support local links.

Component edits require an SDK in a real directory managed by cjv. An external SDK directory referenced by `toolchain link` cannot be edited through component commands. To let cjv manage its components, install the SDK from a [local archive or URL](../install-from-url.md):

```bash
cjv toolchain link my-sdk ./cangjie-sdk.zip
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

Installed stdx supplies `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` to proxies, `exec`, and `envsetup`. Documentation components do not change the runtime environment.

## Open documentation

```bash
cjv doc
cjv doc std
cjv doc dev-guide
cjv doc tools
cjv doc stdx --toolchain nightly
cjv doc --path
```

`doc` opens local HTML in a browser. `--path` and `--json` only return the path. `book` is an alias for `dev-guide`. Without a topic, cjv prefers the main documentation, then extension library documentation. If documentation is missing, it asks you to install the corresponding component.
