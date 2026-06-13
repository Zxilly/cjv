# 快速上手

本章带你跑通从安装工具链到运行仓颉工具的完整流程，并介绍日常会用到的常用命令。读完即可上手，更深入的概念见 [核心概念](concepts/index.md)，每条命令的完整选项见 [命令参考](command-reference.md)。

本章假设你已经装好了 `cjv` 本身。如果还没有，先看 [安装 cjv](installation/index.md)。

## 五分钟跑通

下面四步演示了最典型的用法：安装一个 LTS 工具链、把它设为默认、检查状态，再用它运行命令。

```bash
# 1. 安装最新的 LTS 工具链
cjv install lts

# 2. 设为默认工具链
cjv default lts

# 3. 查看活跃和已安装的工具链
cjv show

# 4. 用指定工具链运行命令
cjv run lts cjc --version
```

`cjv install lts` 下载并安装 LTS 通道的最新工具链。`lts` 是一个通道名，cjv 会把它解析成具体版本。除了 `lts`，你还可以安装 `sts`、`nightly` 或某个确切的版本号。各通道的含义见 [通道](concepts/channels.md)。

`cjv default lts` 把 `lts` 记为默认工具链。设好默认后，未在项目里另行声明工具链时，cjv 都会使用它。

`cjv show` 列出当前活跃的工具链和所有已安装的工具链，便于确认安装结果。只想看其中一项时，可用 `cjv show active` 或 `cjv show installed`。

`cjv run lts cjc --version` 显式用 `lts` 工具链执行 `cjc --version`。`cjv run <工具链> <命令> [参数...]` 临时切到指定工具链运行某条命令，不改变默认设置，便于在多个工具链之间临时对比。

## 直接使用 `cjc` 和 `cjpm`（代理执行）

装好工具链并配置好 `PATH` 后，你不需要每条命令都加 `cjv run` 前缀，直接调用 SDK 工具即可：

```bash
cjc --version
cjpm build
```

cjv 在安装时会在自己的 `bin` 目录里为每个 SDK 工具创建一个代理符号链接。当你调用 `cjc`、`cjpm` 等命令时，实际运行的是这个代理。它先解析出当前应当使用哪个工具链，再把调用透明地转发给该工具链里真正的可执行文件，并自动注入必要的环境变量（如 stdx 已安装时的 `CANGJIE_STDX_PATH_DYNAMIC`、`CANGJIE_STDX_PATH_STATIC`）。

被代理的 SDK 工具包括 `cjc`、`cjc-frontend`、`cjpm`、`cjfmt`、`cjlint`、`cjdb`、`cjcov` 以及若干内部工具。平时怎么用仓颉的命令行工具，装上 cjv 之后照旧用，工具链切换由 cjv 在背后完成。

代理使用的工具链解析规则与 `cjv run` 一致，但不需要你指定工具链。它会按优先级自动判断：环境变量、目录覆盖、项目中的 `cangjie-sdk.toml`，最后回退到默认工具链。完整的解析顺序见 [代理](concepts/proxies.md) 与 [目标与覆盖](concepts/targets-overrides.md)。

如果解析到的工具链尚未安装，且你启用了 `auto_install` 设置，cjv 会在代理前自动把它装上，无需手动 `cjv install`。开启方式：

```bash
cjv set auto-install true
```

该设置的细节见 [配置](configuration.md)。

> 提示：仓颉编译出的二进制文件在运行时还需要正确的库搜索路径。运行自己编译出的程序见 [运行时环境](runtime-environment.md) 中的 `cjv exec` 与 `cjv envsetup`。

## 项目级工具链

把工具链声明写进项目，团队成员进入该目录后就会自动用上同一个工具链，无需各自手动切换。在项目根目录放一个 `cangjie-sdk.toml`：

```toml
[toolchain]
channel = "lts"
```

之后在该目录（或其子目录）里，无论是 `cjv run`、代理执行的 `cjc`/`cjpm`，还是 `cjv show active`，都会优先使用这里声明的工具链。该文件的完整字段（`channel`、`components`、`targets`）见 [工具链文件](toolchain-file.md)。

如果只想给某个目录临时绑定一个工具链而不提交文件，可以用目录覆盖：

```bash
cjv override set nightly
```

详见 [目标与覆盖](concepts/targets-overrides.md)。

## 常用命令一览

下面是日常最常用的命令。每条命令的完整参数和子命令见 [命令参考](command-reference.md)。

| 命令 | 说明 |
| --- | --- |
| `cjv install <工具链>` | 安装工具链（如 `lts`、`sts`、`nightly` 或具体版本） |
| `cjv uninstall <工具链>` | 卸载工具链 |
| `cjv update [工具链]` | 更新已安装的工具链 |
| `cjv default [工具链]` | 设置或显示默认工具链 |
| `cjv show` | 显示活跃和已安装的工具链 |
| `cjv run <工具链> <命令> [参数...]` | 用指定工具链运行命令 |
| `cjv exec [+工具链] <命令> [参数...]` | 在仓颉运行时环境中执行命令 |
| `cjv which <命令>` | 显示活跃工具链中某个 SDK 工具的路径 |
| `cjv check` | 检查可用更新（不安装） |
| `cjv override set <工具链>` | 为当前目录设置工具链覆盖 |
| `cjv component add <名称>...` | 为工具链安装组件（如 `stdx`） |
| `cjv self update` | 更新 cjv 自身到最新版本 |

## 下一步

- 想了解工具链、通道、组件、代理这些核心概念，从 [核心概念](concepts/index.md) 开始。
- 想给项目固定工具链版本，见 [工具链文件](toolchain-file.md)。
- 要做交叉编译，见 [交叉编译](cross-compilation.md)。
- 要运行自己编译出的二进制文件，见 [运行时环境](runtime-environment.md)。
- 查所有命令、选项和环境变量，见 [命令参考](command-reference.md) 与 [环境变量](environment-variables.md)。
