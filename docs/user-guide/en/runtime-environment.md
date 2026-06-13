# Runtime Environment

Binaries compiled by Cangjie are not fully self-contained. They link dynamically to the runtime libraries shipped with the SDK (such as `libcangjie-runtime`), and may depend on other shared libraries the SDK provides. When you run these artifacts directly, the operating system needs to find these `.so`/`.dylib`/`.dll` files in the library search path, otherwise startup fails because the dynamic libraries cannot be found.

cjv's [proxies](concepts/proxies.md) inject these paths automatically when calling SDK tools such as `cjc` and `cjpm`, but the artifacts you compile yourself do not go through a proxy; you run them directly with `./my_binary`. In that case you need to prepare the runtime environment first. cjv provides two ways. `cjv exec` runs a single command in the correct runtime environment and restores afterward, without polluting the current shell. `cjv envsetup` outputs an environment-variable configuration script to the shell, configuring the current session persistently so you can then run compiled artifacts directly.

Both use the same [toolchain resolution priority](concepts/targets-overrides.md) as proxy mode, and both support the `+toolchain` syntax to specify a toolchain explicitly.

## What the runtime environment contains

Whichever way you use, what cjv injects all comes from the SDK directory of the current toolchain. `CANGJIE_HOME` points at that SDK's root directory. The SDK's `bin`, `tools/bin`, and similar directories are prepended to `PATH`. The runtime library directories are prepended to the platform's library search variable: on Linux that is `LD_LIBRARY_PATH`, on macOS `DYLD_LIBRARY_PATH`, and on Windows it goes through `PATH` (Windows has no separate library search variable). If the current toolchain has the `stdx` [component](concepts/components.md) installed, `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` are also injected.

## `cjv exec`: one-off execution

`cjv exec` runs the specified command in a prepared runtime environment and restores everything when it finishes, without altering your current shell:

```bash
cjv exec ./my_binary arg1 arg2
```

The child process's exit code is passed through unchanged, so `cjv exec` can be used in scripts and CI pipelines. Standard input, output, and error streams are also forwarded unchanged.

### Specifying a toolchain

Prefixing the command with `+toolchain` temporarily switches to the specified toolchain without changing the default or currently active toolchain:

```bash
# Execute using the runtime environment of the nightly toolchain
cjv exec +nightly ./my_binary
```

Without `+toolchain`, `cjv exec` selects the toolchain by the standard [resolution priority](concepts/targets-overrides.md) (`CJV_TOOLCHAIN` → directory override → `cangjie-sdk.toml` → default toolchain).

### Running commands that begin with `+`

The `+toolchain` selector consumes only the first argument. If the command you want to run itself starts with `+`, use the `--` terminator to separate it from the selector:

```bash
# Here +foo is the name of the command to run, not a toolchain
cjv exec -- +foo arg1
```

## `cjv envsetup`: configuring the current shell

`cjv envsetup` does not run a command directly; it prints the shell commands needed to configure the runtime environment to standard output. You need to feed its output to the current shell to evaluate, so the environment variables take effect in the current session. Once configured, you can run compiled artifacts directly any number of times in this session, without wrapping each one in `cjv exec`.

The way to evaluate it differs across shells:

```bash
# Bash / Zsh
eval "$(cjv envsetup)"
```

```fish
# Fish
cjv envsetup | source
```

```powershell
# PowerShell
cjv envsetup | Invoke-Expression
```

`cjv envsetup` likewise supports `+toolchain`:

```bash
eval "$(cjv envsetup +nightly)"
```

### Shell auto-detection and `--shell`

`cjv envsetup` determines the current shell type automatically by inspecting the parent process, and chooses the correct output syntax accordingly. It recognizes POSIX shells such as `bash`, `zsh`, `sh`, as well as `fish`, `powershell`/`pwsh`, `cmd`.

If auto-detection fails (for example in certain nested or non-interactive environments), cjv falls back to POSIX syntax and prints a hint to standard error. In that case, or when you want to generate a script for a different shell, specify it explicitly with `--shell`, whose value can be `bash`, `fish`, `powershell`, or `cmd`:

```bash
cjv envsetup --shell=fish | source
```

```powershell
cjv envsetup --shell=powershell | Invoke-Expression
```

## Cross-compilation: `--target`

The `--target=SUFFIX` flag of `cjv envsetup` outputs the runtime environment of an installed target SDK rather than the host SDK. This is useful when a [cross-compilation](cross-compilation.md) artifact needs to run, for example on a target device or emulator.

cjv uses a standalone-SDK model for target SDKs: `CANGJIE_HOME` points to the target SDK directory, and `PATH` and the library search paths are all taken from that directory, independent of the host SDK.

```bash
# Output the runtime environment of the installed ohos target SDK
eval "$(cjv envsetup --target=ohos)"
```

The target SDK must first be installed together with the host toolchain for `--target` to find it:

```bash
cjv install sts --target ohos
```

For the meaning of target suffixes (such as `ohos` and `android`), see [Cross-compilation](cross-compilation.md) and [Targets and overrides](concepts/targets-overrides.md).

## Related chapters

- [Proxies](concepts/proxies.md): how proxy mode injects the runtime environment for SDK tools automatically.
- [Components](concepts/components.md): the `stdx` component and the `CANGJIE_STDX_PATH_*` variables it injects.
- [Environment variables](environment-variables.md): all environment variables cjv deals with.
- [Cross-compilation](cross-compilation.md): installing and using cross-compilation target SDKs.
