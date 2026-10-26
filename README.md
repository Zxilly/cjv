# cjv：仓颉工具链安装与版本管理器

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

[English](README.EN.md) | 中文

_cjv_ 用于从官方发布通道安装[仓颉编程语言（Cangjie）](https://cangjie-lang.cn/) SDK，让你能够轻松地在 LTS、STS 和 nightly 工具链之间切换并保持更新。它同时负责管理扩展库（`stdx`）、离线文档与交叉编译目标 SDK，并通过 `cjc`、`cjpm` 等代理命令为每个项目自动调用匹配的编译器与工具。

`cjv` 支持在 Linux、macOS、Windows 和鸿蒙（HarmonyOS）上运行；各平台具体可安装的 SDK 版本取决于上游分发清单的发布范围。

## 核心特性

- **多通道跟踪与精确版本固定**：支持同时安装并跟踪 `lts`、`sts` 和 `nightly` 通道，也可以按项目需要固定安装 `lts-1.0.5`、`nightly-2026-10-04` 等具体版本。
- **项目级工具链声明文件**：在仓库中提交 `cangjie-sdk.toml`，即可让团队成员与 CI 流水线自动使用完全一致的编译器版本、扩展组件和交叉编译目标。
- **组件管理与自定义 SDK**：一条命令安装官方 `stdx` 扩展库和离线 HTML 文档，或将本机自行编译的 SDK 目录及 `.zip` / `.tar.gz` 归档注册为自定义工具链。
- **面向 CI 与企业内网设计**：原生提供 `--json` 结构化输出、`--offline` 可复用缓存、带历史快照的事务性更新与回滚，以及企业内网镜像分发支持。

## 安装

### 使用安装脚本快速安装

在 Linux、macOS 或鸿蒙 HiShell 中执行：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

在 Windows PowerShell 中执行：

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

安装脚本会自动下载并校验当前平台的 `cjv` 程序，启动交互式安装向导（`cjv init`），并将 `<CJV_HOME>/bin`（默认为 `~/.cjv/bin`）写入 `PATH` 环境变量。安装向导会根据当前操作系统查询可用的 SDK 通道。安装完成后，请打开一个**新终端**使更新后的 `PATH` 生效。

### 使用 GitCode 镜像安装

如果当前网络访问 GitHub 不稳定，可以传入 `--mirror`（PowerShell 为 `-Mirror`）安装使用 GitCode 镜像源的版本：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

### 其他安装方式

你也可以直接从 [GitHub Releases](https://github.com/Zxilly/cjv/releases) 或 [cjv.zxilly.com](https://cjv.zxilly.com) 下载当前平台的归档文件或独立安装器 `cjv-init`，解压后运行 `cjv init`。关于手动安装、使用 Go 从源码构建以及 CI 非交互安装的详细说明，请参阅用户手册中的[安装 cjv](https://cjv.zxilly.com/book/user-guide/zh-CN/installation/) 章节。

## 快速上手

### 查看与选择工具链

完成安装向导后，打开新终端并验证当前激活的工具链与编译器状态：

```bash
cjv show
cjc --version
```

如果在初始化时跳过了 SDK 安装，可以随时查询某个通道下的可用版本，并将其设为全局默认工具链。下面以 `lts` 通道为例：

```bash
cjv toolchain list-remote --channel lts
cjv default lts
```

如果指定的官方工具链尚未安装，`cjv default` 会自动先完成下载和安装，再保存默认选择。如果当前平台尚未发布 LTS 版本，可以改查 `sts` 或 `nightly` 通道。日常管理命令的完整演示见[快速上手](https://cjv.zxilly.com/book/user-guide/zh-CN/basic-usage.html)。

### 为项目固定工具链版本

在多人协作项目中，通常希望所有开发者和 CI 都使用同一个确定的仓颉版本。在项目根目录运行 `cjv pin`：

```bash
cjv pin lts
cjv pin --check
```

`cjv pin lts` 会将 `lts` 解析为具体的发行版名称（例如 `lts-1.0.5`，优先采用本机已安装的该通道版本，不会触发额外下载），并写入 `cangjie-sdk.toml`，同时保留文件中已有的组件和交叉编译目标声明。你可以将生成的 `cangjie-sdk.toml` 提交到版本控制；如果项目依赖 `stdx` 扩展库，只需在 `[toolchain]` 下添加 `components = ["stdx"]`。

此后在该目录中运行 `cjpm build` 或 `cjc` 时，`cjv` 会自动切换到项目声明的工具链，并默认按需补装缺失的 SDK 和组件。日常运行 `cjv update` 会升级已安装的跟踪通道（`lts`、`sts`、`nightly`），而固定版本保持不变。

### 临时切换版本与环境自检

如果想在不修改默认设置或项目文件的情况下，临时使用另一个工具链运行单条命令，可以使用 `cjv run`：

```bash
cjv run --install sts cjc --version
```

加上 `--install` 会在目标工具链缺失时自动安装。关于版本选择的完整优先级规则与本机目录覆盖，请参阅[项目工具链](https://cjv.zxilly.com/book/user-guide/zh-CN/toolchain-file.html)；关于使用 `cjv exec` 运行仓颉编译产物或通过 `cjv envsetup` 配置当前终端环境，请参阅[运行命令与配置环境](https://cjv.zxilly.com/book/user-guide/zh-CN/runtime-environment.html)。

你还可以随时运行以下只读命令，诊断当前安装环境或预览更新计划：

```bash
cjv doctor
cjv update --dry-run
```

这两条命令都不会修改本机安装状态。关于 CI 目录隔离、离线缓存复用、JSON 输出规范和更新历史回滚，请参阅[自动化与 JSON 接口](https://cjv.zxilly.com/book/user-guide/zh-CN/automation.html)。

## 文档

关于安装与使用 `cjv` 的完整说明，请阅读 [**cjv 用户手册**](https://cjv.zxilly.com/book/user-guide/zh-CN/)：

- [**用户手册**](https://cjv.zxilly.com/book/user-guide/zh-CN/) — 安装指南、日常用法、项目工具链配置、组件管理、自定义 SDK 与交叉编译。
- [**命令参考**](https://cjv.zxilly.com/book/user-guide/zh-CN/command-reference.html) — 所有 CLI 子命令语法、参数选项与安装更新策略。
- [**自动化与 JSON 接口**](https://cjv.zxilly.com/book/user-guide/zh-CN/automation.html) — 版本固定校验、CI 环境隔离、离线缓存、历史快照回退与 JSON Schema。
- [**企业与离线部署**](https://cjv.zxilly.com/book/user-guide/zh-CN/enterprise/) — 内部分发源搭建、系统后备配置与完全离线客户端部署。
- [**开发指南**](https://cjv.zxilly.com/book/dev-guide/zh-CN/) — 从源码构建、代码架构、测试套件、落地页与文档站维护及发布流程。

## 编码代理（Coding Agents）

本仓库内置了 [cjv skill](skills/cjv)，帮助 AI 编码代理准确理解 `cjv` 的命令语义、`cangjie-sdk.toml` 规范与只读诊断流程。

可以使用 [skills CLI](https://github.com/vercel-labs/skills) 将其全局安装到 Codex：

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

安装后即可在对话中直接调用 `$cjv`：

```text
使用 $cjv 为当前项目安装 LTS 工具链，并生成可提交的 cangjie-sdk.toml。
```

## 参与贡献

欢迎通过 [GitHub Issues](https://github.com/Zxilly/cjv/issues) 反馈问题或提出改进建议，也欢迎通过 Pull Request 提交代码与文档修改。

本地编译与运行测试需要安装满足 [go.mod](go.mod) 要求的 Go 工具链：

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

关于代码模块划分、测试编写规范与提交前检查要求，请参阅 [**cjv 开发指南**](https://cjv.zxilly.com/book/dev-guide/zh-CN/)。

## 许可证

`cjv` 基于 Apache-2.0 许可证发布，详情见 [LICENSE](LICENSE)。
