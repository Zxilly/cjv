# 配置

cjv 把持久化设置保存在 `~/.cjv/settings.toml` 这个 TOML 文件里。日常使用中你不需要手动编辑它，用 `cjv set` 子命令修改即可，它们会校验取值、写回文件，并打印确认信息。

本章介绍 `cjv set` 的全部子命令、`settings.toml` 中对应的字段，以及 `~/.cjv/` 目录的整体布局。运行时通过环境变量做的临时覆盖（如 `CJV_HOME`、`CJV_DIST_SERVER`）见 [环境变量](environment-variables.md)。

## settings.toml

设置文件始终位于 `<用户主目录>/.cjv/settings.toml`，不会随 `CJV_HOME` 改变。

一个典型的 `settings.toml` 大致长这样：

```toml
version = 1
default_toolchain = "lts"
auto_self_update = "check"
auto_install = true
manifest_url = "https://raw.githubusercontent.com/Zxilly/cangjie-version-manifest/master/versions.json"

[overrides]
"/home/me/project-a" = "sts"
```

字段速览：

| 字段                | 类型   | 对应命令                   | 说明                                                                  |
| ------------------- | ------ | -------------------------- | --------------------------------------------------------------------- |
| `version`           | int    | （自动维护）               | 设置文件格式版本，由 cjv 自动写入和迁移                                |
| `default_toolchain` | string | `cjv default <toolchain>`  | 默认工具链，见 [工具链](concepts/toolchains.md)                       |
| `manifest_url`      | string | 手动编辑 / 系统后备配置     | LTS/STS 清单地址；nightly 从同目录的 `nightly.json` 按需读取            |
| `dist_server`       | string | 手动编辑 / 系统后备配置     | 工具链分发根；包含 `versions.json` 与 `nightly.json`，见[内部分发源](enterprise/distribution-server.md) |
| `auto_self_update`  | string | `cjv set auto-self-update` | `cjv update` 时的自更新行为：`enable` / `disable` / `check`           |
| `auto_install`      | bool   | `cjv set auto-install`     | 代理模式下是否自动安装缺失的工具链                                    |
| `home`              | string | `cjv set home`             | 持久化的 `CJV_HOME` 数据目录路径                                     |
| `default_host`      | string | `cjv set default-host`     | 默认主机平台标识（`goos-goarch` 形式）                                |
| `overrides`         | table  | `cjv override`             | 目录到工具链的覆盖映射，见 [目标与覆盖](concepts/targets-overrides.md) |

> 文件中无法识别的键（例如拼写错误）会以 warn 级别日志提示，但不会阻止 cjv 启动。把 `version` 设成超过当前二进制支持的值则会报错。

`manifest_url` 与 `dist_server` 通过用户设置文件或系统后备配置提供。来源优先级为 `CJV_DIST_SERVER`、`dist_server`、`manifest_url`。根地址对应 `<root>/versions.json` 与 `<root>/nightly.json`；直接配置 `manifest_url` 时，nightly 文件位于其同一目录。完整契约见[内部分发源](enterprise/distribution-server.md)。

## cjv set

`cjv set` 修改 `settings.toml` 中的单项设置，只保存你明确选择的字段，其他字段继续继承系统后备配置或内置默认值。当用户文件中的选择发生变化时，命令写盘并打印 `设置 '<key>' 已更新为 '<value>'`。即使新值等于当前继承的值，第一次显式设置也会将它固定；重复设置已经保存的相同值不会再次写盘。

`false` 和空字符串都是明确的用户值，不会被系统后备值覆盖。例如，`cjv set auto-install false` 会保持关闭，`cjv default none` 会取消默认工具链。要恢复某个字段的继承关系，从用户设置文件中删除该字段即可。环境变量提供的临时覆盖不会随这些修改写入文件。

### cjv set auto-self-update

控制 `cjv update` 在更新工具链之后是否顺带自更新 cjv 本体。

