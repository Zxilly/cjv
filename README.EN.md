# cjv - Cangjie Version Manager

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

English | [中文](README.md)

cjv is a version manager for the [Cangjie](https://cangjie-lang.cn/) SDK. It installs multiple toolchains, selects an SDK for each project, and routes commands such as `cjc` and `cjpm` to that SDK's tools.

- Install and update LTS, STS, and nightly toolchains.
- Declare project versions, components, and cross-compilation SDKs in `cangjie-sdk.toml`.
- Manage stdx extension libraries, offline documentation, and local or custom SDKs.
- Run on Linux, macOS, Windows, and HarmonyOS. SDK availability depends on the distribution source and platform.

## Installation

Linux / macOS / HarmonyOS HiShell:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://cjv.zxilly.dev/install.ps1 | iex
```

The script downloads and verifies cjv, runs the installation wizard, and configures the command path. The wizard selects a channel available for your platform. Open a new terminal after installation to load the updated `PATH`.

If GitHub access is unreliable, use the GitCode mirror:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Mirror
```

You can also download your platform's archive from [GitHub Releases](https://github.com/Zxilly/cjv/releases), extract it, and run `cjv init`. See [installation](https://cjv.zxilly.dev/book/user-guide/en/installation/) for manual setup, source builds, and automated installation.

## Quick start

After completing the installation wizard, inspect the selected toolchain and run the compiler:

```bash
cjv show
cjc --version
```

If you skipped SDK installation, choose an available channel and make it the default. For example, to use LTS:

```bash
cjv toolchain list-remote --channel lts
cjv default lts
```

`default` installs a missing toolchain before saving the selection. If LTS is unavailable for your platform, check STS or nightly.

### Pin a project version

Create `cangjie-sdk.toml` in the project root and commit it to version control:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

Replace the example version with one your project needs and your distribution provides. Running `cjpm build` in that directory uses the declared toolchain. By default, cjv installs missing SDKs and components automatically. `cjv update` updates tracked channels and leaves pinned versions unchanged.

To run a command with another version temporarily:

```bash
cjv run --install sts cjc --version
```

`--install` allows a missing toolchain to be installed. See [project toolchains](https://cjv.zxilly.dev/book/user-guide/en/toolchain-file.html) for selection order and directory overrides.

## Documentation

- [User guide](https://cjv.zxilly.dev/book/user-guide/en/): installation, project configuration, components, and cross-compilation.
- [Command reference](https://cjv.zxilly.dev/book/user-guide/en/command-reference.html): command options and default behavior.
- [Enterprise and offline deployment](https://cjv.zxilly.dev/book/user-guide/en/enterprise/): internal distribution sources and managed client settings.
- [Developer guide](https://cjv.zxilly.dev/book/dev-guide/en/): builds, tests, architecture, and releases.

## Coding agents

The [cjv skill](skills/cjv) gives coding agents a reference for commands and project toolchain rules.

Install it globally for Codex with the [skills CLI](https://github.com/vercel-labs/skills):

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

Then invoke it in a conversation:

```text
Use $cjv to install an LTS toolchain for this project and create a commit-ready cangjie-sdk.toml.
```

## Contributing

Report problems and suggest improvements in [Issues](https://github.com/Zxilly/cjv/issues). Code and documentation changes can be submitted as pull requests.

Local builds require a Go version that satisfies [go.mod](go.mod):

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

See the [developer guide](https://cjv.zxilly.dev/book/dev-guide/en/) for development setup and required checks.

## License

cjv is licensed under Apache-2.0. See [LICENSE](LICENSE).
