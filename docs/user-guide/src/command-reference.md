# 命令参考

本章逐条列出 cjv 的命令，给出用法、参数、标志与可复制的示例。每个命令的简短说明也会出现在 `cjv <command> --help` 中。

## 全局约定

几乎所有命令都接受全局标志 `--json`，它把结果以稳定的 JSON 结构输出到标准输出，便于脚本消费。`cjv run`、`cjv exec`、`cjv init` 不支持 JSON 输出，传入 `--json` 会报错。

未显式指定工具链的命令会按统一优先级解析活跃工具链：`CJV_TOOLCHAIN` 环境变量、目录覆盖、`cangjie-sdk.toml` 工具链文件、默认工具链，按此顺序取第一个生效的。详见 [目标与覆盖](concepts/targets-overrides.md)。

标准通道名为 `lts`、`sts`、`nightly`，也可写成具体版本（如 `lts-1.0.0`）。通过 `cjv toolchain link` 链接的自定义工具链使用任意自定义名，但不得与保留通道名冲突。`cjv exec` 和 `cjv envsetup` 还支持以 `+name` 前缀临时指定工具链，覆盖默认解析。

被代理或被执行的子命令以其原始退出码退出，这适用于 `cjv run` 与 `cjv exec`。

---

## 安装与卸载

### `cjv install`

安装仓颉 SDK 工具链，可附带交叉编译目标与组件。

```text
cjv install <toolchain> [-t target]... [-c component]... [--force]
```

参数：

- `<toolchain>`（必填）：要安装的工具链，如 `lts`、`sts`、`nightly` 或具体版本。它不能用于安装自定义工具链，那种情况请用 `cjv toolchain link`。

标志：

| 标志 | 说明 |
| --- | --- |
| `-t`, `--target <suffix>` | 需要附加安装的交叉编译目标后缀（可重复或逗号分隔），如 `ohos`、`android`、`ohos-arm32` |
| `-c`, `--component <name>` | 需要附加安装的组件（可重复或逗号分隔），如 `stdx`、`docs`、`stdx-docs` |
| `--force` | 强制重新下载并重装，即使已安装 |

示例：

```bash
# 安装最新 LTS 工具链
cjv install lts

# 安装具体版本
cjv install lts-1.0.0

# 安装宿主 STS SDK，并额外安装两个交叉编译目标
cjv install sts -t ohos -t android
cjv install sts --target ohos,android

# 安装时顺带装上组件
cjv install nightly -c stdx,docs

# 强制重装
cjv install lts --force
```

target 只填目标后缀，不要写完整平台 key（如 `linux-x64-ohos`）。交叉编译目标是宿主工具链的附加安装项，不改变活跃工具链。详见 [交叉编译](cross-compilation.md)与 [组件](concepts/components.md)。

### `cjv uninstall`

卸载工具链，并一并清理其 stdx 与离线文档。

```text
cjv uninstall <toolchain> [-y]
```

参数：

- `<toolchain>`（必填）：要卸载的工具链名称。

标志：

| 标志 | 说明 |
| --- | --- |
| `-y`, `--yes` | 跳过确认提示 |

卸载会在交互式终端弹出确认；非交互式终端、`--json` 模式或加 `-y` 时直接执行。如果被卸载的工具链是默认工具链，cjv 会把默认指向另一个已安装的宿主工具链，指向它的目录覆盖也会被清除。卸载会连带删除 `<CJV_HOME>/stdx/<tc>/` 与 `<CJV_HOME>/docs/<tc>/`。

```bash
cjv uninstall sts
cjv uninstall lts-1.0.0 -y
```

> `cjv toolchain uninstall <name>` 与本命令等价，行为一致。

### `cjv update`

将指定工具链或所有已安装工具链更新到对应通道的最新版本。

```text
cjv update [toolchain] [--no-self-update]
```

参数：

- `[toolchain]`（可选）：只更新指定工具链。省略时更新所有已安装工具链。

标志：

| 标志 | 说明 |
| --- | --- |
| `--no-self-update` | 跳过 cjv 自更新检查 |

