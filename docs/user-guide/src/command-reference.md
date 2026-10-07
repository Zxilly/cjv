# 命令参考

用 `cjv <command> --help` 查看本机版本的帮助。名称格式见[工具链与版本](concepts/toolchains.md)，选择顺序见[项目工具链](toolchain-file.md)。

## 全局选项

| 选项 | 作用 |
| --- | --- |
| `--json` | 结构化结果写入 stdout；`run`、`exec`、`init` 不支持 |
| `--quiet`、`-q` | 隐藏进度；`component list -q` 单列打印名称 |
| `--verbose` | 未设置 `CJV_LOG` 时启用 debug 日志，与全局 quiet 互斥 |
| `--help`、`-h` | 显示帮助 |
| `--version`、`-v` | 显示 cjv 版本 |
| `+name` | 放在子命令前，为支持选择工具链的命令指定 SDK |

显式 `--toolchain` 优先于全局 `+name`。`exec` 和 `envsetup` 也接受子命令后的 `+name`。子进程参数由 `run`、`exec` 原样传递，退出码也会保留。

## 安装、更新与卸载

```text
cjv install [toolchain]... [-t target]... [-c component]... [--force | --no-update] [--allow-downgrade]
cjv update [toolchain]... [--force] [--allow-downgrade] [--no-self-update]
cjv uninstall <toolchain>... [-y]
cjv check
```

`install` 省略名称时安装当前选择，指定多个名称时逐项处理。`-t/--target` 和 `-c/--component` 可重复或逗号分隔。

`update` 指定名称时处理这些工具链；无名称时使用全局 `+name`，否则更新所有已跟踪通道。具体版本保持固定，自定义工具链没有官方更新。只有全量更新会按 `auto_self_update` 检查或更新 cjv，`--no-self-update` 可跳过。

| 策略 | 行为 |
| --- | --- |
| 默认 | 保留所需组件和目标；缺少制品时阻止更新，nightly 可查找兼容历史版本 |
| `--force` | 允许跳过并移除未发布的可选组件或目标，不重装未变化的 SDK |
| `install --no-update` | 保留已有发行版，只补齐目标或组件，与 `--force` 互斥 |
| `--allow-downgrade` | 允许选择更旧的兼容 nightly |

通道升级先准备宿主、跟踪目标和组件，再一起发布；失败时保留或恢复原安装。固定版本独立保留。无新增需求的 `--no-update` 可离线完成。

`uninstall` 同时清理 SDK、组件、文档及相关目录覆盖。移除默认工具链时尝试选择其他已装宿主。交互终端会确认，`-y/--yes`、非交互或 JSON 模式直接执行。`toolchain uninstall` 是等价入口。

`check` 只查询 SDK 和 cjv 的可用更新。开发构建跳过自身版本比较。

## 查询与运行

| 命令 | 作用 |
| --- | --- |
| `cjv show` | 当前工具链、默认主机和安装列表 |
| `cjv show active` | 当前工具链及来源 |
| `cjv show installed`、`cjv toolchain list` | 已安装工具链 |
| `cjv show home` | 数据目录及来源 |
| `cjv which [command] [--toolchain <tc>]` | SDK 工具路径；省略 command 时返回 SDK 根目录 |
| `cjv run [--install] <toolchain> <command> [args...]` | 在指定 SDK 环境中运行命令；先查 SDK 工具，再查该环境的 PATH |
| `cjv exec [+toolchain] <command> [args...]` | 在当前或指定工具链环境中运行命令 |
| `cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]` | 输出当前 shell 的环境脚本，JSON 模式返回环境数据 |

`run --install` 允许安装缺失工具链，放在工具链参数前。`exec -- +command` 可执行以 `+` 开头的命令。`envsetup` 支持 `bash`、`fish`、`powershell`、`cmd`；`--target` 要求交叉 SDK 已安装。用法见[运行环境](runtime-environment.md)。

```text
cjv toolchain list-remote [--channel all|lts|sts|nightly] [-t suffix] [--all-platforms] [--limit N]
```

远程列表默认查询所有通道和当前主机平台。`--all-platforms` 按所有平台分组；`--limit 0` 不限制每组版本数。`-t/--target` 过滤交叉目标后缀。

## 自定义工具链

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

目录参数创建链接；归档或 URL 解包为受管安装。三个标志仅适用于归档来源：校验 SHA-256、替换同名安装、跳过随包 stdx。布局和平台限制见[自定义 SDK](install-from-url.md)。

## 目标与组件

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--force]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--force]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [-q]
```

`target add` 保留宿主版本；`target remove` 保留宿主安装。`target list --installed` 可离线查询。组件的 `--target` 只选择已有交叉 SDK，不安装 SDK。

组件名支持 `stdx`、`docs`、`stdx-docs`，可逗号分隔。`component add --force` 重新安装组件，`component link --force` 替换已有组件。只有 stdx 支持本地链接，源目录需包含 `dynamic/` 和 `static/`。外部 SDK 目录链接不支持组件修改。详情见[组件](concepts/components.md)。

## 离线文档

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

主题为 `std`、`dev-guide`（别名 `book`）、`tools`、`stdx`。省略时打开文档首页。`--path` 和 `--json` 只返回路径，其他情况启动浏览器。`docs` 是命令别名，需要先安装文档组件。

## 选择与设置

```text
cjv default [toolchain]
cjv override set <toolchain> [--path <dir>]
cjv override unset [--path <dir>] [--nonexistent]
cjv override list
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set home <path>
```

`default` 省略参数时显示默认选择，`none` 清除选择；指定未安装的官方工具链时先安装，失败则保留原选择。交叉 SDK 不能设为默认。

目录覆盖默认作用于当前目录，`--nonexistent` 清理已不存在目录。设置写入 `~/.cjv/settings.toml`，不随 `CJV_HOME` 改变。字段含义见[配置](configuration.md)。

## 初始化与自管理

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
cjv self update
cjv self uninstall [-y]
```

`init` 安装 cjv 命令入口、配置 PATH，并默认安装 `lts`。`--default-toolchain none` 跳过 SDK，`-c/--component` 选择组件，`-y/--yes` 跳过交互，`--no-modify-path` 保留现有 PATH。非终端输入采用非交互模式。

`self update` 更新 cjv 本体。`self uninstall` 删除整个数据目录、工具链及组件，并清理 PATH；JSON 模式必须加 `-y`。

## Shell 补全

```text
cjv completion <bash|zsh|fish|powershell>
```

输出对应 shell 的补全脚本，加载方法见 `cjv completion <shell> --help`。
