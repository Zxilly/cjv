# 组件

组件（component）是与仓颉 SDK 一同发布、但与 SDK 本体分开管理的扩展资源。装好一个[工具链](toolchains.md)之后，可以按需为它挂上组件，也可以单独卸载，不影响 SDK 本身。

cjv 当前支持三类组件：

| 组件        | 内容                                          | 安装位置（相对 `CJV_HOME`）   |
| ----------- | --------------------------------------------- | ----------------------------- |
| `stdx`      | 仓颉扩展库（Cangjie 扩展库的动态/静态库文件） | `stdx/<tc>/{dynamic,static}`  |
| `docs`      | 仓颉主体离线文档（dev-guide、libs/std、tools）| `docs/<tc>/main/`             |
| `stdx-docs` | 仓颉扩展库的离线文档                           | `docs/<tc>/stdx/`             |

其中 `<tc>` 是工具链名（如 `lts-1.0.5`）。组件按工具链拆分存放，每个工具链各有一份独立的组件。卸载工具链时，对应的 `stdx/<tc>/` 与 `docs/<tc>/` 会一并清理。

组件的下载来源取决于[通道](channels.md)：

- `stdx`：LTS / STS 从 [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases) 下载，nightly 从 [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) 下载。
- `docs`：LTS / STS 从 [`cangjie-docs-bundle`](https://github.com/Zxilly/cangjie-docs-bundle/releases) 的 GitHub release 下载，nightly 从 [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) 下载。
- `stdx-docs`：LTS / STS 从 [`cangjie_stdx`](https://gitcode.com/Cangjie/cangjie_stdx/releases) 下载，nightly 从 [`nightly_build`](https://gitcode.com/Cangjie/nightly_build/releases) 下载。

## 自动注入的环境变量

`stdx` 是唯一向运行时环境贡献变量的组件。某工具链装好 `stdx` 后，cjv 会在[代理执行](proxies.md)以及 `cjv exec` / `cjv envsetup` 中自动注入两个环境变量，无需手动设置：

| 环境变量                    | 指向                          |
| --------------------------- | ----------------------------- |
| `CANGJIE_STDX_PATH_DYNAMIC` | `<CJV_HOME>/stdx/<tc>/dynamic`|
| `CANGJIE_STDX_PATH_STATIC`  | `<CJV_HOME>/stdx/<tc>/static` |

这两个变量只在对应工具链确实装有 `stdx` 时才会出现。`docs` 与 `stdx-docs` 是纯文档数据，不贡献任何运行时环境变量。运行时环境的完整说明见[运行时环境](../runtime-environment.md)。

## 安装与卸载组件

最直接的方式是在安装工具链时用 `-c` / `--component` 一并装上组件：

```bash
# 安装 nightly 工具链，并顺带装上 stdx 和 docs
cjv install nightly -c stdx,docs
```

组件名支持逗号分隔，也支持重复传入（如 `-c stdx -c docs`）。

工具链装好之后也可以单独管理组件。`cjv component` 的子命令默认作用于当前活跃工具链，用 `--toolchain <tc>` 可指定其他工具链：

```bash
# 为 lts 工具链添加 stdx
cjv component add stdx --toolchain lts

# 一次添加多个
cjv component add stdx docs

# 卸载组件（remove 亦可写作 rm / uninstall / delete）
cjv component remove stdx-docs
```

`cjv component add` 在组件已安装时会跳过。如需强制重新下载安装，加 `--force`。

## 查看组件

`cjv component list` 列出组件在当前工具链下的安装与可用情况：

```bash
# 列出某工具链的所有组件及其状态
cjv component list --toolchain nightly

# 只看已安装的组件
cjv component list --installed
```

「可用」与否取决于通道。三类组件在 LTS、STS、nightly 上都受支持，但 custom 工具链没有对应的 release 资产，通过 `cjv component add` 安装会失败（见下一节）。

## 链接本地 stdx

对于通过 `cjv toolchain link` 链接的 custom 工具链，`cjv component add stdx` 无法工作，因为 custom 工具链没有可供下载的 release 资产。这时改用 `cjv component link stdx <path>`，把一个本地的 stdx 目录挂到工具链上：

```bash
# 先链接一个本地编译/获取的 SDK
cjv toolchain link mysdk /path/to/local/sdk

# 再把本地 stdx 链接到这个工具链
cjv component link stdx /path/to/local/stdx --toolchain mysdk
```

标准通道（如 `lts`）也可以用 `link` 替代下载，适合离线环境或调试自编译的 stdx。此时工具链上可能已存在一份下载安装的 stdx，需要加 `--force` 覆盖：

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

`<path>` 必须是一个包含 `dynamic/` 和 `static/` 两个子目录的目录，即解压后的标准 stdx 布局。link 时 cjv 会在 `<CJV_HOME>/stdx/<tc>/` 下为这两个子目录各创建一个符号链接（Windows 上若符号链接需要提权，会回退到 directory junction）。`CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC` 仍按常规注入，指向这些链接。

链接是安全的。`cjv component remove stdx` 和 `cjv toolchain uninstall` 都只删除 cjv 创建的符号链接，不会顺着链接删除原始目录里的数据。

> `link` 目前仅对 `stdx` 有效；`docs` 与 `stdx-docs` 不支持链接，只能下载安装。

## 在工具链文件中声明组件

[工具链文件](../toolchain-file.md) `cangjie-sdk.toml` 的 `components` 字段同样会被识别。开启 `auto_install` 后，[代理执行](proxies.md)会按需补齐当前项目缺失的组件：

```toml
[toolchain]
channel = "nightly"
components = ["stdx", "docs"]
```

团队成员进入项目目录运行 `cjc` 或 `cjpm` 时，cjv 就会在代理前自动装好声明的组件。

## 打开离线文档

装好 `docs` 或 `stdx-docs` 后，`cjv doc` 会在浏览器中打开当前工具链的本地 HTML 文档：

```bash
# 打开文档首页
cjv doc

# 跳转到指定主题
cjv doc stdx        # 扩展库文档（来自 stdx-docs）
cjv doc std         # 标准库
cjv doc dev-guide   # 开发指南（亦可用 book）
cjv doc tools       # 工具文档
```

常用参数：

- `--toolchain <tc>`：打开指定工具链的文档（默认当前活跃工具链）。
- `--path`：只打印解析出的文件路径，不启动浏览器，便于在脚本中使用或确认文档位置。

```bash
# 只打印路径，不打开浏览器
cjv doc --path

# 查看 nightly 工具链的工具文档路径
cjv doc tools --toolchain nightly --path
```

不带主题时 `cjv doc` 打开文档入口（优先 `docs` 的主页，其次 `stdx-docs`）。如果对应工具链尚未安装 `docs` / `stdx-docs`，命令会提示先用 `cjv component add` 安装相应组件。
