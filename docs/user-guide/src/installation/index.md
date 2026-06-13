# 安装 cjv

本章介绍如何安装 cjv 自身。cjv 是一个单文件可执行程序，安装它不需要预先准备任何仓颉 SDK。工具链由 cjv 在安装后按需下载（见[快速上手](../basic-usage.md)）。

共有三种安装方式：

- [一键安装脚本](#一键安装脚本)（`install.sh` / `install.ps1`，推荐）：自动下载对应平台的二进制并完成初始化。
- [下载预编译二进制](#下载预编译二进制)：从 GitHub Releases 手动取回二进制并放入 `PATH`。
- [从源码编译](#从源码编译)：用 `go install` 自行构建。

如果网络环境无法访问 GitHub，可使用文中各方式的镜像源变体，从 [GitCode](https://gitcode.com/Zxilly/cjv) 下载。

## 一键安装脚本

脚本会探测当前平台、下载匹配的二进制、校验 SHA-256，然后运行 `cjv init` 完成初始化（包括把 cjv 的 `bin` 目录加入 `PATH`）。

### Linux / macOS

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh
```

脚本探测不到终端时（例如纯管道执行）会以非交互方式静默安装。若检测到控制终端，`cjv init` 会进入交互式向导，询问安装目录、是否修改 `PATH` 等。把额外参数转发给 `cjv init` 即可跳过交互，例如用 `-y` 接受默认选项：

```bash
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- -y
```

`-s --` 之后的所有参数（除 `--mirror` 外）都会原样传给 `cjv init`，因此 `--no-modify-path`、`--default-toolchain none` 等 `init` 的选项均可在此使用。

### Windows（PowerShell）

```powershell
irm https://cjv.zxilly.dev/install.ps1 | iex
```

`install.ps1` 接受 `-Yes`（跳过确认）、`-DefaultToolchain <name>`（默认安装的工具链，`none` 表示不装）、`-NoModifyPath`（不修改 `PATH`）等参数。通过管道执行时需用脚本块形式传参：

```powershell
& ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Yes
```

> Windows ARM64 没有原生构建，脚本会自动改装 amd64 版本，在系统的 x64 模拟层下运行。

### 镜像源（GitCode）

GitHub 访问不畅时，使用镜像源从 GitCode 下载 `cjv-mirror` 归档：

```bash
# Linux / macOS：加 --mirror 标志
curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror
```

```powershell
# Windows：设置 CJV_MIRROR 环境变量
$env:CJV_MIRROR = "1"; irm https://cjv.zxilly.dev/install.ps1 | iex
```

镜像与默认源装出的是同一个 cjv，区别仅在于下载来源以及后续 `cjv self update` 的更新源。镜像变体可与上面的其它参数自由组合，例如 `curl -sSf https://cjv.zxilly.dev/install.sh | sh -s -- --mirror -y`。

## 下载预编译二进制

前往 [Releases](https://github.com/Zxilly/cjv/releases) 页面，下载与平台匹配的归档并解压，把得到的 `cjv`（Windows 上为 `cjv.exe`）放入 `PATH` 中的任意目录。

归档命名规则为 `cjv_<goos>_<goarch>`，扩展名在 Windows 上是 `.zip`，其余平台是 `.tar.gz`：

| 平台          | 归档文件                  |
| ------------- | ------------------------- |
| Linux x86_64  | `cjv_linux_amd64.tar.gz`  |
| Linux ARM64   | `cjv_linux_arm64.tar.gz`  |
| macOS Apple Silicon | `cjv_darwin_arm64.tar.gz` |
| macOS Intel   | `cjv_darwin_amd64.tar.gz` |
| Windows x86_64 | `cjv_windows_amd64.zip`  |

镜像源用户可从 [GitCode Releases](https://gitcode.com/Zxilly/cjv/releases) 下载对应的 `cjv-mirror_<goos>_<goarch>` 归档。

手动安装的二进制不会自动初始化。首次安装工具链时，cjv 会补上 `PATH` 配置，详见下文[首次安装时的 PATH 配置](#首次安装时的-path-配置)。

## 从源码编译

需要本机已安装 [Go](https://go.dev/)（版本要求见仓库 `go.mod`）：

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

二进制会被装到 `$(go env GOBIN)`（默认 `$(go env GOPATH)/bin`），请确保该目录在 `PATH` 中。国内网络可加上代理：

```bash
GOPROXY=https://goproxy.cn,direct go install github.com/Zxilly/cjv/cmd/cjv@latest
```

与手动下载二进制一样，源码安装不会自动初始化，`PATH` 会在首次安装工具链时配置。

## 首次安装时的 PATH 配置

cjv 在自己的 `bin` 目录（默认 `~/.cjv/bin`）下放置二进制以及指向各工具链的代理符号链接（见[代理](../concepts/proxies.md)）。要让 `cjc`、`cjpm` 等命令在终端中直接可用，这个目录必须在 `PATH` 中。

通过安装脚本安装时，`cjv init` 会立即完成 `PATH` 配置。通过 `go install` 或手动下载二进制安装时，cjv 在你首次安装工具链（如 `cjv install lts`）时才把 `bin` 目录加入 `PATH`。

具体写入方式因平台而异。Windows 上写入用户级注册表 `PATH`；Linux 和 macOS 上把 `PATH` 追加到 shell 配置文件（如 `~/.profile`、`~/.bashrc`、`~/.zshenv` 以及 fish 的 config）。无论哪种方式，配置都要在新开的终端里才生效；当前会话需重启或手动 `source` 才能识别新 `PATH`。

### 跳过自动配置

把环境变量 `CJV_NO_PATH_SETUP` 设为 `1`，即可跳过这一步的 `PATH` 修改，适用于 CI 等不希望改动用户环境的场景：

```bash
CJV_NO_PATH_SETUP=1 cjv install lts
```

使用安装脚本时，等效做法是让底层的 `cjv init` 不要改 `PATH`：Linux / macOS 传 `--no-modify-path`，Windows 用 `-NoModifyPath`。此时需手动把 `bin` 目录加入 `PATH`。`cjv init` 也会打印出可供 `source` 的 `env` 脚本路径（Linux / macOS 为 `~/.cjv/env`，Windows 为 `~/.cjv/env.ps1` 与 `~/.cjv/env.bat`）。

`CJV_NO_PATH_SETUP` 等环境变量的完整说明见[环境变量](../environment-variables.md)。

## 验证安装

```bash
cjv --version
```

能打印出版本号就说明安装成功。接下来可以安装第一个工具链，继续阅读[快速上手](../basic-usage.md)。

> cjv 自身的更新用 `cjv self update`，卸载用 `cjv self uninstall`（会一并移除所有已安装的工具链）。详见[命令参考](../command-reference.md)。
