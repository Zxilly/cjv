# 常见问题

本章解答关于工具链版本选择、项目配置文件、运行时动态库、离线安装以及平台下载的常见问题。

## 版本选择与项目文件

### 为什么选中的 SDK 版本不对？

请先运行 `cjv show active` 查看当前选中的工具链及其生效来源，然后按以下优先级顺序排查：

1. 命令行显式参数（全局 `+name` 或 `--toolchain`）。
2. `CJV_TOOLCHAIN` [环境变量](environment-variables.md)。
3. 当前目录及父目录中的本机目录覆盖（`cjv override list`）和 `cangjie-sdk.toml` 文件。在同一层目录中，目录覆盖优先于 `cangjie-sdk.toml`。完整规则见[项目工具链](toolchain-file.md#工具链选择顺序)。
4. 如果你在当前终端会话中执行过 `cjv envsetup`，当前 shell 的 `PATH` 会直接指向某个具体 SDK。请打开一个新终端恢复自动代理选择，或再次运行 `cjv envsetup` 切换到所需工具链。

### `cangjie-sdk.toml` 写错了字段名会怎样？

遇到未知字段名时，cjv 会打印警告并继续解析其余已知字段。但如果剩余字段没有提供有效的 `channel` 或 `path`，或者存在 TOML 语法错误、文件内容为空，命令会直接报错退出，而不会静默退回默认工具链。如果需要恢复父目录或全局默认选择，请删除该文件。

## 运行环境与组件

### 编译产物提示找不到运行时库怎么办？

编译生成的仓颉程序通常需要加载 SDK 或 `stdx` 中的动态库。请使用 `cjv exec` 运行程序，由 cjv 为子进程自动配置动态库搜索路径（`LD_LIBRARY_PATH`、`DYLD_LIBRARY_PATH` 或 `PATH`）及 `stdx` 变量：

```bash
cjv exec ./my_binary
```

你也可以在当前终端会话中求值执行 `cjv envsetup` 的输出。各 shell 的具体写法见[运行命令与配置环境](runtime-environment.md)。

### 自定义工具链为什么不能用 `component add` 下载组件？

通过 `cjv toolchain link` 注册的自定义 SDK 没有上游官方发行记录，因此 `cjv component add` 无法匹配对应的组件制品：

- 如果自定义工具链是从[本地归档或 URL](install-from-url.md) 安装的，它可以自动安装随包附带的 `stdx`，也支持通过 `cjv component link stdx` 挂载本地 `stdx` 目录。
- 如果是直接链接的外部解压目录，其内容由源 SDK 目录的所有者维护，不支持通过 cjv 组件命令修改。详见[组件与离线文档](concepts/components.md#为自定义-sdk-管理组件)。

## 离线安装、卸载与平台架构

### 离线环境怎样安装和使用 cjv？

请根据现场条件选择对应的离线方案：

- **复用离线缓存**：在联网机器上执行 `cjv install` 生成 `<CJV_HOME>/cache/`，将 `cache/` 与 `cangjie-sdk.toml` 复制到离线机器并加上 `--offline` 运行，见[自动化与 JSON 接口](automation.md#ci-隔离与离线缓存)。
- **使用本地 SDK 归档或目录**：通过 `cjv toolchain link` 注册本地 `.zip`、`.tar.gz` 或已解压目录，见[自定义 SDK](install-from-url.md)。
- **企业内网环境**：部署[内部分发源](enterprise/distribution-server.md)，并在系统后备配置中关闭自动补装和自更新检查，见[企业与离线部署](enterprise/index.md)。

注意：已安装的工具链可完全在本地运行，但离线构建项目时还需提前准备好 `cjpm` 的项目依赖包。

### `cjv uninstall` 和 `cjv self uninstall` 会删除哪些文件？

- `cjv uninstall <name>`：删除指定的 SDK、附属的交叉编译目标、`stdx` 和离线文档组件，并清理引用该名称的目录覆盖。如果卸载的是默认工具链，cjv 会尝试改选其他已安装宿主，否则清除默认选择。通过 `toolchain link` 或 `component link` 指向的外部源目录会被保留。
- `cjv self uninstall`：删除整个 `<CJV_HOME>` 数据目录及其中所有已安装的工具链、组件、缓存和历史快照，并清理 `PATH` 配置。详见[命令参考](command-reference.md#self-update-与-self-uninstall)。

### macOS 下载页为什么让我手动选择架构？

部分 macOS 浏览器在 Apple Silicon 芯片上仍会报告 Intel（`x86_64`）User-Agent，网页无法始终可靠地判断 Mac 的真实 CPU 架构。如果在终端中运行 `install.sh` 安装脚本，脚本会在本机自动识别架构；如果手动下载归档，请在终端运行 `uname -m`：`arm64` 对应 Apple Silicon，`x86_64` 对应 Intel。

### nightly 缺少校验和时怎么处理？

cjv 从 `nightly.json` 读取 nightly 版本和下载地址。如果某个 SDK 条目的 `sha256` 字段为空，cjv 会自动尝试请求 `<url>.sha256`；如果该 sidecar 校验和文件尚未发布，cjv 会明确提示完整性限制。企业自建内网镜像可以直接在 `nightly.json` 中填写完整的 64 位 SHA-256 校验和，详见[内部分发源](enterprise/distribution-server.md#nightlyjson-格式契约)。
