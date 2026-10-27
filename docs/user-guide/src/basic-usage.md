# 快速上手

完成[安装 cjv](installation/index.md)后，可以检查工具链、设置默认版本、配置项目版本，以及更新和卸载工具链。

## 检查安装状态

如果已经完成了安装向导，请打开一个新终端（使更新后的 `PATH` 生效），然后检查当前选中的工具链并运行编译器：

```bash
cjv show
cjc --version
```

`cjv show` 会列出当前默认主机平台、本机已安装的所有工具链以及当前激活的工具链。如果终端提示找不到 `cjc`，请检查 `<CJV_HOME>/bin`（默认路径为 `~/.cjv/bin`）是否已经在 `PATH` 中；命令路径的配置方式见[安装说明](installation/index.md#path-配置与安装验证)。

## 选择默认工具链

如果在安装 `cjv` 时跳过了 SDK，可以先查询当前平台的可用通道，再将其设为全局默认工具链。下面以 `lts` 通道为例：

```bash
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

如果指定的官方工具链尚未安装，`cjv default` 会自动先下载并安装它，再保存默认选择。如果当前平台没有发布 LTS SDK，可以改查 `sts` 或 `nightly` 通道并替换为可用通道；你也可以重新运行 `cjv init`，由安装向导帮你选择可用通道。

全局默认工具链用于所有没有项目声明（`cangjie-sdk.toml`）且没有设置目录覆盖的目录。随时运行 `cjv show active` 即可查看当前生效的工具链及其选择来源。关于通道与版本号的完整语法，请参阅[工具链与版本](concepts/toolchains.md)。

## 为项目选择版本

### 使用 `cangjie-sdk.toml`

在团队协作项目中，可以在项目根目录创建 `cangjie-sdk.toml` 并提交到版本控制，从而让所有成员和 CI 自动使用相同的工具链与组件：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

示例中的 `lts-1.0.5` 请替换为项目实际需要且分发源提供的版本（或者直接运行 `cjv pin lts` 自动解析并写入完整版本名）。

此后在该目录中运行 `cjpm build` 或 `cjc` 时，`cjv` 会使用文件声明的工具链。默认开启的自动安装功能会在首次调用时补齐缺失的 SDK 和组件；已安装的工具链不会因每次运行代理命令而自动升级。

### 设置本机目录覆盖

如果只想在自己的机器上临时切换某个目录使用的工具链，而不修改项目仓库中的配置文件，可以使用目录覆盖：

```bash
cjv override set sts
cjv show active
cjv override unset
```

目录覆盖保存在你个人的 `~/.cjv/settings.toml` 设置文件中，而 `cangjie-sdk.toml` 则可以随代码提交到版本控制。两者的优先级关系见[项目工具链](toolchain-file.md)。

## 临时使用其他版本运行命令

使用 `cjv run` 临时选择工具链，不修改默认设置或目录配置：

```bash
cjv run --install sts cjc --version
```

`cjv run` 会明确使用参数指定的工具链运行该子进程；加上 `--install` 允许在工具链缺失时自动安装。如果你需要运行自己编译出的仓颉可执行程序，并让它自动找到 SDK 或 `stdx` 的运行时动态库，请使用 `cjv exec ./my_binary`。详见[运行命令与配置环境](runtime-environment.md)。

## 查看工具链状态

查看本机工具链状态：

```bash
cjv show active
cjv which cjc
cjv toolchain list
```

- `cjv show active` — 显示当前激活的工具链名称及其选择来源（如项目文件、目录覆盖或全局默认）。
- `cjv which cjc` — 显示当前激活 SDK 中 `cjc` 编译器的实际文件系统路径。
- `cjv toolchain list`（或 `cjv show installed`）— 列出本机已安装的所有宿主工具链及其附加的交叉编译目标。

## 更新与卸载

### 更新工具链与 `cjv` 本体

检查可用更新并更新已安装的跟踪通道：

```bash
cjv check
cjv update
```

`cjv check` 只查询已安装 SDK 和 `cjv` 本体的可用更新，不执行安装。`cjv update` 会更新所有已安装的跟踪通道（`lts`、`sts`、`nightly`），而固定版本（如 `lts-1.0.5`）和自定义工具链保持原样不变。

执行全量 `cjv update` 时是否同时检查或更新 `cjv` 本体，由 [`auto_self_update`](configuration.md#自动更新) 设置控制（默认为 `check`）。你也可以随时手动更新 `cjv` 本体：

```bash
cjv self update
```

### 卸载工具链

要移除不再需要的工具链并释放磁盘空间：

```bash
cjv uninstall sts
```

卸载宿主工具链会同时移除它附属的组件、交叉编译目标和离线文档。更多命令参数与选项请参阅[命令参考](command-reference.md)。
