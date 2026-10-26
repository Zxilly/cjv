# 运行命令与配置环境

完成[安装 cjv](installation/index.md) 后，`<CJV_HOME>/bin` 中的 `cjc`、`cjpm` 等命令入口会通过 cjv 调用当前选中的 SDK。cjv 会按[项目工具链](toolchain-file.md)规则解析版本，配置运行环境，并转发参数和退出码。本章介绍如何调用 SDK 工具、运行仓颉编译产物，以及在当前终端会话中加载 SDK 环境。

## 执行方式对比

请根据使用场景选择合适的命令入口：

| 执行方式 | 命令示例 | 作用范围 | 适用场景 |
| --- | --- | --- | --- |
| 代理命令入口 | `cjc --version` | 单个子进程 | 日常编译、构建与包管理 |
| `cjv run` | `cjv run --install sts cjc --version` | 单个子进程 | 临时指定某个工具链版本运行一次命令 |
| `cjv exec` | `cjv exec ./my_binary arg1` | 单个子进程 | 运行依赖仓颉 SDK 或 `stdx` 动态库的编译产物 |
| `cjv envsetup` | `eval "$(cjv envsetup)"` | 当前终端会话 | 将 SDK 路径与环境变量导入当前交互式终端 |

## 运行 SDK 工具与编译产物

### 使用代理命令与 `cjv run`

直接运行 `cjc` 或 `cjpm` 时，cjv 按[项目工具链](toolchain-file.md)的选择顺序决定版本。如果需要显式指定工具链运行单条命令，请使用 `cjv run`：

```bash
cjc --version
cjv run --install sts cjc --version
```

`run` 会先在指定工具链的 SDK 工具目录中查找命令，再查找该环境的 `PATH`；只有传入 `--install` 时，才会在工具链缺失时自动下载安装。

### 使用 `cjv exec` 运行编译产物

编译生成的仓颉程序通常需要加载 SDK 或 `stdx` 中的动态链接库。使用 `cjv exec` 可以在当前激活工具链（或 `+name` 指定工具链）的运行环境下启动程序：

```bash
cjv exec ./my_binary arg1 arg2
cjv exec +nightly ./my_binary
```

`run` 和 `exec` 都只修改子进程的环境，并原样透传标准输入、标准输出、标准错误和退出码。如果被调用的命令名本身以 `+` 开头，请在命令名前加上 `--`，防止它被解析为工具链选择器：

```bash
cjv exec -- +command
```

## 配置当前终端会话（`envsetup`）

### 执行 `cjv envsetup` 脚本

`cjv envsetup` 会向标准输出打印一段环境配置脚本。在当前终端中求值执行该输出，即可把 SDK 环境导入当前 shell 会话：

```bash
# Bash 或 Zsh
eval "$(cjv envsetup)"
```

```fish
# Fish
cjv envsetup | source
```

```powershell
# PowerShell
cjv envsetup | Invoke-Expression
```

你可以传入 `+nightly` 选择特定工具链，或传入 `--shell=bash|fish|powershell|cmd` 指定脚本格式。省略 `--shell` 时，cjv 会自动检测当前 shell；如果检测失败，会打印提示并回退到 POSIX 格式。运行 `cjv --json envsetup` 可以获取供自动化脚本使用的结构化环境数据，详见[自动化与 JSON 接口](automation.md)。

### 注入的环境变量

SDK 运行环境会设置 `CANGJIE_HOME`，将 SDK 工具目录加入 `PATH`，并按操作系统配置动态库搜索路径：

- **Linux**：设置 `LD_LIBRARY_PATH`。
- **macOS**：设置 `DYLD_LIBRARY_PATH`。
- **Windows**：将运行时动态库目录加入 `PATH`。
- **已安装 `stdx`**：自动注入 `CANGJIE_STDX_PATH_DYNAMIC` 和 `CANGJIE_STDX_PATH_STATIC`。`stdx` 的安装与链接方式见[组件与离线文档](concepts/components.md)。

### 恢复按项目自动切换

在终端中求值执行 `envsetup` 后，当前会话的 `PATH` 会直接指向某个具体 SDK 目录。只要该目录排在 `<CJV_HOME>/bin` 之前，切换工作目录时 `cangjie-sdk.toml` 的代理选择就不会在该终端生效。

如果需要恢复按项目目录自动切换工具链，请打开一个新终端；或者在当前终端再次执行 `envsetup` 切换到其他 SDK。

## 加载交叉编译 SDK 环境

进行跨平台编译时，可以为已安装的交叉编译目标加载对应的工具与库环境：

```bash
cjv install sts --target ohos
eval "$(cjv envsetup +sts --target=ohos)"
```

`--target` 会使用所选宿主工具链同发行版下已安装的交叉 SDK 路径和库环境。它不会自动安装缺失的目标 SDK，也不提供跨平台模拟器来直接运行目标平台的程序。

目标平台所需的编译器参数、系统库和运行条件请查阅对应仓颉 SDK 文档；在 cjv 中管理交叉编译目标的方法见[交叉编译](cross-compilation.md)。
