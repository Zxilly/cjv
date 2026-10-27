# 自定义 SDK：目录、归档与 URL

`cjv toolchain link` 可以从已解压的本机 SDK 目录、本地 `.zip` 或 `.tar.gz` 归档，以及 HTTP(S) URL 注册自定义工具链：

```bash
cjv toolchain link local-sdk /path/to/sdk
cjv toolchain link archive-sdk ./cangjie-sdk.zip
cjv toolchain link remote-sdk https://example.com/cangjie-sdk.tar.gz
```

这三条命令使用各自独立的名称，可以分别执行。注册自定义工具链不会修改当前的全局默认工具链；如果需要将其设为默认，请运行 `cjv default <name>`，或者在项目的 [`cangjie-sdk.toml`](toolchain-file.md) 中配置 `channel = "<name>"`。

指定的名称必须符合[自定义工具链命名规则](concepts/toolchains.md#通道与名称)，不能使用 `lts`、`sts`、`nightly` 等官方名称。

## 目录与归档的区别

根据传入的是已解压目录还是归档/URL，`cjv` 对文件的管理方式有所不同：

| 来源 | 安装方式 | 卸载行为（`cjv uninstall`） |
| --- | --- | --- |
| 已解压目录 | 建立轻量目录引用；要求存在 `bin/cjc`（Windows 为 `cjc.exe`） | 只删除 `cjv` 中的链接记录，保留源 SDK 目录不变 |
| 本地归档或 URL | 解包到 `<CJV_HOME>/toolchains/` 下由 `cjv` 管理的真实目录 | 删除受管的安装副本及附属组件，保留本地源归档文件 |

这一区别也直接决定了[组件管理行为](concepts/components.md#自定义-sdk-的组件管理)：

- **目录链接**的内容由源 SDK 目录的所有者自行维护，`cjv` 的组件命令（`cjv component`）不能修改它。
- **归档或 URL 安装**位于 `cjv` 受管目录中，因此既可以自动安装随包附带的 `stdx`，也可以通过 `cjv component link stdx` 挂载本地 `stdx` 目录。

## 归档选项

当安装来源为本地 `.zip` / `.tar.gz` 归档或 HTTP(S) URL 时，`cjv toolchain link` 支持以下附加选项：

| 参数 | 作用 |
| --- | --- |
| `--sha256 <hex>` | 解包前校验归档内容，值为预期的 64 位十六进制 SHA-256 摘要 |
| `--force` | 替换已存在的同名自定义工具链 |
| `--no-stdx` | 解包外层打包归档时跳过随包附带的 `stdx` |

这三个选项仅适用于本地归档或 URL，与已解压目录参数一起使用会报错。如果未提供 `--sha256`，`cjv` 不会比对特定内容哈希，但仍会验证归档格式与 SDK 目录布局。

## 支持的归档

`cjv toolchain link` 原生支持以下两种归档结构：

1. **裸 SDK 归档**（`.tar.gz` 或 `.zip`）：直接包含一个完整的 `cangjie/` SDK 顶层目录（内含 `bin/`、`lib/`、`tools/`、`runtime/` 等内容）。
2. **`cangjie-build` CI 外层打包 ZIP**：外层 ZIP 内部包含一个 SDK 归档和一个可选的 `stdx` 归档：

```text
cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>
cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip>  # 可选
```

其中内层 `stdx` 归档包含平台目录下的 `dynamic/` 与 `static/`。只要外层包中存在随包 `stdx` 且未指定 `--no-stdx`，`cjv` 就会自动一并安装；如果 `stdx` 安装失败，已成功安装的 SDK 本体仍会保留。

请注意，归档安装要求 SDK 的目标操作系统与当前机器匹配。如果需要跨平台交叉编译，请安装可在当前宿主上运行的[交叉编译目标 SDK](cross-compilation.md)，而不是直接安装其他操作系统的普通宿主 SDK。

安装完成后，可以使用常规命令查看组件或卸载该自定义工具链：

```bash
cjv component list --toolchain archive-sdk
cjv toolchain uninstall archive-sdk
```
