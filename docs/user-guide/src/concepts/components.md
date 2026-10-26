# 组件与离线文档

组件是归属于各个具体[工具链](toolchains.md)的可选扩展包，包括 `stdx` 扩展库和离线 HTML 文档。卸载某个工具链时，它关联的组件也会一并清理。本章介绍如何安装和移除组件、如何链接本地 `stdx` 目录，以及如何打开离线文档。

## 组件类型与存放位置

cjv 按工具链名称（`<tc>`）在 `<CJV_HOME>` 下分别存放各组件：

| 组件名称 | 包含内容 | 相对 `CJV_HOME` 的存放路径 |
| --- | --- | --- |
| `stdx` | 动态与静态扩展库 | `stdx/<tc>/{dynamic,static}` |
| `docs` | 仓颉语言、标准库与工具文档 | `docs/<tc>/main/` |
| `stdx-docs` | 扩展库文档 | `docs/<tc>/stdx/` |

安装 `stdx` 后，代理命令、`cjv exec` 和 `cjv envsetup` 会自动向运行环境注入 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`。文档组件（`docs` 与 `stdx-docs`）不会修改运行时环境变量；运行环境详情见[运行命令与配置环境](../runtime-environment.md)。

## 安装、查看与移除组件

### 使用命令行管理组件

你可以在安装工具链时一并指定组件，也可以后续通过 `cjv component` 子命令管理：

```bash
cjv install nightly -c stdx,docs
cjv component add stdx docs --toolchain lts
cjv component list --toolchain lts
cjv component list --installed -q
cjv component remove stdx-docs
```

- 省略 `--toolchain` 时，命令默认作用于当前激活的工具链。
- 传入 `--target <suffix>` 时，命令作用于该宿主工具链下已安装的[交叉编译](../cross-compilation.md)目标 SDK。
- 组件名称可以传入多个独立参数，也可以用逗号分隔（如 `stdx,docs`）。
- `cjv component add` 默认跳过已安装的组件；传入 `--force` 可强制重新安装。

可下载的组件制品取决于当前分发清单中的通道、版本和平台。你也可以在项目的 [cangjie-sdk.toml](../toolchain-file.md) 中声明 `components`，由 cjv 在运行工具时自动补齐。

### 工具链更新时的组件行为

运行 `cjv update` 更新跟踪通道时，cjv 会将宿主 SDK 和已选组件一起升级；固定版本（如 `lts-1.0.5`）的组件保持不变。如果某个工具链的 `stdx` 是通过本地目录链接挂载的，更新工具链后会继续保留该本地链接。

## 链接本地 `stdx` 目录

### 为受管工具链链接本地 `stdx`

如果你在本机自行编译了 `stdx` 或有现成的解压目录，可以使用 `cjv component link` 将其挂载到工具链：

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

源目录必须同时包含 `dynamic/` 和 `static/` 两个子目录。传入 `--force` 可以替换该工具链上已有的 `stdx` 组件。

运行 `cjv component remove stdx` 移除链接时，cjv 只会删除内部管理的链接条目，保留你的本地源目录不变。`docs` 和 `stdx-docs` 组件不支持本地目录链接。

### 为自定义 SDK 管理组件

组件命令只能修改位于 cjv 受管目录中的 SDK。如果是通过 `cjv toolchain link <name> /path/to/sdk` 直接引用的外部 SDK 目录，其组件由外部目录所有者维护，不支持 `cjv component` 修改。

如果希望由 cjv 管理自定义 SDK 的组件，请先从[本地归档或 URL](../install-from-url.md) 安装该 SDK：

```bash
cjv toolchain link my-sdk ./cangjie-sdk.zip
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

自定义工具链没有官方下载源供 `cjv component add` 使用，但归档安装的自定义工具链会自动安装随包附带的 `stdx`，也支持通过 `cjv component link stdx` 挂载本地扩展库。

## 打开离线文档

安装 `docs` 或 `stdx-docs` 组件后，运行 `cjv doc`（或别名 `cjv docs`）即可在默认浏览器中打开本地 HTML 文档：

```bash
cjv doc
cjv doc std
cjv doc dev-guide
cjv doc tools
cjv doc stdx --toolchain nightly
cjv doc --path
```

| 主题参数 | 打开的文档内容 | 依赖的组件 |
| --- | --- | --- |
| *(省略)* | 若已安装 `docs` 则打开主文档首页，否则打开 `stdx` 文档首页 | `docs` 或 `stdx-docs` |
| `std` | 标准库参考文档 | `docs` |
| `dev-guide` 或 `book` | 仓颉开发指南 | `docs` |
| `tools` | SDK 工具指南 | `docs` |
| `stdx` | 扩展库参考文档 | `stdx-docs` |

传入 `--path` 或 `--json` 时，命令只返回本地文件路径而不启动浏览器。如果对应的文档组件尚未安装，cjv 会提示相应的安装命令。完整选项说明见[命令参考](../command-reference.md#离线文档)。
