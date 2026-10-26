# 安装 cjv

cjv 是单文件可执行程序，安装前不需要预装仓颉 SDK。完成本章安装后，请继续阅读[快速上手](../basic-usage.md)配置并使用工具链。

## 使用安装脚本

### 常规安装

在 Linux、macOS 或鸿蒙 HiShell 中执行：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh
```

在 Windows PowerShell 中执行：

```powershell
irm https://cjv.zxilly.com/install.ps1 | iex
```

脚本会自动识别当前平台，下载并校验 cjv，然后运行 `cjv init`。交互式终端会显示安装向导，非交互输入则使用默认选项。

### 非交互与自动化安装

在自动化脚本或 CI 环境中，可以显式跳过确认，或者先只安装 cjv 本体而不下载 SDK：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Yes -DefaultToolchain none
```

Unix 安装脚本直接处理 `--mirror` 和 `--version`，并将其他参数转发给 `cjv init`。PowerShell 脚本提供对应的 `-Mirror`、`-Version`、`-Yes`、`-DefaultToolchain` 和 `-NoModifyPath` 参数。`cjv init` 的完整参数列表见[命令参考](../command-reference.md#init)。

### 固定 cjv 版本

自动化安装可以指定确切的 cjv 发布标签（包括预发布版本）：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --version v0.4.1 -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Version v0.4.1 -Yes -DefaultToolchain none
```

请将示例中的 `v0.4.1` 替换为实际需要的版本标签。两个脚本也会读取 `CJV_VERSION` 环境变量，但命令行参数优先级更高。未指定版本时，脚本默认下载最新稳定版。

该版本选项仅控制 cjv 本体版本；`--default-toolchain`（或 `-DefaultToolchain`）则独立控制要安装的仓颉 SDK。

如果显式设置了 `CJV_UPDATE_ROOT`、`CJV_GITHUB_ROOT` 或 `CJV_GITCODE_ROOT` 下载地址，它们会优先于版本参数生效。使用企业内部源时，应将下载地址直接指向目标版本的制品目录。相关变量说明见[环境变量](../environment-variables.md)。

### 初始化时的 SDK 选择

默认情况下，`cjv init` 会查询当前平台的可用通道，并按 `lts`、`sts`、`nightly` 的顺序选择第一个可用通道。你可以传入 `--default-toolchain <name>` 指定特定工具链，或传入 `--default-toolchain none` 只安装 cjv。

如果 SDK 下载或安装失败，命令会返回非零退出码，但保留已安装好的 cjv 本体。你可以直接运行错误信息中给出的重试命令，或运行 `cjv toolchain list-remote` 查询可用版本。

### 使用 GitCode 镜像

当 GitHub 访问受限或速度较慢时，可以通过 `--mirror`（或 `-Mirror`）从 GitCode 镜像安装：

```bash
curl -sSf https://cjv.zxilly.com/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.com/install.ps1))) -Mirror
```

镜像构建默认使用 GitCode 获取版本清单和 cjv 自更新。如果需要配置企业自建制品库，请参见[企业与离线部署](../enterprise/index.md)。

## 手动下载归档

如果不希望使用脚本安装，可以手动下载发布归档：

1. 从 [GitHub Releases](https://github.com/Zxilly/cjv/releases) 下载对应平台的归档文件。
2. 将归档中的 `cjv`（Windows 为 `cjv.exe`）解压到 `PATH` 包含的目录中。
3. 运行 `cjv init` 创建代理命令入口并完成初始化。

| 平台 | 归档文件名 |
| --- | --- |
| Linux x86_64 | `cjv_linux_amd64.tar.gz` |
| Linux ARM64 | `cjv_linux_arm64.tar.gz` |
| macOS Apple Silicon | `cjv_darwin_arm64.tar.gz` |
| macOS Intel | `cjv_darwin_amd64.tar.gz` |
| Windows x86_64 | `cjv_windows_amd64.zip` |
| 鸿蒙 x86_64 | `cjv_openharmony_amd64.tar.gz` |
| 鸿蒙 ARM64 | `cjv_openharmony_arm64.tar.gz` |

[GitCode Releases](https://gitcode.com/Zxilly/cjv/releases) 提供前缀为 `cjv-mirror_` 的对应镜像归档。[官网下载页](https://cjv.zxilly.com)还提供独立的 `cjv-init` 可执行文件，运行后直接进入安装向导。

## 从源码编译安装

本机安装了满足仓库 `go.mod` 要求的 Go 工具链时，可以直接从源码安装：

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
cjv init
```

Go 会将可执行文件安装到 `GOBIN`（未设置时为 `$(go env GOPATH)/bin`），请将该目录加入 `PATH`。自定义构建与交叉编译方法见开发指南中的[从源码构建](https://cjv.zxilly.com/book/dev-guide/zh-CN/building.html)。

## 配置 PATH 与验证安装

SDK 代理命令（如 `cjc`、`cjpm`）存放在 `<CJV_HOME>/bin` 目录中，默认路径为 `~/.cjv/bin`。运行 `cjv init` 会自动将该目录写入 `PATH`；如果跳过了 `init` 而直接执行首次工具链安装，cjv 也会尝试配置 `PATH`。

Windows 会更新当前用户的注册表 `PATH`，Unix 系统会更新对应的 shell 配置文件。安装完成后，打开新终端并运行以下命令验证：

```bash
cjv --version
```

如果你需要自行管理 `PATH`，可以给 `cjv init` 传入 `--no-modify-path`（PowerShell 脚本传入 `-NoModifyPath`），或者设置环境变量 `CJV_NO_PATH_SETUP=1`。接下来请阅读[快速上手](../basic-usage.md)安装并管理你的第一个工具链。
