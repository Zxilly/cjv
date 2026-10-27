# Cross-compilation

A host toolchain runs on the host platform (for example, `linux-x64`) and compiles programs for that platform by default. A cross SDK runs on the same host and provides compiler tools and libraries for another target platform, such as `ohos` or `android`.

`cjv` manages cross SDKs as attachments to an installed host [toolchain](concepts/toolchains.md). A target SDK's release version always matches its host toolchain's release, and the set of available targets depends on what the upstream distribution manifest publishes for that release and host.

## Installing and inspecting targets

You can install a cross-compilation target when installing a host toolchain with `--target` (`-t`), or add it to an already-installed host toolchain with `cjv target add`:

```bash
cjv install sts --target ohos
cjv target list --toolchain sts
cjv target add android --toolchain sts
cjv target list --toolchain sts --installed
```

- **`cjv target add`** installs the requested target SDKs for the host toolchain's currently installed release without upgrading the host SDK itself.
- **`cjv target list`** queries the distribution manifest to show all targets published for that host release, marking which ones are already installed.
- **`cjv target list --installed`** reads local installation state only and works completely offline.

When naming a target on the command line or in [`cangjie-sdk.toml`](toolchain-file.md#components-and-targets), always write the target suffix—such as `ohos`, `android`, or `ohos-arm32`—rather than a full host-target platform key like `linux-x64-ohos`. Both `cjv install` and `cjv target add` accept repeated `--target` flags or comma-separated values (such as `--target ohos,android`).

## Using a target SDK

By default, the `cjc` and `cjpm` proxy binaries in `<CJV_HOME>/bin` select your host SDK environment. When you want to invoke tools or link against libraries from an installed cross SDK, load its environment into your current shell with `cjv envsetup`:

```bash
eval "$(cjv envsetup +sts --target=ohos)"
cjc --version
```

In Windows PowerShell:

```powershell
cjv envsetup +sts --target=ohos | Invoke-Expression
```

Keep in mind that installing a target SDK with `cjv` (or listing `targets` in `cangjie-sdk.toml`) prepares the cross SDK files on disk, but does not automatically turn an ordinary `cjpm build` into a cross-compiled build. Compiler flags, target sysroots, system libraries, and linker settings depend on your target platform and project build scripts; see the official Cangjie SDK documentation for target-specific compiler options, and see [Running commands and setting the environment](runtime-environment.md#loading-a-cross-compilation-sdk-environment) for how `envsetup` manages your shell session.

## Target components, updates, and removal

Each installed cross SDK has its own independent component set—host `stdx` libraries do not substitute for target `stdx` libraries. Once a target SDK is installed, pass `--target <suffix>` to `cjv component` commands to manage components for that target:

```bash
cjv component add stdx --toolchain sts --target ohos
cjv component list --toolchain sts --target ohos
cjv target remove ohos --toolchain sts
```

When you run `cjv update` on a tracked channel (`lts`, `sts`, `nightly`), `cjv` stages the updated host SDK, all of its installed cross targets, and all selected components before publishing them together in one atomic step. By default, if a required target or component is missing from the new release, `cjv` blocks the update and preserves your working installation (on `nightly`, it searches published history for a release that includes all required targets and components). If you want `cjv` to skip and remove unavailable optional targets or components during an update, pass `--force`; see the [Command reference](command-reference.md#installation-and-update-policies).

To remove a cross SDK and its target components while keeping the host toolchain intact, run `cjv target remove`. Because a cross SDK is always attached to a host toolchain, it cannot be set as the global default or active toolchain on its own.