```bash
# 自动下载并安装 cjv 的新版本
cjv set auto-self-update enable

# 完全关闭自更新（连提示都不打印）
cjv set auto-self-update disable

# 默认：仅在有新版本时提示，不自动安装
cjv set auto-self-update check
```

`enable` 会在 `cjv update` 完成后自动升级 cjv；`disable` 会跳过自更新；`check`（默认）只打印当前 cjv 版本，不自动升级。

无论此设置如何，你都可以随时用 `cjv self update` 手动升级 cjv。

### cjv set auto-install

控制[代理模式](concepts/proxies.md)下，当解析出的工具链尚未安装时是否自动安装它。默认开启（`true`）。

```bash
# 默认：直接调用 cjc / cjpm 时，缺失的工具链会被自动安装
cjv set auto-install true

# 关闭：缺失工具链时报错，而不是自动安装
cjv set auto-install false
```

开启时，直接运行 `cjc`、`cjpm` 等 SDK 工具，如果当前解析到的工具链没装，cjv 会先把它装上再代理执行。`cangjie-sdk.toml` 中声明的 [组件](concepts/components.md) 与 [目标](cross-compilation.md) 同样适用：开启 `auto-install` 后，代理执行会按需补齐缺失的组件和目标 SDK。

### cjv set home

把 `CJV_HOME` 数据目录路径持久化到 `settings.toml`。传入的相对路径会被转成绝对路径后存储。

```bash
# 把数据目录持久化到指定位置
cjv set home /opt/cjv-data

# 传空字符串可固定使用默认 ~/.cjv，不继承系统配置中的 home
cjv set home ""
```

`CJV_HOME` 环境变量始终优先于此设置。即使在 `settings.toml` 里持久化了 `home`，只要 shell 中设置了 `CJV_HOME` 环境变量，后者仍然生效。`settings.toml` 文件本身不受影响，永远留在 `~/.cjv/`。

### cjv set default-host

设置默认的主机平台标识（`goos-goarch` 形式，如 `linux-amd64`）。一般无需手动设置，cjv 会自动探测当前主机平台；仅在自动探测不符合预期、需要显式指定时才用到。

```bash
cjv set default-host linux-amd64
```

取值必须是 cjv 能识别的合法平台标识，否则命令会报错。

## ~/.cjv 目录结构

cjv 的全部数据都放在 `CJV_HOME`（默认 `~/.cjv`）下，各子目录职责相互解耦：

```text
~/.cjv/
  bin/            # cjv 与 SDK 工具命令入口
  toolchains/     # 已安装的 SDK 工具链（仅 SDK 本体）
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

`bin/` 存放 cjv 与 `cjc`、`cjpm` 等 SDK 工具的命令入口。把这个目录加入 `PATH` 后即可直接调用这些命令，详见[代理](concepts/proxies.md)。

`toolchains/<tc>/` 是每个已安装工具链的 SDK 本体。

`stdx/<tc>/` 按工具链拆分存放 `stdx` 组件，分为 `dynamic/` 与 `static/`。代理或运行时环境中会自动注入 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC` 指向这两个目录，详见 [组件](concepts/components.md)。

`docs/<tc>/` 是离线文档，与工具链目录解耦。`main/` 放 `docs` 组件（dev-guide、libs/std、tools），`stdx/` 放 `stdx-docs` 组件。用 `cjv doc` 在浏览器中打开。

`downloads/` 是下载暂存区，安装成功后即清空，只在安装中断时保留以便恢复。`settings.toml` 是本章描述的用户设置文件。

> `cjv toolchain uninstall <tc>` 会连带清理 `stdx/<tc>/` 与 `docs/<tc>/`，不会留下孤立的组件数据。

如果设置或持久化了自定义 `CJV_HOME`，上述 `bin/`、`toolchains/`、`stdx/`、`docs/`、`downloads/` 都会落在新路径下；唯独 `settings.toml` 始终留在用户主目录的 `~/.cjv/`（见上文 [settings.toml](#settingstoml)）。
