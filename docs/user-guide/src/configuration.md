# 配置

用户设置默认保存在 `~/.cjv/settings.toml`。你可以通过 `--config <path>` 或 `CJV_CONFIG_FILE` 指定另一个用户设置文件，其中命令行参数优先。修改 `CJV_HOME` 或设置中的 `home` 只改变数据目录位置，不会移动 `settings.toml`。本章说明所有设置字段、配置继承与优先级、数据目录结构、下载缓存以及 SDK 文件复用机制。

## 设置文件与字段说明

典型的 `settings.toml` 文件内容如下：

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true

[overrides]
"/home/me/project-a" = "sts"
```

下表列出 `settings.toml` 支持的全部字段：

| 字段 | 默认值或作用 | 修改方式 |
| --- | --- | --- |
| `version` | 设置文件格式版本号，由 cjv 维护 | 由 cjv 自动写入与迁移 |
| `default_toolchain` | 全局默认工具链名称 | 运行 `cjv default <name>`；传入 `none` 可清除 |
| `auto_self_update` | `check`（仅检查并提示更新）；还支持 `enable`、`disable` | 运行 `cjv set auto-self-update <value>` |
| `auto_install` | `true`（自动补齐当前工具链缺失的依赖项） | 运行 `cjv set auto-install <true\|false>` |
| `home` | 数据目录路径，默认为 `~/.cjv` | 运行 `cjv set home <path>` |
| `default_host` | 默认主机平台标识，默认自动检测 | 运行 `cjv set default-host <goos-goarch>` |
| `manifest_url` | 默认 LTS/STS 清单 URL；nightly 使用同目录下的 `nightly.json` | 直接编辑 `settings.toml` |
| `dist_server` | 同时提供 `versions.json` 和 `nightly.json` 的分发根 URL | 直接编辑 `settings.toml` |
| `link_mode` | SDK 文件复用模式：`hardlink` 或 `copy`，默认 `hardlink` | 直接编辑 `settings.toml` |
| `overrides` | 目录路径到工具链名称的映射表 | 运行 `cjv override set` 或 `cjv override unset` |

### 自动更新设置（`auto_self_update`）

`auto_self_update` 仅在执行未指定工具链名称且未带全局 `+name` 的全量 `cjv update` 时生效：

- `enable`：在更新工具链时自动下载并更新 cjv 本体。
- `check`：查询是否有新版 cjv 并打印提示，但不自动替换。
- `disable`：在 `cjv update` 时跳过 cjv 本体更新检查。

本地开发构建（`dev`）会自动跳过发行版检查。无论该项如何设置，你都可以随时运行 `cjv self update` 手动更新 cjv 本体。

### 数据目录与默认主机平台

运行 `cjv set home <path>` 时，cjv 会先将相对路径转换为绝对路径再保存。运行 `cjv set home ""` 可以显式恢复默认数据目录 `~/.cjv`。

`default_host` 字段使用 `linux-amd64`、`darwin-arm64` 这样的 Go 平台命名格式，与[工具链名称](concepts/toolchains.md#主机平台与交叉编译目标)中的 `linux-x64` 写法不同。

## 配置继承与优先级

### 用户设置与系统后备配置

用户设置文件会按字段逐项覆盖[系统后备配置](enterprise/index.md#系统后备配置)；用户文件中未设置的字段继续继承系统后备值。显式写入的 `false` 和空字符串 `""` 都属于明确的用户设置，会覆盖后备配置。

运行 `cjv set` 只会将本次修改的字段写入用户文件，不会把其他继承自系统后备配置的字段一并持久化。要恢复某个字段对系统后备配置的继承，只需从用户 `settings.toml` 中删除该字段。

如果 `settings.toml` 中存在未知键，cjv 会打印警告并读取已知键；如果文件存在语法错误，或 `version` 高于当前版本支持的范围，cjv 会直接报错退出。

### 路径、分发源与临时覆盖优先级

cjv 按以下顺序解析数据目录、分发源地址和单次运行覆盖：

- **数据目录优先级**：`CJV_HOME` 环境变量 > 合并设置中的 `home` > `~/.cjv`。
- **分发源优先级**：`CJV_DIST_SERVER` 环境变量 > 合并设置中的 `dist_server` > `manifest_url`。分发源目录结构见[内部分发源](enterprise/distribution-server.md)。
- **设置文件选择**：`--config <path>` 只替换用户设置文件；系统后备配置、[项目工具链文件](toolchain-file.md)和[环境变量](environment-variables.md)仍然生效。
- **自动补装临时覆盖**：`--no-auto-install` 仅在本次调用中关闭自动补装，`CJV_AUTO_INSTALL=0` 或 `1` 临时覆盖当前进程树的 `auto_install`。这些临时覆盖不会写入 `settings.toml`。

## 数据目录结构

cjv 在 `<CJV_HOME>` 下分类存放已安装的 SDK、组件、下载缓存和历史快照：

```text
<CJV_HOME>/
  bin/              # cjv 本体与 SDK 代理命令入口（cjc、cjpm 等）
  toolchains/<tc>/  # 已安装的宿主 SDK 与交叉编译目标 SDK
  stdx/<tc>/        # 已安装的 stdx dynamic/ 与 static/ 扩展库
  docs/<tc>/main/   # 离线主体文档
  docs/<tc>/stdx/   # 离线扩展库文档
  downloads/        # 安装暂存区与支持断点续传的部分下载文件
  cache/            # 缓存的版本清单与通过 SHA-256 校验的归档文件
  history/<tc>/     # 跟踪通道更新前保存的完整工具链组快照
