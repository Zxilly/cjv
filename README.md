# cjv - Cangjie Version Manager

[English](README.EN.md) | 中文

[仓颉](https://cangjie-lang.cn/)编程语言 SDK 的工具链管理器。

cjv 管理多个仓颉 SDK 安装、处理版本切换，并提供 SDK 工具的透明代理执行。

## 安装

### 从源码编译

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

### 从发布二进制文件

从 [Releases](https://github.com/Zxilly/cjv/releases) 页面下载适合你平台的二进制文件，并将其放入 PATH 中。

## 快速开始

```bash
# 安装最新 LTS 工具链
cjv install lts

# 设为默认
cjv default lts

# 验证安装
cjv show

# 使用指定工具链运行命令
cjv run sts cjc --version
```

## 命令

| 命令                                                  | 说明                                          |
| ----------------------------------------------------- | --------------------------------------------- |
| `cjv install <toolchain> [-t target]`                 | 安装仓颉 SDK 工具链，可附加交叉编译目标       |
| `cjv uninstall <toolchain>`                           | 卸载工具链                                    |
| `cjv update [toolchain]`                              | 更新已安装的工具链                            |
| `cjv default [toolchain]`                             | 设置或显示默认工具链                          |
| `cjv show`                                            | 显示活跃和已安装的工具链                      |
| `cjv show active`                                     | 显示当前活跃的工具链                          |
| `cjv show installed`                                  | 列出已安装的工具链                            |
| `cjv show home`                                       | 显示 CJV_HOME 路径                            |
| `cjv run <toolchain> <command> [args...]`             | 使用指定工具链运行命令                        |
| `cjv exec [+toolchain] <command> [args...]`           | 使用仓颉运行时环境执行命令                    |
| `cjv envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]` | 输出配置仓颉运行时环境的 shell 命令           |
| `cjv which <command>`                                 | 显示活跃工具链中 SDK 工具的路径               |
| `cjv check`                                           | 检查可用更新（不安装）                        |
| `cjv override set <toolchain>`                        | 为当前目录设置工具链覆盖                      |
| `cjv override unset`                                  | 移除当前目录的工具链覆盖                      |
| `cjv override list`                                   | 列出所有目录覆盖                              |
| `cjv toolchain list`                                  | 列出已安装的工具链                            |
| `cjv toolchain link <name> <path>`                    | 将自定义工具链链接到本地目录                  |
| `cjv toolchain uninstall <name>`                      | 卸载工具链                                    |
| `cjv component add <name>... [--toolchain <tc>]`      | 为工具链安装 component（如 stdx）             |
| `cjv component link stdx <path> [--toolchain <tc>]`   | 将本地 stdx 目录链接到工具链（支持 custom）   |
| `cjv component remove <name>... [--toolchain <tc>]`   | 从工具链卸载 component                        |
| `cjv component list [--toolchain <tc>] [--installed]` | 列出 component 的安装与可用情况               |
| `cjv doc [--path] [--toolchain <tc>] [topic]`         | 在浏览器中打开当前工具链的离线文档            |
| `cjv set auto-self-update <enable\|disable\|check>`   | 设置自动自更新行为                            |
| `cjv set auto-install <true\|false>`                  | 设置代理模式下缺失工具链的自动安装            |
| `cjv set gitcode-api-key <key>`                       | 设置 GitCode API 访问令牌（nightly 构建需要） |
| `cjv self update`                                     | 更新 cjv 到最新版本                           |
| `cjv self uninstall`                                  | 卸载 cjv 及所有已安装的工具链                 |

## 工具链解析

cjv 按以下优先级顺序解析活跃工具链（从高到低）：

1. `CJV_TOOLCHAIN` 环境变量
2. 目录覆盖（通过 `cjv override set` 设置）
3. 工具链文件（当前目录或父目录中的 `cangjie-sdk.toml`）
4. 默认工具链（通过 `cjv default` 设置）

## 工具链文件 `cangjie-sdk.toml`

cjv 从当前目录开始向上递归查找 `cangjie-sdk.toml`，找到的第一个文件被视为该项目的工具链声明。所有字段都位于 `[toolchain]` 表中：

```toml
[toolchain]
channel = "lts"                          # 必填，工具链名称（如 lts / sts / nightly / 具体版本）
components = ["stdx", "docs"]            # 可选，需要随工具链安装的 component
targets = ["ohos", "android"]            # 可选，附加的交叉编译目标后缀
```

| 字段         | 类型     | 说明                                                                     |
| ------------ | -------- | ------------------------------------------------------------------------ |
| `channel`    | string   | 工具链名称；空文件等同于未声明，仍会回退到下一级解析                     |
| `components` | string[] | 启用 `auto_install` 时，代理执行会自动补齐缺失的 component               |
| `targets`    | string[] | 仅填写目标后缀（如 `ohos`、`android`、`ohos-arm32`），不要写完整平台 key |

未识别的键（如 `[toolchian]` 拼写错误或 `channal = "lts"`）会以 warn 级别日志提示，但不会阻止解析。`targets` 与 `components` 的具体语义详见下文对应章节。

## 交叉编译 SDK

目标 SDK 是宿主工具链的附加安装项，不改变当前活跃工具链；代理执行 `cjc`、`cjpm` 时仍使用宿主 SDK。

```bash
# 安装宿主 STS SDK，并额外安装当前宿主对应的 OHOS 交叉 SDK
cjv install sts -t ohos

# target 支持重复或逗号分隔
cjv install sts -t ohos -t android
cjv install sts --target ohos,android
```

项目也可以在 `cangjie-sdk.toml` 中声明附加 targets。开启 `auto_install` 时，代理执行会自动补齐缺失的目标 SDK：

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

`targets` 只填写目标后缀，例如 `ohos`、`android`、`ohos-arm32`；不要填写完整平台 key，例如 `linux-x64-ohos`。

## Components

cjv 通过 component 机制管理与 SDK 一同发布的扩展资源。当前支持：

- `stdx`：Cangjie 扩展库；解压后位于 `<CJV_HOME>/stdx/<tc>/{dynamic,static}`，并在代理执行的环境中自动注入 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`。LTS / STS 从 [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases) 下载，nightly 从 [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) 下载。
- `docs`：仓颉主体离线文档（dev-guide、libs/std、tools）。LTS / STS 从 [`cangjie-docs-bundle`](https://github.com/Zxilly/cangjie-docs-bundle/releases) GitHub release 下载，nightly 从 [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) 下载。
- `stdx-docs`：仓颉扩展库离线文档。LTS / STS 从 [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases) 下载，nightly 从 [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) 下载。

```bash
# 安装时顺带装上 component
cjv install nightly -c stdx,docs

# 单独管理
cjv component add stdx --toolchain lts
cjv component remove stdx-docs
cjv component list --toolchain nightly
```

### 链接本地 stdx

对于通过 `cjv toolchain link` 链接的 custom 工具链，`cjv component add stdx` 无法工作（custom 工具链没有对应的 release 资产）。可以改用 `cjv component link stdx <path>` 把本地 stdx 目录挂上去：

```bash
# 先链接一个本地编译/获取的 SDK
cjv toolchain link mysdk /path/to/local/sdk

# 再把本地 stdx 链接到这个工具链
cjv component link stdx /path/to/local/stdx --toolchain mysdk

# 标准 channel 也可用 link 替代下载（如离线环境、调试自编译 stdx）
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

`<path>` 必须是一个包含 `dynamic/` 和 `static/` 两个子目录的目录（即解压后的 stdx 布局）。link 后 cjv 会在 `<CJV_HOME>/stdx/<tc>/` 下创建两个符号链接（Windows 上 fallback 到 directory junction），`CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC` 仍按常规注入。`cjv component remove stdx` 和 `cjv toolchain uninstall` 都只会删除符号链接，不会触及用户原始数据。

`cangjie-sdk.toml` 中的 `components` 字段同样会被识别；当 `auto_install` 开启时，代理执行将按需补齐缺失的 component：

```toml
[toolchain]
channel = "nightly"
components = ["stdx", "docs"]
```

`cjv doc` 在浏览器中打开当前工具链的本地 HTML（默认根 `index.html`，可用 topic `stdx` / `std` / `dev-guide` / `book` / `tools` 跳子页）。`--path` 只打印路径，不打开浏览器；如果对应工具链尚未安装 docs / stdx-docs，会提示用 `cjv component add` 先安装。

## 代理模式

当直接调用 SDK 工具（如 `cjc`、`cjpm`）时，cjv 会透明地将调用代理到相应的工具链。代理符号链接在安装时创建在 cjv 的 bin 目录中。

如果设置中启用了 `auto_install` 且解析到的工具链未安装，cjv 会在代理前自动安装它。

## 运行时环境

仓颉编译出的二进制文件动态链接运行时库（如 `libcangjie-runtime`），需要正确的库搜索路径才能运行。cjv 提供两种方式来配置运行时环境：

**一次性执行**：使用 `cjv exec` 在正确的运行时环境中执行命令，不影响当前 shell：

```bash
cjv exec ./my_binary arg1 arg2

# 指定工具链
cjv exec +nightly ./my_binary
```

**配置当前 shell 会话**：使用 `cjv envsetup` 输出环境变量配置脚本，之后可以直接运行编译产物：

```bash
# Bash/Zsh
eval "$(cjv envsetup)"

# Fish
cjv envsetup | source

# PowerShell
cjv envsetup | Invoke-Expression
```

两个命令都使用与代理模式相同的工具链解析优先级，支持 `+toolchain` 语法指定工具链。`cjv envsetup` 会自动检测当前 shell 类型，也可通过 `--shell=TYPE` 手动指定（支持 `bash`、`fish`、`powershell`、`cmd`）。

交叉编译场景下，可用 `--target=SUFFIX`（如 `ohos`）输出已安装目标 SDK 的环境（独立 SDK 模型：`CANGJIE_HOME` 指向目标 SDK 目录，PATH/库路径均取自该目录）。目标 SDK 需先通过 `cjv install <toolchain> --target <suffix>` 安装。

## 环境变量

| 变量                        | 说明                                                                     |
| --------------------------- | ------------------------------------------------------------------------ |
| `CJV_HOME`                  | 覆盖默认主目录（默认: `~/.cjv`）                                         |
| `CJV_TOOLCHAIN`             | 强制指定工具链，覆盖所有其他解析方式                                     |
| `CJV_LOG`                   | 设置日志级别: `debug`、`info`、`warn`（默认）、`error`                   |
| `CJV_MAX_RETRIES`           | 下载失败最大重试次数（默认: `3`）                                        |
| `CJV_DOWNLOAD_TIMEOUT`      | HTTP 下载超时秒数（默认: `180`）                                         |
| `CJV_GITCODE_API_KEY`       | GitCode API 访问令牌，用于查询和下载 nightly 工具链                      |
| `CJV_NO_PATH_SETUP`         | 设为 `1` 跳过首次安装时的 PATH 自动配置                                  |
| `CANGJIE_STDX_PATH_DYNAMIC` | 由 cjv 自动注入，指向 `<CJV_HOME>/stdx/<tc>/dynamic`（仅当 stdx 已安装） |
| `CANGJIE_STDX_PATH_STATIC`  | 由 cjv 自动注入，指向 `<CJV_HOME>/stdx/<tc>/static`（仅当 stdx 已安装）  |

## 目录结构

```
~/.cjv/
  bin/            # 代理符号链接和 cjv 二进制文件
  toolchains/     # 已安装的 SDK 工具链（仅 SDK 本体）
    <tc>/
      .cjv/components/         # cjv 维护的 component manifest
  stdx/           # stdx component（按工具链拆分，路径通过 CANGJIE_STDX_PATH_* 暴露）
    <tc>/
      dynamic/
      static/
  docs/           # 离线文档（与工具链解耦，docs 与 stdx-docs 各自独占子目录）
    <tc>/
      main/                    # docs component（dev-guide / libs/std / tools 入口）
      stdx/                    # stdx-docs component（libs_stdx 入口）
  downloads/      # 下载暂存区（安装成功后即清空，仅用于中断恢复）
  settings.toml   # 用户设置
```

注：`cjv toolchain uninstall <tc>` 会一并清掉 `stdx/<tc>/` 与 `docs/<tc>/`。

## 配置

设置存储在 `~/.cjv/settings.toml` 中，可通过 `cjv set` 命令修改。

## 许可证

Apache-2.0。详见 [LICENSE](LICENSE)。

