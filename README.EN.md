# cjv: the Cangjie toolchain installer

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

English | [中文](README.md)

_cjv_ installs the [Cangjie Programming Language](https://cangjie-lang.cn/) SDK from the official release channels, enabling you to easily switch between LTS, STS, and nightly toolchains and keep them updated. It manages extension libraries (`stdx`), offline documentation, and cross-compilation target SDKs, and routes everyday commands such as `cjc` and `cjpm` to the right toolchain for each project.

`cjv` runs across Linux, macOS, Windows, and HarmonyOS (availability of specific SDK releases depends on the upstream distribution manifest for each platform).

## Highlights

- **Multiple release channels and pinned versions**: Track `lts`, `sts`, and `nightly` channels side by side, or pin exact versions such as `lts-1.0.5` and `nightly-2026-10-04`.
- **Per-project toolchain files**: Commit a `cangjie-sdk.toml` file to your repository so everyone on your team—and your CI pipeline—uses the exact same compiler, components, and cross-compilation targets.
- **Components and custom SDKs**: Install official `stdx` and documentation packages with a single command, or link local SDK builds and custom `.zip` / `.tar.gz` archives.
- **Built for CI and enterprise networks**: First-class support for `--json` output, `--offline` reusable caches, transactional updates with rollback history, and internal mirror servers.

## Installation

### Quick install via script

On Linux, macOS, or HarmonyOS HiShell, run the shell installer:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

On Windows PowerShell, run the PowerShell installer:

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

The installer downloads and verifies `cjv` for your platform, launches the interactive setup wizard (`cjv init`), and adds `<CJV_HOME>/bin` (default `~/.cjv/bin`) to your `PATH`. The wizard automatically queries the distribution manifest and offers the release channels available for your host platform. Once installation finishes, open a _new_ terminal window so the updated `PATH` takes effect.

### Using the GitCode mirror

If your network connection to GitHub is slow or unreliable, pass `--mirror` (or `-Mirror` in PowerShell) to install the mirror variant backed by GitCode:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

### Other installation methods

You can also download a prebuilt archive or standalone `cjv-init` installer for your platform from [GitHub Releases](https://github.com/Zxilly/cjv/releases) or [cjv.zxilly.com](https://cjv.zxilly.com), extract it, and run `cjv init`. See the [Installation chapter](https://cjv.zxilly.com/book/user-guide/en/installation/) in the user guide for manual installation, building from source with Go, and non-interactive CI options.

## Getting started

### Inspecting and selecting a toolchain

After completing the setup wizard, open a new shell and verify that `cjv` and the Cangjie compiler are ready:

```bash
cjv show
cjc --version
```

If you chose to skip SDK installation during setup, you can query the available releases for a channel and set a global default toolchain at any time. Taking the `lts` channel as an example:

```bash
cjv toolchain list-remote --channel lts
cjv default lts
```

When you pass an uninstalled official toolchain to `cjv default`, `cjv` automatically downloads and installs it before saving the default selection. If `lts` does not publish builds for your platform yet, check `sts` or `nightly` instead. See [Basic usage](https://cjv.zxilly.com/book/user-guide/en/basic-usage.html) for an overview of everyday commands.

### Pinning a version for your project

When working on a team project, you usually want everyone to build with the same Cangjie release. From the root of your project, run `cjv pin`:

```bash
cjv pin lts
cjv pin --check
```

`cjv pin lts` resolves `lts` to a concrete release name (such as `lts-1.0.5`, preferring a release you already have installed locally) and writes it to `cangjie-sdk.toml` without downloading a new SDK. Any existing `components` and `targets` in the file are preserved. You can check `cangjie-sdk.toml` into version control—and if your project uses the `stdx` extension library, simply add `components = ["stdx"]` under `[toolchain]`.

Whenever you run `cjpm build` or `cjc` inside that project directory, `cjv` automatically selects the declared toolchain and installs any missing SDKs or components on demand. Meanwhile, running `cjv update` updates your tracked channels (`lts`, `sts`, `nightly`) while leaving pinned project versions untouched.

### Running commands and checking environment health

To test a command against a different toolchain without changing your default or project configuration, use `cjv run`:

```bash
cjv run --install sts cjc --version
```

Passing `--install` tells `cjv` to install `sts` on the fly if it isn't already present. For full details on how `cjv` chooses a toolchain and how to set local directory overrides, see [Project toolchains and overrides](https://cjv.zxilly.com/book/user-guide/en/toolchain-file.html); for running compiled Cangjie binaries with `cjv exec` or loading SDK variables into your current shell with `cjv envsetup`, see [Running commands and setting the environment](https://cjv.zxilly.com/book/user-guide/en/runtime-environment.html).

You can also run read-only diagnostics on your setup or preview what an update would do before applying it:

```bash
cjv doctor
cjv update --dry-run
```

Neither command modifies your installed toolchains. For CI isolation, offline caches, JSON schemas, and rolling back channel updates, see [Automation and JSON interfaces](https://cjv.zxilly.com/book/user-guide/en/automation.html).

## Documentation

See [**The cjv User Guide**](https://cjv.zxilly.com/book/user-guide/en/) for complete documentation on installing and using `cjv`:

- [**User Guide**](https://cjv.zxilly.com/book/user-guide/en/) — Installation, basic usage, project toolchain files, components, custom SDKs, and cross-compilation.
- [**Command Reference**](https://cjv.zxilly.com/book/user-guide/en/command-reference.html) — Complete CLI syntax, flags, and installation/update policies.
- [**Automation and JSON Interfaces**](https://cjv.zxilly.com/book/user-guide/en/automation.html) — Pinning workflows, CI environment isolation, offline caches, update history rollback, and JSON output contracts.
- [**Enterprise and Offline Deployment**](https://cjv.zxilly.com/book/user-guide/en/enterprise/) — Hosting internal distribution mirrors, system fallback settings, and air-gapped client setups.
- [**Developer Guide**](https://cjv.zxilly.com/book/dev-guide/en/) — Building `cjv` from source, internal architecture, test suites, website maintenance, and release workflows.

## Coding agents

This repository ships with a dedicated [cjv skill](skills/cjv) that gives AI coding agents accurate context on `cjv` commands, `cangjie-sdk.toml` rules, and read-only diagnostics.

You can install it globally for Codex using the [skills CLI](https://github.com/vercel-labs/skills):

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

Once installed, invoke `$cjv` directly in your prompt:

```text
Use $cjv to install an LTS toolchain for this project and create a commit-ready cangjie-sdk.toml.
```

## Contributing

Bug reports and feature suggestions are welcome on [GitHub Issues](https://github.com/Zxilly/cjv/issues), and code or documentation improvements can be submitted through pull requests.

To build and test `cjv` locally, make sure you have a Go toolchain that satisfies [go.mod](go.mod):

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

See [**The cjv Developer Guide**](https://cjv.zxilly.com/book/dev-guide/en/) for a tour of the codebase, testing conventions, and pre-submission checks.

## License

`cjv` is distributed under the terms of the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
