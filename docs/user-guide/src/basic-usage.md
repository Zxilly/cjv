# 快速上手

本章介绍如何验证安装状态、设置全局默认工具链、为项目固定 SDK 版本以及执行日常更新。如果你尚未安装 cjv，请先阅读[安装 cjv](installation/index.md)。

## 检查安装状态

完成安装向导后，打开一个新终端并检查当前激活的工具链：

```bash
cjv show
cjc --version
```

如果终端提示找不到 `cjc`，请确认 `<CJV_HOME>/bin`（默认路径为 `~/.cjv/bin`）已经加入 `PATH`。`PATH` 的设置与排查见[安装 cjv](installation/index.md#配置-path-与验证安装)。

## 设置默认工具链

如果安装 cjv 时跳过了 SDK，可以先查询当前平台的可用通道，再将其设为默认工具链。下面以 LTS 通道为例：

```bash
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

如果目标工具链尚未安装，`default` 命令会先下载并安装它，再保存默认设置。如果当前平台没有 LTS 版本的 SDK，可以查询 `sts` 或 `nightly` 并替换为可用通道。你也可以随时重新运行 `cjv init`，由安装向导帮你选择可用通道。

当工作目录没有项目配置文件（`cangjie-sdk.toml`）或本机目录覆盖时，cjv 就会使用全局默认工具链。运行 `cjv show active` 可以查看当前生效的工具链及其选择来源。版本名称语法见[工具链与版本](concepts/toolchains.md)。

## 为项目选择版本

### 创建项目工具链文件

要在团队内统一项目的 SDK 与组件版本，请在项目根目录创建 `cangjie-sdk.toml`：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

请将示例中的 `lts-1.0.5` 替换为项目实际需要且分发源提供的版本；你也可以运行 `cjv pin lts` 自动写入解析后的完整发行版名称。

在该目录运行 `cjpm build` 时，cjv 会自动使用 `cangjie-sdk.toml` 声明的工具链。默认开启的自动安装功能会自动补齐缺失的 SDK 和组件；已安装的工具链不会因每次运行命令而自动升级。

### 设置本机目录覆盖

如果只想在自己的机器上临时切换某个目录使用的工具链，而不修改项目仓库文件，可以使用目录覆盖：

```bash
cjv override set sts
cjv show active
cjv override unset
```

目录覆盖保存在用户设置文件中，不会污染版本控制。命令行参数、环境变量、目录覆盖与项目文件的完整优先级规则见[项目工具链](toolchain-file.md)。

## 临时使用其他版本运行命令

要单次指定某个工具链运行命令，而不改变默认或项目配置，请使用 `cjv run`：

```bash
cjv run --install sts cjc --version
```

`run` 仅对当前子进程使用指定的工具链；加上 `--install` 会在工具链缺失时自动安装。如果需要运行你自己编译出的仓颉程序并加载 SDK 动态库路径，请使用 `cjv exec ./my_binary`。`run`、`exec` 与 `envsetup` 的区别见[运行命令与配置环境](runtime-environment.md)。

## 查看已安装工具链

使用以下命令查看当前选择、工具路径和所有已安装工具链：

```bash
cjv show active
cjv which cjc
cjv toolchain list
```

- `cjv show active`：显示当前激活的工具链名称及生效原因。
- `cjv which cjc`：显示当前工具链中 `cjc` 编译器的实际文件路径。
- `cjv toolchain list`：列出本机已安装的所有宿主工具链及其附加的交叉编译目标。

## 更新与卸载工具链

### 更新 SDK 与 cjv 本体

运行以下命令检查并应用更新：

```bash
cjv check
cjv update
```

`check` 只查询已安装 SDK 和 cjv 本体的可用更新，不执行安装。`update` 会升级已安装的跟踪通道（如 `lts`、`sts`、`nightly`），而固定版本（如 `lts-1.0.5`）和自定义工具链保持不变。

执行不带参数的 `cjv update` 时，是否同时检查或更新 cjv 本体由 [auto_self_update](configuration.md) 设置控制。你也可以随时手动更新 cjv 本体：

```bash
cjv self update
```

### 卸载工具链

要移除不再需要的工具链，请运行 `cjv uninstall`：

```bash
cjv uninstall sts
```

卸载工具链会同时清理它关联的组件、交叉编译目标和离线文档。所有命令的参数说明见[命令参考](command-reference.md)，或者继续阅读[运行命令与配置环境](runtime-environment.md)。
