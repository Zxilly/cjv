# 代理

你很少会直接键入 `cjv run` 来跑仓颉的 SDK 工具。大多数时候你会像使用一个普通安装的 SDK 那样直接运行 `cjc`、`cjpm`、`cjfmt`，由 cjv 在背后把这次调用转发到正确的工具链。这种机制称为代理执行(proxy)。

代理是 cjv 实现多工具链无缝切换的基础。你切换默认工具链、设置目录覆盖，或在项目里放一个 [工具链文件](../toolchain-file.md)，下一次运行 `cjc` 就会自动落到对应的工具链上，无需改 `PATH`，也无需重新激活。

## 代理符号链接

cjv 安装到 `<CJV_HOME>/bin/`，该目录在首次安装时会被加入 `PATH`(可用 `CJV_NO_PATH_SETUP=1` 跳过这一步)。在这个 bin 目录里，cjv 为每个受支持的 SDK 工具创建一个指向 `cjv` 自身的符号链接(在 Windows 上若无法创建符号链接，则回退到 directory junction 等等效形式)。安装工具链时这些链接会被一并创建。

目前被代理的工具有：

- `cjc`、`cjc-frontend`：编译器
- `cjpm`：包管理器
- `cjfmt`：格式化工具
- `cjlint`：静态检查
- `cjdb`：调试器
- `cjcov`：覆盖率工具
- `cjtrace-recover`、`chir-dis`、`hle`
- `LSPServer`、`LSPMacroServer`：语言服务

当你在终端里输入 `cjc`，shell 在 `PATH` 中找到的其实是 `<CJV_HOME>/bin/cjc`(Windows 上为 `cjc.exe`)这个链接。它指向 `cjv`，因此真正被执行的是 cjv 二进制本身。cjv 通过自己的 `argv[0]`(被调用时的名字)识别出这次是以 `cjc` 的身份被调用的，于是进入代理模式，而不是解析子命令。`cjpm`、`cjfmt` 等其余工具同理。`bin/` 目录里只有 cjv 一个真实可执行文件，其余都是同名链接。

## 工具解析

进入代理模式后，cjv 按以下步骤决定执行哪个二进制：

1. 确定工具名。从 `argv[0]` 取基名，在 Windows 上去掉 `.exe` 后缀，例如 `cjc`、`cjpm`。
2. 解析活跃工具链。使用与 `cjv run`、`cjv exec`、`cjv envsetup` 完全一致的优先级顺序(详见 [目标与覆盖](targets-overrides.md))，从高到低：
   1. `+toolchain` 选择器(见下文)
   2. `CJV_TOOLCHAIN` 环境变量
   3. 目录覆盖(`cjv override set` 设置，见 [目标与覆盖](targets-overrides.md))
   4. 工具链文件(当前目录或父目录中的 `cangjie-sdk.toml`，见 [工具链文件](../toolchain-file.md))
   5. 默认工具链(`cjv default` 设置)
3. 定位工具二进制。在解析出的工具链目录下按固定布局拼出工具路径，例如 `cjc` 位于 `bin/cjc`，`cjpm` 位于 `tools/bin/cjpm`。
4. 注入运行时环境。代理执行会自动配置好该工具链的运行时环境，包括库搜索路径；若该工具链装有 `stdx` [组件](components.md)，还会注入 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`(见 [运行时环境](../runtime-environment.md))。
5. 替换执行。cjv 把控制权交给真正的工具二进制，并原样传递剩余参数、标准输入输出、退出码与信号。在调用方看来就像直接运行了 SDK 自带的 `cjc`。

下面两条命令是等价的：

```bash
# 直接调用(经代理)
cjc --version

# 显式指定工具链运行
cjv run lts cjc --version   # 假设当前解析到的活跃工具链是 lts
```

要查看某个工具最终会落到哪个二进制，用 `cjv which`：

```bash
cjv which cjc
# 打印活跃工具链中 cjc 的真实路径
```

### `+toolchain` 选择器

代理模式支持在参数最前面用 `+` 临时指定工具链，优先级高于其余所有解析方式，只对这一次调用生效：

```bash
# 用 nightly 工具链编译，无论当前默认/覆盖/工具链文件是什么
cjc +nightly main.cj

# 用 sts 跑一次构建
cjpm +sts build
```

`+` 后面的工具链名不能为空，否则报错。该语法与 `cjv exec`、`cjv envsetup` 中的 `+toolchain` 一致。

## `auto_install`：自动补齐缺失项

代理执行时，解析到的工具链(或其声明的目标、组件)可能尚未安装。此时的行为由 `auto_install` 设置决定。

`auto_install = true` 是默认值。cjv 在转发调用之前先把缺失的部分装好，然后照常执行。克隆一个带 `cangjie-sdk.toml` 的项目后，直接运行 `cjpm build` 就会触发首次安装，无需手动 `cjv install`。

`auto_install = false` 时，遇到未安装的工具链、目标或组件，cjv 直接报错退出，不做任何下载。

自动安装会按需覆盖三类缺失项：

1. 活跃工具链本体未安装时自动安装。
2. 工具链文件中 `targets` 声明的交叉编译目标 SDK 缺失时自动补齐(见 [交叉编译](../cross-compilation.md))。
3. 工具链文件中 `components` 声明的组件(如 `stdx`、`docs`)缺失时自动安装(见 [组件](components.md))。

例如，某项目的工具链文件如下：

```toml
[toolchain]
channel = "nightly"
targets = ["ohos"]
components = ["stdx", "docs"]
```

在开启 `auto_install` 的机器上首次运行任意被代理的工具：

```bash
cjpm build
```

cjv 会依次确认 `nightly` 工具链、`ohos` 目标 SDK、`stdx` 与 `docs` 组件是否就绪，补齐所有缺失项后再执行 `cjpm build`。自动安装的进度信息打印到标准错误，不会污染工具自身的标准输出。

### 切换 `auto_install`

```bash
# 关闭与开启(写入 settings.toml)
cjv set auto-install false
cjv set auto-install true
```

该设置存储在 `<CJV_HOME>/settings.toml` 的 `auto_install` 字段，默认值为 `true`。系统级 fallback 设置文件也可提供该字段，详见 [配置](../configuration.md)。

### 不会被自动安装的情形

通过 `cjv toolchain link` 链接的 custom 工具链没有对应的可下载发布资产，cjv 不会、也无法对其执行自动安装；若解析到一个未链接的 custom 名字，直接报错。

`cjv exec` 的交叉编译目标 SDK 必须先通过 `cjv install <toolchain> --target <suffix>` 安装。代理路径只补齐工具链文件中声明的 `targets`，不会凭空为一次性命令安装目标 SDK。

自动安装中任何一步下载或安装失败时，cjv 不会继续转发调用，而是以工具链或组件未安装错误退出，并在标准错误上给出失败原因。

## 递归保护

代理工具最终执行的是真实 SDK，而某些工具内部又可能再调用 `cjc` 等被代理的命令。为避免在配置异常时陷入无限自我调用，cjv 会限制代理的嵌套层数，超过上限即以递归限制错误中止。正常使用中你不会触及这个上限。
