# cjv - Cangjie Version Manager

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

English | [中文](README.md)

cjv is a toolchain manager for the [Cangjie](https://cangjie-lang.cn/) SDK. It installs and manages multiple SDKs, switches default versions, and transparently proxies SDK tools such as `cjc` and `cjpm` to the right toolchain for the current project.

## Documentation

Full documentation is available at [cjv.zxilly.dev](https://cjv.zxilly.dev).

- [User guide](https://cjv.zxilly.dev/book/user-guide/en/)
- [Dev guide](https://cjv.zxilly.dev/book/dev-guide/en/)

## Installation

### Install script

Linux / macOS:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://cjv.zxilly.dev/install.ps1 | iex
```

Use the mirror source when GitHub access is unreliable:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Mirror
```

### Release binaries

Download the archive for your platform from [GitHub Releases](https://github.com/Zxilly/cjv/releases), extract it, and put `cjv` (`cjv.exe` on Windows) on your `PATH`.

### From source

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

## Usage

```bash
# Install the latest LTS toolchain
cjv install lts

# Make it the default toolchain
cjv default lts

# Show the current toolchain state
cjv show

# Run a command with a specific toolchain
cjv run sts cjc --version
```

After setting a default toolchain, you can call `cjc`, `cjpm`, and other SDK tools directly. cjv resolves the active toolchain from environment variables, directory overrides, `cangjie-sdk.toml`, and the default configuration.

See the [user guide](https://cjv.zxilly.dev/book/user-guide/en/) for the full command reference, toolchain resolution, components, cross-compilation, runtime environment, and configuration.

## Agent Skill

The repository includes a concise [cjv user-guide skill](skills/cjv) that helps
coding agents choose the right cjv command and follow its toolchain-resolution
rules.

Install it globally for Codex with the [skills CLI](https://github.com/vercel-labs/skills):

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

Then invoke it in a conversation:

```text
Use $cjv to install an LTS toolchain for this project and create a commit-ready cangjie-sdk.toml.
```

## Development

Build and test locally:

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

See the [dev guide](https://cjv.zxilly.dev/book/dev-guide/en/) for build, test, architecture, and release details.

## License

Apache-2.0. See [LICENSE](LICENSE).
