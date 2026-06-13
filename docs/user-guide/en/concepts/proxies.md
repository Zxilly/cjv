# Proxies

You will rarely type `cjv run` directly to invoke the Cangjie SDK tools. Most of the time you run `cjc`, `cjpm`, `cjfmt` directly, just as you would with an ordinary SDK installation, and cjv forwards the call to the right toolchain behind the scenes. This mechanism is called proxying.

Proxying is what lets cjv switch between multiple toolchains seamlessly. When you change the default toolchain, set a directory override, or place a [toolchain file](../toolchain-file.md) in a project, the next run of `cjc` lands on the corresponding toolchain automatically, without changing `PATH` and without reactivating anything.

## Proxy symbolic links

cjv installs into `<CJV_HOME>/bin/`, which is added to `PATH` on first install (you can skip this step with `CJV_NO_PATH_SETUP=1`). In this bin directory, cjv creates a symlink to `cjv` itself for each supported SDK tool (on Windows, if a symlink cannot be created, it falls back to a directory junction or equivalent). These links are created together when a toolchain is installed.

The tools currently proxied are:

- `cjc`, `cjc-frontend`: the compiler
- `cjpm`: the package manager
- `cjfmt`: the formatter
- `cjlint`: static analysis
- `cjdb`: the debugger
- `cjcov`: the coverage tool
- `cjprof`: the profiler
- `cjtrace-recover`, `chir-dis`, `hle`
- `LSPServer`, `LSPMacroServer`: the language service

When you type `cjc` in a terminal, what the shell finds on `PATH` is actually the `<CJV_HOME>/bin/cjc` link (`cjc.exe` on Windows). It points to `cjv`, so what really runs is the cjv binary itself. cjv uses its own `argv[0]` (the name it was invoked by) to recognize that this time it was called as `cjc`, so it enters proxy mode instead of parsing a subcommand. The same applies to `cjpm`, `cjfmt`, and the other tools. The `bin/` directory contains only one real executable, cjv, and everything else is a same-named link.

## Tool resolution

Once in proxy mode, cjv decides which binary to run in the following steps:

1. Determine the tool name. Take the base name from `argv[0]`, dropping the `.exe` suffix on Windows, for example `cjc`, `cjpm`.
1. Resolve the active toolchain. Using the exact same priority order as `cjv run`, `cjv exec`, `cjv envsetup` (see [Targets and overrides](targets-overrides.md)), from highest to lowest:
   1. the `+toolchain` selector (see below)
   1. the `CJV_TOOLCHAIN` environment variable
   1. directory override (set with `cjv override set`, see [Targets and Overrides](targets-overrides.md))
   1. toolchain file (`cangjie-sdk.toml` in the current or a parent directory, see [Toolchain File](../toolchain-file.md))
   1. the default toolchain (set with `cjv default`)
1. Locate the tool binary. Under the resolved toolchain directory, build the tool path from a fixed layout, for example `cjc` is at `bin/cjc` and `cjpm` is at `tools/bin/cjpm`.
1. Inject the runtime environment. Proxying automatically sets up the runtime environment for that toolchain, including library search paths; if that toolchain has the `stdx` [component](components.md) installed, it also injects `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` (see [Runtime environment](../runtime-environment.md)).
1. Replace execution. cjv hands control over to the real tool binary, passing through the remaining arguments, standard input/output, exit code, and signals unchanged. From the caller's point of view it is as if the SDK's own `cjc` had been run directly.

The following two commands are equivalent:

```bash
# Direct invocation (via proxy)
cjc --version

# Run with an explicitly specified toolchain
cjv run lts cjc --version   # Assuming the currently resolved active toolchain is lts
```

To see which binary a tool ultimately lands on, use `cjv which`:

```bash
cjv which cjc
# Print the real path of cjc in the active toolchain
```

### The `+toolchain` selector

Proxy mode supports temporarily specifying a toolchain by placing `+` at the very front of the arguments; it takes priority over all other resolution methods and applies only to this single invocation:

```bash
# Compile with the nightly toolchain, regardless of the current default/override/toolchain file
cjc +nightly main.cj

# Run a build once with sts
cjpm +sts build
```

The toolchain name after `+` must not be empty, otherwise an error is reported. This syntax is consistent with the `+toolchain` used in `cjv exec` and `cjv envsetup`.

## `auto_install`: filling in missing items automatically

During proxying, the resolved toolchain (or the targets and components it declares) may not be installed yet. The behavior in this case is decided by the `auto_install` setting.

`auto_install = true` is the default. cjv installs the missing parts before forwarding the call, then executes as usual. After cloning a project that has a `cangjie-sdk.toml`, running `cjpm build` directly triggers the first install, with no need to run `cjv install` by hand.

With `auto_install = false`, when it encounters an uninstalled toolchain, target, or component, cjv exits with an error and does not download anything.

Auto-install covers three kinds of missing items on demand:

1. The active toolchain itself is installed automatically when missing.
1. Cross-compilation target SDKs declared in `targets` in the toolchain file are filled in automatically when missing (see [Cross-compilation](../cross-compilation.md)).
1. Components declared in `components` in the toolchain file (such as `stdx`, `docs`) are installed automatically when missing (see [Components](components.md)).

For example, suppose a project's toolchain file is as follows:

```toml
[toolchain]
channel = "nightly"
targets = ["ohos"]
components = ["stdx", "docs"]
```

Running any proxied tool for the first time on a machine with `auto_install` enabled:

```bash
cjpm build
```

cjv checks in turn whether the `nightly` toolchain, the `ohos` target SDK, and the `stdx` and `docs` components are ready, installs any that are missing, and then runs `cjpm build`. Auto-install progress is printed to standard error, so it does not pollute the tool's own standard output.

### Toggling `auto_install`

```bash
# Disable and enable (written to settings.toml)
cjv set auto-install false
cjv set auto-install true
```

This setting is stored in the `auto_install` field of `<CJV_HOME>/settings.toml`, with a default value of `true`. A system-level fallback settings file may also provide this field; see [Configuration](../configuration.md).

### Cases that are not auto-installed

A custom toolchain linked via `cjv toolchain link` has no corresponding downloadable release assets, so cjv will not and cannot auto-install it; if it resolves to an unlinked custom name, it errors out directly.

The cross-compilation target SDK for `cjv exec` must first be installed with `cjv install <toolchain> --target <suffix>`. The proxy path only fills in the `targets` declared in the toolchain file; it will not install a target SDK out of nowhere for a one-off command.

When any download or install step during auto-install fails, cjv does not continue forwarding the call. Instead it exits with a toolchain-or-component-not-installed error and prints the reason for the failure on standard error.

## Recursion protection

A proxied tool ultimately runs the real SDK, and some tools may in turn invoke proxied commands such as `cjc`. To avoid falling into infinite self-invocation when the configuration is broken, cjv limits how deeply proxies may nest and aborts with a recursion-limit error once that limit is exceeded. In normal use you will not reach this limit.
