# 命令参考

`cjv` 内置了完整的命令行帮助系统；你可以随时运行 `cjv help` 或 `cjv <command> --help` 查看本机已安装版本的详细参数。版本名称格式见[工具链与版本](concepts/toolchains.md)，工具链选择顺序见[项目工具链](toolchain-file.md)。

## 全局选项

| 选项 | 作用 |
| --- | --- |
| `--json` | 将结构化 JSON 结果写入 `stdout`；`run`、`exec`、`init` 不支持 |
| `--quiet`、`-q` | 隐藏进度输出；`component list -q` 单列打印组件名称 |
| `--verbose` | 未设置 `CJV_LOG` 时在 `stderr` 启用 `debug` 日志，与 `--quiet` 互斥 |
| `--offline` | 禁止 `cjv` 联网，只使用本地安装和[可复用缓存](configuration.md#下载缓存) |
| `--config <path>` | 选择用户设置文件，仍保留系统后备配置和项目选择 |
| `--no-auto-install` | 仅在本次调用中禁止自动补装缺失项 |
| `--help`、`-h` | 显示命令帮助 |
| `--version`、`-v` | 显示 `cjv` 版本号 |
| `+name` | 放在子命令前，为支持选择工具链的命令显式指定 SDK |

子命令的显式 `--toolchain` 参数优先于前置的全局 `+name`。`exec` 和 `envsetup` 也接受跟在子命令名称后的 `+name`。子进程参数由 `run`、`exec` 原样透传，退出码也会完整保留。

## 安装、更新与卸载

### install

```text
cjv install [toolchain]... [-t target]... [-c component]... [--force | --no-update] [--allow-downgrade] [--reinstall]
```

`install` 省略名称时安装当前选中的工具链；指定多个名称时按顺序逐项处理。`-t`（`--target`）和 `-c`（`--component`）可重复传入或使用逗号分隔：

```bash
cjv install lts sts
cjv install nightly -c stdx,docs
```

### 安装与更新策略

| 策略 | 行为 |
| --- | --- |
| 默认 | 保留所需组件和交叉目标；缺少制品时阻止更新，其中 `nightly` 可在已发布历史中查找兼容版本 |
| `--force` | 允许跳过并移除未发布的可选组件或目标，不会重新安装未发生变化的 SDK |
| `install --no-update` | 保留已有宿主发行版，只补齐新增的目标或组件，与 `--force` 互斥 |
| `install --reinstall` | 重新安装当前已有的发行版并保留组件和目标，用于修复损坏的 SDK |
| `--allow-downgrade` | 允许 `nightly` 安装或更新选择更旧的兼容 nightly 构建 |

没有新增组件或交叉目标需求时，`cjv install --no-update` 可在完全离线状态下完成。

### update

```text
cjv update [toolchain]... [--force] [--allow-downgrade] [--no-self-update] [--dry-run]
```

`update` 指定名称时只处理列出的工具链；省略名称时优先使用全局 `+name`，否则更新所有已安装的跟踪通道（`lts`、`sts`、`nightly`）。固定版本保持不变，自定义工具链没有官方更新源。只有不带名称且不带 `+name` 的全量更新才会按 [`auto_self_update`](configuration.md#自动更新) 检查或更新 `cjv` 本体；传入 `--no-self-update` 可跳过该检查。

通道升级会先在暂存区准备好宿主 SDK、已跟踪目标和组件，全部准备完成后再原子替换安装；若失败则保留或恢复原安装，固定版本独立保留。

```bash
cjv update lts
cjv update --no-self-update
```

第一条命令只更新 `lts`；第二条更新所有已跟踪通道，并跳过 `cjv` 本体的更新检查。

`--dry-run` 只解析更新计划，报告最新发行版、实际兼容版本与阻塞原因，不下载归档、不修改安装也不写入缓存；配合 `--json` 可供 CI 脚本检查。

### uninstall

```text
cjv uninstall <toolchain>... [-y]
```

`uninstall`（或等价入口 `cjv toolchain uninstall`）会同时清理指定 SDK、附属组件、离线文档以及引用它的本机目录覆盖。如果移除的是当前默认工具链，`cjv` 会尝试改选其他已安装宿主，否则清空默认选择。交互式终端会请求确认；`-y`（`--yes`）、非交互输入或 `--json` 模式直接执行。

### check

```text
cjv check
```

`check` 只查询已安装 SDK 和 `cjv` 本体的可用更新，不执行安装。SDK 更新结果采用与实际更新相同的组件与交叉目标兼容性规则。本地 `dev` 开发构建会跳过自身版本比较。

### 更新历史与回退

```text
cjv toolchain history [toolchain]
cjv toolchain history list [toolchain]
cjv toolchain history prune [toolchain] [--keep 2]
cjv toolchain rollback <toolchain> [snapshot]
```

历史快照按跟踪宿主及其交叉目标、组件整组保存在 `<CJV_HOME>/history/` 下，每次更新后默认保留最近 **2** 份。省略查询或清理的工具链名称时会处理所有历史组；`--keep 0` 清空全部历史快照并保留当前安装。`rollback` 省略快照 ID 时默认恢复最新快照，恢复前校验内容并将当前组保存为快照，以便再次撤销。详见[更新历史与回退](automation.md#更新历史与回退)。

### doctor 与缓存

```text
cjv doctor
cjv cache dir
cjv cache list
cjv cache clean
cjv cache prune [--max-age 720h]
```

`doctor` 只读检查配置、代理命令入口及 SDK 完整性，发现问题时返回非零退出码。`cache` 子命令管理 `<CJV_HOME>/cache/` 下已验证 SHA-256 的下载归档和版本清单；清理缓存绝不会删除已安装 SDK。JSON 约定和离线用法见[自动化与 JSON 接口](automation.md)。

## 查询工具链

| 命令 | 作用 |
| --- | --- |
| `cjv show` | 显示当前激活工具链、默认主机平台和已安装列表 |
| `cjv show active` | 显示当前激活工具链及其选择来源 |
| `cjv show installed`、`cjv toolchain list` | 列出已安装的宿主工具链及附加目标 |
| `cjv show home` | 显示实际生效的数据目录及其配置来源 |
| `cjv which [command] [--toolchain <tc>]` | 显示 SDK 工具路径；省略 `[command]` 时返回 SDK 根目录 |

### toolchain list-remote

```text
cjv toolchain list-remote [--channel all|lts|sts|nightly] [-t suffix] [--all-platforms] [--limit N]
```

远程列表默认查询所有通道在当前主机平台下的可用版本。`--all-platforms` 按所有平台分组显示，`--limit 0` 不限制每组显示的版本数，`-t`（`--target`）按交叉目标后缀过滤。

## 运行命令与配置环境

### run

```text
cjv run [--install] <toolchain> <command> [args...]
```

使用显式指定的工具链运行命令。`cjv` 先查找该 SDK 的工具目录，再查找该环境的 `PATH`。需要自动安装缺失工具链时，请把 `--install` 放在 `<toolchain>` 参数前：

```bash
cjv run --install sts cjc --version
```

### exec

```text
cjv exec [+toolchain] <command> [args...]
```

使用当前激活工具链（或 `+name` 指定工具链）的运行环境启动命令。当编译产物需要链接仓颉运行时库或 `stdx` 动态库时，推荐这样运行：

```bash
cjv exec ./my_binary
cjv exec +nightly ./my_binary
```

如果被调用的命令名本身以 `+` 开头，请使用 `cjv exec -- +command`。`run` 和 `exec` 都会原样透传子进程参数并保留退出码。

### envsetup

```text
cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]
```

输出用于配置当前 shell 的脚本，支持 `bash`、`fish`、`powershell` 和 `cmd`。`--target` 要求交叉 SDK 已安装；`--json` 模式返回结构化环境数据。

输出脚本需要在当前 shell 中求值执行（如 `eval "$(cjv envsetup)"`）后才会生效。各 shell 的具体写法和恢复项目代理选择的方式见[运行命令与配置环境](runtime-environment.md)。

## 自定义工具链

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

传入已解压目录时创建引用链接；传入本地归档（`.zip`、`.tar.gz`）或 HTTP(S) URL 时解包为受管安装。`--sha256`、`--force` 和 `--no-stdx` 三个标志仅适用于归档来源，分别用于校验 SHA-256、替换同名安装和跳过随包 `stdx`。归档布局和平台限制见[自定义 SDK](install-from-url.md)。

## 交叉编译目标

```text
cjv target list [--toolchain <tc>] [--installed]
cjv target add <suffix>... [--toolchain <tc>]
cjv target remove <suffix>... [--toolchain <tc>]
```

`target add` 保留宿主当前版本并补装目标；`target remove` 移除交叉 SDK 及其组件并保留宿主安装。`target list --installed` 支持离线查询。详见[交叉编译](cross-compilation.md)。

## 组件

```text
cjv component add <name>... [--toolchain <tc>] [--target <suffix>] [--force]
cjv component link <name> <path> [--toolchain <tc>] [--target <suffix>] [--force]
cjv component remove <name>... [--toolchain <tc>] [--target <suffix>]
cjv component list [--toolchain <tc>] [--target <suffix>] [--installed] [-q]
```

组件命令的 `--target` 只选择已安装的交叉 SDK，不会自动安装该目标 SDK。

组件名支持 `stdx`、`docs`、`stdx-docs`，可传入多个参数或逗号分隔。`component add --force` 重新安装组件，`component link --force` 替换已有组件。只有 `stdx` 支持本地目录链接，且源目录需同时包含 `dynamic/` 和 `static/`。外部 SDK 目录链接不支持通过组件命令修改。详见[组件与离线文档](concepts/components.md)。

## 离线文档

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

支持的主题为 `std`、`dev-guide`（别名 `book`）、`tools`、`stdx`；省略主题时打开文档首页。`--path` 和 `--json` 只返回本地路径，其他情况在默认浏览器中打开文档。`cjv docs` 是命令别名，使用前需先安装对应文档组件。

## 选择与设置

### pin

```text
cjv pin [toolchain] [--check]
```

写入当前目录的 `cangjie-sdk.toml`，固定具体发行版并保留已有的 `components` 和 `targets`。`--check` 只验证项目文件是否固定了具体发行版，不修改文件。详见[自动化与 JSON 接口](automation.md#固定项目版本)。

### default

```text
cjv default [toolchain]
```

省略参数时显示当前全局默认选择，传入 `none` 时清除默认选择；指定未安装的官方工具链时会先安装，若失败则保留原选择。交叉 SDK 不能设为默认。

### override

```text
cjv override set <toolchain> [--path <dir>]
cjv override unset [--path <dir>] [--nonexistent]
cjv override list
```

目录覆盖默认作用于当前工作目录，`--nonexistent` 清理已不存在目录的残留记录。覆盖写入所选用户设置文件（默认 `~/.cjv/settings.toml`），不随 `CJV_HOME` 改变。详见[项目工具链](toolchain-file.md#本机目录覆盖)和[配置](configuration.md)。

### set

```text
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set home <path>
```

修改所选用户设置文件（默认 `~/.cjv/settings.toml`）。各字段默认值、系统后备继承规则和环境变量覆盖见[配置](configuration.md)。

## 初始化与自管理

### init

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
```

`init` 安装 `cjv` 命令入口并配置 `PATH`。它会查询当前平台的可用通道，按 `lts`、`sts`、`nightly` 的顺序自动选择第一个可用通道。交互式菜单只列出当前平台可用的通道；如果无法确认任何可用通道，初始化会报错退出。

| 选项 | 作用 |
| --- | --- |
| `--default-toolchain <name>` | 明确指定默认工具链 |
| `--default-toolchain none` | 只安装 `cjv` 本体，跳过 SDK |
| `-c`、`--component` | 安装初始 SDK 时一并安装指定组件 |
| `-y`、`--yes` | 跳过交互确认 |
| `--no-modify-path` | 保留现有 `PATH` 配置 |

非终端输入会自动采用非交互模式。当初始工具链安装失败时，命令返回非零退出码并保留已安装的 `cjv` 本体；错误信息中会给出保留组件选项的重试命令和查询兼容版本的命令，两者都使用已安装程序的绝对路径，无需先刷新 `PATH` 即可执行。

### self update / self uninstall

```text
cjv self update
cjv self uninstall [-y]
```

`self update` 检查并更新 `cjv` 本体。`self uninstall` 删除整个 `<CJV_HOME>` 数据目录、所有已安装工具链、组件、缓存与快照，并清理 `PATH`（`--json` 模式必须加 `-y`）。

## Shell 补全

```text
cjv completion <bash|zsh|fish|powershell>
```

输出对应 shell 的命令行补全脚本，加载方法见 `cjv completion <shell> --help`。
