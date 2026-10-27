# 工具链与版本

工具链是一份仓颉 SDK 安装，包含编译器（`cjc`）、包管理器（`cjpm`）、构建工具和运行时库。cjv 将工具链安装在 `<CJV_HOME>/toolchains/` 下，将可选[组件](components.md)（`stdx` 和离线文档）存放在独立目录中。

`cjv` 支持多种类型的工具链：最常见的是跟踪官方发布通道（`lts`、`sts` 和 `nightly`）的工具链，同时也可以安装官方清单中的特定固定版本、其他主机平台的工具链，以及来自本机目录或归档文件的[自定义工具链](#自定义-sdk)。

## 通道与名称

标准发布通道与版本工具链的名称遵循以下格式：

```text
[<channel>-]<version>[-<host>]
<channel>[-<date>][-<host>]

<channel>       = lts|sts|nightly
<version>       = <major.minor>|<major.minor.patch>|<nightly-version>
<date>          = YYYY-MM-DD
<host>          = <os>-<arch>
```

下表展示了不同工具链选择器的具体含义：

| 名称 | 含义 |
| --- | --- |
| `lts` | 跟踪长期支持通道（Long-Term Support） |
| `sts` | 跟踪短期支持通道（Short-Term Support） |
| `nightly` | 跟踪已发布清单中的每日开发构建通道 |
| `lts-1.0.5` | 固定通道中的具体发行版本 |
| `1.0.5` | 省略通道前缀，由 `cjv` 在稳定通道中查找匹配版本 |
| `sts-1.2`、`1.2` | 选择该次版本系列中的最新稳定补丁版本 |
| `nightly-2026-10-04` | 选择该日期发布的最新 nightly 构建 |
| `my-sdk` | 使用 `cjv toolchain link` 创建的自定义工具链 |

表中的版本号和日期仅用于演示名称语法。要查询当前分发源为你的平台实际发布了哪些版本，请运行 `cjv toolchain list-remote`：

```bash
cjv toolchain list-remote --channel lts
cjv toolchain list-remote --channel nightly --limit 10
```

官方通道名称不区分大小写。自定义工具链名称不能与官方通道或版本格式冲突，不能包含路径分隔符或以 `+` 开头，也不能是 `.`、`..` 或空字符串。

## 跟踪通道与固定版本

当你分别安装通道名 `lts` 和具体版本 `lts-1.0.5` 时，`cjv` 会将它们作为两个完全独立的安装实例管理：

```bash
cjv install lts
cjv install lts-1.0.5
```

之后运行 `cjv update lts`（或 `cjv update`）时，`cjv` 只会将跟踪通道 `lts` 更新到该通道的最新版本，而固定的 `lts-1.0.5` 保持原样不变；两者的附属组件和交叉编译目标也分别独立管理。日常运行 `cjc` 时始终使用本机已安装的版本，只有显式执行 `cjv install` 或 `cjv update` 时才会升级。

次版本简写（如 `sts-1.2`）和日期简写（如 `nightly-2026-10-04`）在安装时会被解析为具体的发行版并固定安装。如果需要团队成员与 CI 使用完全一致的发行版，请在 [`cangjie-sdk.toml`](../toolchain-file.md) 中写入完整版本名（或运行 `cjv pin` 自动解析写入）。

### Nightly 通道与组件兼容性

由于 `nightly` 构建发布频繁，有时最新一次 nightly 构建可能尚未包含某个可选组件（如 `stdx`、`docs`）或交叉编译 SDK。

在安装或更新 `nightly` 通道时，`cjv` 会综合考虑你请求的（或该通道上已安装的）组件与交叉 SDK 集合，自动在 `nightly.json` 保留的历史记录中查找最新的兼容版本。默认情况下，`cjv` 不会将已安装的 `nightly` 降级到更旧的版本；如果允许选择更旧的兼容 nightly 构建，可以传入 `--allow-downgrade`。其他安装与更新策略详见[命令参考](../command-reference.md#安装与更新策略)。

## 主机平台

你可以在工具链名称末尾附加主机平台后缀，例如 `sts-linux-x64` 或 `sts-1.2.0-darwin-arm64`。不同主机平台作为独立工具链安装；显式写出当前机器默认主机平台的名称（例如在 Linux x86_64 机器上写 `lts-linux-x64`），与省略平台后缀的名称（`lts`）指向同一个安装实例。

请注意，安装其他操作系统的宿主 SDK 并不能让该平台的程序直接在本机运行。如果需要在当前机器上为其他平台编译代码，请通过[交叉编译目标管理](../cross-compilation.md)将交叉 SDK 附加到当前宿主工具链上。交叉编译 SDK 依附于宿主工具链，不能单独设为全局默认或当前活跃工具链。

## 自定义 SDK

对于自行编译仓颉 SDK 或处于离线环境的开发者，`cjv` 支持管理本机 SDK 目录与归档文件。要链接本机已解压的 SDK 目录，可以运行：

```bash
cjv toolchain link my-sdk /path/to/local/sdk
cjv default my-sdk
```

目录链接直接引用你自行维护的已有 SDK 目录；运行 `cjv uninstall my-sdk` 只会移除 `cjv` 中的链接记录，保留你的源目录不变。

如果需要由 `cjv` 托管独立的 SDK 副本，或者为自定义 SDK 挂载本地 `stdx` 扩展库，可以改向 `cjv toolchain link` 传入本地归档（`.zip` / `.tar.gz`）或 HTTP(S) URL，详见[自定义 SDK](../install-from-url.md)。自定义工具链不会通过 `cjv update` 更新，也没有可供 `cjv component add` 下载的官方组件。

运行 `cjv toolchain list` 可以查看本机安装的所有工具链，运行 `cjv show active` 可以查看当前激活的工具链及其选择来源。完整选择规则见[项目工具链](../toolchain-file.md)。
