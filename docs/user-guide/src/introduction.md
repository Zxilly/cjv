# 简介

cjv(Cangjie Version Manager)是[仓颉](https://cangjie-lang.cn/)编程语言 SDK 的工具链管理器。它在一台机器上管理多个仓颉 SDK 安装,处理版本切换,并对 `cjc`、`cjpm` 等 SDK 工具提供透明的代理执行。

## cjv 解决什么问题

直接安装仓颉 SDK 时,系统里往往只能保留一个版本,切换版本意味着重新下载、解包、改 `PATH`。当你手头有多个项目、各自需要不同的 SDK 时,这会很麻烦。

cjv 把 LTS、STS、nightly 和具体版本的 SDK 并排安装在同一个主目录下,互不干扰。你可以为整机设置一个默认工具链,也可以为单个目录或单条命令指定工具链;cjv 会按环境变量、目录覆盖、工具链文件、默认值的优先级,解析出每次该用哪个 SDK。

直接运行 `cjc`、`cjpm` 等命令时,cjv 会把调用代理到解析出的工具链,并注入运行时和组件所需的环境变量(如 `CANGJIE_STDX_PATH_DYNAMIC`)。你不需要手动改 `PATH` 或导出变量。

在此之上,cjv 还支持交叉编译目标 SDK、`stdx` 等扩展组件、离线文档,以及配置运行时环境以直接运行编译产物。

## 适合谁用

cjv 面向需要在 LTS、STS、nightly 之间来回切换的仓颉开发者,以及在同一台机器上维护多个项目、各项目锁定不同 SDK 版本的人。如果你做交叉编译(如鸿蒙 OHOS、Android),需要在宿主 SDK 之外管理目标 SDK,cjv 也能帮上忙。团队可以通过工具链文件让版本切换可复现、可随项目走。

## 上手路径

- [安装 cjv](installation/index.md):获取并安装 cjv 本体。
- [快速上手](basic-usage.md):安装第一个工具链、设默认、运行命令。
- [核心概念](concepts/index.md):了解工具链、通道、组件、代理与覆盖等术语,理解 cjv 的工作方式。

cjv 以 Apache-2.0 协议开源,源码见 <https://github.com/Zxilly/cjv>。
