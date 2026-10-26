# Common questions

## Why is the wrong SDK version selected?

Run `cjv show active` to see which toolchain is currently active and which rule selected it. Then check the selection sources in priority order:

1. Command-line selectors (`+name` or `--toolchain`).
2. The `CJV_TOOLCHAIN` [environment variable](environment-variables.md).
3. Per-directory overrides (`cjv override list`) and `cangjie-sdk.toml` files in your current directory and its parent directories. Remember that `cjv` walks upward one directory level at a time, and an override takes precedence over `cangjie-sdk.toml` at the same directory level. See [Project toolchains](toolchain-file.md#selection-order).
4. If you evaluated `cjv envsetup` earlier in the current terminal session, your shell's `PATH` points directly to that SDK's `bin/` directory ahead of `<CJV_HOME>/bin`. Open a new terminal window to return to automatic proxy selection, or run `cjv envsetup` again for the toolchain you want.

## What happens when a field in `cangjie-sdk.toml` is misspelled?

Unrecognized fields under `[toolchain]` produce a warning on `stderr` while recognized fields are still parsed. However, if the remaining fields do not provide a valid `channel` or `path`—or if the file is empty or contains invalid TOML syntax—the command fails with an error instead of silently falling back to the global default toolchain. Delete the broken `cangjie-sdk.toml` file if you want to restore selection from a parent directory or the default toolchain.

## My compiled program cannot find runtime libraries

Compiled Cangjie binaries often dynamically link against runtime libraries in the SDK or in `stdx`. Use `cjv exec` to run your binary with the active toolchain's library paths configured automatically:

```bash
cjv exec ./my_binary
```

Alternatively, you can evaluate `cjv envsetup` in your current shell session so that `LD_LIBRARY_PATH` (Linux/HarmonyOS), `DYLD_LIBRARY_PATH` (macOS), or `PATH` (Windows)—along with `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC`—are exported directly into your terminal. See [Running commands and setting the environment](runtime-environment.md) for shell-specific commands.

## How do I install and use `cjv` offline?

Depending on your environment, you have three options:

- **Pre-populated cache (`--offline`)** — Populate `<CJV_HOME>/cache/` on a connected machine with `cjv install`, copy `cache/` and `cangjie-sdk.toml` to the offline machine, and run with `--offline` (or `CJV_OFFLINE=1`). See [Automation and JSON interfaces](automation.md#configuration-and-network-access).
- **Local SDK archive or directory (`toolchain link`)** — Register a local `.zip` / `.tar.gz` SDK archive or an unpacked SDK directory with `cjv toolchain link`. See [Custom SDKs](install-from-url.md).
- **Internal enterprise mirror (`dist_server`)** — Host manifests and archives on an [internal distribution server](enterprise/distribution-server.md) and disable automatic installation and self-update checks in system fallback settings. See [Enterprise and offline deployment](enterprise/index.md).

Keep in mind that while installed toolchains run completely locally, any external package dependencies used by `cjpm` must also be prepared in advance for offline builds.

## Why can't a custom toolchain download components with `component add`?

A custom toolchain registered with `cjv toolchain link` has no official release entry in the upstream distribution manifest, so `cjv component add` cannot look up component artifacts for it. Instead:

- If you installed the custom toolchain from a [local archive or URL](install-from-url.md), it can automatically install a bundled `stdx` archive or have a local `stdx` directory attached via `cjv component link stdx /path/to/stdx --toolchain <name>`.
- If you linked an unpacked external SDK directory, that directory is maintained by its owner and cannot be modified with `cjv component` commands. See [Components and offline documentation](concepts/components.md#managing-components-on-custom-sdks).

## What does `uninstall` remove?

- **`cjv uninstall <name>`** removes the selected SDK installation, its attached cross-compilation targets, its `stdx` and offline documentation components, and any directory overrides referring to `<name>`. If `<name>` was your global default toolchain, `cjv` tries to select another installed host toolchain as the default or clears the default setting. External source directories linked via `toolchain link` or `component link` are always preserved.
- **`cjv self uninstall`** removes the entire `<CJV_HOME>` data directory—including all installed toolchains, components, caches, and rollback snapshots—and cleans up `PATH`. See the [Command reference](command-reference.md#self-update--self-uninstall) for options.

## Why does the macOS download page ask me to choose an architecture?

Many macOS browsers report an Intel (`x86_64`) User-Agent string even when running on Apple Silicon (`arm64`) hardware, so a web page cannot always determine your Mac's native CPU architecture reliably. If you run the `install.sh` script in your terminal, it detects your native architecture locally. For manual archive downloads, run `uname -m` in your terminal: `arm64` means Apple Silicon and `x86_64` means Intel.

## What happens when a nightly release has no checksum in `nightly.json`?

`cjv` reads nightly versions and download URLs from `nightly.json`. If an upstream nightly SDK entry temporarily has an empty `sha256` field, `cjv` automatically attempts to fetch `<url>.sha256`. If that sidecar file has not been published yet, `cjv` reports the integrity limitation. When operating an internal enterprise mirror, you can compute and write the 64-character SHA-256 directly into `nightly.json`; see [Internal distribution](enterprise/distribution-server.md#nightlyjson-contract).
