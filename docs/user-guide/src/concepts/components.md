# 组件与离线文档

每个[工具链](toolchains.md)都可以安装附属的可选**组件（components）**——包括 `stdx` 扩展库和离线 HTML 文档。组件按工具链分别独立存放，当卸载某个工具链时，它的附属组件也会一并清理：

| 组件 | 内容 | 相对 `CJV_HOME` 的位置 |
| --- | --- | --- |
| `stdx` | 动态与静态扩展库 | `stdx/<tc>/{dynamic,static}` |
| `docs` | 仓颉语言、标准库和工具文档 | `docs/<tc>/main/` |
| `stdx-docs` | 扩展库（`stdx`）文档 | `docs/<tc>/stdx/` |

当所选工具链安装了 `stdx` 后，代理命令（`cjc`、`cjpm`）、`cjv exec` 和 `cjv envsetup` 会自动向运行环境注入 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`。文档组件（`docs` 与 `stdx-docs`）供 `cjv doc` 打开阅读，不会修改运行时环境变量。

## 安装与移除

你可以在安装工具链时通过 `-c`（`--component`）参数一并安装组件，也可以随时通过 `cjv component` 子命令为已安装的工具链增删组件：

```bash
cjv install nightly -c stdx,docs
cjv component add stdx docs --toolchain lts
cjv component list --toolchain lts
cjv component list --installed -q
cjv component remove stdx-docs
```

- 未指定 `--toolchain` 时，命令默认作用于当前激活的工具链。
- 传入 `--target <suffix>` 时，命令作用于该宿主工具链下已安装的[交叉编译](../cross-compilation.md)目标 SDK（例如 `cjv component add stdx --toolchain sts --target ohos`）。
- 组件名可以传入多个参数，也可以用逗号分隔（如 `stdx,docs`）。
- `cjv component add` 默认跳过已安装的组件；加上 `--force` 可以强制重新下载安装。

实际可下载的组件制品取决于所选通道、版本和平台。自定义工具链没有上游官方清单记录，因此无法通过 `cjv component add` 下载官方组件。在团队项目中，也可以在 [`cangjie-sdk.toml`](../toolchain-file.md) 中声明 `components`，由自动安装按需补齐。

### 更新行为

运行 `cjv update` 升级跟踪通道（`lts`、`sts`、`nightly`）时，`cjv` 会将宿主 SDK 和已选组件作为同一事务一并更新；固定版本（如 `lts-1.0.5`）的组件保持不变。通过 `component link` 挂载的本地 `stdx` 在通道更新后仍保留原本地链接。

## 链接本地 `stdx`

如果你在本机编译了 `stdx` 或有现成的本地目录，可以使用 `cjv component link` 将其挂载到指定工具链：

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

指定的源目录必须同时包含 `dynamic/` 和 `static/` 子目录。`--force` 用于替换该工具链上已有的 `stdx` 组件；运行 `cjv component remove stdx` 移除链接时，只会删除 `cjv` 内部管理的链接记录，保留你的本地源目录不变。注意 `docs` 和 `stdx-docs` 不支持本地目录链接。

### 自定义 SDK 的组件管理

组件修改要求目标 SDK 位于 `cjv` 管理的真实目录中。通过 `cjv toolchain link <name> /path/to/sdk` 直接引用的外部解压目录由该目录的所有者维护，不支持通过 `cjv component` 命令修改。

如果需要让 `cjv` 为自定义 SDK 管理组件（例如挂载本地 `stdx`），请先从[本地归档或 URL](../install-from-url.md) 安装该自定义 SDK：

```bash
cjv toolchain link my-sdk ./cangjie-sdk.zip
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

## 打开离线文档（`cjv doc`）

安装好 `docs` 或 `stdx-docs` 组件后，使用 `cjv doc`（或命令别名 `cjv docs`）即可在默认浏览器中打开本地 HTML 文档：

```bash
cjv doc
cjv doc std
cjv doc dev-guide
cjv doc tools
cjv doc stdx --toolchain nightly
cjv doc --path
```

你可以指定要打开的文档主题：

- **省略主题（`cjv doc`）** — 优先打开主体文档首页（`docs`）；若仅安装了扩展库文档，则打开 `stdx-docs` 首页。
- **`std`** — 打开标准库 API 参考文档（需安装 `docs`）。
- **`dev-guide`**（或别名 **`book`**）— 打开仓颉开发指南（需安装 `docs`）。
- **`tools`** — 打开 SDK 工具指南（需安装 `docs`）。
- **`stdx`** — 打开扩展库参考文档（需安装 `stdx-docs`）。

传入 `--path` 或 `--json` 时，命令只向标准输出返回本地文件路径而不启动浏览器。如果缺少对应的文档组件，`cjv` 会提示安装命令。
