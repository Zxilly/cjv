---
name: cjv
description: "Use cjv to install, select, and troubleshoot Cangjie SDK toolchains, including cangjie-sdk.toml, components, targets, and runtime setup."
---

# cjv

Help users operate cjv and choose commands appropriate to their shell and
requested outcome. Match the user's language.

## Work from current state

- For an existing setup, start with the relevant subset of `cjv --version`,
  `cjv show active`, `cjv show installed`, and `cjv show home`.
- Treat the installed `cjv <command> --help` as authoritative. Use the
  [Chinese](https://cjv.zxilly.dev/book/user-guide/zh-CN/) or
  [English](https://cjv.zxilly.dev/book/user-guide/en/) manual for more detail.
- Match action to authorization: use read-only inspection for explanation and
  diagnosis, and make persistent changes when the user requests them.

## Preserve these semantics

- Use `cjv run <toolchain> ...` for one command with an explicit SDK. Use
  `cjv exec [+toolchain] ...` for a compiled program that needs Cangjie runtime
  libraries. Use `cjv envsetup` only when the current shell needs that runtime.
- Automatic selection first honors `CJV_TOOLCHAIN`, then walks upward from the
  current directory considering overrides and `cangjie-sdk.toml`, then falls
  back to `cjv default`. At one directory an override wins, but a nearer project
  file wins over a farther ancestor override.
- A found `cangjie-sdk.toml` is not merged with parent files and must have a
  non-empty `[toolchain].channel`. Prefer an exact version for reproducible
  projects; use `lts`, `sts`, or `nightly` only when tracking a channel is wanted.
- `components` and `targets` in that file apply only when it selected the active
  toolchain. Targets extend the host SDK and use suffixes such as `ohos` or
  `android` rather than full platform tuples.
- `cjv toolchain link` preserves a linked source directory. Archive or URL
  installs are cjv-managed copies and are removed when that toolchain is removed.

## Protect and verify state

- `cjv init` and first installation may change `PATH`; mention when a new shell
  or a no-modify option is needed. Redact proxy credentials.
- Before `cjv uninstall`, report that its managed components and docs are also
  removed. Treat `cjv self uninstall` as removal of cjv and all managed SDK data.
- Verify selection with `cjv show active`, resolution with `cjv which cjc`, and
  the SDK with `cjc --version` or `cjv run <toolchain> cjc --version`.
- Prefer `--json` for automation where supported. `run`, `exec`, and `init`
  emit their native output. Summarize persistent changes after execution.
