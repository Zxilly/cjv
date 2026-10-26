# 配置

`cjv` 的用户设置默认保存在 `~/.cjv/settings.toml`（Windows 上为 `%USERPROFILE%\.cjv\settings.toml`）。你可以通过命令行选项 `--config <path>` 或环境变量 `CJV_CONFIG_FILE` 为当前进程指定另一个设置文件，其中命令行选项优先。请注意，`CJV_HOME` 和设置中的 `home` 字段只改变数据目录（`toolchains/`、`cache/` 等）的位置，不会移动 `settings.toml` 本身。

典型的 `settings.toml` 文件如下：

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true

[overrides]
"/home/me/project-a" = "sts"
```

## 设置项

大多数日常设置可以通过 `cjv` 子命令（`cjv default`、`cjv set`、`cjv override`）查询和修改，高级分发源选项则可以直接编辑 `settings.toml`：

| 字段 | 默认值或作用 | 修改方式 |
| --- | --- | --- |
| `version` | 设置格式版本号，由 `cjv` 维护 | 自动写入与迁移 |
| `default_toolchain` | 全局默认工具链 | `cjv default <name>`；传入 `none` 可清除 |
| `auto_self_update` | `check`（仅提示更新）；另支持 `enable`、`disable` | `cjv set auto-self-update <value>` |
| `auto_install` | `true`（自动补齐所选工具链的缺失项） | `cjv set auto-install <true\|false>` |
| `home` | 数据目录路径，默认为 `~/.cjv` | `cjv set home <path>` |
| `default_host` | 默认主机平台，默认自动检测 | `cjv set default-host <goos-goarch>` |
| `manifest_url` | 默认 LTS/STS 清单地址；`nightly` 使用同目录的 `nightly.json` | 编辑 `settings.toml` |
| `dist_server` | 同时提供 `versions.json` 和 `nightly.json` 的根地址 | 编辑 `settings.toml` |
| `link_mode` | `hardlink` 或 `copy`，默认 `hardlink` | 编辑 `settings.toml` |
| `overrides` | 目录路径到工具链名称的映射表 | `cjv override set` / `cjv override unset` |

## 自动更新

`auto_self_update` 仅在执行未指定工具链名称且未带全局 `+name` 的全量 `cjv update` 时生效：

- **`check`** *(默认)* — 查询是否有新版本 `cjv` 并打印提示，但不自动替换。
- **`enable`** — 在更新工具链的同时自动下载并更新 `cjv` 本体。
- **`disable`** — 在 `cjv update` 时跳过 `cjv` 本体的更新检查。

本地开发构建（版本号为 `dev`）会自动跳过发行检查。无论该项如何配置，你都可以随时运行 `cjv self update` 手动更新 `cjv` 本体，或者通过 `cjv update --no-self-update` 在单次更新中跳过检查。

## 数据目录与主机平台

运行 `cjv set home <path>` 时，`cjv` 会先将相对路径转换为绝对路径再写入设置文件。运行 `cjv set home ""` 可以显式恢复默认数据目录 `~/.cjv`。

`default_host` 使用 `linux-amd64`、`darwin-arm64` 这样的 Go 平台命名格式，这与[工具链名称](concepts/toolchains.md#主机平台)中的 `linux-x64` 写法不同。

## 继承与临时覆盖

在配置了系统级设置的机器上，用户设置文件（`~/.cjv/settings.toml`）会按字段逐项覆盖[系统后备配置](enterprise/index.md#系统后备配置)（Unix 为 `/etc/cjv/settings.toml`，Windows 为 `C:\ProgramData\cjv\settings.toml`）。用户文件中未设置的字段会继续继承系统后备配置；而显式写入的 `false` 和空字符串 `""` 都属于明确的用户值，会覆盖后备配置。

运行 `cjv set` 只会将本次指定的字段写入用户文件，不会把其他继承自系统后备配置的值一并写入。要恢复某个字段对系统后备配置的继承，只需从用户 `settings.toml` 中删除该字段即可。

各路径、分发源与临时覆盖的优先级规则如下：

- **数据目录优先级**：`CJV_HOME` 环境变量 > 合并设置中的 `home` > `~/.cjv`。
- **分发源优先级**：`CJV_DIST_SERVER` 环境变量 > 合并设置中的 `dist_server` > `manifest_url`。环境变量属于临时覆盖，不会因运行 `cjv set` 而被持久化到文件中。
- **自定义设置文件（`--config`）**：`--config <path>`（或 `CJV_CONFIG_FILE`）只替换用户设置文件；系统后备配置、[项目工具链文件](toolchain-file.md)和[环境变量](environment-variables.md)仍然正常生效。
- **单次自动补装覆盖**：`--no-auto-install` 仅在本次命令调用中关闭自动补装；`CJV_AUTO_INSTALL=0` 或 `1` 临时覆盖 `auto_install`。这些选项不会写入设置文件，并会传递给通过 `cjv` 启动的子进程。

设置文件中的未知键会打印警告；无法解析的语法错误和高于当前支持范围的格式版本号（`version`）会报错退出。

## 数据布局

`cjv` 将所有已安装的工具链、组件、缓存和历史快照统一组织在 `<CJV_HOME>` 下：

```text
<CJV_HOME>/
  bin/              # cjv 与 SDK 代理命令入口（cjc、cjpm 等）
  toolchains/<tc>/  # 已安装的宿主 SDK 与交叉编译目标 SDK
  stdx/<tc>/        # dynamic/ 与 static/ 扩展库
  docs/<tc>/main/   # 主体离线文档
  docs/<tc>/stdx/   # 扩展库（stdx）离线文档
  downloads/        # 安装暂存与可断点续传的部分下载文件
  cache/            # 缓存的版本清单与已验证的归档
  history/<tc>/     # 跟踪通道更新前保存的完整组快照
