# cjv - Cangjie Version Manager

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

[English](README.EN.md) | 中文

cjv 是[仓颉](https://cangjie-lang.cn/) SDK 的版本管理器。它可以在本机安装多套工具链，为每个项目选择所需的 SDK 版本，并通过 `cjc`、`cjpm` 等代理命令调用当前 SDK 的工具。

## 核心功能

- 安装和更新 LTS、STS 与 nightly 工具链。
- 使用 `cangjie-sdk.toml` 声明项目所需的版本、组件和交叉编译目标 SDK。
- 管理 `stdx` 扩展库、离线 HTML 文档，以及本地目录或归档形式的自定义 SDK。
- 支持 Linux、macOS、Windows 和鸿蒙（HarmonyOS）；各平台可用的 SDK 版本以分发源为准。

## 安装

### 运行安装脚本

在 Linux、macOS 或鸿蒙 HiShell 中执行：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

在 Windows PowerShell 中执行：

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

安装脚本会下载并校验 cjv，启动安装向导，并配置命令路径 `PATH`。向导会根据当前平台选择可用的 SDK 通道。安装完成后，请打开新终端使 `PATH` 生效。

### 使用 GitCode 镜像

如果访问 GitHub 不稳定，可以使用 GitCode 镜像安装：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

### 其他安装方式

你也可以从 [GitHub Releases](https://github.com/Zxilly/cjv/releases) 下载当前平台的归档，解压后运行 `cjv init`。手动安装、从源码编译和自动化安装的详细步骤见[安装 cjv](https://cjv.zxilly.com/book/user-guide/zh-CN/installation/)。

## 快速上手

### 检查当前工具链

完成安装向导后，检查当前激活的工具链并运行编译器：

```bash
cjv show
cjc --version
```

如果安装时跳过了 SDK，可以先查询可用通道并设为默认。下面以 LTS 通道为例：

```bash
cjv toolchain list-remote --channel lts
cjv default lts
```

`default` 命令会先安装缺失的工具链，再保存默认选择。如果当前平台没有 LTS 版本，可以改查 `sts` 或 `nightly`。日常管理操作见[快速上手](https://cjv.zxilly.com/book/user-guide/zh-CN/basic-usage.html)。

### 为项目固定版本

在项目根目录解析并固定所需的 SDK 版本：

```bash
cjv pin lts
cjv pin --check
```

`pin` 命令会将完整的发行版名称写入 `cangjie-sdk.toml`。它优先使用本机已安装的通道版本，不会触发下载安装，并保留文件中已有的组件和目标列表。请将生成的 `cangjie-sdk.toml` 提交到版本控制。如果项目需要 `stdx`，可以在 `[toolchain]` 下添加 `components = ["stdx"]`。

在该目录运行 `cjpm build` 时，cjv 会使用项目声明的工具链，并默认自动安装缺失的 SDK 和组件。运行 `cjv update` 会更新跟踪通道，而固定版本保持不变。

### 临时使用其他版本运行命令

要单次使用其他工具链版本运行命令，请使用 `cjv run`：

```bash
cjv run --install sts cjc --version
```

`--install` 允许自动安装缺失的工具链。工具链的选择顺序和本机目录覆盖见[项目工具链](https://cjv.zxilly.com/book/user-guide/zh-CN/toolchain-file.html)；`cjv exec` 与 `cjv envsetup` 的用法见[运行命令与配置环境](https://cjv.zxilly.com/book/user-guide/zh-CN/runtime-environment.html)。

### 诊断环境与预览更新

运行以下只读命令检查当前环境或预览更新计划：

```bash
cjv doctor
cjv update --dry-run
```

这两条命令都不会修改已安装的工具链。CI 隔离配置、离线缓存、JSON 输出与快照回退见[自动化与 JSON 接口](https://cjv.zxilly.com/book/user-guide/zh-CN/automation.html)。

## 文档导航

| 文档章节 | 主要内容 |
| --- | --- |
| [用户手册](https://cjv.zxilly.com/book/user-guide/zh-CN/) | 安装、项目配置、组件管理与交叉编译 |
| [命令参考](https://cjv.zxilly.com/book/user-guide/zh-CN/command-reference.html) | CLI 命令语法、选项参数与默认策略 |
| [自动化与 JSON 接口](https://cjv.zxilly.com/book/user-guide/zh-CN/automation.html) | 版本固定、CI 隔离、离线缓存与更新回退 |
| [企业与离线部署](https://cjv.zxilly.com/book/user-guide/zh-CN/enterprise/) | 内部分发源搭建、系统后备配置与离线客户端 |
| [开发指南](https://cjv.zxilly.com/book/dev-guide/zh-CN/) | 源码构建、模块架构、测试检查与发布流程 |

## 编码代理（AI Agents）

仓库内置了 [cjv skill](skills/cjv)，供编码代理查询 CLI 命令和项目工具链规则。

使用 [skills CLI](https://github.com/vercel-labs/skills) 将该技能全局安装到 Codex：

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

安装后即可在对话中调用：

```text
使用 $cjv 为当前项目安装 LTS 工具链，并生成可提交的 cangjie-sdk.toml。
```

## 参与开发

问题反馈和功能建议请提交到 [GitHub Issues](https://github.com/Zxilly/cjv/issues)，代码与文档改动请通过 Pull Request 提交。

本地构建与测试需要安装满足 [go.mod](go.mod) 要求的 Go 版本：

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

代码架构说明和完整检查要求见[开发指南](https://cjv.zxilly.com/book/dev-guide/zh-CN/)。

## 许可证

cjv 采用 Apache-2.0 许可证，完整条款见 [LICENSE](LICENSE)。
