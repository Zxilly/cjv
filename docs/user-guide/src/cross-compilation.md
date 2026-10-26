# 交叉编译

交叉 SDK 在当前宿主机器上提供面向其他目标平台的编译工具和库文件。cjv 将交叉 SDK 作为宿主[工具链](concepts/toolchains.md)的附加目标进行管理：交叉 SDK 的发行版必须与宿主工具链一致，可用目标列表由分发清单决定。

## 安装与查询目标 SDK

你可以在安装宿主工具链时同时指定交叉目标，也可以稍后通过 `cjv target add` 为已有宿主补装目标：

```bash
cjv install sts --target ohos
cjv target list --toolchain sts
cjv target add android --toolchain sts
cjv target list --toolchain sts --installed
```

- `cjv target add`：为已安装的宿主版本补齐交叉目标 SDK，不会升级宿主本身。
- `cjv target list`：查询分发清单中为该宿主发行版发布的所有可用目标。
- `cjv target list --installed`：仅读取本机安装记录，支持离线执行。

指定目标参数时，请只写目标后缀（如 `ohos`、`android`、`ohos-arm32`），不要写 `linux-x64-ohos` 这样的完整平台名。安装命令支持重复传入 `--target` 或使用逗号分隔列表（如 `--target ohos,android`）。

你也可以在项目的 [cangjie-sdk.toml](toolchain-file.md#声明组件与目标-sdk) 中声明 `targets`，由 cjv 在运行工具时自动补齐。

## 加载目标 SDK 构建环境

默认情况下，`cjc` 和 `cjpm` 代理命令仍使用宿主 SDK 环境。需要调用交叉 SDK 中的编译工具和库时，请在当前终端会话中通过 `cjv envsetup` 加载目标环境：

```bash
eval "$(cjv envsetup +sts --target=ohos)"
cjc --version
```

在 Windows PowerShell 中执行：

```powershell
cjv envsetup +sts --target=ohos | Invoke-Expression
```

安装目标 SDK 或在 `cangjie-sdk.toml` 中声明 `targets` 只负责把交叉 SDK 文件准备就绪，不会自动将普通的 `cjpm build` 转换为交叉编译构建。后续编译所需的目标平台编译参数、系统库路径（sysroot）和链接器配置仍由目标 SDK 及项目构建脚本决定。终端环境的切换与恢复方式见[运行命令与配置环境](runtime-environment.md#加载交叉编译-sdk-环境)。

## 管理目标组件、更新与移除

### 目标 SDK 的独立组件

每个交叉 SDK 独立维护自己的 `stdx` 组件，宿主工具链的 `stdx` 不会自动充当目标平台的 `stdx`。安装好目标 SDK 后，给 `cjv component` 命令加上 `--target <suffix>` 即可管理目标组件：

```bash
cjv component add stdx --toolchain sts --target ohos
cjv component list --toolchain sts --target ohos
```

组件的存放位置和链接方式见[组件与离线文档](concepts/components.md)。

### 附带交叉目标的通道更新

运行 `cjv update` 更新跟踪通道时，cjv 会在暂存区一并准备新的宿主 SDK、已跟踪的交叉目标和已选组件，全部就绪后再统一替换当前安装。

默认情况下，如果新版本缺少所需的目标或组件制品，cjv 会阻止本次更新并保留原有可用安装；对于 `nightly` 通道，cjv 会在已发布的历史版本中查找同时满足所有目标和组件的兼容构建。如果希望在更新时跳过并移除未发布的可选目标或组件，请传入 `--force`，具体规则见[命令参考](command-reference.md#安装与更新策略)。

### 移除交叉目标

要移除某个交叉 SDK 及其目标组件并保留宿主工具链，请运行 `cjv target remove`：

```bash
cjv target remove ohos --toolchain sts
```

交叉 SDK 始终依附于宿主工具链，不能直接设为全局默认工具链或当前活跃工具链。
