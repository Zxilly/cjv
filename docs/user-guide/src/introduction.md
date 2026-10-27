# cjv 用户手册

cjv 是[仓颉](https://cangjie-lang.cn/) SDK 的版本管理器，可以在本机安装多套工具链，并为不同项目选择各自需要的版本。配置好命令路径后，`cjc`、`cjpm` 等命令会通过 cjv 调用所选 SDK。

## 开始使用

首次使用请先[安装 cjv](installation/index.md)，然后按[快速上手](basic-usage.md)检查工具链、配置项目和运行命令。

安装向导会根据当前平台查询可用通道。cjv 支持的平台与仓颉 SDK 的发布范围可能不同，具体版本可以通过 `cjv toolchain list-remote` 查询。

## 配置项目与管理 SDK

[工具链与版本](concepts/toolchains.md)说明通道、固定版本和名称格式。在团队项目中，可以提交 [cangjie-sdk.toml](toolchain-file.md)，统一声明版本、组件和目标 SDK。

扩展库和离线文档见[组件](concepts/components.md)，其他目标平台的 SDK 见[交叉编译](cross-compilation.md)。已有的 SDK 目录或归档也可以注册为[自定义工具链](install-from-url.md)。

运行编译产物或配置当前 shell 时，参见[运行命令与配置环境](runtime-environment.md)。

## 配置与参考

[配置](configuration.md)和[环境变量](environment-variables.md)说明用户设置与临时覆盖。[网络代理](network-proxies.md)适用于需要代理访问上游的环境；内部分发源、受管终端和完全离线安装见[企业与离线部署](enterprise/index.md)。

查阅具体参数时使用[命令参考](command-reference.md)，排查常见问题时使用[常见问题](faq.md)。

版本固定校验、CI 隔离、离线缓存、更新回退和 JSON 输出见[自动化与 JSON 接口](automation.md)。参与开发请阅读[开发指南](https://cjv.zxilly.com/book/dev-guide/zh-CN/)。
