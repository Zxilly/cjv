# Getting Started

This chapter walks you through the complete flow from installing a toolchain to running Cangjie tools, and introduces the common commands you'll use day to day. By the end you'll be up and running; for deeper concepts see [Core Concepts](concepts/index.md), and for the full set of options for each command see the [Command Reference](command-reference.md).

This chapter assumes you have already installed `cjv` itself. If you haven't, start with [Installing cjv](installation/index.md).

## Up and running in five minutes

The following four steps demonstrate the most typical usage: install an LTS toolchain, set it as the default, check the status, and then run a command with it.

```bash
# 1. Install the latest LTS toolchain
cjv install lts

# 2. Set it as the default toolchain
cjv default lts

# 3. Show the active and installed toolchains
cjv show

# 4. Run a command with a specific toolchain
cjv run lts cjc --version
```

`cjv install lts` downloads and installs the latest toolchain from the LTS channel. `lts` is a channel name, which cjv resolves to a specific version. Besides `lts`, you can also install `sts`, `nightly`, or an exact version number. For the meaning of each channel, see [Channels](concepts/channels.md).

`cjv default lts` records `lts` as the default toolchain. Once a default is set, cjv uses it whenever no toolchain is declared separately in the project.

`cjv show` lists the currently active toolchain and all installed toolchains, which makes it easy to confirm the installation. To see just one of these, use `cjv show active` or `cjv show installed`.

`cjv run lts cjc --version` runs `cjc --version` explicitly with the `lts` toolchain. `cjv run <toolchain> <command> [args...]` temporarily switches to the given toolchain to run a command without changing the default, which makes it easy to compare across toolchains temporarily.

## Using `cjc` and `cjpm` directly (proxy execution)

Once a toolchain is installed and `PATH` is configured, you do not need to prefix every command with `cjv run`; just call the SDK tools directly:

```bash
cjc --version
cjpm build
```

When installing, cjv creates a proxy symlink for each SDK tool in its own `bin` directory. When you call commands such as `cjc` or `cjpm`, what actually runs is this proxy. It first resolves which toolchain should be used, then transparently forwards the call to the real executable in that toolchain, and automatically injects the necessary environment variables (such as `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` when stdx is installed).

The proxied SDK tools include `cjc`, `cjc-frontend`, `cjpm`, `cjfmt`, `cjlint`, `cjdb`, `cjcov`, and several internal tools. Use the Cangjie command-line tools the way you normally would; after installing cjv they work as before, with toolchain switching handled by cjv behind the scenes.

The proxy resolves the toolchain by the same rules as `cjv run`, but without requiring you to specify a toolchain. It decides automatically by priority: environment variable, directory override, the project's `cangjie-sdk.toml`, and finally falling back to the default toolchain. For the full resolution order, see [Proxies](concepts/proxies.md) and [Targets and overrides](concepts/targets-overrides.md).

If the resolved toolchain is not yet installed and you have enabled the `auto_install` setting, cjv installs it automatically before proxying, with no manual `cjv install` needed. To enable it:

```bash
cjv set auto-install true
```

For details on this setting, see [Configuration](configuration.md).

 >
 > Tip: a binary compiled by Cangjie still needs the correct library search paths at runtime. To run a program you compiled yourself, see `cjv exec` and `cjv envsetup` in [Runtime environment](runtime-environment.md).

## Project-level toolchains

Write the toolchain declaration into the project so that team members automatically use the same toolchain once they enter the directory, without each switching manually. Place a `cangjie-sdk.toml` in the project root:

```toml
[toolchain]
channel = "lts"
```

After that, within this directory (or any of its subdirectories), `cjv run`, the proxied `cjc`/`cjpm`, and `cjv show active` all prefer the toolchain declared here. For the full set of fields in this file (`channel`, `components`, `targets`), see [Toolchain file](toolchain-file.md).

If you only want to temporarily bind a toolchain to a directory without committing a file, you can use a directory override:

```bash
cjv override set nightly
```

For details, see [Targets and overrides](concepts/targets-overrides.md).

## Common commands at a glance

Below are the commands you'll use most in day-to-day work. For the full set of arguments and subcommands of each command, see [Command reference](command-reference.md).

|Command|Description|
|-------|-----------|
|`cjv install <toolchain>`|Install a toolchain (such as `lts`, `sts`, `nightly`, or a specific version)|
|`cjv uninstall <toolchain>`|Uninstall a toolchain|
|`cjv update [toolchain]`|Update installed toolchains|
|`cjv default [toolchain]`|Set or show the default toolchain|
|`cjv show`|Show the active and installed toolchains|
|`cjv run <toolchain> <command> [args...]`|Run a command with a specific toolchain|
|`cjv exec [+toolchain] <command> [args...]`|Execute a command in the Cangjie runtime environment|
|`cjv which <command>`|Show the path of an SDK tool in the active toolchain|
|`cjv check`|Check for available updates (without installing)|
|`cjv override set <toolchain>`|Set a toolchain override for the current directory|
|`cjv component add <name>...`|Install a component for a toolchain (such as `stdx`)|
|`cjv self update`|Update cjv itself to the latest version|

## Next steps

- To learn about core concepts such as toolchains, channels, components, and proxies, start from [Core concepts](concepts/index.md).
- To pin a toolchain version for a project, see [Toolchain file](toolchain-file.md).
- For cross-compilation, see [Cross-compilation](cross-compilation.md).
- To run binaries you compiled yourself, see [Runtime environment](runtime-environment.md).
- For all commands, options, and environment variables, see [Command reference](command-reference.md) and [Environment variables](environment-variables.md).
