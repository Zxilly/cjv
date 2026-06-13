# cjv - Cangjie Version Manager

[English](README.EN.md) | 中文

[仓颉](https://cangjie-lang.cn/)编程语言 SDK 的工具链管理器。

cjv 管理多个仓颉 SDK 安装、处理版本切换，并提供 SDK 工具的透明代理执行。

完整文档见 cjv 用户手册：<https://cjv.zxilly.dev/book/user-guide/zh-CN/>（[English](https://cjv.zxilly.dev/book/user-guide/en/)）。

## 安装

### 从源码编译

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

### 从发布二进制文件

从 [Releases](https://github.com/Zxilly/cjv/releases) 页面下载适合你平台的二进制文件，并将其放入 PATH 中。

### 一键安装脚本

落地页 <https://cjv.zxilly.dev> 提供 `install.sh` / `install.ps1` 一键安装（含镜像源变体）。详见文档的[安装 cjv](https://cjv.zxilly.dev/book/user-guide/zh-CN/installation/index.html)。

## 快速开始

```bash
# 安装最新 LTS 工具链
cjv install lts

# 设为默认
cjv default lts

# 验证安装
cjv show

# 使用指定工具链运行命令
cjv run sts cjc --version
```

安装并设为默认后，可直接调用 `cjc`、`cjpm` 等工具，cjv 会透明代理到对应工具链。

## 常用命令

| 命令                                         | 说明                                    |
| -------------------------------------------- | --------------------------------------- |
| `cjv install <toolchain> [-t target] [-c c]` | 安装工具链，可附加交叉编译目标与组件    |
| `cjv uninstall <toolchain>`                  | 卸载工具链                              |
| `cjv update [toolchain]`                      | 更新已安装的工具链                      |
| `cjv default [toolchain]`                     | 设置或显示默认工具链                    |
| `cjv show`                                    | 显示活跃和已安装的工具链                |
| `cjv run <toolchain> <command> [args...]`     | 使用指定工具链运行命令                  |
| `cjv toolchain link <name> <path\|url>`       | 从本地目录或 URL 添加自定义工具链       |
| `cjv component add <name>...`                 | 为工具链安装 component（如 stdx、docs） |
| `cjv exec [+toolchain] <command>`             | 在运行时环境中执行命令                  |
| `cjv envsetup [+toolchain]`                   | 输出配置运行时环境的 shell 命令         |

完整命令参考，以及工具链解析、`cangjie-sdk.toml` 格式、从 URL 安装、组件、交叉编译、运行时环境、环境变量与配置等说明，都在 cjv 用户手册里：

- 中文：<https://cjv.zxilly.dev/book/user-guide/zh-CN/>
- English：<https://cjv.zxilly.dev/book/user-guide/en/>

想参与开发（构建、测试、架构、发布流程），见开发指南：<https://cjv.zxilly.dev/book/dev-guide/zh-CN/>（[English](https://cjv.zxilly.dev/book/dev-guide/en/)）。

文档源码在 [`docs/`](docs/) 目录（mdBook，简体中文源 + 英文翻译）：用户手册 `docs/user-guide/`，开发指南 `docs/dev-guide/`。

## 许可证

Apache-2.0。详见 [LICENSE](LICENSE)。
