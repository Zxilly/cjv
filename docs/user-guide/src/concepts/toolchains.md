# 工具链

工具链(toolchain)是 cjv 管理的基本单位，也就是一份完整、可独立运行的仓颉 SDK 安装。一个工具链至少包含编译器 `cjc`、包管理器 `cjpm` 以及配套的运行时库。在它之上还可以挂载 [组件](components.md)(如 `stdx`、`docs`)和 [交叉编译目标](targets-overrides.md)。

cjv 可以同时安装多个工具链，并在它们之间切换。每个工具链都有一个唯一的名称，你在几乎所有命令里都用这个名称来指代它：

```bash
cjv install lts          # 安装名为 lts 的工具链
cjv default sts          # 把默认工具链设为 sts
cjv run nightly cjc -V   # 用 nightly 工具链运行 cjc
cjv uninstall lts        # 卸载 lts
```

所有已安装的工具链都位于 `<CJV_HOME>/toolchains/<名称>/` 下，`CJV_HOME` 默认是 `~/.cjv`。

## 工具链名称的形式

工具链名称可以是以下几种形式之一。前几种由 cjv 直接识别，从官方源下载；最后一种 `custom` 由你显式创建。

| 形式 | 示例 | 说明 |
| ---- | ---- | ---- |
| 通道名 | `lts`、`sts`、`nightly` | 解析为该 [通道](channels.md) 当前的最新版本 |
| 通道名 + 版本 | `lts-1.0.5`、`sts-1.1.0-beta.23` | 该通道下的一个具体版本 |
| 裸版本号 | `1.0.5` | 不带通道前缀的版本号，跨所有通道查找 |
| custom(自定义) | `my-sdk`、`local-build` | 由 `cjv toolchain link` 创建，见下文 |

### 通道名

`lts`、`sts`、`nightly` 是三个通道。单独使用通道名时，cjv 把它解析为该通道当前的最新版本。通道名不区分大小写，`LTS`、`Lts`、`lts` 等价。

```bash
cjv install lts      # 安装最新 LTS
cjv install nightly  # 安装最新 nightly
```

各通道的语义、更新节奏与下载来源详见 [通道](channels.md)。

### 通道名 + 版本

在通道名后加 `-` 和版本号，即可锁定该通道下的一个具体版本。版本号可以带预发布后缀：

```bash
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23
cjv install nightly-1.1.0-alpha.20260306010001
```

这样安装的工具链名称就是你输入的完整字符串，例如 `lts-1.0.5`，后续命令也用它来指代：

```bash
cjv default lts-1.0.5
cjv uninstall sts-1.1.0-beta.23
```

### 裸版本号

如果只给出版本号(以数字开头，不带通道前缀)，cjv 会在所有通道中查找匹配该版本的已安装工具链：

```bash
cjv run 1.0.5 cjc --version
```

用它来引用某个已安装的版本，不必记住它属于哪个通道。

### custom：自定义工具链

凡是不匹配上述任何形式的名称(既不是通道名、不是 `通道-版本`、也不以数字开头)，都被视为 custom(自定义)工具链。这类工具链不来自官方源，而是由你通过 `cjv toolchain link` 显式创建：

```bash
cjv toolchain link my-sdk /path/to/local/sdk
```

自定义工具链有两种来源，区别在于数据归谁所有，详见下文 [自定义工具链](#自定义工具链) 一节。

## 命名规则

无论哪种形式，工具链名称都必须满足以下约束，否则命令会报错：

- 不能为空；
- 不能包含路径分隔符 `/` 或 `\`(防止逃逸到 `toolchains/` 目录之外)；
- 不能是 `.` 或 `..`；
- 不能以 `+` 前缀开头，`+` 是 `cjv exec`、`cjv envsetup` 等命令里的工具链选择语法，直接写名称即可；
- 末尾多余的 `/`、`\` 会被自动去掉。

此外，在 `cjv toolchain link` 中，自定义工具链的名称不能与保留的通道名冲突：`lts`、`sts`、`nightly` 已被官方通道占用，不能用作链接名。

## 自定义工具链

自定义工具链让你把官方源之外的 SDK 纳入 cjv 管理，例如本地编译的 SDK、内部分发的构建，或某个临时下载的归档。它有两种创建方式，区别在于这份数据是否由 cjv 拥有。

### 来源一：链接本地目录(cjv 不拥有数据)

第一种方式是把一个已存在的本地 SDK 目录链接进来。cjv 在 `<CJV_HOME>/toolchains/<名称>/` 下创建一个指向你目录的符号链接(Windows 上回退为 directory junction)，并不复制任何文件：

```bash
cjv toolchain link my-sdk /path/to/local/sdk
```

被链接的目录必须是一个真正的仓颉 SDK。cjv 会校验其中存在 `bin/cjc`，否则拒绝链接。

因为只是一个链接，原始数据仍归你所有。你在源目录里做的任何改动都会立刻通过 cjv 生效；`cjv toolchain uninstall my-sdk`(以及 `cjv uninstall my-sdk`)只删除这个链接，不会动你的原始目录。这种方式适合调试自编译的 SDK，或在多个工具之间共享同一份安装。

### 来源二：从 URL 安装(cjv 拥有数据)

当 `cjv toolchain link` 的第二个参数是一个 `http://` 或 `https://` 链接时，cjv 会下载归档、解压，并把内容物化到 `<CJV_HOME>/toolchains/<名称>/` 下，成为一份由 cjv 完整拥有的安装：

```bash
cjv toolchain link my-sdk https://example.com/cangjie-sdk.tar.gz
```

与本地链接相反，这种工具链的数据由 cjv 管理：`cjv toolchain uninstall my-sdk` 会真正删除这份目录及其组件。

URL 安装支持几个额外参数，它们仅对 URL 来源有效，搭配本地路径使用会被拒绝：

- `--sha256 <hash>`：校验下载归档的 SHA-256；
- `--force`：覆盖同名的已安装工具链；
- `--no-stdx`：跳过自动探测并安装随包的 stdx。

完整的 URL 格式约定、归档布局要求、校验行为与示例，见 [从 URL 安装工具链](../install-from-url.md)。

### 为自定义工具链挂载 stdx

通过链接本地目录创建的自定义工具链没有对应的官方 release 资产，因此 `cjv component add stdx` 对它无效。需要时改用 `cjv component link stdx` 把一个本地 stdx 目录挂上去：

```bash
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

详见 [组件](components.md)。

## 查看与管理工具链

列出所有已安装的工具链，自定义工具链也会出现在列表中：

```bash
cjv toolchain list
# 等价于
cjv show installed
```

查看当前活跃的工具链以及整体状态：

```bash
cjv show
cjv show active
```

设置默认工具链、为目录设置覆盖，以及通过环境变量或 `cangjie-sdk.toml` 选择工具链等机制，决定了在某个上下文里哪个工具链处于活跃状态。这部分的优先级规则详见 [目标与覆盖](targets-overrides.md) 与 [工具链文件](../toolchain-file.md)。