传入通道名（如 `lts`）时，更新该通道当前已安装版本到最新版本。传入具体版本时，等同于安装该版本，已安装则跳过。自定义（链接）工具链无法更新，会被跳过或报错。更新到新版本后，原指向旧版本的默认工具链与目录覆盖会自动改指向新版本，旧目录被删除。更新结束后会根据 `auto-self-update` 设置决定是否自更新 cjv 本身，可用 `--no-self-update` 关闭。

```bash
# 更新所有工具链
cjv update

# 只更新 LTS
cjv update lts

# 更新但不触发 cjv 自更新
cjv update --no-self-update
```

### `cjv check`

检查已安装工具链是否有可用更新，但不执行安装。

```text
cjv check
```

逐个列出已安装工具链：有更新显示 `当前 → 最新`，已是最新显示 `✓`，并在末尾显示 cjv 自身版本。`--json` 模式输出结构化结果，含 `update_available`、`latest` 等字段。

```bash
cjv check
cjv check --json
```

---

## 查看与运行

### `cjv show`

显示活跃工具链、默认主机平台与已安装工具链列表。

```text
cjv show
cjv show active
cjv show installed
cjv show home
```

子命令：

| 子命令 | 说明 |
| --- | --- |
| `cjv show` | 显示活跃工具链 + 默认主机 + 已安装列表（含各工具链已装组件） |
| `cjv show active` | 仅显示当前活跃工具链及其来源 |
| `cjv show installed` | 仅列出已安装工具链 |
| `cjv show home` | 显示 `CJV_HOME` 路径及其来源 |

```bash
cjv show
cjv show active
cjv show home
```

### `cjv run`

使用指定工具链运行命令，不影响当前 shell。

```text
cjv run [--install] <toolchain> <command> [args...]
```

参数：

- `<toolchain>`（必填）：用于运行命令的工具链。
- `<command>`（必填）：要运行的命令；可以是工具链自带工具（如 `cjc`、`cjpm`），也可以是该工具链环境下 PATH 中的任意命令。
- `[args...]`：传给命令的参数。

标志：

| 标志 | 说明 |
| --- | --- |
| `--install` | 当目标工具链未安装时，先自动安装再运行 |

命令在该工具链的运行时环境中执行，cjv 会注入正确的 PATH 与库路径，并应用已安装组件的环境（如 `CANGJIE_STDX_PATH_*`）。该命令不支持 `--json`。

```bash
# 用 sts 工具链查看 cjc 版本
cjv run sts cjc --version

# 工具链未装则先装再运行
cjv run --install nightly cjpm build
```

### `cjv exec`

在仓颉运行时环境中执行任意命令，便于直接运行编译产物。

```text
cjv exec [+toolchain] <command> [args...]
```

参数：

- `[+toolchain]`（可选）：以 `+name` 前缀临时指定工具链；省略时按标准优先级解析活跃工具链。
- `<command>`（必填）：要执行的命令。
- `[args...]`：传给命令的参数。

仓颉编译出的二进制动态链接运行时库，需要正确的库搜索路径。`cjv exec` 在注入了运行时库路径的环境中执行命令，但不影响当前 shell。该命令不支持 `--json`。

```bash
# 在活跃工具链的运行时环境中运行编译产物
cjv exec ./my_binary arg1 arg2

# 指定工具链
cjv exec +nightly ./my_binary

# "--" 之后的内容原样传递，可运行以 "+" 开头的命令名
cjv exec -- +weird-command
```

详见 [运行时环境](runtime-environment.md)。

### `cjv envsetup`

输出用于配置仓颉运行时环境的 shell 命令，供当前 shell 会话 `eval`。

```text
cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]
```

参数与标志：

| 参数 / 标志 | 说明 |
| --- | --- |
| `[+toolchain]` | 以 `+name` 临时指定工具链 |
| `--shell=TYPE` | 手动指定 shell 类型：`bash`、`fish`、`powershell`、`cmd`；省略时自动检测 |
| `--target=SUFFIX` | 输出已安装目标 SDK 的运行时环境（独立 SDK 模型），如 `--target=ohos` |

