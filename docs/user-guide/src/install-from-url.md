# 从 URL 或本地归档安装工具链

`cjv toolchain link <name> <path>` 的 `<path>` 有三种形态：本地目录、本地归档文件（`.zip` / `.tar.gz`），或一个 `http(s)://` URL。传入目录时，cjv 只创建一个指向它的链接；传入归档时——无论是本地文件还是 URL——cjv 会解包它，并把它落地成一个由 cjv 拥有的真实工具链。

## 两种行为：引用与物化

传入本地目录时使用引用模式；传入本地归档或 HTTP(S) URL 时使用物化模式。

| 维度 | 引用模式（本地目录） | 物化模式（本地归档 / URL） |
| --- | --- | --- |
| `<path>` 形态 | 本地目录，如 `/path/to/sdk` | 本地归档 `sdk.zip`，或 `https://...` |
| `toolchains/<name>` 内容 | 引用原目录 | 由 cjv 管理的安装目录 |
| 数据归属 | cjv 不拥有，只是引用 | cjv 拥有 |
| 随包 stdx | 不涉及 | 可选，自动安装（见下文） |
| 卸载行为 | 只删链接，原目录保留 | 删除整个目录（含 stdx） |
| 是否改默认工具链 | 否 | 否 |

本地归档会就地读取，不会被移动或删除。引用本地目录的说明见[工具链](concepts/toolchains.md)与[组件](concepts/components.md)。

```bash
# 物化:从 URL 下载、解包,落地为 cjv 拥有的真实目录
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip

# 物化:从本地归档解包,落地为 cjv 拥有的真实目录(源文件保留)
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip

# 引用(对照):只创建一个指向本地目录的链接
cjv toolchain link mysdk /path/to/local/sdk
```

## 名称必须是自定义名

`<name>` 必须是自定义名，不能与保留的通道名 `lts`、`sts`、`nightly` 冲突，也不能包含路径分隔符、`+` 前缀，或为空、`.`、`..` 等非法名称：

```bash
# 报错：lts 是保留通道名
cjv toolchain link lts https://example.com/sdk.zip
```

## 标志

物化模式支持三个标志，本地归档与 URL 同样适用：

| 标志 | 作用 |
| --- | --- |
| `--sha256 <hex>` | 校验归档的 SHA-256。缺省时只校验归档格式是否合法 |
| `--force` | 当 `toolchains/<name>` 已存在时覆盖重装 |
| `--no-stdx` | 即便归档内含 stdx，也不安装随包 stdx |

这三个标志只在物化模式（本地归档或 URL）下有效。如果你在传入本地目录时带上其中任意一个，cjv 会直接报错，提示该标志不适用于链接本地目录，而不是默默忽略。

```bash
# 校验归档的 SHA-256(本地归档同样适用)
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip \
  --sha256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

# 覆盖已存在的同名工具链
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.1.0.zip --force

# 只装 SDK,跳过随包 stdx
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip --no-stdx
```

未提供 `--sha256` 时，URL 依赖 TLS，本地归档则视为可信文件。需要校验内容完整性时，请始终提供 SHA-256。

## 期望的归档格式

物化模式支持 `cangjie-build` CI 产物和裸 SDK 归档。

### CI 构建产物(嵌套布局)

从 GitHub Actions 下载的构建产物 `cangjie-<target>-<version>` 是一个外层 ZIP，可包含以下归档：

```text
<外层 .zip>
├── cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>            （内层 SDK,必需）
└── cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip> （内层 stdx,可选）
```

- 外层归档为 `.zip`。
- 内层归档在 Linux 上通常为 `.tar.gz`，在 Windows 上通常为 `.zip`。
- 内层 SDK 解开后是单一顶层目录 `cangjie/`，内含 `bin/`、`lib/`、`tools/`、`runtime/` 等。
- 内层 stdx 解开后是单一顶层目录 `<platform>_cjnative/`(如 `linux_x86_64_cjnative`、`windows_x86_64_cjnative`)，内含 `dynamic/` 与 `static/`。

### 裸 SDK 归档

归档也可以直接包含一个 SDK 顶层目录。该目录必须包含完整 SDK 布局；这种格式不包含随包 stdx。

## 随包 stdx 自动安装

当归档内含 `cangjie-stdx-*` 且未指定 `--no-stdx` 时，cjv 会同时安装 stdx 组件。

```bash
# 归档含 stdx → SDK 与 stdx 一并装好
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip

# 验证 stdx 已就位
cjv component list --toolchain mysdk
```

stdx 的管理方式见[组件](concepts/components.md)。

## 仅支持当前系统

物化安装只支持与当前操作系统匹配的 SDK。尝试安装其他系统的 SDK 会报错。

```bash
# 在 Linux 上尝试安装 Windows SDK → 落地前报错,toolchains/<name> 不会被创建
cjv toolchain link winsdk https://example.com/cangjie-windows-x64-1.0.0.zip
```

如果你需要为另一个平台准备 SDK，请在那个平台上执行安装，或使用[交叉编译](cross-compilation.md)的目标 SDK 机制。

## 卸载：cjv 拥有，真删除

物化安装的工具链由 cjv 拥有，卸载时会真正删除落地的目录，包括随包安装的 stdx：

```bash
cjv toolchain uninstall mysdk
```

这会删除 `toolchains/mysdk/`、`stdx/mysdk/` 以及 `docs/mysdk/`(若存在)。引用模式的卸载只删除链接条目，原始目录不受影响。

更多相关内容：工具链解析优先级见 [工具链](concepts/toolchains.md)，组件与 stdx 见 [组件](concepts/components.md)，运行时环境注入见 [运行时环境](runtime-environment.md)，完整命令签名见 [命令参考](command-reference.md)。
