# 自定义 SDK：目录、归档与 URL

使用 `cjv toolchain link` 可以将本机已解压的 SDK 目录、本地 `.zip` / `.tar.gz` 归档，或 HTTP(S) URL 注册为自定义工具链。本章对比目录链接与归档安装的行为差异，并说明受支持的归档目录结构。

## 注册自定义 SDK

运行 `cjv toolchain link` 并指定自定义名称及本地路径或 URL：

```bash
cjv toolchain link local-sdk /path/to/sdk
cjv toolchain link archive-sdk ./cangjie-sdk.zip
cjv toolchain link remote-sdk https://example.com/cangjie-sdk.tar.gz
```

上面三条命令使用不同的自定义名称，可以独立执行。自定义名称必须符合[工具链与版本](concepts/toolchains.md#通道与版本名称)中的命名规则，不能使用 `lts`、`sts`、`nightly` 等官方名称。

注册自定义工具链不会改变当前的全局默认选择。如果需要将其设为默认，请运行 `cjv default <name>`；如果需要在项目中使用，请将其名称写入 [cangjie-sdk.toml](toolchain-file.md)。

## 目录链接与归档安装的区别

目录链接与归档安装在文件所有权、组件管理支持和卸载行为上存在以下差异：

| 来源类型 | 安装方式 | 组件支持 | `cjv uninstall` 卸载行为 |
| --- | --- | --- | --- |
| 已解压目录 | 建立目录引用；要求包含 `bin/cjc`（Windows 为 `cjc.exe`） | 由源目录所有者维护，`cjv component` 不能修改 | 仅删除 cjv 链接记录，保留源 SDK 目录 |
| 本地归档或 URL | 解包到 `<CJV_HOME>/toolchains/` 下的受管目录 | 自动安装随包 `stdx`，或通过 `cjv component link stdx` 挂载 | 删除受管安装副本及组件，保留本地源归档文件 |

为归档安装的自定义 SDK 挂载本地 `stdx` 的示例见[组件与离线文档](concepts/components.md#为自定义-sdk-管理组件)。

## 归档安装专属选项

以下选项仅在来源为本地归档文件或 HTTP(S) URL 时可用：

| 选项 | 作用 |
| --- | --- |
| `--sha256 <hex>` | 校验归档文件的 64 位十六进制 SHA-256 摘要 |
| `--force` | 替换已存在的同名自定义工具链 |
| `--no-stdx` | 跳过安装外层打包归档中附带的 `stdx` |

将 `--sha256`、`--force` 或 `--no-stdx` 用于已解压目录会直接报错。如果不传 `--sha256`，cjv 不会比对特定哈希值，但仍会校验归档完整性与 SDK 目录布局。

## 支持的归档结构

### 裸 SDK 归档

裸 `.zip` 或 `.tar.gz` 归档直接包含一个完整的 `cangjie/` SDK 顶层目录（内含 `bin/`、`lib/`、`tools/`、`runtime/` 等子目录）。

### `cangjie-build` 外层打包 ZIP

cjv 也支持直接传入 `cangjie-build` CI 生成的外层 `.zip` 归档，其内部结构如下：

```text
cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>
cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip>  # 可选
```

其中内层 `stdx` 归档在平台子目录下包含 `dynamic/` 与 `static/`。只要存在随包 `stdx` 且未传入 `--no-stdx`，cjv 就会自动安装该 `stdx`；如果 `stdx` 解包失败，已成功安装的 SDK 本体仍会保留。

### 宿主平台要求

归档安装要求 SDK 的目标操作系统与当前机器一致。如果需要跨平台交叉编译，请安装可在当前宿主上运行的[交叉编译目标 SDK](cross-compilation.md)，而不是直接安装其他操作系统的普通宿主 SDK。

## 查看与卸载归档 SDK

使用标准组件与工具链命令即可查询或卸载归档安装的自定义工具链：

```bash
cjv component list --toolchain archive-sdk
cjv toolchain uninstall archive-sdk
```

在无网企业环境中使用自定义归档的部署流程见[企业与离线部署](enterprise/index.md#完全离线客户端)。