```

只有当 `<CJV_HOME>` 使用默认路径 `~/.cjv` 时，`settings.toml` 才会与上述目录位于同一处。运行 `cjv show home` 可以查看当前生效的数据目录及其配置来源。

## 下载暂存与可复用缓存

### 断点续传与暂存清理

下载带有 SHA-256 校验和的归档时，如果网络中断，cjv 会在 `downloads/` 中保留已下载的部分文件，并在下次执行时断点续传、完成后校验完整归档的 SHA-256。

每次安装或更新成功后，cjv 会自动清理本次操作使用的暂存文件；当不带参数的全量 `cjv update` 全部成功时，还会一并清理 `downloads/` 中残留的历史暂存文件。

### 可复用离线缓存（`cache/`）

当带有 SHA-256 的制品成功安装后，cjv 会将版本清单和已通过 SHA-256 校验的归档文件保存到 `cache/` 目录中。缓存归档按来源 URL 和 SHA-256 摘要索引，每次复用前都会再次校验摘要。没有 SHA-256 校验和的归档不会进入可复用缓存。

传入 `--offline` 或设置 `CJV_OFFLINE=1` 时，cjv 禁止发起任何网络请求，仅从本地安装和 `cache/` 读取数据。缓存的清单反映最后一次联网查询时的内容，不一定包含上游最新发布版本；如果所需的清单或归档不在缓存中，离线命令会直接报错。

使用以下命令查看或清理可复用缓存：

```bash
cjv cache dir
cjv cache list
cjv cache prune --max-age 720h
cjv cache clean
```

清理缓存不会删除任何已安装的 SDK。在 CI 流水线中准备和复用离线缓存的做法见[自动化与 JSON 接口](automation.md#ci-隔离与离线缓存)。

## SDK 文件硬链接复用

默认的 `link_mode = "hardlink"` 会在具有相同发行版、平台和 SHA-256 摘要的多份 SDK 安装之间，自动通过硬链接复用内容相同的普通文件。安装元数据和组件目录保持独立；如果当前文件系统不支持硬链接，cjv 会自动回退为保留独立复制文件。

> **注意：** 硬链接共享底层文件内容。请勿手动原地编辑 `<CJV_HOME>/toolchains/` 中的 SDK 文件，否则会同时影响共享同一硬链接的其他工具链安装。

如果需要原地修改 SDK 文件做实验，请在安装前将 `settings.toml` 中的 `link_mode` 设为 `"copy"`。修改 `link_mode` 只影响后续新安装的 SDK，不会拆开已存在的硬链接。`cjv update` 在更新工具链时始终通过替换整个目录树发布，不会原地改写共享文件。
