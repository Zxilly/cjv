# 快速上手

## 检查安装

如果已经完成安装向导，打开新终端后检查当前工具链：

```bash
cjv show
cjc --version
```

如果找不到 `cjc`，检查 `<CJV_HOME>/bin` 是否在 `PATH` 中，默认路径为 `~/.cjv/bin`。命令路径的配置方式见[安装说明](installation/index.md)。

## 选择默认工具链

如果安装时跳过了 SDK，先查询可用通道，再设置默认工具链。下面以 LTS 为例：

```bash
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

`default` 会安装缺失的工具链。所选平台没有 LTS SDK 时，可查询 STS 或 nightly，并将命令中的通道替换为可用通道。也可以运行 `cjv init`，由向导选择可用通道。

默认工具链用于没有项目声明或覆盖设置的目录。运行 `cjv show active` 可以查看选择及来源。

## 为项目选择版本

在项目根目录创建 `cangjie-sdk.toml`：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

这里的版本是示例，请替换为项目需要且分发源提供的版本。在该目录运行 `cjpm build` 时，cjv 使用文件声明的工具链。默认开启的自动安装会补齐缺失的 SDK 和组件；已有工具链不会因每次运行命令而自动升级。

只想在本机切换某个目录时，可使用目录覆盖：

```bash
cjv override set sts
cjv show active
cjv override unset
```

目录覆盖写入用户设置，项目文件可以提交到版本控制。两者的优先级见[项目工具链](toolchain-file.md)。

## 临时使用其他版本

```bash
cjv run --install sts cjc --version
```

`run` 明确指定工具链，`--install` 允许安装缺失的版本。要运行自己的编译产物，用 `cjv exec ./my_binary` 配置该进程的运行时库路径。

## 查看工具链状态

```bash
cjv show active
cjv which cjc
cjv toolchain list
```

这三条命令分别显示当前工具链及来源、编译器路径和已安装工具链列表。

## 更新与卸载

```bash
cjv check
cjv update
```

`check` 只查询 SDK 和 cjv 的可用更新。`update` 更新已跟踪的通道，固定版本和自定义工具链保留原样。全量更新是否检查或更新 cjv 本身，由 [auto_self_update](configuration.md) 设置控制。

手动更新 cjv 本身：

```bash
cjv self update
```

移除不再需要的工具链：

```bash
cjv uninstall sts
```

卸载工具链会同时移除它的组件和文档。更多选项见[命令参考](command-reference.md)。
