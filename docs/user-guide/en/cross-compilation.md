# Cross-compilation

A cross SDK provides compilers, libraries, and tools on your current host machine so you can build programs for another target platform. cjv manages each cross SDK as an attachment to an installed host [toolchain](concepts/toolchains.md). The cross SDK release must match the host release, and available targets depend on your distribution manifest.

## Install and list target SDKs

You can install a cross target when you install a host toolchain, or add it later with `cjv target add`:

```bash
cjv install sts --target ohos
cjv target list --toolchain sts
cjv target add android --toolchain sts
cjv target list --toolchain sts --installed
```

- `cjv target add` installs the target SDK for the existing host release without upgrading the host SDK.
- `cjv target list` queries the distribution manifest for all targets published for that host release.
- `cjv target list --installed` reads local metadata only and works offline.

Always write target suffixes such as `ohos`, `android`, or `ohos-arm32`. Do not write a full platform tuple such as `linux-x64-ohos`. You can pass `--target` more than once or use a comma-separated list such as `--target ohos,android`.

You can also declare needed targets in [cangjie-sdk.toml](toolchain-file.md#components-and-target-sdks) so cjv installs them automatically when a project command runs.

## Load the target build environment

By default, proxy commands such as `cjc` and `cjpm` select the host SDK environment. When you want to call tools or link against libraries from an installed cross SDK, load its environment into your current shell with `cjv envsetup`:

```bash
eval "$(cjv envsetup +sts --target=ohos)"
cjc --version
```

In Windows PowerShell, run:

```powershell
cjv envsetup +sts --target=ohos | Invoke-Expression
```

Installing a target SDK or listing `targets` in `cangjie-sdk.toml` prepares the files on disk. It does not turn a regular `cjpm build` into a cross-platform build on its own. Your build scripts and project configuration must still pass the target compiler flags, system sysroot, and linker options for the target platform. Read [Run commands and set the environment](runtime-environment.md#load-a-cross-compilation-environment) for details on shell sessions.

## Manage target components, updates, and removal

### Target components

Each cross SDK manages its own `stdx` component separately from the host SDK. Pass `--target <suffix>` to `cjv component` commands after you install the target SDK:

```bash
cjv component add stdx --toolchain sts --target ohos
cjv component list --toolchain sts --target ohos
```

Host `stdx` libraries do not substitute for target `stdx` libraries. Read [Components and offline documentation](concepts/components.md) for component storage and options.

### Channel updates with targets

When you run `cjv update` on a tracked channel, cjv prepares the updated host SDK, all installed cross targets, and all selected components before it replaces the active installation.

By default, if a needed target or component is missing from the new release, cjv blocks the update and keeps your working setup. On the `nightly` channel, cjv searches the published history for a release that provides all needed targets and components. Pass `--force` if you want cjv to skip and remove unavailable optional targets or components during the update; read [Command reference](command-reference.md#installation-and-update-policies).

### Remove a target SDK

To delete a cross SDK and its target components while keeping the host toolchain, run `cjv target remove`:

```bash
cjv target remove ohos --toolchain sts
```

A cross SDK always belongs to a host toolchain and cannot be set as the global default or active toolchain directly.
