# 命令参考

运行 `cjv <command> --help` 可查看本机已安装版本的内置帮助。版本名称语法见[工具链与版本](concepts/toolchains.md)，工具链选择顺序见[项目工具链](toolchain-file.md)。

## 全局选项

以下全局选项可与支持该选项的子命令搭配使用：

| 选项 | 作用 |
| --- | --- |
| `--json` | 将结构化 JSON 结果写入 `stdout`；`run`、`exec` 和 `init` 不支持 |
| `--quiet`、`-q` | 隐藏进度输出；`component list -q` 以单列纯文本打印组件名称 |
| `--verbose` | 未设置 `CJV_LOG` 时在 `stderr` 启用 `debug` 日志；与 `--quiet` 互斥 |
| `--offline` | 禁止 cjv 发起网络请求，仅使用本地安装和[可复用缓存](configuration.md#可复用离线缓存cache) |
| `--config <path>` | 选择用户设置文件 `settings.toml`；仍保留系统后备配置和项目工具链选择 |
| `--no-auto-install` | 仅在本次调用中禁止自动补装缺失的项目依赖 |
| `--help`、`-h` | 显示命令帮助 |
| `--version`、`-v` | 显示 `cjv` 版本号 |
| `+name` | 放在子命令前，为支持选择工具链的命令显式指定 SDK |

子命令的显式 `--toolchain` 参数优先于前置的全局 `+name` 选择器。`exec` 和 `envsetup` 也接受跟在子命令名称后面的 `+name`。`run` 和 `exec` 会将子进程参数与退出码原样透传。

## 安装、更新与卸载

### install

```text
cjv install [toolchain]... [-t target]... [-c component]... [--force | --no-update] [--allow-downgrade] [--reinstall]
```

省略 `[toolchain]` 时，`install` 安装当前选中的工具链；指定多个名称时按顺序逐项处理。`-t`（`--target`）和 `-c`（`--component`）支持重复传入或逗号分隔：

```bash
cjv install lts sts
cjv install nightly -c stdx,docs
```

### 安装与更新策略

| 策略或标志 | 行为说明 |
| --- | --- |
| 默认策略 | 保留已选组件和交叉目标；当所需制品缺失时阻止更新，其中 `nightly` 会在已发布历史中查找兼容构建 |
| `--force` | 允许跳过并移除未发布的可选组件或交叉目标；不会重新安装未发生变化的 SDK |
| `install --no-update` | 保留已安装的宿主发行版，仅补装新增的交叉目标或组件；与 `--force` 互斥 |
| `install --reinstall` | 重新安装当前已有的宿主发行版并保留组件和交叉目标，用于修复损坏的 SDK |
| `--allow-downgrade` | 允许 `nightly` 安装或更新选择更旧的兼容 nightly 构建 |

没有新增组件或交叉目标需求时，`cjv install --no-update` 可在离线状态下完成。

### update

```text
cjv update [toolchain]... [--force] [--allow-downgrade] [--no-self-update] [--dry-run]
```

- 指定工具链名称时，仅更新列出的工具链。
- 省略名称时，优先使用全局 `+name`；若未指定 `+name`，则更新所有已安装的跟踪通道（`lts`、`sts`、`nightly`）。
- 固定版本（如 `lts-1.0.5`）保持不变，[自定义工具链](install-from-url.md)没有官方更新源。
- 只有不带名称且不带 `+name` 的全量 `cjv update` 才会按 `auto_self_update` 设置检查或更新 cjv 本体；传入 `--no-self-update` 可跳过该检查。

更新跟踪通道时，cjv 会先在暂存区准备好宿主 SDK、已跟踪交叉目标和组件，全部就绪后再替换安装；若中途失败，原安装保持完整可用。

```bash
cjv update lts
cjv update --no-self-update
```

传入 `--dry-run` 只计算更新计划，报告最新发行版、实际兼容版本与阻塞原因，不下载归档也不写入缓存。搭配 `--json` 可供 CI 脚本检查。

### uninstall

```text
cjv uninstall <toolchain>... [-y]
```

`uninstall`（或等价入口 `cjv toolchain uninstall`）会一并删除指定 SDK、附属组件、离线文档以及指向它的本机目录覆盖。如果卸载的是当前默认工具链，cjv 会尝试自动改选其他已安装宿主工具链，否则清空默认设置。交互式终端会请求确认；传入 `-y`（`--yes`）、非交互输入或 `--json` 模式则直接执行。

### check

```text
cjv check
```

`check` 仅查询已安装 SDK 和 cjv 本体的可用更新，不执行安装。SDK 更新判断采用与 `cjv update` 相同的组件及交叉目标兼容性规则。本地 `dev` 开发构建会跳过自身版本比较。

### 更新历史与回退

```text
cjv toolchain history [toolchain]
cjv toolchain history list [toolchain]
cjv toolchain history prune [toolchain] [--keep 2]
cjv toolchain rollback <toolchain> [snapshot]
```

跟踪通道更新时，cjv 会将旧的宿主 SDK、交叉目标与组件整组保存为 `<CJV_HOME>/history/` 下的历史快照，每个通道默认保留最近 2 份：

- 执行 `history list` 或 `history prune` 时省略 `[toolchain]` 会处理所有通道的历史组；`--keep 0` 会清空全部历史快照并保留当前安装。
- `cjv toolchain rollback <toolchain>` 省略快照 ID 时默认恢复最新快照，恢复前校验快照内容并将当前组保存为快照，以便再次撤销。详见[自动化与 JSON 接口](automation.md#更新历史与回退)。

### doctor 与缓存

```text
cjv doctor
cjv cache dir
cjv cache list
cjv cache clean
cjv cache prune [--max-age 720h]
```

- `cjv doctor` 只读检查配置、代理命令入口与 SDK 完整性，发现问题时返回非零退出码。
- `cjv cache` 管理 `<CJV_HOME>/cache/` 下已通过 SHA-256 校验的归档和版本清单；清理缓存绝不会删除已安装的 SDK。详见[配置](configuration.md#可复用离线缓存cache)和[自动化与 JSON 接口](automation.md)。

## 查询工具链

| 命令 | 输出内容 |
| --- | --- |
| `cjv show` | 当前激活工具链、默认主机平台与已安装工具链列表 |
| `cjv show active` | 当前激活工具链名称及其选择来源 |
| `cjv show installed`、`cjv toolchain list` | 已安装的宿主工具链及其附加的交叉编译目标 |
| `cjv show home` | 当前生效的 `CJV_HOME` 数据目录及其配置来源 |
| `cjv which [command] [--toolchain <tc>]` | SDK 工具的完整路径；省略 `[command]` 时返回 SDK 根目录路径 |

### toolchain list-remote

```text
cjv toolchain list-remote [--channel all|lts|sts|nightly] [-t suffix] [--all-platforms] [--limit N]
```

查询分发清单中的远程可用版本。默认查询当前主机平台下的所有通道；`--all-platforms` 按全部平台分组显示，`-t`（`--target`）按交叉目标后缀过滤，`--limit 0` 不限制每组显示的版本数量。

## 运行命令与配置环境

各 shell 下的具体用法与切换方式见[运行命令与配置环境](runtime-environment.md)。

### run

```text
cjv run [--install] <toolchain> <command> [args...]
```

使用显式指定的 `<toolchain>` 运行命令。cjv 先查找该 SDK 的工具目录，再查找该环境的 `PATH`。将 `--install` 放在 `<toolchain>` 参数前，即可在工具链缺失时自动安装：

```bash
cjv run --install sts cjc --version
```

### exec

```text
cjv exec [+toolchain] <command> [args...]
```

在当前激活工具链（或 `+toolchain`）的运行环境和动态库路径下运行命令，适合运行依赖仓颉运行时库或 `stdx` 的编译产物：

```bash
cjv exec ./my_binary
cjv exec +nightly ./my_binary
```

如果命令名称以 `+` 开头，请使用 `cjv exec -- +command`。`run` 和 `exec` 均原样透传子进程参数并保留退出码。

### envsetup

```text
cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]
```

输出用于配置当前 shell 的环境脚本，支持 `bash`、`fish`、`powershell` 和 `cmd`。`--target` 要求对应的交叉 SDK 已安装；`--json` 模式返回结构化环境数据。输出脚本需要在当前 shell 中求值执行后才会生效。

## 自定义工具链

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

传入已解压目录时创建引用链接；传入本地归档（`.zip`、`.tar.gz`）或 HTTP(S) URL 时解包为受管安装。`--sha256`、`--force` 和 `--no-stdx` 三个标志仅适用于归档来源。归档目录布局和平台限制见[自定义 SDK](install-from-url.md)。

## 交叉编译目标

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
```

管理附加在宿主工具链上的交叉 SDK。`target add` 保留当前宿主版本并补装目标；`target remove` 移除交叉 SDK 及其组件并保留宿主安装。`target list --installed` 支持离线查询。详见[交叉编译](cross-compilation.md)。

## 组件

```text
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--force]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--force]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [-q]
```

管理宿主工具链或已安装 `--target <suffix>` 交叉 SDK 的 `stdx`、`docs` 和 `stdx-docs` 组件：

- `component add --force` 重新安装已存在的组件。
- `component link` 链接包含 `dynamic/` 和 `static/` 的本地 `stdx` 目录（`--force` 替换已有组件）；`docs` 和 `stdx-docs` 不支持本地目录链接。
- 通过 `cjv toolchain link` 引用的外部 SDK 目录不支持组件修改。详见[组件与离线文档](concepts/components.md)。

## 离线文档

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

在浏览器中打开离线 HTML 文档（传入 `--path` 或 `--json` 时仅返回本地路径）。支持的主题为 `std`、`dev-guide`（别名 `book`）、`tools` 和 `stdx`；省略 `[topic]` 时打开文档首页。`cjv docs` 是命令别名，使用前需先安装对应文档组件。

## 选择与设置

### pin

```text
cjv pin [toolchain] [--check]
```

将解析后的具体发行版名称写入当前目录的 `cangjie-sdk.toml`，并保留已有的 `components` 和 `targets`。`--check` 仅验证项目文件是否固定了具体发行版，不修改文件。详见[项目工具链](toolchain-file.md)和[自动化与 JSON 接口](automation.md#固定项目版本)。

### default

```text
cjv default [toolchain]
```

省略参数时显示当前全局默认工具链，传入 `none` 时清除默认选择。如果指定的官方工具链尚未安装，`default` 会先执行安装，若安装失败则保留原有默认设置。交叉 SDK 不能设为默认工具链。

### override

```text
cjv override set <toolchain> [--path <dir>]
cjv override unset [--path <dir>] [--nonexistent]
cjv override list
```

设置、移除或列出本机目录覆盖。默认作用于当前工作目录；`unset --nonexistent` 可清理磁盘上已不存在目录的记录。覆盖写入所选用户设置文件（默认 `~/.cjv/settings.toml`），不随 `CJV_HOME` 改变。详见[项目工具链](toolchain-file.md#本机目录覆盖)。

### set

```text
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set home <path>
```

修改当前用户设置文件（默认 `~/.cjv/settings.toml`）中的对应字段。各字段默认值、继承规则与环境变量覆盖见[配置](configuration.md)。

## 初始化与自管理

### init

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
```

安装 cjv 命令入口并配置 `PATH`。`init` 默认查询当前平台的可用通道，并按 `lts`、`sts`、`nightly` 的顺序选择第一个可用通道。交互式菜单只列出当前平台可用的通道；如果无法确认任何可用通道，初始化会报错退出。

| 选项 | 作用 |
| --- | --- |
| `--default-toolchain <name>` | 显式指定默认工具链 |
| `--default-toolchain none` | 只安装 cjv 本体，跳过 SDK 安装 |
| `-c`、`--component` | 安装默认 SDK 时一并安装指定组件 |
| `-y`、`--yes` | 跳过交互确认 |
| `--no-modify-path` | 保留现有 `PATH` 配置不变 |

如果工具链安装失败，`init` 返回非零退出码并保留已安装的 cjv 本体，同时在错误信息中给出使用程序绝对路径的重试命令和 `list-remote` 查询命令。安装脚本参数见[安装 cjv](installation/index.md)。

### self update 与 self uninstall

```text
cjv self update
cjv self uninstall [-y]
```

- `cjv self update`：检查并更新 cjv 本体。
- `cjv self uninstall`：删除整个 `<CJV_HOME>` 数据目录、所有已安装工具链和组件，并清理 `PATH` 配置；在非交互或 `--json` 模式下必须传入 `-y`。

> **注意：** `cjv self uninstall` 会永久删除 `<CJV_HOME>` 中的所有 SDK、组件、缓存和历史快照。

## Shell 补全

```text
cjv completion <bash|zsh|fish|powershell>
```

输出指定 shell 的命令行自动补全脚本，加载方法请运行 `cjv completion <shell> --help` 查看。
