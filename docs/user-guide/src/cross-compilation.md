# 交叉编译

仓颉支持交叉编译：在宿主机器上为另一个平台（如 OpenHarmony、Android）生成可执行文件。除了宿主工具链外，这还需要对应平台的 target SDK（交叉编译 SDK）。

本章介绍如何安装、声明和使用 target SDK。`targets` 与目录覆盖在工具链解析中的位置，参见[目标与覆盖](concepts/targets-overrides.md)。

## target SDK 是附加安装项

target SDK 不是一条独立的工具链，而是挂在某条宿主工具链上的附加安装项。安装 target SDK 不会改变当前活跃工具链，也不会改变 `cjv default`。直接调用 `cjc`、`cjpm` 等工具时（[代理模式](concepts/proxies.md)），用的仍然是宿主 SDK，target SDK 只在你显式请求交叉编译环境时才会被使用。

target SDK 的版本锁定到宿主工具链已解析出的版本。如果该版本没有对应的 target 资产，安装会失败，而不会装上一个版本错配的 SDK。`cjv install sts -t ohos` 给你的是 STS 宿主 SDK，加上与之配套的 OHOS 交叉 SDK，宿主开发体验完全不变。

## 安装 target SDK

用 `cjv install` 的 `-t` / `--target` 标志在安装宿主工具链时附带交叉编译目标：

```bash
# 安装宿主 STS SDK，并额外安装当前宿主对应的 OHOS 交叉 SDK
cjv install sts -t ohos
```

一次安装多个目标有两种等价写法，可以混用：

```bash
# 重复标志
cjv install sts -t ohos -t android

# 逗号分隔
cjv install sts --target ohos,android

# 两者混用也可以
cjv install sts -t ohos,android -t ohos-arm32
```

`--target` 只接受目标后缀，例如 `ohos`、`android`、`ohos-arm32`，不要填写完整的平台 key（如 `linux-x64-ohos`）。平台前缀由 cjv 根据宿主自动补全。

target SDK 可以和[组件](concepts/components.md)在同一条命令里一起安装：

```bash
# 宿主 STS + OHOS 交叉 SDK + stdx 组件
cjv install sts -t ohos -c stdx
```

> target SDK 是附加的：你随时可以对一条已安装的工具链再跑一次 `cjv install <tc> -t <新后缀>` 来补装新的目标，已装好的部分不受影响。

## 在工具链文件中声明 targets

项目可以把交叉编译目标写进 `cangjie-sdk.toml` 的 `[toolchain]` 表，让协作者无需记忆安装命令。`targets` 与命令行同样只填后缀：

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

`targets` 是附加语义：它在宿主工具链之上声明需要哪些 target SDK，不会改变 `channel` 解析出的活跃工具链。

当设置中启用了 `auto_install` 时，[代理执行](concepts/proxies.md)会在调用 SDK 工具前自动补齐缺失的 target SDK；未启用时，你需要手动用上文的 `cjv install … -t …` 安装。`targets` 字段的完整语义见[工具链文件](toolchain-file.md)与[目标与覆盖](concepts/targets-overrides.md)。

## 独立 SDK 模型与 `cjv envsetup --target`

每个 target SDK 都是自包含的：它有自己的 `CANGJIE_HOME`、自己的 `bin` 目录和运行时库路径。要进入某个 target SDK 的交叉编译环境，给 [`cjv envsetup`](runtime-environment.md) 传 `--target=SUFFIX`：

```bash
# 输出 OHOS 交叉编译环境（独立 SDK 模型）
eval "$(cjv envsetup --target=ohos)"

# 其他 shell
cjv envsetup --target=ohos | source             # Fish
cjv envsetup --target=ohos | Invoke-Expression   # PowerShell
```

不带 `--target` 时输出的环境指向宿主工具链。带上 `--target` 后，输出的环境整体重新指向 target SDK 目录：`CANGJIE_HOME` 指向 target SDK 自己的目录，而非宿主工具链目录；`PATH` 与库搜索路径全部取自该 target SDK 目录。逻辑上使用的仍是同一条宿主工具链（如 `lts-1.0.5`），只是底层根目录换成了交叉 SDK。

`--target` 同样遵循与代理模式一致的工具链解析优先级，并支持 `+toolchain` 语法指定宿主工具链：

```bash
# 为 +nightly 宿主工具链输出 OHOS 交叉环境
eval "$(cjv envsetup +nightly --target=ohos)"
```

> 注意：`cjv envsetup --target` 不会自动安装 target SDK。对应 target 必须已经通过 `cjv install <toolchain> --target <suffix>` 安装，否则命令会报错。

配置好环境后，就可以直接调用交叉编译工具链了：

```bash
eval "$(cjv envsetup --target=ohos)"
cjc --version          # 此处的 cjc 来自 OHOS target SDK
cjpm build             # 产物面向 OHOS 平台
```

环境变量注入、不同 shell 的写法，以及一次性执行（`cjv exec`）与配置当前会话（`cjv envsetup`）之间的取舍，详见[运行时环境](runtime-environment.md)。

## 卸载

target SDK 随宿主工具链一同清理。卸载宿主工具链时，挂在它上面的 target SDK 会一并移除：

```bash
cjv toolchain uninstall sts
```
