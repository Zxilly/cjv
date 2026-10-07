# 自定义 SDK：目录、归档与 URL

`cjv toolchain link` 接受已有 SDK 目录、本地 `.zip` / `.tar.gz` 归档或 HTTP(S) URL：

```bash
cjv toolchain link local-sdk /path/to/sdk
cjv toolchain link archive-sdk ./cangjie-sdk.zip
cjv toolchain link remote-sdk https://example.com/cangjie-sdk.tar.gz
```

这三条命令使用不同名称，可以分别执行。它们不会修改默认工具链；需要时执行 `cjv default <name>`。

## 目录与归档的区别

| 来源 | 安装方式 | 卸载行为 |
| --- | --- | --- |
| 已解压目录 | 建立引用，要求存在 `bin/cjc`（Windows 为 `cjc.exe`） | 只删除链接，保留源 SDK |
| 本地归档或 URL | 解包到 cjv 管理的真实目录 | 删除安装副本及组件；保留本地源归档 |

目录链接的内容由源 SDK 所有者维护，cjv 的组件命令不能修改它。归档安装可使用随包 stdx，或通过 `component link` 添加本地 stdx。

名称必须是[自定义工具链名](concepts/toolchains.md)，不能使用 `lts`、`sts`、`nightly` 等官方名称。

## 归档选项

| 参数 | 作用 |
| --- | --- |
| `--sha256 <hex>` | 校验归档内容，值为预期的 64 位十六进制 SHA-256 |
| `--force` | 替换已有的同名工具链 |
| `--no-stdx` | 跳过随包 stdx |

这些选项仅适用于本地归档或 URL，与目录参数一起使用会报错。未提供 SHA-256 时不做预期内容哈希校验，仍会验证归档与 SDK 布局。

## 支持的归档

裸 SDK 归档包含一个完整 SDK 顶层目录。`cangjie-build` 的 CI 外层 ZIP 也受支持，其内部包含：

```text
cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>
cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip>  # 可选
```

内层 SDK 包含 `cangjie/` 目录及其 `bin/`、`lib/`、`tools/`、`runtime/` 等内容。stdx 包含平台目录下的 `dynamic/` 与 `static/`。存在随包 stdx 且未指定 `--no-stdx` 时自动安装；stdx 安装失败会保留已成功安装的 SDK。

归档安装要求 SDK 的操作系统与本机匹配。交叉编译使用宿主可运行的[目标 SDK](cross-compilation.md)，不通过安装其他操作系统的普通 SDK 实现。

```bash
cjv component list --toolchain archive-sdk
cjv toolchain uninstall archive-sdk
```
