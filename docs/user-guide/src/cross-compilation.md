# 交叉编译

交叉 SDK 在当前宿主上提供面向其他平台的编译工具。cjv 把它作为宿主工具链的附加安装管理，版本必须与宿主一致，可用目标由分发清单决定。

## 安装和查询

```bash
cjv install sts --target ohos
cjv target list --toolchain sts
cjv target add android --toolchain sts
cjv target list --toolchain sts --installed
```

`target add` 为宿主已安装的版本补齐目标，不升级宿主。`target list --installed` 只读本地状态，可以离线执行；普通列表会查询该发行版的可用目标。

目标参数只写后缀，如 `ohos`、`android`、`ohos-arm32`，不要写 `linux-x64-ohos`。安装时可重复 `--target` 或传入逗号分隔列表。

## 使用目标 SDK

默认的 `cjc`、`cjpm` 代理仍使用宿主 SDK。需要调用交叉 SDK 中的工具时，在当前 shell 中加载它的环境：

```bash
eval "$(cjv envsetup +sts --target=ohos)"
cjc --version
```

PowerShell 使用：

```powershell
cjv envsetup +sts --target=ohos | Invoke-Expression
```

后续编译参数、系统库和链接器配置由目标 SDK 及项目决定。安装目标或在项目文件中声明 `targets` 只负责准备 SDK，不会自动把普通 `cjpm build` 转为目标平台构建。

## 组件、更新和移除

```bash
cjv component add stdx --toolchain sts --target ohos
cjv component list --toolchain sts --target ohos
cjv target remove ohos --toolchain sts
```

`component --target` 管理交叉 SDK 自己的组件，要求目标已安装。宿主组件不会自动充当目标组件。

更新跟踪通道时，cjv 一起准备宿主、已跟踪目标和组件，成功后统一发布。默认情况下，缺少所需制品会阻止更新；nightly 可以选择清单中的兼容版本。`--force` 允许跳过并移除未发布的可选项，具体规则见[命令参考](command-reference.md)。

`target remove` 移除所选交叉 SDK 及其组件，保留宿主。交叉 SDK 不能直接设为默认或活跃工具链。
