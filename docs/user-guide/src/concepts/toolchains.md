# 工具链与版本

一个工具链是一套完整的仓颉 SDK，包含编译器、构建工具和运行时库。cjv 将多套 SDK 保存在 `<CJV_HOME>/toolchains/` 下，并将[组件](components.md)和离线文档存放在独立目录中。本章介绍通道与版本名称语法、跟踪通道与固定版本的区别、主机平台后缀以及自定义工具链。

## 通道与版本名称

使用以下名称格式选择发布通道、具体版本或自定义 SDK：

| 名称格式 | 含义 |
| --- | --- |
| `lts` | 跟踪长期支持通道（Long-Term Support） |
| `sts` | 跟踪短期支持通道（Short-Term Support） |
| `nightly` | 跟踪已发布清单中的每日开发构建通道 |
| `lts-1.0.5` | 固定某个通道中的具体发行版 |
| `1.0.5` | 省略通道名，由 cjv 在稳定通道中查找匹配版本 |
| `sts-1.2`、`1.2` | 选择该次版本系列中的最新稳定补丁版本 |
| `nightly-2026-10-04` | 选择指定日期发布的最新 nightly 构建 |
| `my-sdk` | 选择通过 `cjv toolchain link` 注册的自定义工具链 |

上表中的版本号和日期仅用于演示语法格式。请运行以下命令查询当前分发源中实际可用的版本：

```bash
cjv toolchain list-remote --channel lts
cjv toolchain list-remote --channel nightly --limit 10
```

通道名称不区分大小写。自定义工具链名称不能与官方通道或版本名称冲突，不能包含路径分隔符，不能以 `+` 开头，也不能为空字符串、`.` 或 `..`。

## 跟踪通道与固定版本

### 独立管理与更新

跟踪通道与固定版本是两个彼此独立的安装实例：

```bash
cjv install lts
cjv install lts-1.0.5
```

运行 `cjv update lts` 只会升级跟踪通道 `lts`，而固定的 `lts-1.0.5` 保持不变；两者的附属组件也各自独立管理。日常运行 `cjc` 时会直接使用本机已安装的版本，只有显式执行 `cjv install` 或 `cjv update` 时才会升级。

### 版本解析与 nightly 策略

像 `sts-1.2` 这样的次版本简写和 `nightly-2026-10-04` 这样的日期简写，在安装时都会解析为具体的发行版并固定下来。如果需要团队所有成员使用完全一致的发行版，请在 [cangjie-sdk.toml](../toolchain-file.md) 中写入完整版本名，或运行 `cjv pin`。

安装或更新 `nightly` 通道时，cjv 会结合所需的组件和交叉编译目标，在清单保留的历史记录中查找兼容版本。默认情况下，cjv 不会对已安装的 nightly 工具链执行降级；如果需要允许回退到更旧的兼容 nightly 构建，请传入 `--allow-downgrade`。所有安装与更新策略见[命令参考](../command-reference.md#安装与更新策略)。

## 主机平台与交叉编译目标

你可以在工具链名称末尾附加主机平台后缀，例如 `sts-linux-x64` 或 `sts-1.2.0-darwin-arm64`。不同主机平台会作为独立工具链安装；显式写出当前默认主机平台的名称，与省略平台后缀的名称指向同一个安装。

安装其他操作系统的宿主 SDK 并不能让该系统的程序直接在本机运行。如果需要在当前机器上编译其他平台的目标程序，请通过[交叉编译](../cross-compilation.md)将目标 SDK 附加到当前宿主工具链上。交叉编译 SDK 不能单独设为全局默认或当前活跃工具链。

## 自定义 SDK

你可以将本机已有的 SDK 目录注册为具名的自定义工具链：

```bash
cjv toolchain link my-sdk /path/to/local/sdk
cjv default my-sdk
```

目录链接直接指向你自行维护的 SDK 目录；运行 `cjv uninstall my-sdk` 只会删除 cjv 中的链接记录，保留源目录内容不变。

如果希望由 cjv 托管 SDK 副本或为其挂载本地 `stdx` 扩展库，请改从本地归档或 URL 安装。`cjv update` 不会更新自定义工具链，`cjv component add` 也无法为自定义工具链下载官方组件。目录链接与归档安装的详细对比见[自定义 SDK](../install-from-url.md)。

## 查看已安装的工具链

运行 `cjv toolchain list` 可以查看本机已安装的所有工具链；运行 `cjv show active` 可以查看当前激活的工具链及生效来源。接下来请阅读[项目工具链](../toolchain-file.md)，了解 cjv 如何为不同项目目录自动选择工具链。
