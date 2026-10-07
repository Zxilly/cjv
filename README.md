# cjv - Cangjie Version Manager

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

[English](README.EN.md) | 中文

cjv 管理 [仓颉](https://cangjie-lang.cn/) SDK，让不同项目使用各自需要的版本。安装后直接运行 `cjc`、`cjpm`，由 cjv 选择工具链。

## 文档

完整文档请访问 [cjv.zxilly.dev](https://cjv.zxilly.dev)。

- [用户手册](https://cjv.zxilly.dev/book/user-guide/zh-CN/)
- [开发指南](https://cjv.zxilly.dev/book/dev-guide/zh-CN/)

## 安装

### 一键安装脚本

Linux / macOS:

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://cjv.zxilly.dev/install.ps1 | iex
```

GitHub 访问不稳定时可使用镜像源：

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Mirror
```

### 预编译二进制

从 [GitHub Releases](https://github.com/Zxilly/cjv/releases) 下载适合当前平台的归档，解压后把 `cjv`（Windows 为 `cjv.exe`）放入 `PATH`。

### 从源码编译

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

## 使用

```bash
# 安装最新 LTS 工具链
cjv install lts

# 设为默认工具链
cjv default lts

# 查看当前工具链状态
cjv show

# 使用指定工具链运行命令
cjv run --install sts cjc --version
```

项目可提交 `cangjie-sdk.toml` 固定版本：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

版本仅为示例。进入项目运行 `cjc`、`cjpm` 时，cjv 按项目声明选择工具链，默认会安装缺失的 SDK 和组件。版本选择、交叉编译及完整参数见[用户手册](https://cjv.zxilly.dev/book/user-guide/zh-CN/)。

## Agent Skill

仓库提供 [cjv skill](skills/cjv)，供编码代理查询命令和项目工具链规则。

使用 [skills CLI](https://github.com/vercel-labs/skills) 全局安装到 Codex：

```bash
npx skills add Zxilly/cjv --skill cjv --agent codex --global --yes
```

安装后可在对话中调用：

```text
使用 $cjv 为当前项目安装 LTS 工具链，并生成可提交的 cangjie-sdk.toml。
```

## 开发

本地构建和测试：

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

更多构建、测试、架构和发布流程见[开发指南](https://cjv.zxilly.dev/book/dev-guide/zh-CN/)。

## 许可证

Apache-2.0。详见 [LICENSE](LICENSE)。
