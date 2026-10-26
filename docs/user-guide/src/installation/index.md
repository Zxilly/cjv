# 安装 cjv

`cjv` 是独立分发的单文件可执行程序，安装前不需要在系统中预装仓颉 SDK。完成安装后，`cjv` 会将自身以及 `cjc`、`cjpm` 等标准 SDK 工具的代理入口放置在 `<CJV_HOME>/bin` 目录下（Unix 默认为 `~/.cjv/bin`，Windows 默认为 `%USERPROFILE%\.cjv\bin`）。

## 使用安装脚本

安装 `cjv` 最简单的方式是运行对应平台的官方安装脚本。

在 Linux、macOS 或鸿蒙 HiShell 中执行：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

在 Windows PowerShell 中执行：

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

安装脚本会自动识别当前操作系统与 CPU 架构，下载并校验 `cjv` 可执行文件的 SHA-256 校验和，随后运行 `cjv init`。在交互式终端中，`cjv init` 会显示安装向导；当标准输入不是终端时，则会直接使用默认选项完成初始化。

### 非交互与自动化安装参数

两个安装脚本都支持传入命令行参数，以便在 CI 流水线或自动化脚本中跳过交互提示，或者先只安装 `cjv` 本体（后续再由项目的 `cangjie-sdk.toml` 按需选择并安装 SDK）：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Yes -DefaultToolchain none
```

Unix 安装脚本（`install.sh`）自身消费 `--mirror` 和 `--version` 两个参数，并将其余参数原样转发给 `cjv init`。PowerShell 安装脚本（`install.ps1`）则提供对应的 `-Mirror`、`-Version`、`-Yes`、`-DefaultToolchain` 和 `-NoModifyPath` 参数。`cjv init` 支持的全部选项见[命令参考](../command-reference.md#init)。

### 固定 cjv 本体版本

在需要严格复现构建环境的自动化场景中，可以指定要安装的 `cjv` 发布标签（包括预发布版本）：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --version v0.4.1 -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Version v0.4.1 -Yes -DefaultToolchain none
```

请将示例中的 `v0.4.1` 替换为实际需要的版本标签。两个脚本也会读取 `CJV_VERSION` [环境变量](../environment-variables.md)，但命令行显式传入的版本参数优先级更高；两者都未指定时，默认下载最新稳定版。

请注意，`--version` / `CJV_VERSION` 选择的是 `cjv` 管理器本体的版本，而 `--default-toolchain` / `-DefaultToolchain` 选择的是仓颉 SDK 工具链，两者彼此独立。此外，显式设置的下载地址变量（`CJV_UPDATE_ROOT`、`CJV_GITHUB_ROOT`、`CJV_GITCODE_ROOT`）优先于版本参数；如果使用企业内部制品源，应将地址直接指向所选版本的制品目录。

### 初始化时的 SDK 选择规则

默认情况下，`cjv init` 会查询当前主机平台可用的发布通道，并按 `lts`、`sts`、`nightly` 的顺序自动选择第一个可用通道。你可以使用 `--default-toolchain <name>` 明确指定要安装的工具链，或者使用 `--default-toolchain none` 只安装 `cjv` 本体。

如果初始化过程中下载或安装 SDK 失败，命令会返回非零退出码，但已安装好的 `cjv` 本体会被保留下来。错误输出中会直接给出带程序绝对路径的重试命令（无需先刷新 `PATH` 即可运行），你也可以运行 `cjv toolchain list-remote` 查询当前分发源提供的可用版本。

### 使用 GitCode 镜像

当所处网络环境访问 GitHub 不稳定时，可以加上 `--mirror`（或 `-Mirror`）从 GitCode 镜像源安装：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

镜像构建默认使用 GitCode 获取仓颉版本清单和 `cjv` 自更新。如果在企业内网自建制品库，请参阅[企业与离线部署](../enterprise/index.md)。

## 手动下载归档安装

如果不希望在终端直接运行在线脚本，也可以从 [GitHub Releases](https://github.com/Zxilly/cjv/releases) 下载对应平台的归档文件，解压后将 `cjv`（Windows 为 `cjv.exe`）放入 `PATH` 目录，再运行 `cjv init`：

| 平台 | 发布归档文件名 |
| --- | --- |
| Linux x86_64 | `cjv_linux_amd64.tar.gz` |
| Linux ARM64 | `cjv_linux_arm64.tar.gz` |
| macOS Apple Silicon | `cjv_darwin_arm64.tar.gz` |
| macOS Intel | `cjv_darwin_amd64.tar.gz` |
| Windows x86_64 | `cjv_windows_amd64.zip` |
| 鸿蒙 x86_64 | `cjv_openharmony_amd64.tar.gz` |
| 鸿蒙 ARM64 | `cjv_openharmony_arm64.tar.gz` |

[GitCode Releases](https://gitcode.com/Zxilly/cjv/releases) 提供前缀为 `cjv-mirror_` 的对应镜像归档。此外，[官方网站下载页](https://cjv.zxilly.com)还提供 `cjv-init`（及 `cjv-init.exe`）独立安装器，运行后会直接启动安装向导。

## 从源码编译安装

如果本机已安装满足仓库 `go.mod` 版本要求的 Go 工具链，可以直接从源码编译安装：

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
cjv init
```

Go 会将编译出的程序放入 `GOBIN`（未配置时为 `$(go env GOPATH)/bin`）；将该目录加入 `PATH` 或运行 `cjv init` 即可。关于自定义构建标签与跨平台编译，请参阅开发指南中的[从源码构建](https://cjv.zxilly.com/book/dev-guide/zh-CN/building.html)。

## PATH 配置与安装验证

SDK 代理命令（`cjc`、`cjpm` 等）位于 `<CJV_HOME>/bin`，默认路径为 `~/.cjv/bin`。运行 `cjv init` 会自动配置 `PATH`（Windows 写入当前用户注册表 `PATH`，Unix 写入对应的 shell 配置文件）。即使跳过 `init` 直接安装 SDK，首次工具链安装也会自动尝试配置 `PATH`。

安装完成后，打开一个**新终端**并运行以下命令验证：

```bash
cjv --version
```

如果希望自行管理 `PATH`，可以在运行 `cjv init` 时传入 `--no-modify-path`（PowerShell 安装脚本传入 `-NoModifyPath`），或者设置环境变量 `CJV_NO_PATH_SETUP=1`。接下来请继续阅读[快速上手](../basic-usage.md)安装并切换工具链。
