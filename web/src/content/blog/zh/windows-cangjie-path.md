安装完成后，PowerShell 提示“无法将 cjc 识别为 cmdlet、函数、脚本文件或可运行程序的名称”，通常意味着终端还没有找到编译器入口。先判断命令在哪一步找不到，再决定是否需要安装 SDK。

本文适用于通过 cjv 管理 SDK 的 Windows PowerShell 环境。手动解压的 SDK 请同时参考官方安装说明；不要混用两套环境配置而不检查命令来源。

## 先打开一个新的终端

cjv 初始化会尝试更新当前用户的 PATH，但已经运行的 PowerShell 或编辑器进程可能仍保留旧环境。关闭原终端，重新打开 PowerShell；如果使用 VS Code 集成终端，必要时重新启动编辑器。

```powershell
Get-Command cjv,cjc,cjpm -All -ErrorAction SilentlyContinue |
  Select-Object Name,Source
```

如果这时已经能找到命令，继续运行 `cjv show active` 和 `cjc --version`。前者说明版本选择来源，后者确认编译器能够启动。

## 确认 cjv 代理入口是否存在

默认的代理目录是 `%USERPROFILE%\.cjv\bin`。如果设置过 `CJV_HOME`，应检查该目录下的 `bin`，而不是默认路径。

```powershell
$cjvRoot = if ($env:CJV_HOME) { $env:CJV_HOME } else {
  Join-Path $env:USERPROFILE '.cjv'
}
$cjvBin = Join-Path $cjvRoot 'bin'
Test-Path (Join-Path $cjvBin 'cjv.exe')
Test-Path (Join-Path $cjvBin 'cjc.exe')
$env:Path -split ';'
```

如果文件存在，但 PATH 中没有这个目录，可以先在当前会话临时加入它，验证是否确实是路径问题：

```powershell
$env:Path = "$cjvBin;$env:Path"
cjv --version
```

这条赋值只影响当前终端。要重新执行初始化并配置后续终端的 PATH，运行 `cjv init`，按向导完成配置，再打开新终端。使用了 `--no-modify-path` 或 `CJV_NO_PATH_SETUP=1` 时，需要自行维护 PATH。

如果 `cjv.exe` 也不存在，回到[安装指南](https://cjv.zxilly.com/book/user-guide/zh-CN/installation/)选择脚本或预编译文件安装方式。不要把不存在的目录反复加入 PATH。

## 区分找不到命令与没有可用 SDK

当 `cjv --version` 正常，而 `cjc --version` 返回 cjv 的工具链错误时，命令入口已经工作，下一步是检查 SDK 的选择和安装状态：

```powershell
cjv show active
cjv toolchain list
cjv doctor
```

如果没有默认工具链，先查询当前平台可用的版本，再选择需要的通道。下面以有 LTS 可用的主机为例：

```powershell
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

`cjv default lts` 在需要时会安装工具链。如果当前平台未提供 LTS，请选择分发源实际提供的通道。如果项目已经有 `cangjie-sdk.toml`，优先检查文件声明；更改全局默认版本不会覆盖项目配置。自动安装关闭时，可显式安装项目要求的版本。

## 排查旧 SDK 抢占命令

PATH 中残留的手动安装目录可能让终端绕过 cjv。比较操作系统找到的命令入口与 cjv 最终选中的编译器：

```powershell
Get-Command cjc -All | Select-Object Name,Source
cjv which cjc
cjv show active
```

使用 cjv 时，`Get-Command` 应先找到 cjv 的代理入口，`cjv which cjc` 则指向选中 SDK 内的实际编译器。这两个路径不同是正常的。如果先找到旧 SDK，调整 PATH 顺序，并检查终端启动脚本是否重新添加了旧目录。

验收时，在实际项目目录运行 `cjc --version` 和 `cjpm build`。如果编译器已经能启动但项目构建失败，应根据构建错误继续排查；运行产物缺少动态库则参考[运行命令与配置环境](https://cjv.zxilly.com/book/user-guide/zh-CN/runtime-environment.html)。
