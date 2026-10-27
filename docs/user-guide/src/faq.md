# 常见问题

## 为什么选中的 SDK 版本不对？

请先运行 `cjv show active` 查看当前选中的工具链及其生效来源，再按以下优先级依次检查：

1. 命令行显式参数（全局 `+name` 或 `--toolchain`）。
2. `CJV_TOOLCHAIN` [环境变量](environment-variables.md)。
3. 当前目录及父目录中的本机目录覆盖（`cjv override list`）和 `cangjie-sdk.toml`。`cjv` 从当前目录逐层向上检查，在同一层目录中目录覆盖优先于项目文件；完整规则见[项目工具链](toolchain-file.md#选择顺序)。
4. 全局默认工具链（`cjv default`）。

如果在当前终端中执行过 `cjv envsetup`，shell 的 `PATH` 会直接指向具体 SDK 的工具目录，绕过代理选择。打开新终端可恢复代理自动选择，也可以重新加载所需工具链的环境。

## 项目文件拼错字段后会怎样？

`cangjie-sdk.toml` 中的未知字段会打印警告，其余已识别字段继续解析。但如果剩余字段没有提供有效的 `channel` 或 `path`（或者文件为空、存在 TOML 语法错误），命令会直接报错退出，而不会静默退回默认工具链。需要恢复上级目录或全局默认选择时，请删除该文件。

## 编译产物提示找不到运行时库怎么办？

编译生成的仓颉程序通常需要动态链接 SDK 或 `stdx` 中的运行时库。使用 `cjv exec` 运行程序，即可由 `cjv` 自动为该子进程配置动态库搜索路径：

```bash
cjv exec ./my_binary
```

你也可以为当前终端会话求值加载 `cjv envsetup` 的输出。各 shell 的具体写法见[运行命令与配置环境](runtime-environment.md)。

## 离线环境怎样安装和使用？

可以根据实际环境选择以下三种方案之一：

- **复用预填充缓存（`--offline`）** — 在联网机器上执行安装后，将 `<CJV_HOME>/cache/` 与 `cangjie-sdk.toml` 一起复制到离线环境，加上 `--offline`（或 `CJV_OFFLINE=1`）运行，见[自动化与 JSON 接口](automation.md#配置与网络)。
- **使用本地 SDK 归档或目录（`toolchain link`）** — 通过 `cjv toolchain link` 注册本地 `.zip` / `.tar.gz` 归档或已解压目录，见[自定义 SDK](install-from-url.md)。
- **部署企业内部分发源（`dist_server`）** — 受管终端配置[内部分发源](enterprise/distribution-server.md)，并在系统后备配置中关闭自动补装和自更新检查，见[企业与离线部署](enterprise/index.md)。

请注意，已安装的工具链可以完全在本地运行，但离线构建项目时还需提前准备好 `cjpm` 的项目依赖包。

## 自定义工具链为什么不能用 `component add` 下载组件？

通过 `cjv toolchain link` 注册的自定义 SDK 没有上游官方发行记录，因此 `cjv component add` 无法确定对应的组件制品。替代方式如下：

- **归档安装**可自动安装随包附带的 `stdx`，也可通过 `cjv component link stdx` 挂载本地 `stdx` 目录。
- **外部目录链接**由源 SDK 目录的所有者自行维护，不支持通过 `cjv` 组件命令修改。详见[组件与离线文档](concepts/components.md#自定义-sdk-的组件管理)。

## 卸载会删除什么？

- **`cjv uninstall <name>`** 删除所选 SDK、附属的交叉编译目标、`stdx` 和离线文档组件，并清理引用它的本机目录覆盖。如果它是当前默认工具链，`cjv` 会尝试改选其他已安装宿主，否则清除默认选择。通过 `toolchain link` 或 `component link` 引用的外部源目录会完整保留。
- **`cjv self uninstall`** 删除整个 `<CJV_HOME>` 数据目录及其中所有安装、缓存和历史快照，并清理 `PATH`。命令选项见[命令参考](command-reference.md#self-update--self-uninstall)。

## macOS 下载页为什么让我选架构？

部分浏览器在 Apple Silicon Mac 上仍会报告 Intel（`x86_64`）User-Agent，导致网页无法始终可靠地判断 Mac 的原生 CPU 架构。如果在终端中运行 `install.sh` 安装脚本，脚本会在本机自动识别；如果手动下载归档，可在终端运行 `uname -m`：`arm64` 对应 Apple Silicon，`x86_64` 对应 Intel。

## nightly 缺少校验和时怎么处理？

`cjv` 从 `nightly.json` 读取版本和下载地址。如果某个 SDK 条目暂时没有 `sha256` 值，`cjv` 会自动尝试请求 `<url>.sha256`；如果该 sidecar 文件尚未发布，则会提示完整性限制。企业内网镜像可以直接在 `nightly.json` 清单中填写完整的 64 位 SHA-256 校验和，详见[内部分发源格式](enterprise/distribution-server.md#nightlyjson)。
