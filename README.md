# cjv - Cangjie Version Manager

[![CI](https://img.shields.io/github/actions/workflow/status/Zxilly/cjv/ci.yml?branch=master&style=flat-square)](https://github.com/Zxilly/cjv/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Zxilly/cjv?style=flat-square)](https://github.com/Zxilly/cjv/releases)
[![License](https://img.shields.io/github/license/Zxilly/cjv?style=flat-square)](LICENSE)

[English](README.EN.md) | 中文

cjv 是 [仓颉](https://cangjie-lang.cn/) SDK 的工具链管理器：安装并管理多套 SDK，切换默认版本，并让 `cjc`、`cjpm` 等 SDK 工具按当前项目透明代理到正确工具链。

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
cjv run sts cjc --version
```

设置默认工具链后，可以直接调用 `cjc`、`cjpm` 等命令；cjv 会根据环境变量、目录覆盖、`cangjie-sdk.toml` 和默认配置解析应使用的工具链。

完整命令参考、工具链解析、组件、交叉编译、运行时环境和配置说明见[用户手册](https://cjv.zxilly.dev/book/user-guide/zh-CN/)。

## 开发

本地构建和测试：

```bash
go build ./cmd/cjv
go test -race -count=1 ./...
```

更多构建、测试、架构和发布流程见[开发指南](https://cjv.zxilly.dev/book/dev-guide/zh-CN/)。

## 许可证

Apache-2.0。详见 [LICENSE](LICENSE)。
