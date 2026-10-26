# 简介

_cjv_ 用于从官方发布通道安装[仓颉编程语言（Cangjie）](https://cangjie-lang.cn/) SDK，让你能够轻松地在 LTS、STS 和 nightly 工具链之间切换并保持更新。安装好 `cjv` 并将代理目录加入 `PATH` 后，日常运行的 `cjc`、`cjpm` 等命令会自动调用当前项目所选的工具链。

首次使用时，请先阅读[安装 cjv](installation/index.md) 了解如何在 Linux、macOS、Windows 和鸿蒙（HarmonyOS）上完成安装，随后跟随[快速上手](basic-usage.md)查看当前工具链、配置项目版本并管理更新。

安装向导会根据当前主机平台自动查询可用的发布通道。需要注意的是，`cjv` 本体支持的运行平台与仓颉 SDK 各通道的发布范围可能不同；你可以随时运行 `cjv toolchain list-remote` 查询当前平台实际发布了哪些版本。

## 手册章节导览

### 入门与运行环境

- **[安装 cjv](installation/index.md)** 介绍一键安装脚本、固定 `cjv` 本体版本、使用 GitCode 镜像、手动下载归档以及配置 `PATH` 环境变量。
- **[快速上手](basic-usage.md)** 演示最核心的日常操作：验证安装状态、设置全局默认工具链、为项目固定版本，以及更新和卸载工具链。
- **[运行命令与配置环境](runtime-environment.md)** 解释代理命令的工作机制，说明如何通过 `cjv run` 或 `cjv exec` 运行工具及编译后的仓颉程序（自动配置运行时动态库路径），以及如何通过 `cjv envsetup` 将 SDK 环境直接加载到当前终端会话。

### 管理工具链、组件与交叉编译

- **[工具链与版本](concepts/toolchains.md)** 介绍发布通道（`lts`、`sts`、`nightly`）、固定版本、次版本与日期简写、主机平台后缀以及自定义工具链的概念。
- **[项目工具链](toolchain-file.md)** 说明 `cjv` 选择工具链的优先级顺序，演示如何在团队项目中提交 `cangjie-sdk.toml` 以统一编译器、组件和交叉编译目标，并介绍本机目录覆盖（`cjv override`）。
- **[组件与离线文档](concepts/components.md)** 介绍如何安装和管理工具链附属的 `stdx` 扩展库与离线 HTML 文档（`cjv doc`），以及如何链接本机自行构建的 `stdx` 目录。
- **[自定义 SDK](install-from-url.md)** 说明如何将本机已解压的 SDK 目录注册为自定义工具链，或从本地 `.zip` / `.tar.gz` 归档及 HTTP(S) URL 安装自定义 SDK。
- **[交叉编译](cross-compilation.md)** 阐明宿主平台（host）与编译目标（target）的区别，演示如何为宿主工具链安装和使用 `ohos`、`android` 等交叉编译 SDK。

### 配置、自动化与企业部署

- **[配置](configuration.md)** 与 **[环境变量](environment-variables.md)** 详细说明 `~/.cjv/settings.toml`、系统后备配置继承、`<CJV_HOME>` 数据目录布局、可复用离线下载缓存以及 SDK 文件硬链接复用。
- **[自动化与 JSON 接口](automation.md)** 面向 CI 流水线与工具集成开发者，涵盖 `cjv pin --check`、CI 目录与配置隔离、`--offline` 离线构建、`cjv doctor` 环境自检、更新历史回滚以及 `--json` 输出兼容规范。
- **[网络代理](network-proxies.md)** 说明如何配置 HTTP、HTTPS 和 SOCKS5 代理变量及企业私有 TLS 证书。
- **[企业与离线部署](enterprise/index.md)** 与 **[内部分发源](enterprise/distribution-server.md)** 介绍如何搭建内网 SDK 镜像源、下发系统后备配置以及在完全离线的物理隔离终端上部署工具链。

### 命令参考与参与开发

需要查阅具体子命令参数与安装更新策略时，请翻阅 **[命令参考](command-reference.md)**；遇到常见报错或疑问时，请查看 **[常见问题](faq.md)**。

`cjv` 与本手册的源码托管在 <https://github.com/Zxilly/cjv>。如果在使用中发现问题或希望参与开发，欢迎访问 [Issue 列表](https://github.com/Zxilly/cjv/issues) 并阅读 [**cjv 开发指南**](https://cjv.zxilly.com/book/dev-guide/zh-CN/)。
