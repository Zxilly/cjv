# Cross-compilation

Cangjie supports cross-compilation: producing executables for another platform (such as OpenHarmony, Android) on a host machine. In addition to the host toolchain, this requires the target SDK (cross-compilation SDK) for that platform.

This chapter covers how to install, declare, and use a target SDK. For where `targets` and directory overrides sit in toolchain resolution, see [Targets and overrides](concepts/targets-overrides.md).

## A target SDK is an add-on install

A target SDK is not a separate toolchain but an additional install attached to a host toolchain. Installing a target SDK does not change the active toolchain, nor does it change `cjv default`. When you call `cjc`, `cjpm`, and other tools directly ([proxy mode](concepts/proxies.md)), the host SDK is still used; the target SDK is used only when you explicitly request a cross-compilation environment.

A target SDK's version is locked to the version the host toolchain has resolved to. If that version has no matching target asset, the install fails rather than installing a version-mismatched SDK. `cjv install sts -t ohos` gives you the STS host SDK plus the matching OHOS cross SDK, with the host development experience unchanged.

## Installing a target SDK

Use the `-t` / `--target` flag of `cjv install` to attach cross-compilation targets while installing a host toolchain:

```bash
# Install the host STS SDK, and additionally install the OHOS cross SDK matching the current host
cjv install sts -t ohos
```

There are two equivalent ways to install multiple targets at once, and they can be mixed:

```bash
# Repeated flags
cjv install sts -t ohos -t android

# Comma-separated
cjv install sts --target ohos,android

# Mixing the two also works
cjv install sts -t ohos,android -t ohos-arm32
```

`--target` accepts only the target suffix, for example `ohos`, `android`, `ohos-arm32`. Do not write a full platform key (such as `linux-x64-ohos`). cjv fills in the platform prefix automatically based on the host.

A target SDK can be installed together with [components](concepts/components.md) in the same command:

```bash
# Host STS + OHOS cross SDK + stdx component
cjv install sts -t ohos -c stdx
```

 >
 > A target SDK is additive: you can run `cjv install <tc> -t <new-suffix>` again on an already installed toolchain at any time to add new targets, and the parts already installed are unaffected.

## Declaring targets in the toolchain file

A project can write cross-compilation targets into the `[toolchain]` table of `cangjie-sdk.toml`, so collaborators do not have to remember the install command. As on the command line, `targets` takes only the suffix:

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

`targets` is additional semantics: it declares which target SDKs are needed on top of the host toolchain, and does not change the active toolchain that `channel` resolves to.

When `auto_install` is enabled in settings, [proxy execution](concepts/proxies.md) automatically installs any missing target SDK before invoking an SDK tool; when it is disabled, you need to install them manually with the `cjv install … -t …` shown above. For the full semantics of the `targets` field, see the [toolchain file](toolchain-file.md) and [Targets and overrides](concepts/targets-overrides.md).

## The standalone-SDK model and `cjv envsetup --target`

Each target SDK is self-contained: it has its own `CANGJIE_HOME`, its own `bin` directory, and its own runtime library paths. To enter the cross-compilation environment of a target SDK, pass `--target=SUFFIX` to [`cjv envsetup`](runtime-environment.md):

```bash
# Output the OHOS cross-compilation environment (standalone-SDK model)
eval "$(cjv envsetup --target=ohos)"

# Other shells
cjv envsetup --target=ohos | source             # Fish
cjv envsetup --target=ohos | Invoke-Expression   # PowerShell
```

Without `--target`, the environment that is output points at the host toolchain. With `--target`, the whole output environment is redirected to the target SDK directory: `CANGJIE_HOME` points at the target SDK's own directory rather than the host toolchain directory; `PATH` and the library search paths all come from that target SDK directory. Logically the same host toolchain is still in use (such as `lts-1.0.5`); only the underlying root directory is switched to the cross SDK.

`--target` also follows the same toolchain resolution priority as proxy mode, and supports the `+toolchain` syntax for specifying the host toolchain:

```bash
# Output the OHOS cross environment for the +nightly host toolchain
eval "$(cjv envsetup +nightly --target=ohos)"
```

 >
 > Note: `cjv envsetup --target` does not install the target SDK automatically. The corresponding target must already be installed via `cjv install <toolchain> --target <suffix>`, or the command reports an error.

Once the environment is set up, you can invoke the cross-compilation toolchain directly:

```bash
eval "$(cjv envsetup --target=ohos)"
cjc --version          # Here cjc comes from the OHOS target SDK
cjpm build             # The output targets the OHOS platform
```

For environment variable injection, the syntax for different shells, and the trade-off between one-off execution (`cjv exec`) and configuring the current session (`cjv envsetup`), see [Runtime environment](runtime-environment.md).

## Uninstalling

A target SDK is cleaned up together with its host toolchain. When you uninstall the host toolchain, the target SDKs attached to it are removed along with it:

```bash
cjv toolchain uninstall sts
```
