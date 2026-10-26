# 项目工具链

将 `cangjie-sdk.toml` 放在项目根目录并提交到版本控制，即可让团队成员与 CI 流水线共享同一套工具链、组件与交叉编译目标声明。本章介绍如何创建和固定项目工具链文件、工具链的选择优先级，以及本机目录覆盖的用法。

## 声明与固定项目工具链

在项目根目录创建 `cangjie-sdk.toml`：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

示例中的 `lts-1.0.5` 仅作演示；支持的所有版本与通道语法见[工具链与版本](concepts/toolchains.md)。

你也可以运行 `cjv pin` 自动将解析后的具体发行版名称写入 `cangjie-sdk.toml`：

```bash
cjv pin lts
cjv pin --check
```

- `cjv pin lts`：将 `lts` 解析为完整发行版名称并写入 `cangjie-sdk.toml`，同时保留文件中已有的 `components` 和 `targets` 列表。
- `cjv pin`（省略名称）：将当前激活的工具链固定到 `cangjie-sdk.toml`。
- `cjv pin --check`：只检查当前目录的 `cangjie-sdk.toml` 是否固定了具体发行版，不修改文件也不发起网络请求。该选项适合在 CI 中执行校验，详见[自动化与 JSON 接口](automation.md#固定项目版本)。

## 工具链选择顺序

运行代理命令（如 `cjc`、`cjpm`）或 cjv 子命令时，cjv 按以下优先级从高到低确定当前工具链：

1. 命令行显式指定的工具链，例如全局 `+name` 选择器或子命令支持的 `--toolchain` 参数。
2. `CJV_TOOLCHAIN` 环境变量（见[环境变量](environment-variables.md)）。
3. 从当前工作目录向文件系统根目录逐层向上查找：在每一层目录中，先检查本机目录覆盖，再检查 `cangjie-sdk.toml`；一旦找到匹配项立即停止。
4. `cjv default` 设置的全局默认工具链。

由于 cjv 在第一个匹配的目录层停止查找，**同一目录**下的本机目录覆盖优先于该目录的 `cangjie-sdk.toml`；但**子目录**中的 `cangjie-sdk.toml` 优先于**父目录**的本机目录覆盖。不同目录层级的 `cangjie-sdk.toml` 文件不会合并。随时运行 `cjv show active` 即可查看最终选中的工具链及其来源。

`cjv run <toolchain> <command>` 始终使用命令参数指定的 `<toolchain>`。当命令行显式选择或 `CJV_TOOLCHAIN` 环境变量生效时，`cangjie-sdk.toml` 中声明的 `components` 和 `targets` 不会附加到该选择上。

## 配置字段说明

`cangjie-sdk.toml` 中的所有字段均位于 `[toolchain]` 表下：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `channel` | `string` | 通道名、版本选择器或已注册的自定义工具链名称 |
| `path` | `string` | 外部 SDK 目录的绝对路径，与 `channel` 互斥 |
| `components` | `string[]` | 项目需要的组件列表：支持 `stdx`、`docs`、`stdx-docs`，默认为空 |
| `targets` | `string[]` | 项目需要的交叉编译目标后缀列表（如 `ohos`、`android`），默认为空 |

### `channel` 与外部 `path` 互斥

`channel` 和 `path` 必须且只能设置其中一个。当设置 `path` 直接指向本机外部 SDK 目录时，组件由该 SDK 目录的所有者自行维护，`components` 与 `targets` 不会触发自动安装：

```toml
[toolchain]
path = 'C:\SDKs\cangjie'
```

### 声明组件与目标 SDK

`targets` 字段只接受目标后缀（如 `ohos`、`android`），不接受 `linux-x64-ohos` 这样的完整平台标识。cjv 会自动规范化大小写和下划线、展开逗号分隔项并去除重复项；空字符串目标会报错。

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos", "android"]
```

当该文件选中当前活跃工具链时，上述声明会自动准备宿主 SDK、宿主[组件](concepts/components.md)以及[交叉编译](cross-compilation.md)目标 SDK。编译时如何加载目标 SDK 的环境见[运行命令与配置环境](runtime-environment.md#加载交叉编译-sdk-环境)。

### 文件校验与错误处理

如果 `cangjie-sdk.toml` 包含未知字段，cjv 会打印警告并继续解析已知字段。但如果出现 TOML 语法错误、未知的组件名称、文件内容为空，或者缺少有效的 `channel` / `path`，cjv 会直接报错退出，而不会静默回退到全局默认工具链。如果需要恢复父目录或全局默认选择，请删除该文件。

## 自动安装缺失依赖

`auto_install` 设置默认开启（`true`）。当运行 `cjc`、`cjpm` 等代理命令时，cjv 会自动下载并安装当前 `cangjie-sdk.toml` 中声明但尚未安装的 SDK、组件和交叉编译目标。

如果关闭了 `auto_install`，遇到缺失项时命令会直接报错退出：

```bash
cjv set auto-install false
```

如果只想对单次命令临时关闭自动补装而不修改保存的[配置](configuration.md)，可以传入 `--no-auto-install`，或为代理命令设置环境变量 `CJV_AUTO_INSTALL=0`：

```bash
cjv --no-auto-install exec cjc --version
```

显式执行 `cjv install` 不受 `auto_install` 设置影响。

## 本机目录覆盖

如果只想在自己的开发机上为某个项目目录指定其他工具链，而不修改项目中的 `cangjie-sdk.toml`，请使用 `cjv override`：

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
cjv override list
cjv override unset
cjv override unset --nonexistent
```

目录覆盖保存在用户设置文件 `settings.toml` 中，作用于指定目录及其所有子目录。传入 `--nonexistent` 可以一键清理磁盘上已删除目录的残留覆盖记录。完整参数说明见[命令参考](command-reference.md#override)。
