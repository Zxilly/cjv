# 项目工具链

每次运行 `cjc`、`cjpm` 等代理命令时，`cjv` 都会自动决定使用哪个[工具链](concepts/toolchains.md)。在团队项目中，将 `cangjie-sdk.toml` 放在项目根目录并提交到版本控制，即可让所有项目成员和 CI 流水线自动使用同一套工具链、组件与交叉编译目标声明：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

示例中的 `lts-1.0.5` 仅作演示；支持的所有通道与版本名称格式见[工具链与版本](concepts/toolchains.md)。

除了手动编写外，你也可以在项目根目录直接运行 `cjv pin lts` 自动写入解析后的具体发行版名称，或运行不带参数的 `cjv pin` 固定当前激活的工具链选择，文件中已有的 `components` 与 `targets` 列表会被完整保留。在 CI 环境中，可以运行 `cjv pin --check` 验证当前目录的项目文件是否已经固定了具体发行版（不修改文件也不访问网络），详见[自动化与 JSON 接口](automation.md#固定项目版本)。

## 选择顺序

有多种方式可以控制和覆盖当前使用的工具链。`cjv` 按以下顺序从高到低选择工具链，并采用第一个指定的来源：

1. 命令行中显式指定的工具链，例如全局 `+name` 简写（如 `cjv +nightly show`）或子命令支持的 `--toolchain` 参数。
2. `CJV_TOOLCHAIN` [环境变量](environment-variables.md)。
3. 从当前工作目录向文件系统根目录逐层向上查找：在每一层目录中，先检查[本机目录覆盖](#本机目录覆盖)，再检查 `cangjie-sdk.toml`，找到第一个匹配项后立即停止。
4. `cjv default` 设置的全局默认工具链。

请特别留意第 3 步向上遍历目录树时的就近原则：目录覆盖和 `cangjie-sdk.toml` 都优先采用离当前工作目录更近的配置。因此在**同一层目录**中，本机目录覆盖优先于该目录的 `cangjie-sdk.toml`；但**子目录**中的 `cangjie-sdk.toml` 优先于**父目录**的本机目录覆盖。不同目录层级的 `cangjie-sdk.toml` 文件不会合并。

要验证当前目录选中的工具链及其生效来源，可以随时运行：

```bash
cjv show active
```

注意：`cjv run <toolchain> <command>` 始终使用命令参数中的 `<toolchain>`。当命令行显式选择或 `CJV_TOOLCHAIN` 环境变量优先于项目文件生效时，`cangjie-sdk.toml` 中声明的组件和目标要求不会附加到该选择上。

## 字段说明

`cangjie-sdk.toml` 采用 [TOML](https://toml.io/) 格式，所有配置字段均位于 `[toolchain]` 表下：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `channel` | `string` | 通道名、版本选择器或已注册的自定义工具链名称 |
| `path` | `string` | 外部 SDK 目录的绝对路径，与 `channel` 互斥 |
| `components` | `string[]` | 需要的附加组件（`stdx`、`docs`、`stdx-docs`），默认为空 |
| `targets` | `string[]` | 需要的交叉编译 SDK 后缀（例如 `ohos`），默认为空 |

### 工具链配置项详解

#### `channel` 与 `path`

每个 `cangjie-sdk.toml` 必须且只能指定 `channel` 和 `path` 中的一个：

- **`channel`** 指定官方通道（`lts`、`sts`、`nightly`）、版本选择器（`lts-1.0.5`、`1.2`、`nightly-2026-10-04`）或通过 `cjv toolchain link` 注册的自定义工具链名称。
- **`path`** 使用绝对路径直接指向本机已解压的外部 SDK 目录：

```toml
[toolchain]
path = 'C:\SDKs\cangjie'
```

由于 `path` 直接引用外部 SDK 目录，其组件由该 SDK 目录的所有者自行维护，因此使用 `path` 时 `components` 与 `targets` 不会触发自动安装。

#### 组件与目标 SDK（`components` 与 `targets`）

当 `channel` 选中受管工具链时，可以声明项目所需的[组件](concepts/components.md)和[交叉编译目标 SDK](cross-compilation.md)：

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos", "android"]
```

在 `targets` 中请只写目标后缀（如 `ohos`、`android`），不要写 `linux-x64-ohos` 这样的完整宿主-目标平台名。`cjv` 会自动规范化大小写和下划线、展开逗号分隔项并去除重复项；空字符串目标会报错。

当该文件选中当前活跃工具链时，这一声明会自动准备宿主 SDK、宿主组件和目标 SDK。编译时如何加载目标 SDK 环境见[交叉编译](cross-compilation.md)。

## 自动安装

`auto_install` 设置默认开启（`true`）。当运行 `cjc`、`cjpm` 等代理命令时，`cjv` 会自动下载并补齐所选 `cangjie-sdk.toml` 中声明的缺失项。

如果希望缺少依赖时直接报错而不是自动下载，可以在用户设置中关闭 `auto_install`：

```bash
cjv set auto-install false
```

如果只想对单次调用关闭自动补装而不修改配置文件，可以使用 `cjv --no-auto-install exec cjc --version`，或为代理命令设置环境变量 `CJV_AUTO_INSTALL=0`。显式执行的 `cjv install` 不受该选项影响。

## 项目文件错误

`cangjie-sdk.toml` 中如果包含未知键名，`cjv` 会向 `stderr` 打印警告并继续解析已识别字段。但如果存在 TOML 语法错误、未知组件名称、文件内容为空，或缺少有效的 `channel` / `path`，命令会直接报错退出，而不会退回默认工具链。要恢复上级目录或全局默认选择，请删除该文件。

## 本机目录覆盖

使用 `cjv override` 可以在不修改项目文件的情况下，为本机的某个目录指定专用的工具链覆盖。设置后，在该目录及其子目录中运行 `cjc` 或 `cjpm` 都会调用覆盖指定的工具链：

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
cjv override list
cjv override unset
cjv override unset --nonexistent
```

目录覆盖保存在用户设置文件 [`settings.toml`](configuration.md) 中。`cjv override unset --nonexistent` 可以一键清除磁盘上已不存在目录的残留覆盖。
