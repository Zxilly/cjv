# 交叉编译

宿主工具链运行在主机平台（如 `linux-x64`）上，默认编译该平台的程序。交叉 SDK 运行在同一主机上，提供面向其他目标平台（如 `ohos` 或 `android`）的编译工具和目标库。

`cjv` 将交叉 SDK 作为宿主[工具链](concepts/toolchains.md)的附加安装进行管理：交叉 SDK 的发行版本必须与所依附的宿主工具链完全一致，可用目标列表由分发清单决定。

## 安装和查询

你可以在安装宿主工具链时通过 `--target`（`-t`）参数一并安装交叉目标，也可以稍后通过 `cjv target add` 为已安装的宿主工具链补装目标：

```bash
cjv install sts --target ohos
cjv target list --toolchain sts
cjv target add android --toolchain sts
cjv target list --toolchain sts --installed
```

- **`cjv target add`** — 为宿主工具链当前已安装的版本补齐目标 SDK，不会升级宿主本身。
- **`cjv target list`** — 查询分发清单中为该宿主发行版发布的所有可用目标，并标出哪些已安装。
- **`cjv target list --installed`** — 只读取本地安装状态，支持完全离线执行。

指定目标参数（或在 [`cangjie-sdk.toml`](toolchain-file.md#组件与目标-sdkcomponents-与-targets) 中配置 `targets`）时，请只写目标后缀，例如 `ohos`、`android` 或 `ohos-arm32`，不要写 `linux-x64-ohos` 这样的完整宿主-目标平台名。安装命令支持重复传入 `--target` 或传入逗号分隔列表（如 `--target ohos,android`）。

## 使用目标 SDK

默认情况下，`<CJV_HOME>/bin` 中的 `cjc`、`cjpm` 代理命令仍选择宿主 SDK 环境。需要调用交叉 SDK 中的编译工具和目标库时，请在当前 shell 中通过 `cjv envsetup` 加载目标环境：

```bash
eval "$(cjv envsetup +sts --target=ohos)"
cjc --version
```

在 Windows PowerShell 中执行：

```powershell
cjv envsetup +sts --target=ohos | Invoke-Expression
```

请注意，安装目标 SDK 或在 `cangjie-sdk.toml` 中声明 `targets` 只负责在磁盘上准备好交叉 SDK 文件，不会自动把普通的 `cjpm build` 转换为目标平台构建。后续编译参数、系统库（sysroot）和链接器配置由目标 SDK 及项目构建脚本决定；终端会话环境的切换与恢复详见[运行命令与配置环境](runtime-environment.md#交叉-sdk-环境)。

## 组件、更新和移除

每个交叉 SDK 独立管理自己的组件，宿主工具链的组件不会自动充当目标平台的组件。安装好目标 SDK 后，给 `cjv component` 命令加上 `--target <suffix>` 即可管理该交叉目标的组件：

```bash
cjv component add stdx --toolchain sts --target ohos
cjv component list --toolchain sts --target ohos
cjv target remove ohos --toolchain sts
```

运行 `cjv update` 更新跟踪通道（`lts`、`sts`、`nightly`）时，`cjv` 会先一并准备好新的宿主 SDK、已跟踪目标和已选组件，全部就绪后再统一原子发布。默认情况下，如果新版本缺少所需的目标或组件制品，`cjv` 会阻止更新并保留原有可用安装（`nightly` 通道会自动在已发布历史中查找满足全部目标和组件的兼容版本）。如果希望允许跳过并移除未发布的可选目标或组件，请传入 `--force`；具体规则见[命令参考](command-reference.md#安装与更新策略)。

运行 `cjv target remove` 会移除所选交叉 SDK 及其目标组件，同时保留宿主工具链。交叉 SDK 依附于宿主工具链，不能直接设为全局默认或当前活跃工具链。
