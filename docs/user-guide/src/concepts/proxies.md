# 代理

你很少会直接键入 `cjv run` 来跑仓颉的 SDK 工具。大多数时候你会像使用一个普通安装的 SDK 那样直接运行 `cjc`、`cjpm`、`cjfmt`，由 cjv 在背后把这次调用转发到正确的工具链。这种机制称为代理执行(proxy)。

代理是 cjv 实现多工具链无缝切换的基础。你切换默认工具链、设置目录覆盖，或在项目里放一个 [工具链文件](../toolchain-file.md)，下一次运行 `cjc` 就会自动落到对应的工具链上，无需改 `PATH`，也无需重新激活。

## 支持的工具

`<CJV_HOME>/bin/` 加入 `PATH` 后，可以直接调用以下 SDK 工具：

- `cjc`、`cjc-frontend`：编译器
- `cjpm`：包管理器
- `cjfmt`：格式化工具
- `cjlint`：静态检查
- `cjdb`：调试器
- `cjcov`：覆盖率工具
- `cjprof`：性能分析工具
- `cjtrace-recover`、`chir-dis`、`hle`
- `LSPServer`、`LSPMacroServer`：语言服务

首次安装默认会配置 `PATH`；若使用 `CJV_NO_PATH_SETUP=1` 跳过了这一步，需要手动加入该目录。

## 工具链选择

直接调用 SDK 工具时，cjv 按以下顺序选择工具链：

1. `+toolchain` 选择器(见下文)
2. `CJV_TOOLCHAIN` 环境变量
3. 目录覆盖(`cjv override set`)
4. 当前目录或父目录中的 `cangjie-sdk.toml`
5. 默认工具链(`cjv default`)

cjv 会配置所选工具链的运行环境，并原样传递命令参数、标准输入输出和退出码。完整优先级规则见[目标与覆盖](targets-overrides.md)，环境配置见[运行时环境](../runtime-environment.md)。

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

该设置保存在 `~/.cjv/settings.toml` 的 `auto_install` 字段，默认值为 `true`，详见[配置](../configuration.md)。

### 不会被自动安装的情形

通过 `cjv toolchain link` 链接的 custom 工具链没有对应的可下载发布资产，cjv 不会、也无法对其执行自动安装；若解析到一个未链接的 custom 名字，直接报错。

`cjv exec` 的交叉编译目标 SDK 必须先通过 `cjv install <toolchain> --target <suffix>` 安装。代理路径只补齐工具链文件中声明的 `targets`，不会凭空为一次性命令安装目标 SDK。

自动安装中任何一步下载或安装失败时，cjv 不会继续转发调用，而是以工具链或组件未安装错误退出，并在标准错误上给出失败原因。