```

设置文件只有在使用默认数据目录（`~/.cjv`）时才与上述目录位于同一处。运行 `cjv show home` 可以随时查看当前实际生效的数据目录及其配置来源。

### 下载缓存

带有 SHA-256 校验和的下载如果因网络故障失败或中断，会在 `downloads/` 中保留可续传的部分文件，下次继续下载完成后校验整个归档。成功的操作会自动清理本次使用的暂存下载；当不带参数的全量 `cjv update` 全部成功时，还会顺带清理 `downloads/` 中残留的历史下载文件。

与此同时，`cache/` 目录独立保留版本清单和带 SHA-256 的已验证归档，在成功安装后仍可复用。归档按来源 URL 和 SHA-256 摘要索引，每次复用前都会重新校验摘要（没有校验和的归档不会进入可复用缓存）。

传入 `--offline` 或设置 `CJV_OFFLINE=1` 会禁止 `cjv` 发起任何网络请求，仅使用本地安装和 `cache/` 缓存。缓存的清单以下载时的内容为准，不保证包含上游最新发布的版本；如果所需的清单或归档不在缓存中，离线命令会直接报错。

可以使用 `cjv cache` 子命令管理可复用缓存：

- `cjv cache dir` — 查看缓存目录路径。
- `cjv cache list` — 查看当前缓存的清单与归档列表。
- `cjv cache prune --max-age 720h` — 删除超过指定时长的缓存条目。
- `cjv cache clean` — 清空全部可复用缓存。

清理缓存绝不会删除任何已安装的 SDK。更多 CI 缓存用法见[自动化与 JSON 接口](automation.md#配置与网络)。

## SDK 文件复用

默认的 `link_mode = "hardlink"` 会对具有相同发行版、平台和 SHA-256 摘要的 SDK 安装尝试通过硬链接复用内容相同的普通文件。安装元数据和组件目录保持独立；如果底层文件系统不支持硬链接，会自动回退并保留复制文件。

由于硬链接共享同一份底层文件内容，手动原地编辑 `<CJV_HOME>/toolchains/` 中的某份 SDK 文件可能会同时影响其他共享该文件的安装。如果需要完全独立的副本以便原地修改文件，请在 `settings.toml` 中设置 `link_mode = "copy"`（它只影响后续新安装，不会拆开已有的硬链接）。`cjv update` 更新工具链时始终通过替换整个目录树发布，不会原地改写共享文件。
