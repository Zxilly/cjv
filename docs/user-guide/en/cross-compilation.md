# Cross-compilation

A cross SDK provides tools on the current host for compiling to another platform. cjv manages it as an addition to a host toolchain. Its release must match the host, and available targets depend on the distribution manifest.

## Install and inspect

```bash
cjv install sts --target ohos
cjv target list --toolchain sts
cjv target add android --toolchain sts
cjv target list --toolchain sts --installed
```

`target add` supplies targets for the installed host release without upgrading it. `target list --installed` reads local state and works offline. The regular list queries targets published for that release.

Use suffixes such as `ohos`, `android`, or `ohos-arm32`, not a full platform name such as `linux-x64-ohos`. Installation accepts repeated `--target` flags or comma-separated values.

## Use the target SDK

The default `cjc` and `cjpm` proxies still select the host SDK. To call tools from a cross SDK, load its environment in the current shell:

```bash
eval "$(cjv envsetup +sts --target=ohos)"
cjc --version
```

In PowerShell:

```powershell
cjv envsetup +sts --target=ohos | Invoke-Expression
```

Compiler options, system libraries, and linker configuration depend on the target SDK and project. Installing a target or declaring `targets` in the project file prepares SDKs; it does not turn a regular `cjpm build` into a target-platform build.

## Components, updates, and removal

```bash
cjv component add stdx --toolchain sts --target ohos
cjv component list --toolchain sts --target ohos
cjv target remove ohos --toolchain sts
```

`component --target` manages the cross SDK's own components and requires the target to be installed. Host components do not substitute for target components.

Updating a tracked channel prepares its host, tracked targets, and components before publishing them together. Missing required artifacts block updates by default; nightly may select a compatible release from published history. `--force` allows unavailable optional entries to be skipped and removed. See the [command reference](command-reference.md).

`target remove` removes the cross SDK and its components while keeping the host. Cross SDKs cannot be set as the default or active toolchain.
