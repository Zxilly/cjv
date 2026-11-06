# Running commands and setting the environment

The `cjc`, `cjpm`, and other executables in `<CJV_HOME>/bin` are proxies for SDK tools. Each proxy selects an SDK using the [project toolchain](toolchain-file.md) rules, prepares environment variables and library paths, and forwards command-line arguments and the child process exit code.

## Running SDK tools and compiled programs

In addition to calling proxy binaries like `cjc` directly, `cjv` provides two subcommands—`cjv run` and `cjv exec`—for running commands in an isolated child process environment:

```bash
cjc --version
cjv exec +sts cjc --version
cjv exec ./my_binary arg1 arg2
cjv exec +nightly ./my_binary
```

Understanding the difference between `run` and `exec` makes it easy to pick the right tool:

- **`cjv run [--install] <toolchain> <command> [args...]`** retains the explicit positional toolchain form and runs any program. It shares the runtime environment and `auto_install` policy with `exec`. `--install` explicitly enables installation, `--install=false` disables it, and enabling it conflicts with `--no-auto-install`.
- **`cjv exec [+toolchain] <command> [args...]`** is the recommended entry point, using the project selection or an explicit toolchain, setting SDK and runtime library paths, and installing missing requirements according to `auto_install`. An explicit toolchain replaces project selection; project components and targets apply when using the project selection.

Both `run` and `exec` modify the environment for the spawned child process only, and both forward `stdin`, `stdout`, `stderr`, and the child's exit code directly.

If you need to run a command whose executable name literally begins with a `+` character, pass `--` before the command name so `cjv exec` does not interpret it as a `+toolchain` selector:

```bash
cjv exec -- +command
```

`which` reads a tool path (or the SDK root with no command), without installation or requiring unrelated project components or targets.

## Configuring the current shell with `envsetup`

`cjv envsetup` prints a script that loads SDK paths and environment variables into the current shell. Use it with external build systems, Makefiles, or CMake scripts that need the SDK environment:

```bash
# Bash or Zsh
eval "$(cjv envsetup)"
```

```fish
# Fish
cjv envsetup | source
```

```powershell
# Windows PowerShell
cjv envsetup | Invoke-Expression
```

You can pass `+nightly` (or any installed toolchain name) to `cjv envsetup +nightly` to load a specific toolchain rather than the active selection, or pass `--shell=bash|fish|powershell|cmd` to choose the output syntax explicitly. When `--shell` is omitted, `cjv` detects your current shell automatically; if detection fails, it prints a notice to `stderr` and falls back to POSIX shell syntax.

For automation tools and IDE integrations, you can also run `cjv --json envsetup` to get the resolved environment variables as structured JSON; see [Automation and JSON interfaces](automation.md) for details.

`envsetup` reads installed SDKs and validates declared project components and targets. Missing requirements produce explicit installation hints without automatic installation. It accepts at most one valid `+toolchain` positional argument.

### Environment variables configured by `cjv`

Whether you run a command via a proxy, `cjv run`, `cjv exec`, or `cjv envsetup`, `cjv` configures the following variables for the selected SDK:

- **`CANGJIE_HOME`** — Points to the root directory of the selected SDK installation.
- **`PATH`** — Prepends the SDK's compiler and tool directories so that internal tools resolve directly to that SDK.
- **Dynamic library search paths** — Configures runtime library paths according to the host operating system: `LD_LIBRARY_PATH` on Linux and HarmonyOS, `DYLD_LIBRARY_PATH` on macOS, and `PATH` on Windows.
- **`stdx` library paths** — When the [`stdx` component](concepts/components.md) is installed for the selected toolchain, `cjv` automatically sets `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` to point to the component's `dynamic/` and `static/` directories.

### Switching projects after `envsetup`

Keep in mind that once you evaluate `cjv envsetup` in a terminal, your current shell session's `PATH` points directly to that specific SDK's `bin` directory ahead of `<CJV_HOME>/bin`. As a result, `cd`-ing into another project with a different `cangjie-sdk.toml` will not switch compilers in that terminal session.

To return to automatic per-project proxy selection, open a new terminal window—or evaluate `cjv envsetup` again in the new directory to switch the active session.

## Loading a cross-compilation SDK environment

When cross-compiling for another platform, you can use `cjv envsetup` with the `--target` flag to load the paths and libraries of an installed cross SDK:

```bash
cjv install sts --target ohos
eval "$(cjv envsetup +sts --target=ohos)"
```

`--target` selects the installed cross SDK whose release matches the selected host toolchain. Note that `envsetup --target` does not install a missing target automatically, nor does it provide an emulator to execute binaries built for another operating system or CPU architecture.

Consult the official Cangjie SDK documentation for the compiler flags and target linker requirements of your target platform, and see the [Cross-compilation](cross-compilation.md) chapter for how to install and manage target SDKs with `cjv`.
