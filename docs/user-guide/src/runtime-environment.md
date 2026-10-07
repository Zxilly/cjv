# 运行命令与配置环境

安装后，`<CJV_HOME>/bin` 中的 `cjc`、`cjpm` 等命令入口会通过 cjv 调用所选 SDK。cjv 按[项目工具链](toolchain-file.md)的规则选择版本、配置环境并转发参数和退出码。

## SDK 工具与自己的程序

```bash
cjc --version
cjv run --install sts cjc --version
cjv exec ./my_binary arg1 arg2
cjv exec +nightly ./my_binary
```

`run` 明确指定工具链，先查 SDK 工具，再查该环境的 PATH；只有加 `--install` 才会安装缺失版本。`exec` 按当前选择或 `+name` 准备 SDK 环境，适合运行需要仓颉动态库的编译产物。这两个命令只改变子进程环境，并透传标准流和退出码。

如果命令名以 `+` 开头，用 `cjv exec -- +command` 避免它被当作工具链选择器。

## 配置当前 shell

`envsetup` 输出环境脚本，需要在当前 shell 中执行：

```bash
# Bash / Zsh
eval "$(cjv envsetup)"
```

```fish
cjv envsetup | source
```

```powershell
cjv envsetup | Invoke-Expression
```

可以用 `cjv envsetup +nightly` 指定版本，或用 `--shell=bash|fish|powershell|cmd` 指定输出格式。省略 `--shell` 时自动检测，检测失败时提示并回退到 POSIX 格式。`cjv --json envsetup` 输出结构化环境数据。

环境包含 `CANGJIE_HOME`、SDK 工具路径和运行时库路径。Linux 使用 `LD_LIBRARY_PATH`，macOS 使用 `DYLD_LIBRARY_PATH`，Windows 使用 `PATH`。已安装的 stdx 会提供 `CANGJIE_STDX_PATH_DYNAMIC` 和 `CANGJIE_STDX_PATH_STATIC`。

执行 `envsetup` 的输出后，当前会话的 PATH 会指向具体 SDK。需要重新按项目选择工具链时，可打开新终端；也可以再次执行 `envsetup` 切换当前会话。

## 交叉 SDK 环境

```bash
cjv install sts --target ohos
eval "$(cjv envsetup +sts --target=ohos)"
```

`--target` 使用已安装交叉 SDK 的路径与库环境，目标必须属于所选宿主的相同发行版。它不会自动安装该目标，也不提供模拟器或跨平台程序执行能力。SDK 的编译参数和目标设备运行条件见对应仓颉 SDK 文档；cjv 的安装方式见[交叉编译](cross-compilation.md)。
