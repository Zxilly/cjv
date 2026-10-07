# 配置

用户设置始终保存在 `~/.cjv/settings.toml`。`CJV_HOME` 和 `home` 只改变数据目录，不改变这个文件的位置。

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true

[overrides]
"/home/me/project-a" = "sts"
```

## 设置项

| 字段 | 默认值或作用 | 修改方式 |
| --- | --- | --- |
| `version` | 设置格式版本，由 cjv 维护 | 自动写入与迁移 |
| `default_toolchain` | 默认工具链 | `cjv default <name>`；`none` 清除 |
| `auto_self_update` | `check`：仅提示更新；另有 `enable`、`disable` | `cjv set auto-self-update <value>` |
| `auto_install` | `true`：自动补齐所选工具链的缺失项 | `cjv set auto-install <true\|false>` |
| `home` | 数据目录，默认 `~/.cjv` | `cjv set home <path>` |
| `default_host` | 默认主机平台，默认自动检测 | `cjv set default-host <goos-goarch>` |
| `manifest_url` | 默认 LTS/STS 清单；nightly 使用同目录的 `nightly.json` | 编辑设置文件 |
| `dist_server` | 同时提供 `versions.json` 和 `nightly.json` 的根地址 | 编辑设置文件 |
| `link_mode` | `hardlink` 或 `copy`，默认 `hardlink` | 编辑设置文件 |
| `overrides` | 目录到工具链的映射 | `cjv override` |

`auto_self_update` 只影响未指定名称和全局 `+name` 的全量 `cjv update`。`enable` 自动更新 cjv，`check` 查询并提示，`disable` 跳过。开发构建跳过发行检查；`cjv self update` 可手动执行。

`cjv set home` 将相对路径转为绝对路径。`cjv set home ""` 显式选择默认数据目录。`default_host` 使用 `linux-amd64` 这样的 Go 平台格式，和工具链名称中的 `linux-x64` 格式不同。

## 继承与临时覆盖

用户设置按字段覆盖[系统后备配置](enterprise/index.md)，未设置字段继续继承。`false` 和空字符串也是明确的用户值。`cjv set` 只保存指定字段，不把其他继承值写入用户文件；要恢复某个字段的继承，删除该字段。

数据目录优先级为 `CJV_HOME`、`home`、`~/.cjv`。分发源优先级为 `CJV_DIST_SERVER`、合并设置中的 `dist_server`、`manifest_url`。环境变量不会因运行 `cjv set` 而持久化。

未知设置键会警告。无法解析的设置和高于当前支持范围的格式版本会报错。

## 数据布局

```text
<CJV_HOME>/
  bin/              # cjv 与 SDK 命令入口
  toolchains/<tc>/  # SDK
  stdx/<tc>/        # dynamic/ 与 static/
  docs/<tc>/main/   # 主体文档
  docs/<tc>/stdx/   # 扩展库文档
  downloads/        # 下载归档与可续传部分文件
```

设置文件只有在使用默认数据目录时才与上述目录位于同一处。用 `cjv show home` 查看实际数据目录及来源。

带 SHA-256 的下载失败或中断时会保留可续传文件，下次完成后校验整个归档。成功的操作清理自己使用的下载；全量更新全部成功时还会清理残留下载。

## SDK 文件复用

默认 `link_mode = "hardlink"` 对相同发行版、平台和 SHA-256 的 SDK 尝试复用内容相同的普通文件。元数据和组件保持独立，硬链接不可用时保留复制文件。

硬链接共享文件内容，手工原地编辑某份 SDK 可能影响其他安装。需要独立副本时设置 `link_mode = "copy"`；它只影响后续安装，不拆开已有硬链接。cjv 的更新通过替换目录发布，不原地改写共享文件。
