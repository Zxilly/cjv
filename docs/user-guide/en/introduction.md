# cjv user guide

cjv is a version manager for the [Cangjie](https://cangjie-lang.cn/) SDK. It installs multiple toolchains on your machine and selects a version for each project. Once the command path is configured, commands such as `cjc` and `cjpm` use cjv to invoke the selected SDK.

## Get started

Start with [installation](installation/index.md), then follow the [quick start](basic-usage.md) to check your toolchain, configure a project, and run commands.

The installation wizard queries channels available for your platform. cjv's platform support may differ from the SDK's release coverage. Use `cjv toolchain list-remote` to check available versions.

## Configure projects and manage SDKs

[Toolchains and versions](concepts/toolchains.md) explains channels, pinned versions, and name syntax. Commit a [cangjie-sdk.toml](toolchain-file.md) in a team project to share version, component, and target SDK requirements.

See [components](concepts/components.md) for extension libraries and offline documentation, and [cross-compilation](cross-compilation.md) for target SDKs. Existing SDK directories and archives can be registered as [custom toolchains](install-from-url.md).

For running compiled programs or setting up your current shell, see [running commands and setting the environment](runtime-environment.md).

## Configuration and reference

[Configuration](configuration.md) and [environment variables](environment-variables.md) cover user settings and temporary overrides. Use [network proxies](network-proxies.md) when upstream access requires a proxy. See [enterprise and offline deployment](enterprise/index.md) for internal sources, managed clients, and fully offline installation.

Use the [command reference](command-reference.md) for command options and [common questions](faq.md) for troubleshooting.

See [Automation and JSON interfaces](automation.md) for version pinning checks, CI isolation, offline caches, rollback, and JSON output. For development, see the [developer guide](https://cjv.zxilly.com/book/dev-guide/en/).
