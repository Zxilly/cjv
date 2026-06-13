# 简介

cjv 是[仓颉](https://cangjie-lang.cn/)编程语言 SDK 的工具链管理器，用 Go 编写。它管理多个仓颉 SDK 安装、处理版本切换，并为 `cjc`、`cjpm` 等 SDK 工具提供透明代理执行。面向用户的功能说明在 cjv 用户手册：<https://cjv.zxilly.dev/book/user-guide/zh-CN/>。

这份开发指南面向 cjv 的贡献者和开发者。它讲怎么从源码构建、代码如何组织、怎么写和跑测试、文档站和落地页如何工作、CI 与发布流程是什么样的。如果你只是想用 cjv 装一个工具链，看用户手册就够了；这里讲的是 cjv 自己怎么造出来。

## 仓库构成

仓库根目录下分成四块。

Go CLI 是核心。命令入口在 `cmd/cjv/main.go`，实际逻辑都在 `internal/` 下按子系统拆分：`cli` 是 cobra 命令定义，`resolve` 做工具链版本解析，`toolchain`、`component` 管理已安装的 SDK 与组件，`proxy` 实现透明代理，`env` 处理运行时环境，`config`、`dist`、`selfupdate` 等各司其职。模块路径是 `github.com/Zxilly/cjv`。

`web/` 是落地页，一个 Vite + React 19 + TypeScript 项目，托管在 <https://cjv.zxilly.dev>，提供 `install.sh` / `install.ps1` 一键安装脚本和按平台分发的下载入口。文案用 Lingui 做国际化。

`docs/` 是两本 mdBook：`docs/user-guide/` 用户手册，`docs/dev-guide/` 就是你正在读的这本开发指南。每本都维护中英两份独立源：`src/` 是简体中文（源语言），`en/` 是英文。

`tests/` 放跨包的集成测试与冒烟测试：`tests/integration/` 是端到端集成测试，`tests/smoke/` 验证真实下载，`tests/install-scripts/` 测安装脚本。单元测试则按惯例和被测代码同目录，`_test.go` 紧挨着源文件。

## 上手前置

构建 Go CLI 需要 Go，所需版本以 `go.mod` 里的 `go` 指令为准（当前是 1.26.0）。装好 Go 后 `go build ./...` 就能编译整个项目，细节见[从源码构建](building.md)。

开发落地页需要 Node 和 pnpm。包管理器锁定在 `web/package.json` 的 `packageManager` 字段（pnpm 10.x）。在 `web/` 下 `pnpm install` 装依赖、`pnpm dev` 起开发服务器，见[落地页](web.md)。

本地构建文档只需要 [mdBook](https://github.com/rust-lang/mdBook)。CI 用的版本写在 `.github/workflows/pages.yml` 里，见[文档站](documentation.md)。

## 接下来

[从源码构建](building.md)讲怎么编译和运行 CLI，[代码架构](architecture.md)讲 `internal/` 各子系统如何协作，[测试](testing.md)和[代码检查与格式化](linting.md)讲怎么保证改动正确且符合风格。[文档站](documentation.md)、[落地页](web.md)、[持续集成](ci.md)、[发布流程](releasing.md)分别讲四个子系统。准备提交改动时，看[贡献指南](contributing.md)。
