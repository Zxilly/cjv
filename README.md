# cjv - Cangjie Version Manager

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

[English](README.EN.md) | 中文

cjv 是 [仓颉](https://cangjie-lang.cn/) SDK 的版本管理器。它可以安装多套工具链，为项目选择 SDK 版本，并通过 `cjc`、`cjpm` 等命令入口调用对应的工具。

- 安装、更新 LTS、STS 和 nightly 工具链。
- 使用 `cangjie-sdk.toml` 声明项目需要的版本、组件和交叉编译 SDK。
- 管理 stdx 扩展库、离线文档，以及本地或自定义 SDK。
- 在 Linux、macOS、Windows 和鸿蒙上使用；各平台可用的 SDK 以分发源为准。

## 安装

Linux / macOS / 鸿蒙 HiShell：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

安装脚本会下载并校验 cjv，运行安装向导，配置命令路径。向导根据当前平台选择可用的 SDK 通道。安装完成后打开新终端，使 `PATH` 生效。

GitHub 访问不稳定时，可以使用 GitCode 镜像：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

也可以从 [GitHub Releases](https://github.com/Zxilly/cjv/releases) 下载当前平台的归档，解压后运行 `cjv init`。手动安装、从源码编译和自动化安装的步骤见[安装说明](https://cjv.zxilly.com/book/user-guide/zh-CN/installation/)。

## 快速上手

完成安装向导后，检查当前工具链并运行编译器：

```bash
cjv show
cjc --version
```

如果安装时跳过了 SDK，可以选择一个可用通道并设为默认。下面以 LTS 为例：

```bash
cjv toolchain list-remote --channel lts
cjv default lts
```

`default` 会先安装缺失的工具链，再保存默认选择。所选平台没有 LTS 时，可查询 STS 或 nightly。

### 为项目固定版本

在项目根目录创建 `cangjie-sdk.toml`，并提交到版本控制：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

将示例版本替换为项目需要且分发源提供的版本。在该目录运行 `cjpm build` 时，cjv 会使用项目声明的工具链，默认自动安装缺失的 SDK 和组件。跟踪通道可通过 `cjv update` 更新，固定版本保持不变。

临时使用其他版本运行命令：

```bash
cjv run --install sts cjc --version
```

`--install` 允许安装缺失的工具链。版本选择顺序和目录覆盖见[项目工具链](https://cjv.zxilly.com/book/user-guide/zh-CN/toolchain-file.html)。

## 文档

- [用户手册](https://cjv.zxilly.com/book/user-guide/zh-CN/)：安装、项目配置、组件和交叉编译。
- [命令参考](https://cjv.zxilly.com/book/user-guide/zh-CN/command-reference.html)：命令参数与默认行为。
- [企业与离线部署](https://cjv.zxilly.com/book/user-guide/zh-CN/enterprise/)：内部分发源与受管终端配置。
- [开发指南](https://cjv.zxilly.com/book/dev-guide/zh-CN/)：构建、测试、架构和发布。

## 编码代理

仓库提供 [cjv skill](skills/cjv)，供编码代理查询命令和项目工具链规则。

使用 [skills CLI](https://github.com/vercel-labs/skills) 全局安装到 Codex：

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

安装后可在对话中调用：

```text
使用 $cjv 为当前项目安装 LTS 工具链，并生成可提交的 cangjie-sdk.toml。
```

## 参与开发

问题反馈和改进建议可以提交到 [Issues](https://github.com/Zxilly/cjv/issues)，代码与文档修改可以通过 Pull Request 提交。

本地构建需要满足 [go.mod](go.mod) 要求的 Go 版本：

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

开发环境与检查要求见[开发指南](https://cjv.zxilly.com/book/dev-guide/zh-CN/)。

## 许可证

cjv 使用 Apache-2.0 许可证，详见 [LICENSE](LICENSE)。
