# 工具链文件

`cangjie-sdk.toml` 是放在项目里的工具链声明文件。有了它，一个项目就固定使用某个工具链，并可一并声明该项目需要的交叉编译目标和组件。任何人在这个目录（或其子目录）下运行 `cjc`、`cjpm` 等命令，cjv 都会自动切换到声明的工具链，不必手动 `cjv default` 或设置环境变量。

它在工具链解析链中的位置参见[目标与覆盖](concepts/targets-overrides.md)。环境变量 `CJV_TOOLCHAIN` 与目录覆盖的优先级更高，默认工具链的优先级更低。

```toml
[toolchain]
channel = "lts"                  # 必填
components = ["stdx", "docs"]    # 可选
targets = ["ohos", "android"]    # 可选
```

## 文件位置与查找

cjv 从当前工作目录开始，沿目录树逐级向上查找名为 `cangjie-sdk.toml` 的文件，使用遇到的第一个文件并就此停止，不会合并多个层级的文件。因此子目录里的工具链文件会覆盖父目录里的。

举例，目录结构如下：

```text
~/work/
  cangjie-sdk.toml        # channel = "lts"
  project/
    cangjie-sdk.toml      # channel = "sts"
    src/
```

在 `~/work/project/src/` 下运行命令，cjv 向上找到的第一个文件是 `~/work/project/cangjie-sdk.toml`，于是使用 `sts`，`~/work/` 里那个 `lts` 文件被遮蔽。查找会一直向上走到文件系统根目录为止。

> 同一层级的优先级：如果某一层级同时存在目录覆盖（`cjv override set`）和 `cangjie-sdk.toml`，该层级的目录覆盖优先。但更靠近当前目录的工具链文件，仍然胜过更上层的目录覆盖。

## 字段参考

所有字段都位于 `[toolchain]` 表下。表名必须正好是 `toolchain`。

### `channel`

| 项目     | 值                                                  |
| -------- | --------------------------------------------------- |
| 类型     | string                                              |
| 是否必填 | 是                                                  |
| 默认值   | 无                                                  |

工具链名称，也就是平时传给 `cjv install` 的那个标识符。它可以是通道名（`lts`、`sts`、`nightly`），也可以是带版本的精确名称（如 `lts-1.0.5`、`nightly-1.1.0-alpha.20260306010001`）。通道与版本的写法详见[通道](concepts/channels.md)与[工具链](concepts/toolchains.md)。

```toml
[toolchain]
channel = "lts"
```

```toml
[toolchain]
channel = "lts-1.0.5"
```