`envsetup` 与代理模式使用相同的工具链解析优先级。`--target` 对应的目标 SDK 需先通过 `cjv install <toolchain> --target <suffix>` 安装。`--json` 模式输出结构化的环境描述（变量、PATH 前后置、库路径键），不打印 shell 脚本。

```bash
# Bash / Zsh
eval "$(cjv envsetup)"

# Fish
cjv envsetup | source

# PowerShell
cjv envsetup | Invoke-Expression

# 指定工具链并强制 bash 格式
cjv envsetup +nightly --shell=bash

# 输出已安装 ohos 目标 SDK 的环境
cjv envsetup --target=ohos
```

### `cjv which`

显示活跃工具链中某个 SDK 工具的路径；不带参数时打印工具链根目录。

```text
cjv which [command]
```

参数：

- `[command]`（可选）：要查询的工具名，如 `cjc`、`cjpm`。省略时打印活跃工具链根目录。

```bash
# 打印活跃工具链根目录
cjv which

# 打印 cjc 的绝对路径
cjv which cjc
```

`cjv which` 与 `cjv run` 使用一致的工具解析逻辑：除固定代理工具外，也能解析 `bin/` 与 `tools/bin/` 下的二进制。

### `cjv doc`

在浏览器中打开当前工具链的离线文档。

```text
cjv doc [topic] [--path] [--toolchain <tc>]
```

参数：

- `[topic]`（可选）：要跳转的子页主题，如 `stdx`、`std`、`dev-guide`、`book`、`tools`。省略时打开根 `index.html`。

标志：

| 标志 | 说明 |
| --- | --- |
| `--path` | 只打印文档路径或 URL，不打开浏览器 |
| `--toolchain <tc>` | 指定要打开文档的工具链（默认为当前活跃工具链） |

若目标工具链尚未安装 `docs` / `stdx-docs`，会提示先用 `cjv component add` 安装。`--json` 模式同样只返回路径，不启动浏览器。命令别名：`cjv docs`。

```bash
cjv doc
cjv doc std
cjv doc --path
cjv doc stdx --toolchain nightly
```

---

## 工具链管理

### `cjv toolchain list`

列出已安装的工具链（等价于 `cjv show installed`）。

```text
cjv toolchain list
```

### `cjv toolchain link`

将自定义工具链链接到本地目录，或从 URL 下载并安装为 cjv 拥有的工具链。

```text
cjv toolchain link <name> <path|url> [--sha256 <hash>] [--force] [--no-stdx]
```

参数：

- `<name>`（必填）：自定义工具链名。必须是自定义名，不能与保留通道名 `lts`、`sts`、`nightly` 冲突，也不能含路径分隔符、`+` 前缀或为非法名。
- `<path|url>`（必填）：本地目录路径，或 `http(s)://` URL。命令根据是否匹配 `^https?://` 自动分流为两种模式。

两种模式：

| 维度 | 本地路径模式 | URL 模式 |
| --- | --- | --- |
| `<path>` 形态 | 本地目录 | `https://...` 或 `http://...` |
| `toolchains/<name>` 内容 | 符号链接 / junction（Windows 回退到 junction） | 下载、解包后落地的真实目录 |
| 数据归属 | cjv 不拥有，只引用 | cjv 拥有 |
| 卸载行为 | 只删链接，原目录保留 | 删除整个目录（含 stdx） |

标志（仅 URL 模式）：

| 标志 | 说明 |
| --- | --- |
| `--sha256 <hash>` | 用该 SHA-256 校验下载的归档 |
| `--force` | 覆盖同名的已存在工具链 |
| `--no-stdx` | 跳过安装随包的 stdx 组件 |

这三个标志只对 URL 模式有效，与本地路径一起使用会报错，而不是静默忽略。本地路径模式要求目录是一个真实的仓颉 SDK（须存在 `bin/cjc`）。

```bash
# 本地路径模式：只创建链接，原目录保留
cjv toolchain link mysdk /path/to/local/sdk

# URL 模式：下载、解包，落地为 cjv 拥有的真实目录
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip

# URL 模式 + 校验 + 覆盖同名 + 跳过随包 stdx
cjv toolchain link mysdk https://example.com/sdk.zip \
  --sha256 <hash> --force --no-stdx
```

