# Run commands and set the environment

After you [install cjv](installation/index.md), proxy commands such as `cjc` and `cjpm` live in `<CJV_HOME>/bin`. When you run a proxy command, cjv selects an SDK, sets the runtime environment, and runs the tool. This chapter explains how to run SDK tools, start compiled programs, and configure your current shell session.

## Compare execution methods

Choose the command that matches your task:

| Method | Example | Scope | Typical use |
| --- | --- | --- | --- |
| Proxy binary | `cjc --version` | Child process | Daily compilation and package management |
| `cjv run` | `cjv run --install sts cjc --version` | Child process | Run a tool once with a specific toolchain |
| `cjv exec` | `cjv exec ./my_binary arg1` | Child process | Run a compiled Cangjie program that needs runtime libraries |
| `cjv envsetup` | `eval "$(cjv envsetup)"` | Current shell session | Load SDK paths and variables into your active terminal |

## Run SDK tools and compiled programs

### Use proxy commands and `cjv run`

Proxy commands in `<CJV_HOME>/bin` obey the [Project toolchains](toolchain-file.md) selection order. To run a command with an explicit toolchain instead, use `cjv run`:

```bash
cjc --version
cjv run --install sts cjc --version
```

The `run` command searches the tools of the selected SDK first, and then searches the `PATH` of that environment. It installs a missing toolchain only when you pass `--install`.

### Run compiled programs with `cjv exec`

Compiled Cangjie binaries often need dynamic libraries from the SDK or from `stdx`. Use `cjv exec` to run a program with the runtime environment of the active SDK or of a `+name` toolchain:

```bash
cjv exec ./my_binary arg1 arg2
cjv exec +nightly ./my_binary
```

Both `run` and `exec` change the environment for the child process only. They forward standard input, output, error streams, and exit codes directly. If the command name starts with `+`, pass `--` before the command name:

```bash
cjv exec -- +command
```

This separator stops cjv from reading `+command` as a toolchain selector.

## Configure the current shell session

### Evaluate `cjv envsetup`

The `cjv envsetup` command prints a shell script that exports SDK variables into your current terminal. Evaluate its output in your shell:

```bash
# Bash or Zsh
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

Pass `+nightly` to select a specific toolchain, or pass `--shell=bash|fish|powershell|cmd` to select the script syntax. If you omit `--shell`, cjv detects your shell automatically. If shell detection fails, cjv prints a warning and uses POSIX syntax. Run `cjv --json envsetup` to get structured environment data for scripts; read [Automation and JSON interfaces](automation.md) for details.

### Variables set in the environment

The SDK environment sets `CANGJIE_HOME`, adds SDK tool directories to `PATH`, and configures dynamic library paths:

- **Linux**: sets `LD_LIBRARY_PATH`.
- **macOS**: sets `DYLD_LIBRARY_PATH`.
- **Windows**: adds runtime library directories to `PATH`.
- **Installed `stdx`**: sets `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC`. Read [Components and offline documentation](concepts/components.md) for `stdx` management.

### Restore project-based selection

After you evaluate `cjv envsetup`, the `PATH` in your current shell points directly to one SDK directory. Proxy selection from `cangjie-sdk.toml` will not take effect in that terminal while the SDK directory precedes `<CJV_HOME>/bin`.

Open a new terminal to return to automatic proxy selection, or evaluate `cjv envsetup` again to switch the active session to another SDK.

## Load a cross-compilation environment

When you compile for another target platform, you can load the environment of an installed cross SDK:

```bash
cjv install sts --target ohos
eval "$(cjv envsetup +sts --target=ohos)"
```

The `--target` flag uses the tool paths and runtime libraries of the cross SDK that matches the release of the selected host. It does not install a missing target automatically, and it does not emulate another platform to run target binaries.

Consult the Cangjie SDK documentation for target compiler flags and device runtime setup. Read [Cross-compilation](cross-compilation.md) for instructions on how to install and manage target SDKs in cjv.
