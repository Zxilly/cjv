# Common questions

This chapter answers frequent questions about toolchain selection, project files, runtime libraries, offline setups, and platform downloads.

## Toolchain selection and project files

### Why did cjv select the wrong SDK version?

Run `cjv show active` to see the selected toolchain and the rule that chose it. Next, check these sources in order:

1. Command-line selectors (`+name` or `--toolchain`).
2. The `CJV_TOOLCHAIN` [environment variable](environment-variables.md).
3. Local directory overrides (`cjv override list`) and `cangjie-sdk.toml` files in the current directory and parent directories. At the same directory level, an override takes precedence over `cangjie-sdk.toml`. Read [Project toolchains](toolchain-file.md#toolchain-selection-order).
4. If you evaluated `cjv envsetup` earlier in the current terminal, your shell `PATH` points directly to that SDK. Open a new terminal to return to automatic proxy selection, or evaluate `cjv envsetup` again for the toolchain you want.

### What happens if `cangjie-sdk.toml` has a misspelled field?

cjv prints a warning for unknown keys and continues to parse known keys. However, if the file has invalid TOML syntax, an empty body, or no valid `channel` or `path`, cjv fails with an error instead of falling back to the global default. Delete the broken `cangjie-sdk.toml` file if you want to restore parent-directory or default selection.

## Runtime environment and components

### Why does my compiled program fail to find runtime libraries?

Compiled Cangjie binaries often link against dynamic libraries in the SDK or in `stdx`. Run your binary through `cjv exec` so cjv sets the platform library path (`LD_LIBRARY_PATH`, `DYLD_LIBRARY_PATH`, or `PATH`) and `stdx` variables for the child process:

```bash
cjv exec ./my_binary
```

You can also evaluate `cjv envsetup` to load those variables into your current shell session. Read [Run commands and set the environment](runtime-environment.md) for shell-specific commands.

### Why can't a custom toolchain download components with `component add`?

A custom toolchain registered with `cjv toolchain link` has no official release entry in the upstream manifest, so `cjv component add` cannot look up component archives for it:

- If you installed the custom SDK from a [local archive or URL](install-from-url.md), it can install bundled `stdx` automatically or accept a local `stdx` directory via `cjv component link stdx`.
- If you linked an unpacked external SDK directory, the owner of that directory maintains its contents and `cjv component` commands cannot edit it. Read [Components and offline documentation](concepts/components.md#manage-components-on-custom-sdks).

## Installation, offline use, and removal

### How do I install and use cjv on an offline machine?

Choose the method that matches your environment:

- **Pre-populated cache**: Populate `<CJV_HOME>/cache/` on a connected machine with `cjv install`, copy `cache/` and `cangjie-sdk.toml` to the offline machine, and run with `--offline`. Read [Automation and JSON interfaces](automation.md#ci-isolation-and-offline-caches).
- **Local archive or directory**: Register a local SDK `.zip`, `.tar.gz`, or unpacked directory with `cjv toolchain link`. Read [Custom SDKs](install-from-url.md).
- **Enterprise internal network**: Host manifests and archives on an [internal distribution server](enterprise/distribution-server.md) and disable automatic installation and self-updates in system fallback settings. Read [Enterprise and offline deployment](enterprise/index.md).

Note that installed SDKs run locally, but you must also cache or vendor your `cjpm` project dependencies for offline builds.

### What files do `cjv uninstall` and `cjv self uninstall` delete?

- `cjv uninstall <name>` deletes the named SDK, its attached cross targets, its `stdx` and documentation components, and any directory overrides that reference `<name>`. If `<name>` was your default toolchain, cjv selects another installed host SDK or clears the default. External directories linked with `toolchain link` or `component link` are never deleted.
- `cjv self uninstall` deletes the entire `<CJV_HOME>` data directory, including all installed toolchains, components, caches, and snapshots, and cleans up `PATH`. Read [Command reference](command-reference.md#self-update-and-self-uninstall).

### Why does the macOS download page ask me to choose an architecture?

Many macOS browsers report an Intel user agent even on Apple Silicon Macs, so a web page cannot always detect the CPU architecture reliably. If you run the `install.sh` script in your terminal, it detects your native CPU architecture automatically. For a manual archive download, run `uname -m` in your terminal: `arm64` means Apple Silicon and `x86_64` means Intel.

### What happens when a nightly release has no SHA-256 in `nightly.json`?

cjv reads nightly versions and URLs from `nightly.json`. If an upstream nightly entry has an empty `sha256` field, cjv fetches `<url>.sha256` automatically. If that sidecar file is not published yet, cjv reports the integrity limitation. On an internal enterprise mirror, always write the 64-character SHA-256 directly into `nightly.json`; read [Internal distribution](enterprise/distribution-server.md#nightlyjson-schema).
