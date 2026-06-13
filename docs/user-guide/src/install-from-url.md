# 从 URL 安装工具链

`cjv toolchain link <name> <path>` 的 `<path>` 既可以是本地目录,也可以是一个 `http(s)://` URL。传入 URL 时,cjv 会下载归档、解包,并把它落地成一个由 cjv 拥有的真实工具链。这与传入本地路径只创建一个链接的行为不同。

## 一条命令,两种模式

`cjv toolchain link` 根据 `<path>` 是否匹配 `^https?://` 自动分流:

| 维度 | 本地路径模式 | URL 模式 |
| --- | --- | --- |
| `<path>` 形态 | 本地目录,如 `/path/to/sdk` | `https://...` 或 `http://...` |
| `toolchains/<name>` 内容 | 符号链接 / junction,指向你的目录 | 下载、解包后落地的真实目录 |
| 数据归属 | cjv 不拥有,只是引用 | cjv 拥有 |
| 随包 stdx | 不涉及 | 可选,自动安装(见下文) |
| 卸载行为 | 只删链接,原目录保留 | 删除整个目录(含 stdx) |
| 是否改默认工具链 | 否 | 否 |

两种模式共用同一条命令、同一道名称校验,区别在于数据归属与卸载语义。本章只讲 URL 模式;本地链接见 [工具链](concepts/toolchains.md)与 [组件](concepts/components.md)。

```bash
# URL 模式:下载、解包,落地为 cjv 拥有的真实目录
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip

# 本地路径模式(对照):只创建一个指向本地目录的链接
cjv toolchain link mysdk /path/to/local/sdk
```

## 名称必须是自定义名

`<name>` 必须是自定义名,不能与保留的通道名 `lts`、`sts`、`nightly` 冲突,也不能包含路径分隔符、`+` 前缀,或为空、`.`、`..` 等非法名称。这道校验对两种模式都生效,且在下载发生之前执行:

```bash
# 报错:lts 是保留通道名,不会触发任何下载
cjv toolchain link lts https://example.com/sdk.zip
```

## 标志

URL 模式支持三个标志:

| 标志 | 作用 |
| --- | --- |
| `--sha256 <hex>` | 校验外层 zip 的 SHA-256。缺省时只做归档格式嗅探 |
| `--force` | 当 `toolchains/<name>` 已存在时覆盖重装 |
| `--no-stdx` | 即便归档内含 stdx,也不安装随包 stdx |

这三个标志仅对 URL 模式生效。如果你在传入本地路径时带上其中任意一个,cjv 会直接报错,提示该标志仅适用于 URL 安装,而不是默默忽略。

```bash
# 校验外层 zip 的 SHA-256
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip \
  --sha256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

# 覆盖已存在的同名工具链
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.1.0.zip --force

# 只装 SDK,跳过随包 stdx
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip --no-stdx
```

未提供 `--sha256` 时,cjv 依赖 TLS 传输安全,并在下载后嗅探归档魔数(必须是合法的 zip 或 gzip),以拒绝明显损坏或非归档的内容。内层的 SDK 与 stdx 归档没有独立校验,外层完整即认为内层可信。如需端到端的完整性保证,请提供 `--sha256`。

## 期望的归档格式

URL 模式期望指向 `cangjie-build` CI 产出的构建产物,也兼容直接指向一个裸 SDK 归档。

### CI 构建产物(嵌套布局)

从 GitHub Actions 下载的构建产物 `cangjie-<target>-<version>` 是一个外层 ZIP,里面恰好包含两个文件:

```text
<外层 .zip>
├── cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>            （内层 SDK,必需）
└── cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip> （内层 stdx,可选）
```

- 外层包装永远是 `.zip`。这是 GitHub `actions/upload-artifact` 在下载时的打包行为,不是 cangjie-build 自己产出的。
- 内层归档在 Linux 上为 `.tar.gz`,在 Windows 上为 `.zip`。cjv 按内容嗅探格式,同一套解包逻辑同时处理两者。
- 内层 SDK 解开后是单一顶层目录 `cangjie/`,内含 `bin/`、`lib/`、`tools/`、`runtime/` 等。
- 内层 stdx 解开后是单一顶层目录 `<platform>_cjnative/`(如 `linux_x86_64_cjnative`、`windows_x86_64_cjnative`),内含 `dynamic/` 与 `static/`。

