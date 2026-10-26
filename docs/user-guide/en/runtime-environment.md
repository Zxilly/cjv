# Running commands and setting the environment

When `cjv` is installed, it places wrapper executables—called _proxies_—for standard Cangjie SDK tools such as `cjc` and `cjpm` into `<CJV_HOME>/bin`. Whenever you invoke one of these proxies, `cjv` determines which [project toolchain](toolchain-file.md) is active for your current directory, prepares the SDK's environment variables and library paths, and forwards your command-line arguments and exit code transparently.

## Running SDK tools and compiled programs

In addition to calling proxy binaries like `cjc` directly, `cjv` provides two subcommands—`cjv run` and `cjv exec`—for running commands in an isolated child process environment:

```bash
cjc --version
cjv run --install sts cjc --version
cjv exec ./my_binary arg1 arg2
cjv exec +nightly ./my_binary
```

Understanding the difference between `run` and `exec` makes it easy to pick the right tool:

- **`cjv run [--install] <toolchain> <command> [args...]`** explicitly selects the `<toolchain>` you name on the command line. It searches that SDK's `bin` and tool directories first, and then searches the `PATH` of that environment. It only installs a missing toolchain when you pass `--install` before the toolchain name.
- **`cjv exec [+toolchain] <command> [args...]`** prepares the runtime environment for the currently active toolchain (or for `+toolchain` if specified), including the dynamic library paths required by compiled Cangjie programs and `stdx`. This makes `cjv exec` the ideal way to run your own compiled binaries without polluting your interactive shell environment.

Both `run` and `exec` modify the environment for the spawned child process only, and both forward `stdin`, `stdout`, `stderr`, and the child's exit code directly.

If you need to run a command whose executable name literally begins with a `+` character, pass `--` before the command name so `cjv exec` does not interpret it as a `+toolchain` selector:

```bash
cjv exec -- +command
```

## Configuring the current shell with `envsetup`

Sometimes—for example, when working with external build systems, custom Makefiles, or CMake scripts—you want the SDK's tool paths and environment variables loaded directly into your current shell session. The `cjv envsetup` command prints a shell script that you can evaluate in your terminal:

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

### Environment variables configured by `cjv`

Whether you run a command via a proxy, `cjv run`, `cjv exec`, or `cjv envsetup`, `cjv` configures the following variables for the selected SDK:

- **`CANGJIE_HOME`** — Points to the root directory of the selected SDK installation.
- **`PATH`** — Prepends the SDK's compiler and tool directories so that internal tools resolve directly to that SDK.
- **Dynamic library search paths** — Configures runtime library paths according to the host operating system: `LD_LIBRARY_PATH` on Linux and HarmonyOS, `DYLD_LIBRARY_PATH` on macOS, and `PATH` on Windows.
- **`stdx` library paths** — When the [`stdx` component](concepts/components.md) is installed for the selected toolchain, `cjv` automatically sets `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` to point to the component's `dynamic/` and `static/` directories.

### Switching projects after `envsetup`

Keep in mind that once you evaluate `cjv envsetup` in a terminal, your current shell session's `PATH` points directly to that specific SDK's `bin` directory ahead of `<CJV_HOME>/bin`. As a result, `cd`-ing into another project with a different `cangjie-sdk.toml` will not switch compilers in that terminal session.

To return to automatic per-project proxy selection, simply open a new terminal window—or evaluate `cjv envsetup` again in the new directory to switch the active session.

## Loading a cross-compilation SDK environment

When cross-compiling for another platform, you can use `cjv envsetup` with the `--target` flag to load the paths and libraries of an installed cross SDK:

```bash
cjv install sts --target ohos
eval "$(cjv envsetup +sts --target=ohos)"
```

`--target` selects the installed cross SDK whose release matches the selected host toolchain. Note that `envsetup --target` does not install a missing target automatically, nor does it provide an emulator to execute binaries built for another operating system or CPU architecture.

Consult the official Cangjie SDK documentation for the compiler flags and target linker requirements of your target platform, and see the [Cross-compilation](cross-compilation.md) chapter for how to install and manage target SDKs with `cjv`.