`channel` 不能为空。一个被找到但 `channel` 为空的工具链文件（空文件、`channel = ""`、或只写了无法识别的键）会被视为配置不完整并直接报错，cjv 不会跳过它去继续解析下一级。详见下文[空文件与空 channel](#空文件与空-channel)。

### `components`

| 项目     | 值          |
| -------- | ----------- |
| 类型     | string[]    |
| 是否必填 | 否          |
| 默认值   | `[]`（空）  |

声明该项目需要随工具链一起就绪的[组件](concepts/components.md)，例如扩展库 `stdx`、离线文档 `docs`、`stdx-docs`。

```toml
[toolchain]
channel = "lts"
components = ["stdx", "docs"]
```

满足自动安装条件（见下文[与 `auto_install` 的关系](#与-auto_install-的关系)）时，cjv 会在代理执行前自动补齐这里列出但尚未安装的组件。条件不满足时，缺失的组件会让命令以“组件未安装”错误终止，提示你手动运行 `cjv component add`。

组件名会经过校验，未知组件名会报错。可用组件清单见[组件](concepts/components.md)。

### `targets`

| 项目     | 值          |
| -------- | ----------- |
| 类型     | string[]    |
| 是否必填 | 否          |
| 默认值   | `[]`（空）  |

声明该项目需要的[交叉编译目标](cross-compilation.md)，每一项只写目标后缀，例如 `ohos`、`android`、`ohos-arm32`。

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

需要遵守以下规则：

- 只写后缀，不要写完整平台 key。`ohos` 正确，`linux-x64-ohos` 这种完整 SDK 目标元组会被拒绝并报错。
- 大小写与下划线会被规范化：`OHOS`、`ohos_arm32` 会被分别归一为 `ohos`、`ohos-arm32`。
- 单个字符串里支持逗号分隔，等价于写成多项：`targets = ["ohos,android"]` 与 `targets = ["ohos", "android"]` 等价。
- 不允许空目标：`targets = ["ohos", ""]` 或 `targets = [","]` 会报错。
- 重复项会自动去重。

与 `components` 一样，满足自动安装条件时，cjv 会在代理执行前自动安装这里声明但缺失的目标 SDK，否则会以“工具链未安装”错误提示你手动运行 `cjv install <toolchain> --target <suffix>`。

目标 SDK 是宿主工具链的附加安装项，不改变当前活跃工具链，`cjc`、`cjpm` 仍用宿主 SDK 运行。完整说明见[交叉编译](cross-compilation.md)。

## 未识别的键

无法识别的键不会让解析失败，只会以 `warn` 级别日志提示（可通过 `CJV_LOG` 控制日志级别，见[环境变量](environment-variables.md)），其余可识别字段照常生效。常见诱因是拼写错误：

```toml
[toolchian]        # 表名拼错，应为 [toolchain]
channal = "lts"    # 键名拼错，应为 channel
```

上例中没有任何可识别的字段被读到，于是 `channel` 实际为空，这又会触发[空 channel 报错](#空文件与空-channel)。当工具链文件看起来配置了却不生效时，先检查是否有这类 warn 日志。

> 注意：键名拼写错误只警告不报错，但 TOML 语法错误会报错。例如缺少右括号的 `[toolchain` 会让整次解析失败并终止命令。

## 空文件与空 channel

只要 `cangjie-sdk.toml` 被找到，cjv 就认定你打算在此声明工具链。如果该文件存在、但解析后 `channel` 为空，cjv 会报错而不是悄悄回退。以下几种情况都属于空 `channel`：

- 完全空的文件；
- 写了 `channel = ""`；
- 只写了无法识别的键（如上一节的拼写错误），导致没有有效的 `channel`。

这几种情况都会得到类似下面的错误，并指出具体文件路径：

```text
…/cangjie-sdk.toml: toolchain.channel is empty; please specify a channel (e.g. lts, sts, nightly)
```

这样设计是为了避免一类难以察觉的问题：以为切到了某个工具链，实际上悄悄用了默认工具链。如果你确实想让某个目录回退到上一级或默认工具链，请删除该文件，而不是把它清空。

## 与 `auto_install` 的关系

`channel` 决定用哪个工具链，`targets` 与 `components` 决定附带就绪哪些目标和组件。后两者是否会被自动补齐，取决于用户设置中的 `auto_install`：

- `auto_install` 开启（默认，对应 `cjv set auto-install true`）时，代理执行（直接调用 `cjc`、`cjpm` 等）会在运行前自动安装工具链文件里声明、但本机尚缺的目标 SDK 与组件。
- `auto_install` 关闭（`cjv set auto-install false`）时，cjv 不会自动安装，缺失的目标或组件会让命令以相应的“未安装”错误终止，并提示你手动安装。

`auto_install` 的含义与设置方法见[代理](concepts/proxies.md)与[配置](configuration.md)。

`targets` 与 `components` 只在工具链由 `cangjie-sdk.toml` 解析得到时才生效。如果当前活跃工具链来自更高优先级的来源，例如 `CJV_TOOLCHAIN` 环境变量，或 `cjv run`/`cjv exec` 的 `+toolchain` 显式指定，那么工具链文件里的 `targets` 与 `components` 不会被应用（此时连这个文件都未必会被读取）。它们是项目工具链声明的一部分，随 `channel` 一同来自同一个文件。

## 完整示例

一个为 OpenHarmony 交叉编译、需要扩展库和离线文档的项目：

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos"]
```

配合开启自动安装，团队成员首次在该目录运行 `cjpm build` 时，cjv 会自动安装 `sts` 工具链、`ohos` 目标 SDK，以及 `stdx`、`docs` 组件，无需任何额外步骤：

```bash
cjv set auto-install true
cd my-ohos-project
cjpm build        # 缺失的工具链 / 目标 / 组件会被自动补齐
```

## 相关章节

- [目标与覆盖](concepts/targets-overrides.md)：工具链文件在整条解析链中的位置
- [工具链](concepts/toolchains.md) 与[通道](concepts/channels.md)：`channel` 可填的取值
- [组件](concepts/components.md)：`components` 可填的取值
- [交叉编译](cross-compilation.md)：`targets` 的语义与目标 SDK 模型
- [配置](configuration.md) 与[代理](concepts/proxies.md)：`auto_install` 的设置与行为
- [环境变量](environment-variables.md)：`CJV_TOOLCHAIN`、`CJV_LOG` 等
