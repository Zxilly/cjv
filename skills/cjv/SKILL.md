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
  non-empty `[toolchain].channel` or an absolute `[toolchain].path`, never both.
  A path selects an external SDK and ignores component/target auto-install.
  Prefer an exact version for reproducible
  projects; use `lts`, `sts`, or `nightly` only when tracking a channel is wanted.
- `components` and `targets` in that file apply only when it selected the active
  toolchain. Targets extend the host SDK and use suffixes such as `ohos` or
  `android` rather than full platform tuples.
- `cjv toolchain link` preserves a linked source directory. Archive or URL
  installs are cjv-managed copies and are removed when that toolchain is removed.

- Install/update/uninstall accept multiple names. A leading global `+name`
  selects the SDK for management commands; explicit `--toolchain` wins.
  `which --toolchain` and `target list/add/remove` follow this selection.
- `default` installs a missing official SDK before saving the selection.
  `check` queries SDK and cjv releases without installing updates.
- Install/update `--force` permits skipping and removing unavailable components
  or targets, without reinstalling unchanged SDKs. `install --no-update` keeps
  the current release. Nightly searches published history without downgrading
  unless `--allow-downgrade` is set. Hosts and tracking targets update together.
  Component and custom `toolchain link` replacement flags retain their own semantics.
- Use `component --target <suffix>` for an installed matching cross SDK.
  Global `--quiet/-q` hides progress and `--verbose` enables debug logging;
  honor an existing `CJV_LOG` value. Version selectors support minor patches,
  dated nightly, and host suffixes only when the manifest publishes them.

## Protect and verify state

- `cjv init` and first installation may change `PATH`; mention when a new shell
  or a no-modify option is needed. Redact proxy credentials.
- Before `cjv uninstall`, report that its managed components and docs are also
  removed. Treat `cjv self uninstall` as removal of cjv and all managed SDK data.
- Verify selection with `cjv show active`, resolution with `cjv which cjc`, and
  the SDK with `cjc --version` or `cjv run <toolchain> cjc --version`.
- Prefer `--json` for automation where supported. `run`, `exec`, and `init`
  emit their native output. Summarize persistent changes after execution.
