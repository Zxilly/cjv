# Introduction

_cjv_ installs the [Cangjie Programming Language](https://cangjie-lang.cn/) SDK from the official release channels, enabling you to easily switch between LTS, STS, and nightly toolchains and keep them updated. Once `cjv` is installed and its proxy directory is on your `PATH`, everyday commands like `cjc` and `cjpm` automatically route to the toolchain selected for your current project.

Check out the [Install cjv](installation/index.md) chapter for instructions on installing `cjv` across Linux, macOS, Windows, and HarmonyOS, and then follow the [Quick start](basic-usage.md) to inspect your active toolchain, pin a project version, and keep your SDKs up to date.

Note that `cjv` itself runs on platforms where a Cangjie SDK channel might not yet publish prebuilt archives. During setup, the installation wizard queries the upstream manifest for channels available on your host platform, and you can run `cjv toolchain list-remote` at any time to see which releases are published for your machine.

## How this book is organized

### Getting started and running tools

- **[Install cjv](installation/index.md)** covers the one-line shell and PowerShell installers, version pinning for `cjv` itself, the GitCode mirror, manual archive downloads, and `PATH` configuration.
- **[Quick start](basic-usage.md)** walks through everyday operations: checking your installation, choosing a default toolchain, pinning a project version, and updating or removing toolchains.
- **[Running commands and setting the environment](runtime-environment.md)** explains how proxy binaries work, how to run a single command with `cjv run` or `cjv exec` (including runtime library paths for compiled Cangjie binaries), and how to load an SDK into your interactive shell with `cjv envsetup`.

### Managing toolchains, components, and targets

- **[Toolchains and versions](concepts/toolchains.md)** introduces the core concepts behind release channels (`lts`, `sts`, `nightly`), pinned releases, minor-version and date selectors, host platforms, and custom toolchains.
- **[Project toolchains](toolchain-file.md)** explains how `cjv` chooses the active toolchain for a directory, how to check a `cangjie-sdk.toml` file into version control so your whole team shares the same setup, and how to use local directory overrides.
- **[Components and offline documentation](concepts/components.md)** covers optional packages attached to a toolchain—such as the `stdx` extension library and offline HTML documentation (`cjv doc`)—as well as linking local `stdx` builds.
- **[Custom SDKs](install-from-url.md)** shows how to register unpacked local SDK directories or install custom SDKs from `.zip` / `.tar.gz` archives and HTTP(S) URLs.
- **[Cross-compilation](cross-compilation.md)** explains the distinction between host platforms and compilation targets, and shows how to attach target SDKs such as `ohos` and `android` to your host toolchain.

### Configuration, automation, and enterprise deployment

- **[Configuration](configuration.md)** and **[Environment variables](environment-variables.md)** document `~/.cjv/settings.toml`, inheritance from system fallback settings, the `<CJV_HOME>` directory layout, reusable download caches, and hard-link deduplication.
- **[Automation and JSON interfaces](automation.md)** is designed for CI pipelines and tooling authors, covering `cjv pin --check`, isolated CI directories, `--offline` builds, `cjv doctor`, update history and rollback, and the `--json` output contract.
- **[Network proxies](network-proxies.md)** explains how to use HTTP, HTTPS, and SOCKS5 proxies and custom TLS certificates.
- **[Enterprise and offline deployment](enterprise/index.md)** and **[Internal distribution](enterprise/distribution-server.md)** describe how organizations can host private SDK mirrors, distribute system fallback settings, and support air-gapped developer workstations.

### Reference and contributing

For a complete overview of every subcommand and flag, see the **[Command reference](command-reference.md)**, or check **[Common questions](faq.md)** for troubleshooting tips.

The source code for `cjv` and this manual is hosted on GitHub at <https://github.com/Zxilly/cjv>. If you run into a bug or want to contribute, check out the [issue tracker](https://github.com/Zxilly/cjv/issues) and the [**cjv Developer Guide**](https://cjv.zxilly.com/book/dev-guide/en/).
