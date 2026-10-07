# 安装 cjv

cjv 是单文件程序，不需要预装仓颉 SDK。

## 安装脚本

Linux / macOS：

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://cjv.zxilly.dev/install.ps1 | iex
```

脚本检测平台、下载并校验 cjv，再运行 `cjv init`。交互终端会显示向导，非交互输入使用默认选项。自动化安装可以显式跳过确认，或先只安装 cjv：

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- -y --default-toolchain none
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Yes -DefaultToolchain none
```

Unix 脚本将 `--mirror` 以外的参数传给 `cjv init`。PowerShell 脚本提供对应的 `-Yes`、`-DefaultToolchain`、`-NoModifyPath` 等参数。

### GitCode 镜像

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror
```

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Mirror
```

镜像版使用 GitCode 的默认版本清单和 cjv 自更新源。内部制品库的配置见[企业部署](../enterprise/index.md)。

## 手动下载

从 [GitHub Releases](https://github.com/Zxilly/cjv/releases) 下载归档，解压后把 `cjv` 或 `cjv.exe` 放入 `PATH`，再运行 `cjv init`。

| 平台 | 归档 |
| --- | --- |
| Linux x86_64 | `cjv_linux_amd64.tar.gz` |
| Linux ARM64 | `cjv_linux_arm64.tar.gz` |
| macOS Apple Silicon | `cjv_darwin_arm64.tar.gz` |
| macOS Intel | `cjv_darwin_amd64.tar.gz` |
| Windows x86_64 | `cjv_windows_amd64.zip` |

[GitCode Releases](https://gitcode.com/Zxilly/cjv/releases) 提供前缀为 `cjv-mirror_` 的对应归档。下载页提供的 `cjv-init` 程序在启动时直接进入安装向导。

## 从源码安装

安装满足仓库 `go.mod` 要求的 Go 后执行：

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
cjv init
```

Go 将程序放入 `GOBIN`，未配置时为 `$(go env GOPATH)/bin`；该目录需要加入 `PATH`。

## PATH 与验证

SDK 代理命令位于 `<CJV_HOME>/bin`，默认是 `~/.cjv/bin`。`init` 会配置 PATH；跳过初始化而直接安装 SDK 时，首次工具链安装也会尝试配置。Windows 写入用户 PATH，Unix 修改 shell 配置。打开新终端后检查：

```bash
cjv --version
```

自行管理 PATH 时，给 `init` 传 `--no-modify-path`，PowerShell 脚本用 `-NoModifyPath`。`CJV_NO_PATH_SETUP=1` 也可跳过自动配置。继续阅读[快速上手](../basic-usage.md)安装工具链。
