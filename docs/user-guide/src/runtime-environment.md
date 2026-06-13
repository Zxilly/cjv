# 运行时环境

仓颉编译出来的二进制文件并不是完全自包含的。它们动态链接到 SDK 自带的运行时库（例如 `libcangjie-runtime`），还可能依赖 SDK 提供的其它共享库。直接运行这些产物时，操作系统需要在库搜索路径里找到这些 `.so`/`.dylib`/`.dll`，否则会因为找不到动态库而启动失败。

cjv 的[代理](concepts/proxies.md)在调用 `cjc`、`cjpm` 等 SDK 工具时会自动注入这些路径，但你自己编译出来的产物并不经过代理，你是直接 `./my_binary` 运行它的。这时需要先把运行时环境准备好。cjv 提供两种方式。`cjv exec` 在正确的运行时环境里执行一条命令，运行结束即恢复，不会污染当前 shell。`cjv envsetup` 把环境变量配置脚本输出到 shell，持久配置当前会话，之后可以直接运行编译产物。

两者都使用与代理模式一致的[工具链解析优先级](concepts/targets-overrides.md)，并支持 `+toolchain` 语法显式指定工具链。

## 运行时环境包含什么

无论用哪种方式，cjv 注入的内容都来自当前工具链对应的 SDK 目录。`CANGJIE_HOME` 指向该 SDK 的根目录。SDK 的 `bin`、`tools/bin` 等目录被前置到 `PATH`。运行时库目录被前置到平台对应的库搜索变量：Linux 上是 `LD_LIBRARY_PATH`，macOS 上是 `DYLD_LIBRARY_PATH`，Windows 上则通过 `PATH`（Windows 没有独立的库搜索变量）。如果当前工具链装了 `stdx` [组件](concepts/components.md)，还会注入 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`。

## `cjv exec`：一次性执行

`cjv exec` 在准备好的运行时环境里运行指定命令，运行结束即恢复，不会改动你当前的 shell：

```bash
cjv exec ./my_binary arg1 arg2
```

子进程的退出码会被原样透传，因此 `cjv exec` 可以用在脚本和 CI 流水线里。标准输入、输出、错误流也会原样转发。

### 指定工具链

在命令前加 `+toolchain` 即可临时切换到指定工具链，而不改变默认或当前激活的工具链：

```bash
# 用 nightly 工具链的运行时环境执行
cjv exec +nightly ./my_binary
```

不带 `+toolchain` 时，`cjv exec` 按标准[解析优先级](concepts/targets-overrides.md)选择工具链（`CJV_TOOLCHAIN` → 目录覆盖 → `cangjie-sdk.toml` → 默认工具链）。

### 运行以 `+` 开头的命令

`+toolchain` 选择器只会消费第一个参数。如果你要执行的命令本身就以 `+` 开头，用 `--` 终止符把它和选择器隔开：

```bash
# 此处 +foo 是要执行的命令名，不是工具链
cjv exec -- +foo arg1
```

## `cjv envsetup`：配置当前 shell

`cjv envsetup` 不直接执行命令，而是把配置运行时环境所需的 shell 命令打印到标准输出。你需要把它的输出喂给当前 shell 求值，让环境变量在当前会话中生效。配置一次之后，本会话内就可以反复直接运行编译产物，无需每次都套一层 `cjv exec`。

不同 shell 的求值方式不同：

```bash
# Bash / Zsh
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

`cjv envsetup` 同样支持 `+toolchain`：

```bash
eval "$(cjv envsetup +nightly)"
```

### Shell 自动检测与 `--shell`

`cjv envsetup` 通过检查父进程来自动判断当前 shell 类型，并据此选择正确的输出语法。它识别 `bash`、`zsh`、`sh` 等 POSIX shell，以及 `fish`、`powershell`/`pwsh`、`cmd`。

如果自动检测失败（例如在某些嵌套或非交互环境中），cjv 会回退到 POSIX 语法并在标准错误打印一条提示。此时，或当你想为另一个 shell 生成脚本时，用 `--shell` 显式指定，取值为 `bash`、`fish`、`powershell`、`cmd`：

```bash
cjv envsetup --shell=fish | source
```

```powershell
cjv envsetup --shell=powershell | Invoke-Expression
```

## 交叉编译：`--target`

`cjv envsetup` 的 `--target=SUFFIX` 用于输出已安装目标 SDK 的运行时环境，而不是宿主 SDK 的。当[交叉编译](cross-compilation.md)产物需要运行（例如在目标设备或模拟器上）时，这很有用。

cjv 对目标 SDK 采用独立 SDK 模型：`CANGJIE_HOME` 指向目标 SDK 目录，`PATH` 与库搜索路径也全部取自该目录，与宿主 SDK 互不干扰。

```bash
# 输出已安装 ohos 目标 SDK 的运行时环境
eval "$(cjv envsetup --target=ohos)"
```

目标 SDK 必须先随宿主工具链一并安装，`--target` 才能找到它：

```bash
cjv install sts --target ohos
```

目标后缀（如 `ohos`、`android`）的含义详见[交叉编译](cross-compilation.md)与[目标与覆盖](concepts/targets-overrides.md)。

## 相关章节

- [代理](concepts/proxies.md)：代理模式如何为 SDK 工具自动注入运行时环境。
- [组件](concepts/components.md)：`stdx` 组件及其注入的 `CANGJIE_STDX_PATH_*` 变量。
- [环境变量](environment-variables.md)：cjv 涉及的全部环境变量。
- [交叉编译](cross-compilation.md)：安装与使用交叉编译目标 SDK。
