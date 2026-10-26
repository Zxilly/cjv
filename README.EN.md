# cjv - Cangjie Version Manager

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

English | [中文](README.md)

cjv is a version manager for the [Cangjie](https://cangjie-lang.cn/) SDK. It installs toolchains on your machine and selects an SDK version for each project. It also routes commands such as `cjc` and `cjpm` to the tools of the active SDK.

## Key functions

- Install and update LTS, STS, and nightly toolchains.
- Declare the SDK version, components, and target SDKs for a project in `cangjie-sdk.toml`.
- Manage `stdx` extension libraries, offline documentation, and local or custom SDKs.
- Operate on Linux, macOS, Windows, and HarmonyOS. Available SDK releases depend on the distribution source and the host platform.

## Installation

### Run the installer script

On Linux, macOS, or HarmonyOS HiShell, run this command:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

On Windows PowerShell, run this command:

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

The script downloads cjv, checks the file hash, starts the setup wizard, and sets the command path. The wizard selects a channel that is available for your platform. Open a new terminal after installation to load the updated `PATH`.

### Use the GitCode mirror

If GitHub access is unstable, use the GitCode mirror:

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

### Other installation methods

You can also download the archive for your platform from [GitHub Releases](https://github.com/Zxilly/cjv/releases). Extract the archive and run `cjv init`. Read [Install cjv](https://cjv.zxilly.com/book/user-guide/en/installation/) for manual setup, source builds, and CI scripts.

## Quick start

### Do a check of your toolchain

After you complete the setup wizard, check the active toolchain and run the compiler:

```bash
cjv show
cjc --version
```

If you skipped SDK setup, find an available channel and set it as the default. For example, run these commands to use LTS:

```bash
cjv toolchain list-remote --channel lts
cjv default lts
```

The `default` command installs a missing toolchain and then saves the selection. If LTS is not available for your platform, check STS or nightly. Read [Quick start](https://cjv.zxilly.com/book/user-guide/en/basic-usage.html) for daily operations.

### Pin a project version

In the project root directory, resolve and pin the SDK version:

```bash
cjv pin lts
cjv pin --check
```

The `pin` command writes the full release name to `cangjie-sdk.toml`. It uses an installed channel release first and does not install a new SDK. It also keeps existing component and target lists. Commit `cangjie-sdk.toml` to version control. If the project needs `stdx`, add `components = ["stdx"]` under `[toolchain]`.

When you run `cjpm build` in that directory, cjv uses the declared toolchain. By default, cjv installs missing SDKs and components automatically. The `cjv update` command updates tracked channels and keeps pinned versions unchanged.

### Run a command with another version

To run one command with a different toolchain, use `cjv run`:

```bash
cjv run --install sts cjc --version
```

The `--install` flag installs the toolchain if it is missing. Read [Project toolchains](https://cjv.zxilly.com/book/user-guide/en/toolchain-file.html) for the selection sequence and directory overrides. Read [Run commands and set the environment](https://cjv.zxilly.com/book/user-guide/en/runtime-environment.html) for `cjv exec` and `cjv envsetup`.

### Check the environment and preview updates

Run these read-only commands to check your setup or preview an update:

```bash
cjv doctor
cjv update --dry-run
```

Neither command changes your installed toolchains. Read [Automation and JSON interfaces](https://cjv.zxilly.com/book/user-guide/en/automation.html) for CI setup, offline caches, JSON output, and rollback.

## Documentation

| Guide | Topics |
| --- | --- |
| [User guide](https://cjv.zxilly.com/book/user-guide/en/) | Installation, project files, components, and cross-compilation |
| [Command reference](https://cjv.zxilly.com/book/user-guide/en/command-reference.html) | CLI syntax, flags, and default behavior |
| [Automation and JSON interfaces](https://cjv.zxilly.com/book/user-guide/en/automation.html) | Version pins, CI isolation, offline caches, and rollback |
| [Enterprise and offline deployment](https://cjv.zxilly.com/book/user-guide/en/enterprise/) | Internal distribution servers and managed client settings |
| [Developer guide](https://cjv.zxilly.com/book/dev-guide/en/) | Source builds, architecture, tests, and releases |

## Code agents

The repository includes the [cjv skill](skills/cjv) for code agents. It gives agents the rules for commands and project toolchains.

Install the skill globally for Codex with the [skills CLI](https://github.com/vercel-labs/skills):

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

Then call the skill in your chat session:

```text
Use $cjv to install an LTS toolchain for this project and create a commit-ready cangjie-sdk.toml.
```

## Contribute to cjv

Report bugs and send feature requests in [GitHub Issues](https://github.com/Zxilly/cjv/issues). Send code and documentation changes through pull requests.

To build and test locally, install the Go version listed in [go.mod](go.mod) and run:

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

Read the [Developer guide](https://cjv.zxilly.com/book/dev-guide/en/) for the architecture and the full check suite.

## License

cjv uses the Apache-2.0 License. Read [LICENSE](LICENSE) for the full text.
