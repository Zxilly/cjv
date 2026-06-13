# 常见问题

本章汇总使用 cjv 时最常遇到的问题。命令的完整说明见[命令参考](command-reference.md)，背景概念见[核心概念](concepts/index.md)。

## 安装 nightly 时为什么提示需要 GitCode API 密钥？

nightly 工具链发布在 GitCode 的 [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) 仓库。cjv 要查询该仓库的 `releases/latest` 接口才能解析出最新的 nightly 版本号，而这个 API 端点要求携带访问令牌。没有令牌时，`cjv install nightly`、`cjv update nightly`、`cjv check` 等命令会直接报错：

```text
查询 nightly 版本需要 GitCode API 密钥。请通过以下命令设置: cjv set gitcode-api-key <your-token>
```

在 [GitCode](https://gitcode.com/) 上生成一个个人访问令牌后，按以下任一方式提供：

```bash
# 持久化到 settings.toml（推荐）
cjv set gitcode-api-key <your-token>

# 或仅对当前会话生效（环境变量优先级高于持久化设置）
export CJV_GITCODE_API_KEY=<your-token>
```

只有 nightly 通道需要这个令牌。`lts`、`sts` 和具体版本号不依赖 GitCode API，无需配置即可安装。通道的区别见[通道](concepts/channels.md)。

> 注意：nightly 资产的 sha256 校验文件（sidecar）可能并不总是发布。当上游未提供校验文件时，cjv 会在打印明确提示后继续安装，仅依赖 TLS 保证传输完整性。

## 在 macOS 上为什么没有自动识别我的 CPU 架构？

浏览器无法跨浏览器可靠地读取 Mac 的 CPU 架构（Apple Silicon 还是 Intel）。Safari 与 Firefox 完全不暴露架构信息，Safari 甚至在 Apple Silicon 上仍把平台标识冻结为 `MacIntel`。因此 cjv 的网页安装向导在无法确定架构时不会猜测，而是采用两种回退策略。

命令安装给出的 `install.sh` 一行命令不写死架构，由脚本在你的机器上自行探测后下载匹配的二进制。手动下载则在下载页同时给出 Apple Silicon（arm64）与 Intel（x86_64）两个选项，由你按自己的机器选择。

如果你不确定本机架构，在终端运行 `uname -m`：输出 `arm64` 选 Apple Silicon，输出 `x86_64` 选 Intel。安装 cjv 自身之后，工具链的下载会由 cjv 根据本机真实架构自动解析，不再有这个问题。

## 在离线或受限网络环境下怎么用 cjv？

cjv 的多数操作都需要从上游下载资产，但有几种方式可以适配离线、内网或镜像环境。

cjv 提供 `mirror` 构建变体，其默认的工具链清单（manifest）指向 GitCode 而非 GitHub，适合 GitHub 访问不稳定的环境。两种构建的区别仅在默认 manifest 源。

工具链清单地址保存在 `~/.cjv/settings.toml` 的 `manifest_url` 字段，可手动改为你的内网镜像地址。清空该字段会恢复内置默认值。详见[配置](configuration.md)。

如果你已经拿到解压好的 SDK 目录，用 `cjv toolchain link` 直接挂载，不会触发下载：

```bash
cjv toolchain link mysdk /path/to/local/sdk
```

离线环境下无法 `cjv component add stdx`（需要下载 release 资产），改用 `cjv component link` 把本地 stdx 目录挂上去。标准通道也可用 `--force` 以本地目录替代下载：

```bash
cjv component link stdx /path/to/local/stdx --toolchain mysdk
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

`<path>` 必须是包含 `dynamic/` 和 `static/` 两个子目录的目录。详见[组件](concepts/components.md)。

也可以把 SDK 归档放到内网可访问的地址，再用 URL 安装，见[从 URL 安装工具链](install-from-url.md)。

此外，`CJV_MAX_RETRIES` 与 `CJV_DOWNLOAD_TIMEOUT` 可调节下载的重试次数与超时，应对慢速链路。完整列表见[环境变量](environment-variables.md)。

## 为什么 custom 工具链不能用 `cjv component add stdx`？

通过 `cjv toolchain link` 创建的 custom 工具链没有对应的官方 release 资产，cjv 不知道该从哪里下载 stdx，因此 `cjv component add stdx` 对它无效。请改用下列两种方式之一：

```bash
# 方式一：链接一个本地的 stdx 目录
cjv component link stdx /path/to/local/stdx --toolchain mysdk

# 方式二：从一个 SDK 归档安装时，归档内若自带 cangjie-stdx-* 内层包，
# cjv 会把它作为 stdx 组件一并安装（见“从 URL 安装工具链”）
```

`cjv component link` 创建的是指向你原始目录的符号链接（Windows 上回退为目录联接 / junction），`cjv component remove stdx` 与 `cjv toolchain uninstall` 都只删除链接，不会触碰你的原始数据。标准通道（lts / sts / nightly）也可以用 `cjv component link stdx ... --force` 以本地目录替代下载，适合离线或调试自编译 stdx 的场景。详见[组件](concepts/components.md)。

## 卸载一个工具链会清理哪些目录？

执行 `cjv uninstall <tc>`（等价于 `cjv toolchain uninstall <tc>`）时，与该工具链相关的三处目录都会被一并删除：

| 目录                         | 内容                                  |
| ---------------------------- | ------------------------------------- |
| `<CJV_HOME>/toolchains/<tc>` | SDK 本体                              |
| `<CJV_HOME>/stdx/<tc>`       | stdx 组件                            |
| `<CJV_HOME>/docs/<tc>`       | docs 与 stdx-docs 离线文档          |

stdx 与 docs 与工具链解耦、各自独立存放，但卸载时会被当作该工具链的附属物一起清掉，以免下次重装时残留过期的扩展资源。如果 `stdx/<tc>` 是通过 `cjv component link` 链接的本地目录，删除的只是符号链接，你的原始数据不受影响。

卸载同时还会清理 `settings.toml` 中指向该工具链的引用：如果它是默认工具链，默认设置会被清空；指向它的目录覆盖（override）也会被移除。

> 提示：`cjv self uninstall` 会删除整个 `~/.cjv` 目录，连同 cjv 自身、所有工具链、组件、文档和设置一起卸载。

## 从 URL 安装的工具链能跨操作系统吗？

不能。`cjv toolchain link` 的物化安装（本地归档或 URL）只支持与当前操作系统匹配的 SDK；若 SDK 面向的系统与本机不符，cjv 会在安装前拒绝：

```text
无法在 windows 上安装 linux SDK；此安装仅支持与当前系统匹配的 SDK
```

cjv 安装后要立刻校验 SDK 可用（运行其中的工具），异系统的二进制无法在本机执行。需要给其他系统准备 SDK 时，请到对应系统上安装。

这条限制只针对操作系统，不针对交叉编译目标。在同一台机器上为其他平台编译产物是支持的，那通过附加的目标 SDK 实现，见[交叉编译](cross-compilation.md)。

## 直接运行 `cjc`、`cjpm` 时 cjv 是怎么介入的？

cjv 在自己的 `bin/` 目录里为每个 SDK 工具创建了代理符号链接。直接调用这些工具时，cjv 会按既定优先级解析出活跃工具链，再把调用透明地转发给对应 SDK。解析优先级从高到低为：

1. `CJV_TOOLCHAIN` 环境变量
2. 目录覆盖（`cjv override set`）
3. 工具链文件 `cangjie-sdk.toml`（当前或父目录）
4. 默认工具链（`cjv default`）

如果设置里开启了 `auto-install` 而解析到的工具链尚未安装，cjv 会在转发前自动把它装好。详见[代理](concepts/proxies.md)与[工具链文件](toolchain-file.md)。

## 编译出的二进制运行时报找不到运行时库怎么办？

仓颉编译产物动态链接运行时库（如 `libcangjie-runtime`），需要正确的库搜索路径。用 `cjv exec` 在正确环境中一次性运行，或用 `cjv envsetup` 配置当前 shell 会话：

```bash
# 一次性执行，不影响当前 shell
cjv exec ./my_binary arg1 arg2

# 或为当前 shell 注入运行时环境（Bash/Zsh）
eval "$(cjv envsetup)"
```

详见[运行时环境](runtime-environment.md)。

## 我把 `channel` 拼成了 `channal`，为什么没报错只是个警告？

`cangjie-sdk.toml` 里未识别的键（如把表名写成 `[toolchian]`、把字段写成 `channal`）会以 warn 级别日志提示，但不会中断解析。cjv 会忽略它们并回退到下一级解析方式。如果发现工具链选择不符合预期，请先把日志级别调到 `warn`（默认）或 `debug` 检查这类提示：

```bash
CJV_LOG=debug cjv show active
```

工具链文件的字段语义见[工具链文件](toolchain-file.md)，环境变量见[环境变量](environment-variables.md)。
