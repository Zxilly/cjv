# Frequently Asked Questions

This chapter gathers the questions most frequently encountered when using cjv. For the full description of commands, see the [Command Reference](command-reference.md); for background concepts, see [Core Concepts](concepts/index.md).

## Why does installing nightly say a GitCode API key is required?

This message identifies compatibility mode. That mode queries the `releases/latest` endpoint of GitCode's [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) repository, whose authentication uses an access token. Supply the token to `cjv install nightly`, `cjv update nightly`, and `cjv check`:

```text
GitCode API key is required to query nightly versions. Set it with: cjv set gitcode-api-key <your-token>
```

After generating a personal access token on [GitCode](https://gitcode.com/), provide it in any of the following ways:

```bash
# Persist it to settings.toml (recommended)
cjv set gitcode-api-key <your-token>

# Or apply it only to the current session (env var takes precedence over the persisted setting)
export CJV_GITCODE_API_KEY=<your-token>
```

Compatibility-mode nightly discovery uses this token. An enterprise source resolves every channel through its unified manifest. See [Internal distribution source](enterprise/distribution-server.md).

 >
 > Note: cjv verifies nightly assets when a sha256 sidecar is published. Other releases display an integrity notice and use TLS transport.

## Why wasn't my CPU architecture detected automatically on macOS?

Browsers cannot reliably read a Mac's CPU architecture (Apple Silicon or Intel) across browsers. Safari and Firefox do not expose architecture information at all, and Safari even freezes the platform identifier to `MacIntel` on Apple Silicon. So cjv's web install wizard does not guess when it cannot determine the architecture, and instead uses two fallback strategies.

The `install.sh` one-line command given by command install does not hardcode the architecture; the script detects it on your machine and downloads the matching binary. Manual download instead offers both Apple Silicon (arm64) and Intel (x86_64) options on the download page, for you to choose according to your own machine.

If you are unsure of your machine's architecture, run `uname -m` in a terminal: output `arm64` means choose Apple Silicon, output `x86_64` means choose Intel. After cjv itself is installed, toolchain downloads are resolved automatically by cjv based on the machine's real architecture, so this problem no longer arises.

## How do I use cjv in an offline or restricted network environment?

Most cjv operations download assets from upstream, but there are several ways to adapt to offline, intranet, or mirror environments.

See [Enterprise deployment](enterprise/index.md) for the complete artifact layout, system fallback settings, automated installation procedure, and acceptance checklist. This section lists only the quick offline options.

cjv provides a `mirror` build variant with GitCode defaults for the toolchain manifest and self-update backend. It suits networks where GitHub access is unreliable; enterprise mirrors use `dist_server`.

A complete enterprise mirror should set `dist_server`, making LTS, STS, nightly, and components share `<root>/versions.json`. The older `manifest_url` setting only overrides LTS/STS in compatibility mode. See [Configuration](configuration.md).

If you already have an extracted SDK directory, mount it directly with `cjv toolchain link`, which does not trigger a download:

```bash
cjv toolchain link mysdk /path/to/local/sdk
```

In an offline environment you cannot `cjv component add stdx` (which needs to download a release asset); use `cjv component link` instead to mount a local stdx directory. Standard channels can also use `--force` to replace the download with a local directory:

```bash
cjv component link stdx /path/to/local/stdx --toolchain mysdk
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

`<path>` must be a directory containing the two subdirectories `dynamic/` and `static/`. See [Components](concepts/components.md).

You can also place the SDK archive at an intranet-accessible address and install from the URL; see [Installing a toolchain from a URL](install-from-url.md).

In addition, `CJV_MAX_RETRIES` and `CJV_DOWNLOAD_TIMEOUT` can adjust the download retry count and timeout to handle slow links. For the full list, see [Environment variables](environment-variables.md).

## Why can't a custom toolchain use `cjv component add stdx`?

A custom toolchain created with `cjv toolchain link` has no corresponding official release asset, so cjv does not know where to download stdx from, which makes `cjv component add stdx` ineffective for it. Use one of the following two approaches instead:

```bash
# Approach 1: link a local stdx directory
cjv component link stdx /path/to/local/stdx --toolchain mysdk

# Approach 2: when installing from an SDK archive, if the archive bundles an inner cangjie-stdx-* package,
# cjv installs it as the stdx component as well (see "Installing a toolchain from a URL")
```

`cjv component remove stdx` and `cjv toolchain uninstall` do not delete the source directory. Standard channels (lts / sts / nightly) can also use `cjv component link stdx ... --force` to substitute a local directory for the download, which suits offline environments or debugging a self-compiled stdx. See [Components](concepts/components.md).

## Which directories are cleaned up when a toolchain is uninstalled?

When you run `cjv uninstall <tc>` (equivalent to `cjv toolchain uninstall <tc>`), all three directories related to that toolchain are deleted together:

|Directory|Content|
|---------|-------|
|`<CJV_HOME>/toolchains/<tc>`|The SDK itself|
|`<CJV_HOME>/stdx/<tc>`|The stdx component|
|`<CJV_HOME>/docs/<tc>`|The docs and stdx-docs offline documentation|

stdx and docs are decoupled from the toolchain and stored independently, but on uninstall they are treated as attachments of that toolchain and cleaned up together, so that no stale extension resources remain on the next reinstall. If `stdx/<tc>` is a local directory linked via `cjv component link`, only the symlink is deleted and your original data is unaffected.

Uninstalling also cleans up references to that toolchain in `settings.toml`: if it is the default toolchain, the default setting is cleared; any directory overrides pointing to it are also removed.

 >
 > Tip: `cjv self uninstall` deletes the entire `~/.cjv` directory, uninstalling cjv itself along with all toolchains, components, documentation, and settings.

## Can a toolchain installed from a URL span operating systems?

No. The materialize install of `cjv toolchain link` (a local archive or a URL) only supports an SDK that matches the current operating system; if the SDK targets a different OS than the local machine, cjv refuses it before installing:

```text
cannot install a linux SDK on windows; this install supports only SDKs matching the current system
```

After installing, cjv immediately verifies that the SDK works (by running one of its tools), and a binary for a different system cannot run on the local machine. To prepare an SDK for another system, install it on that system.

This restriction applies only to the operating system, not to cross-compilation targets. Building artifacts for other platforms on the same machine is supported, through an additional target SDK; see [Cross-compilation](cross-compilation.md).

## How does cjv get involved when I run `cjc` or `cjpm` directly?

cjv creates proxy symlinks for each SDK tool in its own `bin/` directory. When you call these tools directly, cjv resolves the active toolchain by a fixed priority and forwards the call transparently to the corresponding SDK. The resolution priority, from highest to lowest, is:

1. the `CJV_TOOLCHAIN` environment variable
1. Directory override (`cjv override set`)
1. The toolchain file `cangjie-sdk.toml` (in the current or a parent directory)
1. The default toolchain (`cjv default`)

If `auto-install` is enabled in the settings and the resolved toolchain is not yet installed, cjv installs it automatically before forwarding. See [Proxies](concepts/proxies.md) and [Toolchain File](toolchain-file.md).

## What do I do when a compiled binary reports that it can't find the runtime library?

Cangjie build artifacts dynamically link against runtime libraries (such as `libcangjie-runtime`) and need the correct library search path. Use `cjv exec` to run once in the correct environment, or use `cjv envsetup` to configure the current shell session:

```bash
# Run once, without affecting the current shell
cjv exec ./my_binary arg1 arg2

# Or inject the runtime environment into the current shell (Bash/Zsh)
eval "$(cjv envsetup)"
```

See [Runtime Environment](runtime-environment.md).

## I misspelled `channel` as `channal`—why was there no error, only a warning?

Unrecognized keys in `cangjie-sdk.toml` (such as writing the table name as `[toolchian]` or a field as `channal`) are reported at the warn log level but do not interrupt parsing. cjv ignores them and falls back to the next resolution method. If toolchain selection does not match what you expect, first set the log level to `warn` (the default) or `debug` to check for these messages:

```bash
CJV_LOG=debug cjv show active
```

For the field semantics of the toolchain file, see [Toolchain File](toolchain-file.md); for environment variables, see [Environment Variables](environment-variables.md).
