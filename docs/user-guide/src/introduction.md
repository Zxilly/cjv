# cjv 用户手册

cjv 是[仓颉](https://cangjie-lang.cn/) SDK 的版本管理器。它可以在本机安装多套工具链，并为每个项目选择各自需要的 SDK 版本。配置好命令路径后，`cjc`、`cjpm` 等代理命令会通过 cjv 调用当前选中的 SDK。

## 开始使用

首次使用 cjv 时，请按以下顺序完成安装和基础验证：

1. 阅读[安装 cjv](installation/index.md)，通过安装脚本、预编译归档或源码完成安装。
2. 阅读[快速上手](basic-usage.md)，设置默认工具链、配置项目版本并执行日常更新。
3. 阅读[运行命令与配置环境](runtime-environment.md)，了解如何调用 SDK 工具、运行仓颉编译产物或加载当前终端环境。

安装向导会根据当前平台查询可用通道。cjv 支持的运行平台与仓颉 SDK 的发布范围可能不同，可随时运行 `cjv toolchain list-remote` 查询当前平台可用的版本。

## 管理 SDK 与项目工具链

使用以下章节管理不同版本、扩展组件和跨平台编译目标：

- [工具链与版本](concepts/toolchains.md)：说明发布通道、固定版本、主机平台后缀与名称语法。
- [项目工具链](toolchain-file.md)：说明如何在团队项目中提交 `cangjie-sdk.toml`，共享版本、组件和目标 SDK 声明。
- [组件与离线文档](concepts/components.md)：说明如何管理 `stdx` 扩展库与本地离线 HTML 文档。
- [自定义 SDK](install-from-url.md)：说明如何链接本机已解压的 SDK 目录，或从本地归档与 URL 安装自定义工具链。
- [交叉编译](cross-compilation.md)：说明如何为宿主工具链安装和使用 `ohos`、`android` 等交叉目标 SDK。

## 配置与部署

使用以下章节调整本机设置、构建 CI 流水线或部署企业内网环境：

- [配置](configuration.md)与[环境变量](environment-variables.md)：说明用户设置文件、数据目录结构、下载缓存与单进程临时覆盖。
- [自动化与 JSON 接口](automation.md)：说明版本固定校验、CI 环境隔离、离线缓存复用、更新历史回退与 JSON 输出约定。
- [网络代理](network-proxies.md)：说明 HTTP、HTTPS、SOCKS5 代理变量及企业私有 CA 证书配置。
- [企业与离线部署](enterprise/index.md)与[内部分发源](enterprise/distribution-server.md)：说明系统后备设置、内网镜像清单格式及完全离线部署方案。

## 参考与支持

- [命令参考](command-reference.md)：汇总所有 CLI 子命令、参数选项和更新策略。
- [常见问题](faq.md)：提供版本选择异常、找不到运行时库、离线安装等问题的排查方法。
- 如果你需要从源码编译或参与开发 cjv 本体，请阅读 [cjv 开发指南](https://cjv.zxilly.com/book/dev-guide/zh-CN/)。