URL 模式的完整语义（名称校验时机、随包 stdx、跨系统限制等）见 [从 URL 安装工具链](install-from-url.md)。链接本地 stdx 见 [组件](concepts/components.md)。

### `cjv toolchain uninstall`

卸载工具链（等价于 `cjv uninstall`）。

```text
cjv toolchain uninstall <name> [-y]
```

| 标志 | 说明 |
| --- | --- |
| `-y`, `--yes` | 跳过确认提示 |

---

## 组件管理

`cjv component` 的子命令统一支持持久标志 `--toolchain <tc>` 指定目标工具链；省略时使用当前活跃工具链。

### `cjv component add`

为工具链安装一个或多个组件（如 `stdx`、`docs`、`stdx-docs`）。

```text
cjv component add <name>... [--toolchain <tc>] [--force]
```

| 标志 | 说明 |
| --- | --- |
| `--toolchain <tc>` | 目标工具链（默认为当前活跃工具链） |
| `--force` | 强制重新下载并重装，即使已安装 |

`<name>` 可重复或逗号分隔。通过 `cjv toolchain link` 链接的自定义工具链没有对应的 release 资产，`component add` 对其不可用，请改用 `cjv component link`。

```bash
cjv component add stdx --toolchain lts
cjv component add stdx,docs
cjv component add stdx --force
```

### `cjv component link`

将本地组件目录链接到工具链，而非通过下载安装。当前用于 `stdx`。

```text
cjv component link <name> <path> [--toolchain <tc>] [--force]
```

| 标志 | 说明 |
| --- | --- |
| `--toolchain <tc>` | 目标工具链（默认为当前活跃工具链） |
| `--force` | 替换已存在的组件安装（无论它是 link 还是下载得到的） |

`<path>` 必须是一个解压后的 stdx 布局目录，包含 `dynamic/` 与 `static/` 两个子目录。link 后 cjv 在 `<CJV_HOME>/stdx/<tc>/` 下创建符号链接（Windows 回退到 junction），`CANGJIE_STDX_PATH_DYNAMIC` / `CANGJIE_STDX_PATH_STATIC` 仍按常规注入。`cjv component remove` 与卸载工具链只会删除符号链接，不触及原始数据。

```bash
# 自定义工具链没有 release 资产，用 link 挂上本地 stdx
cjv toolchain link mysdk /path/to/local/sdk
cjv component link stdx /path/to/local/stdx --toolchain mysdk

# 标准通道也可用 link 替代下载（离线 / 调试自编译 stdx）
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

### `cjv component remove`

从工具链卸载一个或多个组件。

```text
cjv component remove <name>... [--toolchain <tc>]
```

`<name>` 可重复或逗号分隔。别名：`uninstall`、`rm`、`delete`、`del`。

```bash
cjv component remove stdx-docs
cjv component remove stdx,docs --toolchain nightly
```

### `cjv component list`

列出组件的已安装与可安装情况。

```text
cjv component list [--toolchain <tc>] [--installed] [-q]
```

| 标志 | 说明 |
| --- | --- |
| `--toolchain <tc>` | 目标工具链（默认为当前活跃工具链） |
| `--installed` | 仅列出已安装的组件 |
| `-q`, `--quiet` | 以单列形式输出（只打印名字，便于脚本） |

```bash
cjv component list
cjv component list --toolchain nightly
cjv component list --installed -q
```

详见 [组件](concepts/components.md)。

---

## 默认工具链与覆盖

### `cjv default`

设置或显示默认工具链。

```text
cjv default [toolchain]
```

参数：

- `[toolchain]`（可选）：要设为默认的工具链。省略时显示当前默认。传入 `none` 清除默认设置。

交叉编译目标变体（如 `lts-1.0.0-ohos`）不能设为活跃或默认工具链，请用宿主工具链并通过 targets 配置。若设为一个尚未安装的工具链，会给出 warn 但不阻止。

```bash
# 显示当前默认
cjv default

