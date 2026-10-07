# Running commands and setting the environment

After installation, commands such as `cjc` and `cjpm` in `<CJV_HOME>/bin` invoke cjv. It follows the [project toolchain](toolchain-file.md) selection rules, prepares the SDK environment, and forwards arguments and exit codes.

## SDK tools and your programs

```bash
cjc --version
cjv run --install sts cjc --version
cjv exec ./my_binary arg1 arg2
cjv exec +nightly ./my_binary
```

`run` selects a toolchain explicitly, searches its SDK tools and then its environment's PATH, and installs a missing version only with `--install`. `exec` prepares the active or `+name` SDK's environment, including library paths needed by compiled Cangjie programs. Both affect only the child process and forward standard streams and exit codes.

For a command whose name starts with `+`, use `cjv exec -- +command` to prevent it from being read as a toolchain selector.

## Configure the current shell

`envsetup` prints a script to evaluate in the current shell:

```bash
# Bash / Zsh
eval "$(cjv envsetup)"
```

```fish
cjv envsetup | source
```

```powershell
cjv envsetup | Invoke-Expression
```

Use `cjv envsetup +nightly` to select a version, or `--shell=bash|fish|powershell|cmd` to select an output format. Without `--shell`, cjv detects the shell; if detection fails, it reports this and falls back to POSIX syntax. `cjv --json envsetup` returns structured environment data.

The environment includes `CANGJIE_HOME`, SDK tool paths, and runtime library paths. Linux uses `LD_LIBRARY_PATH`, macOS uses `DYLD_LIBRARY_PATH`, and Windows uses `PATH`. Installed stdx provides `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC`.

After evaluating the output, the current session's PATH points to a specific SDK. Open a new terminal to return to project-based proxy selection, or run `envsetup` again to switch the session.

## Cross SDK environments

```bash
cjv install sts --target ohos
eval "$(cjv envsetup +sts --target=ohos)"
```

`--target` uses the paths and libraries of an installed cross SDK matching the selected host's release. It does not install the target or provide emulation for running another platform's programs. Consult the SDK documentation for compiler options and target runtime requirements. See [cross-compilation](cross-compilation.md) for cjv installation commands.
