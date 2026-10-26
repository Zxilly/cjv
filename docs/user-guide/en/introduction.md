# cjv user guide

cjv is a version manager for the [Cangjie](https://cangjie-lang.cn/) SDK. It installs toolchains on your machine and selects an SDK version for each project. When the command path is set, proxy commands such as `cjc` and `cjpm` call the active SDK through cjv.

## Get started

Start with these chapters to install cjv and run your first Cangjie commands:

1. Read [Install cjv](installation/index.md) to run the installer script, download an archive, or build from source.
2. Read [Quick start](basic-usage.md) to select a default toolchain, configure a project, and update SDKs.
3. Read [Run commands and set the environment](runtime-environment.md) to run SDK tools, start compiled programs, or load SDK variables into your shell.

The setup wizard queries the channels that are available for your platform. Platform support in cjv can differ from the release coverage of the Cangjie SDK. Run `cjv toolchain list-remote` to check the available versions for your host.

## Manage SDKs and project toolchains

Use these chapters to control SDK versions, extensions, and target platforms across your projects:

- [Toolchains and versions](concepts/toolchains.md) explains release channels, pinned versions, host platforms, and version names.
- [Project toolchains](toolchain-file.md) shows how to commit `cangjie-sdk.toml` so your team shares the same SDK, components, and targets.
- [Components and offline documentation](concepts/components.md) covers the `stdx` extension library and local HTML documentation.
- [Custom SDKs](install-from-url.md) explains how to link a local SDK directory or install from a `.zip`, `.tar.gz`, or URL.
- [Cross-compilation](cross-compilation.md) shows how to add target SDKs such as `ohos` and `android` to a host toolchain.

## Configure and deploy

Use these chapters to customize cjv for local workstations, CI pipelines, and enterprise networks:

- [Configuration](configuration.md) and [Environment variables](environment-variables.md) describe user settings, data directories, caches, and process overrides.
- [Automation and JSON interfaces](automation.md) covers CI isolation, offline caches, update history, rollback, and JSON output schemas.
- [Network proxies](network-proxies.md) explains HTTP, HTTPS, and SOCKS5 proxy settings and private TLS certificates.
- [Enterprise and offline deployment](enterprise/index.md) and [Internal distribution](enterprise/distribution-server.md) describe fallback settings, air-gapped hosts, and private mirrors.

## Reference and support

- [Command reference](command-reference.md) lists all CLI commands, flags, and installation policies.
- [Common questions](faq.md) gives solutions for version selection, runtime library errors, and offline setups.
- If you want to build or change cjv itself, read the [cjv developer guide](https://cjv.zxilly.com/book/dev-guide/en/).