# 设为 lts
cjv default lts

# 清除默认
cjv default none
```

### `cjv override set`

为某个目录设置工具链覆盖。进入该目录（或其子目录）时，cjv 优先使用该工具链。

```text
cjv override set <toolchain> [--path <dir>]
```

| 标志 | 说明 |
| --- | --- |
| `--path <dir>` | 为指定目录设置覆盖，而非当前目录 |

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
```

### `cjv override unset`

移除目录的工具链覆盖。

```text
cjv override unset [--path <dir>] [--nonexistent]
```

| 标志 | 说明 |
| --- | --- |
| `--path <dir>` | 移除指定目录的覆盖，而非当前目录 |
| `--nonexistent` | 移除所有指向已不存在目录的覆盖 |

```bash
cjv override unset
cjv override unset --path /path/to/project
cjv override unset --nonexistent
```

### `cjv override list`

列出所有目录覆盖。

```text
cjv override list
```

工具链解析优先级与覆盖语义详见 [目标与覆盖](concepts/targets-overrides.md)。

---

## 配置

### `cjv set`

修改 cjv 设置（存储在 `<CJV_HOME>/settings.toml`）。

```text
cjv set auto-self-update <enable|disable|check>
cjv set auto-install <true|false>
cjv set default-host <goos-goarch>
cjv set gitcode-api-key <key>
cjv set home <path>
```

子命令：

| 子命令 | 取值 | 说明 |
| --- | --- | --- |
| `auto-self-update` | `enable` / `disable` / `check` | 设置自动自更新行为；`check` 只检查不更新 |
| `auto-install` | `true` / `false` | 代理模式下，解析到的工具链未安装时是否自动安装 |
| `default-host` | `<goos-goarch>` | 设置默认主机平台标识（如 `linux-amd64`），用于解析下载平台 |
| `gitcode-api-key` | `<key>` | 设置 GitCode API 访问令牌（查询和下载 nightly 构建需要）；显示时会被掩码 |
| `home` | `<path>` | 持久化 `CJV_HOME` 到 settings.toml；传空字符串清除该覆盖；`CJV_HOME` 环境变量仍优先生效 |

```bash
cjv set auto-self-update check
cjv set auto-install true
cjv set default-host linux-amd64
cjv set gitcode-api-key <your-token>
cjv set home /opt/cjv
```

详见 [配置](configuration.md)与 [环境变量](environment-variables.md)。

---

## 自管理

### `cjv self update`

将 cjv 自身更新到最新版本，并刷新代理符号链接与托管的 env 脚本。

```text
cjv self update
```

```bash
cjv self update
```

### `cjv self uninstall`

卸载 cjv 自身以及所有已安装的工具链（删除整个 `<CJV_HOME>/` 并清理 PATH 配置）。

```text
cjv self uninstall [-y]
```

| 标志 | 说明 |
| --- | --- |
| `-y`, `--yes` | 跳过确认提示 |

交互式终端会弹出确认。`--json` 模式下必须配合 `-y` 才能执行。

```bash
cjv self uninstall
cjv self uninstall -y
```

---

## 安装引导

### `cjv init`

交互式引导首次安装：配置数据目录、PATH，并可选安装默认工具链与组件。通常由安装脚本调用，也可手动运行。

```text
cjv init [-y] [--default-toolchain <name>] [-c component]... [--no-modify-path]
```

| 标志 | 说明 |
| --- | --- |
| `-y`, `--yes` | 跳过交互菜单，按默认选项非交互安装 |
| `--default-toolchain <name>` | 要安装的默认工具链（默认 `lts`；用 `none` 跳过安装工具链） |
| `-c`, `--component <name>` | 随默认工具链安装的组件（可重复或逗号分隔） |
| `--no-modify-path` | 不修改 PATH |

标准输入不是终端时（如 `curl ... | sh` 引导），自动回退为非交互安装。该命令不支持 `--json`。

```bash
cjv init
cjv init -y --default-toolchain lts -c stdx,docs
cjv init -y --default-toolchain none --no-modify-path
```

安装方式详见 [安装 cjv](installation/index.md)。