cjv 扫描外层 zip 顶层时按前缀匹配:文件名以 `cangjie-sdk-` 开头的内层归档为 SDK(必需),以 `cangjie-stdx-` 开头的为 stdx(可选);其余多余文件(如 README、校验和)被静默忽略。

### 直归档兜底

如果 URL 直接指向一个裸 SDK 归档,即归档解开后顶层恰好是单一目录,而非两个 `cangjie-*` 内层归档,cjv 会把这个目录当作 SDK 根直接落地,不再二次解包。这种情形下没有随包 stdx。

如果外层既没有 `cangjie-sdk-*` 归档,顶层也不是单一目录(例如一堆松散文件),cjv 会报错"未找到 cangjie-sdk-* 归档",不做进一步尝试。

## 随包 stdx 自动安装

当归档内含 `cangjie-stdx-*` 且未指定 `--no-stdx` 时,cjv 会把这个 stdx 作为该工具链的 stdx 组件自动安装,落地到 `<CJV_HOME>/stdx/<name>/` 下的 `dynamic/` 与 `static/`,并写入组件 manifest。之后代理执行时会自动注入 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`,无需手动配置。

```bash
# 归档含 stdx → SDK 与 stdx 一并装好
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip

# 验证 stdx 已就位
cjv component list --toolchain mysdk
```

本地链接的 stdx(`cjv component link stdx`)只创建符号链接,而 URL 安装的 stdx 是 cjv 拥有的真实数据,卸载时会一并删除。组件机制的更多细节见 [组件](concepts/components.md)。

## 仅支持当前系统

URL 安装只支持与当前操作系统匹配的 SDK,不支持跨 OS 安装。cjv 读取 SDK 里 `cjc` 可执行文件的格式(ELF / Mach-O / PE)判断它面向哪个系统;若与当前系统不符(例如在 Linux 上安装 Windows 或 macOS 的 SDK),会在落地之前直接报错。这对嵌套产物和裸归档都生效。

```bash
# 在 Linux 上尝试安装 Windows SDK → 落地前报错,toolchains/<name> 不会被创建
cjv toolchain link winsdk https://example.com/cangjie-windows-x64-1.0.0.zip
```

如果你需要为另一个平台准备 SDK,请在那个平台上执行安装,或使用[交叉编译](cross-compilation.md)的目标 SDK 机制。

## 卸载:cjv 拥有,真删除

URL 安装的工具链由 cjv 拥有,卸载时会真正删除落地的目录,包括随包安装的 stdx:

```bash
cjv toolchain uninstall mysdk
```

这会删除 `toolchains/mysdk/`、`stdx/mysdk/` 以及 `docs/mysdk/`(若存在)。本地链接模式的卸载只删除链接条目,原始目录不受影响。

## 行为要点

- 不改变默认工具链:URL 安装与本地链接一致,不会把新工具链设为默认。需要时用 `cjv default mysdk` 显式设置。
- 下载暂存:外层 zip 下载到 `<CJV_HOME>/downloads/` 暂存,解包与落地经事务完成,成功后清理。任何中途失败都会回滚,不会留下半成品的 `toolchains/<name>`。
- 下载失败保留暂存:若 `--sha256` 不符或下载中断,外层 zip 会被保留以便重试;只有安装成功才清理暂存文件。
- Windows 无符号链接权限:proxy 链接会自动降级为 junction,stdx 内部的符号链接会物化为复制,安装不会中断。

更多相关内容:工具链解析优先级见 [工具链](concepts/toolchains.md),组件与 stdx 见 [组件](concepts/components.md),运行时环境注入见 [运行时环境](runtime-environment.md),完整命令签名见 [命令参考](command-reference.md)。
