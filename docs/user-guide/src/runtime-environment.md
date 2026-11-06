# 运行命令与配置环境

`<CJV_HOME>/bin` 中的 `cjc`、`cjpm` 等程序是 SDK 工具的代理入口。代理按[项目工具链](toolchain-file.md)规则选择 SDK，准备环境变量和动态库路径，并转发命令行参数和子进程退出码。

## 运行 SDK 工具与自己的程序

除了直接运行 `cjc`、`cjpm` 等代理命令外，`cjv` 还提供了 `cjv run` 和 `cjv exec` 两个子命令，用于在独立的子进程环境中执行命令：

```bash
cjc --version
cjv exec +sts cjc --version
cjv exec ./my_binary arg1 arg2
cjv exec +nightly ./my_binary
```

理解两者的定位可以帮助你快速选择合适的命令：

- **`cjv run [--install] <toolchain> <command> [args...]`** — 保留显式工具链位置参数的兼容入口，可执行任意程序。它与 `exec` 使用相同运行环境和 `auto_install` 补装策略；`--install` 显式开启补装，`--install=false` 关闭补装，显式开启与 `--no-auto-install` 互斥。
- **`cjv exec [+toolchain] <command> [args...]`** — 推荐入口，使用当前项目选择或指定工具链，配置 SDK、运行时及 stdx 库路径，并按 `auto_install` 补齐缺失需求。显式指定工具链时使用该选择；当前项目中的组件和目标声明在使用项目选择时生效。

`run` 和 `exec` 都只改变子进程的运行环境，不会污染当前终端会话，并且会原样转发标准输入、标准输出、标准错误和退出码。

如果待运行的命令名称本身以 `+` 开头，请在命令名前加上 `--`，避免它被当作 `+toolchain` 选择器解析：

```bash
cjv exec -- +command
```

`which` 只读取工具路径（省略工具名时读取 SDK 根目录），不安装缺失项，也不要求补齐无关的项目组件或目标。

## 使用 `envsetup` 配置当前 shell

在与外部构建系统（如 Makefile、CMake 或自定义脚本）配合时，有时需要将 SDK 的工具路径和环境变量直接导入当前交互式终端会话。`cjv envsetup` 会向标准输出打印一段环境配置脚本，在当前 shell 中求值执行即可生效：

```bash
# Bash 或 Zsh
eval "$(cjv envsetup)"
```

```fish
# Fish
cjv envsetup | source
```

```powershell
# Windows PowerShell
cjv envsetup | Invoke-Expression
```

你可以运行 `cjv envsetup +nightly` 显式指定工具链版本，或者传入 `--shell=bash|fish|powershell|cmd` 指定输出脚本的语法格式。省略 `--shell` 时，`cjv` 会自动检测当前 shell；如果检测失败，会在 `stderr` 打印提示并回退到 POSIX shell 格式。此外，运行 `cjv --json envsetup` 可以输出结构化的环境变量 JSON 数据，供编辑器插件或自动化脚本读取（详见[自动化与 JSON 接口](automation.md)）。

`envsetup` 只读取已安装 SDK 并验证项目声明的组件和目标；缺失时给出显式安装命令，不会触发自动补装。它只接受至多一个 `+toolchain` 位置参数，多余参数会报错。

### 设置的环境变量

无论是通过代理命令、`cjv run`、`cjv exec` 还是 `cjv envsetup`，`cjv` 都会为所选 SDK 配置以下环境信息：

- **`CANGJIE_HOME`** — 指向所选 SDK 安装的根目录。
- **`PATH`** — 将 SDK 的编译器与工具目录前置加入 `PATH`。
- **运行时动态库路径** — 按宿主操作系统设置动态库搜索路径：Linux 与鸿蒙使用 `LD_LIBRARY_PATH`，macOS 使用 `DYLD_LIBRARY_PATH`，Windows 使用 `PATH`。
- **`stdx` 扩展库路径** — 如果该工具链已安装 [`stdx` 组件](concepts/components.md)，`cjv` 会自动注入 `CANGJIE_STDX_PATH_DYNAMIC` 和 `CANGJIE_STDX_PATH_STATIC`，分别指向组件的 `dynamic/` 与 `static/` 目录。

### 切换项目与恢复代理选择

请注意，在当前终端中求值执行 `envsetup` 的输出后，当前会话的 `PATH` 会直接指向某个具体 SDK 的工具目录（排在 `<CJV_HOME>/bin` 之前）。此时在该终端中切换到其他项目目录，将不再按 `cangjie-sdk.toml` 自动切换编译器。

如果需要恢复按项目自动选择工具链，打开一个新终端；或者在当前终端再次执行 `envsetup` 切换到目标工具链。

## 交叉 SDK 环境

进行跨平台交叉编译时，可以通过 `cjv envsetup --target` 加载已安装交叉编译 SDK 的工具路径与库环境：

```bash
cjv install sts --target ohos
eval "$(cjv envsetup +sts --target=ohos)"
```

`--target` 会使用所选宿主工具链同一发行版下已安装的交叉 SDK 路径和库环境。它不会自动安装缺失的目标 SDK，也不提供模拟器或跨平台程序直接运行能力。

目标平台所需的编译器参数、系统库与运行条件请参阅对应仓颉 SDK 文档；使用 `cjv` 安装和管理交叉编译目标的方法见[交叉编译](cross-compilation.md)。
