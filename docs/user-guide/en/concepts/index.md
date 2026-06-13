# Core Concepts

cjv manages the Cangjie SDK with a few concepts that work together. A _toolchain_ is an installation of a particular version of the Cangjie SDK. You usually specify which toolchain to install through a _channel_ (such as `lts`, `sts`, `nightly`) rather than remembering a specific version number. Each toolchain can also carry several _components_ (such as `stdx`, `docs`), which are extension resources released together with the SDK.

When you call SDK tools such as `cjc` or `cjpm` directly, a _proxy_ forwards the call to the currently active toolchain. Which toolchain is active is decided jointly by several sources according to priority. A directory-level _override_ pins a toolchain for a given project directory. A _target_ is an additional cross-compilation SDK on top of the host toolchain, used to build artifacts for other platforms.

The following subsections cover these concepts one by one:

- [Toolchains](toolchains.md): the unit of Cangjie SDK installation, and how the active toolchain is resolved.
- [Channels](channels.md): rolling aliases such as `lts` / `sts` / `nightly`.
- [Components](components.md): extension resources managed with the toolchain, such as `stdx`, `docs`, and `stdx-docs`.
- [Proxies](proxies.md): transparent forwarding and on-demand installation for SDK tool calls.
- [Targets and overrides](targets-overrides.md): cross-compilation target SDKs and directory-level toolchain overrides.
