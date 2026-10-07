# 快速上手

安装 LTS 工具链并设为默认：

```bash
cjv install lts
cjv default lts
cjc --version
```

如果找不到 `cjc`，先打开新终端，使安装时写入的 `PATH` 生效。仍无法找到时，检查 `<CJV_HOME>/bin` 是否在 `PATH` 中，默认路径为 `~/.cjv/bin`。

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

## 临时运行与检查

```bash
cjv run --install sts cjc --version
cjv show
cjv which cjc
cjv toolchain list
```

`run` 明确指定工具链，`--install` 允许安装缺失的版本。要运行自己的编译产物，用 `cjv exec ./my_binary` 配置该进程的运行时库路径。

## 更新与卸载

```bash
cjv check
cjv update
cjv self update
cjv uninstall sts
```

`check` 查询 SDK 和 cjv 的可用更新。`update` 更新已跟踪的通道，固定版本和自定义工具链保留原样；cjv 自身的自动更新行为由 [auto_self_update](configuration.md) 设置控制。`uninstall` 同时移除所选工具链的组件和文档。
