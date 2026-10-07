# Common questions

## Why is the wrong version selected?

Run `cjv show active` to see the selection and its source. Check `CJV_TOOLCHAIN`, directory overrides, and `cangjie-sdk.toml` in the current directory and its parents. cjv searches one level at a time, with an override taking precedence at the same level. See [project toolchains](toolchain-file.md).

If you evaluated `envsetup`, the shell may be calling a specific SDK directly. Open a new terminal to return to proxy selection, or load the environment for the required toolchain again.

## What happens when a project field is misspelled?

Unknown fields produce warnings. If the remaining fields do not provide a valid `channel` or `path`, the command fails instead of falling back to the default. Empty files also fail. Delete the file to restore parent-directory selection.

## My compiled program cannot find runtime libraries

```bash
cjv exec ./my_binary
```

You can also evaluate `cjv envsetup` for the current shell. See [runtime environments](runtime-environment.md) for shell-specific syntax.

## How do I install offline?

Use a [local SDK archive or directory](install-from-url.md). Installed toolchains run locally, but project dependencies must also be available. Managed clients can use an [internal distribution](enterprise/distribution-server.md) and disable automatic installation and update checks. See [enterprise deployment](enterprise/index.md).

## Why can't a custom toolchain download components?

A custom SDK has no official release record, so `component add` cannot identify its artifacts. Archive installations can include stdx or link local stdx. External directory links are maintained by the source SDK owner and cannot be edited with cjv component commands. See [components](concepts/components.md).

## What does uninstall remove?

`cjv uninstall <name>` removes the selected SDK, attached stdx and documentation, and directory overrides referring to it. If it is the default, cjv tries to select another installed host or clears the default. Link sources are preserved.

`cjv self uninstall` removes the cjv data directory and all installations, and cleans up PATH. See the [command reference](command-reference.md) for options.

## Why does the macOS download page ask for an architecture?

Some browsers cannot reliably report a Mac's CPU architecture. The installer script detects it locally. For manual downloads, run `uname -m`: `arm64` means Apple Silicon and `x86_64` means Intel.

## What happens when nightly has no checksum?

cjv reads versions and download URLs from `nightly.json`. If an SDK entry has no SHA-256, it tries `<url>.sha256`. If the sidecar is not published, cjv reports the integrity limitation. Internal mirrors can put checksums directly in the manifest; see the [distribution format](enterprise/distribution-server.md).
