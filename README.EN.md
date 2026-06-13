# cjv - Cangjie Version Manager

English | [中文](README.md)

A toolchain manager for the [Cangjie](https://cangjie-lang.cn/) programming language SDK.

cjv manages multiple Cangjie SDK installations, handles version switching, and provides transparent proxy execution of SDK tools.

Full documentation is in the cjv user guide: <https://cjv.zxilly.dev/book/user-guide/en/> ([中文](https://cjv.zxilly.dev/book/user-guide/zh-CN/)).

## Installation

### From source

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

### From release binaries

Download the binary for your platform from the [Releases](https://github.com/Zxilly/cjv/releases) page and put it on your PATH.

### Install script

The landing page <https://cjv.zxilly.dev> offers one-line `install.sh` / `install.ps1` installers (with mirror variants). See [Installing cjv](https://cjv.zxilly.dev/book/user-guide/en/installation/index.html) in the docs.

## Quick start

```bash
# Install the latest LTS toolchain
cjv install lts

# Make it the default
cjv default lts

# Verify the installation
cjv show

# Run a command with a specific toolchain
cjv run sts cjc --version
```

Once a toolchain is installed and set as default, you can invoke `cjc`, `cjpm`, and other tools directly. cjv proxies them to the right toolchain.

## Common commands

| Command                                       | Description                                          |
| --------------------------------------------- | --------------------------------------------------- |
| `cjv install <toolchain> [-t target] [-c c]`  | Install a toolchain, optionally with cross targets and components |
| `cjv uninstall <toolchain>`                   | Uninstall a toolchain                               |
| `cjv update [toolchain]`                       | Update installed toolchains                         |
| `cjv default [toolchain]`                      | Set or show the default toolchain                  |
| `cjv show`                                     | Show active and installed toolchains               |
| `cjv run <toolchain> <command> [args...]`      | Run a command with a specific toolchain            |
| `cjv toolchain link <name> <path\|url>`        | Add a custom toolchain from a local dir or a URL   |
| `cjv component add <name>...`                  | Install a component (e.g. stdx, docs) for a toolchain |
| `cjv exec [+toolchain] <command>`              | Execute a command in the runtime environment       |
| `cjv envsetup [+toolchain]`                    | Print shell commands to configure the runtime env  |

The full command reference, plus toolchain resolution, the `cangjie-sdk.toml` format, installing from a URL, components, cross-compilation, the runtime environment, environment variables, and configuration, are all in the cjv user guide:

- English: <https://cjv.zxilly.dev/book/user-guide/en/>
- 中文: <https://cjv.zxilly.dev/book/user-guide/zh-CN/>

To work on cjv itself (building, testing, architecture, releasing), see the dev guide: <https://cjv.zxilly.dev/book/dev-guide/en/> ([中文](https://cjv.zxilly.dev/book/dev-guide/zh-CN/)).

The book sources live in [`docs/`](docs/) (mdBook, Simplified Chinese source + English translation): the user guide in `docs/user-guide/`, the dev guide in `docs/dev-guide/`.

## License

Apache-2.0. See [LICENSE](LICENSE).
