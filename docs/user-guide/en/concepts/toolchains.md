# Toolchains and versions

A toolchain is one Cangjie SDK installation containing the compiler (`cjc`), package manager (`cjpm`), build tools, and runtime libraries. cjv stores toolchains under `<CJV_HOME>/toolchains/` and optional [components](components.md) (`stdx` and offline documentation) in separate directories.

`cjv` supports several kinds of toolchains. The most common track the official release channels—`lts`, `sts`, and `nightly`—but `cjv` can also install specific pinned versions from the release manifests, toolchains for alternate host platforms, and [custom toolchains](#custom-sdks) from local directories or archives.

## Channels and names

Standard release channel and version toolchain names follow this form:

```text
[<channel>-]<version>[-<host>]
<channel>[-<date>][-<host>]

<channel>       = lts|sts|nightly
<version>       = <major.minor>|<major.minor.patch>|<nightly-version>
<date>          = YYYY-MM-DD
<host>          = <os>-<arch>
```

Here is how the different toolchain selectors work in practice:

| Name | Meaning |
| --- | --- |
| `lts` | Track the Long-Term Support release channel |
| `sts` | Track the Short-Term Support release channel |
| `nightly` | Track development builds published in the nightly manifest |
| `lts-1.0.5` | Pin a specific version within a channel |
| `1.0.5` | Omit the channel prefix and let `cjv` find the matching release |
| `sts-1.2`, `1.2` | Select the latest stable patch within that minor version series |
| `nightly-2026-10-04` | Select the latest published nightly build for that date |
| `my-sdk` | A custom toolchain registered with `cjv toolchain link` |

The version numbers and dates in the table above illustrate the naming syntax. To see which versions are actually published by your distribution source for your platform, run `cjv toolchain list-remote`:

```bash
cjv toolchain list-remote --channel lts
cjv toolchain list-remote --channel nightly --limit 10
```

Official channel names are case-insensitive. Toolchain names that do not match an official channel or version pattern can be used to name custom toolchains—provided they do not conflict with official names, contain path separators, begin with `+`, or equal `.`, `..`, or an empty string.

## Tracked channels and pinned versions

When you install a channel name like `lts` and a specific version like `lts-1.0.5`, `cjv` treats them as two completely independent installations:

```bash
cjv install lts
cjv install lts-1.0.5
```

When you later run `cjv update lts` (or `cjv update`), `cjv` updates the tracked `lts` installation to the newest release on that channel, while leaving `lts-1.0.5` untouched. Their attached components and cross-compilation targets are also managed independently. Running proxy commands like `cjc` always uses the release currently installed on disk; upgrading an installed toolchain requires an explicit `cjv install` or `cjv update`.

Shorthand selectors—such as a minor version (`sts-1.2`) or a nightly date (`nightly-2026-10-04`)—are resolved against the manifest at installation time and installed as concrete pinned versions. When you want a team project to build with the exact same compiler across machines, write the full concrete release name in [`cangjie-sdk.toml`](../toolchain-file.md) (or run `cjv pin` to resolve and write it for you).

### Nightly availability and component compatibility

Nightly builds are published frequently, and occasionally a newest nightly build may not yet include every optional component (`stdx`, `docs`) or cross-compilation target SDK.

When you install or update the `nightly` channel, `cjv` checks which components and cross targets are required (either requested on the command line or already installed on your local `nightly` toolchain) and automatically searches the published history in `nightly.json` for the newest compatible release. By default, `cjv` will not downgrade an already-installed `nightly` toolchain to an older build; if you want to permit stepping back to an older compatible nightly, pass `--allow-downgrade`. See [Installation and update policies](../command-reference.md#installation-and-update-policies) in the Command reference for full details.

## Host platforms

You can optionally append a host platform suffix to a toolchain name, such as `sts-linux-x64` or `sts-1.2.0-darwin-arm64`. Each host platform is installed separately. Specifying your machine's default host platform explicitly (for example, `lts-linux-x64` on an x86_64 Linux machine) refers to the exact same installation as `lts` without a platform suffix.

Keep in mind that installing another operating system's host SDK does not make that platform's binaries executable on your machine. If you want to compile code for another platform from your current machine, attach a cross-compilation SDK to your host toolchain using [`cjv target`](../cross-compilation.md). A cross-compilation target SDK is an add-on to a host toolchain and cannot be set as the global default or active toolchain on its own.

## Custom SDKs

For developers working on custom Cangjie SDK builds or operating in offline environments, `cjv` can manage local SDK directories and archives. To link an unpacked local SDK directory, run:

```bash
cjv toolchain link my-sdk /path/to/local/sdk
cjv default my-sdk
```

Linking a directory creates a lightweight reference to the SDK directory that you maintain yourself; running `cjv uninstall my-sdk` removes the reference in `cjv` while leaving your source directory untouched.

If you want `cjv` to manage an isolated copy of a custom SDK—or if you want to attach a local `stdx` directory to it—you can pass a `.zip` / `.tar.gz` archive or HTTP(S) URL to `cjv toolchain link` instead. See [Custom SDKs](../install-from-url.md) for details. Note that custom toolchains are skipped during `cjv update` and do not have official component downloads via `cjv component add`.

To inspect all toolchains installed on your machine, run `cjv toolchain list`. To see which toolchain is active in your current directory and why, run `cjv show active`. Next, see [Project toolchains](../toolchain-file.md) for how `cjv` selects a toolchain for each project.
