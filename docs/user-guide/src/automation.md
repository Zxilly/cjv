# 自动化与 JSON 接口

`cjv` 针对 CI 流水线、自动化构建脚本与编辑器工具链集成进行了专门设计。本章介绍如何固定并校验项目版本、隔离 CI 配置与离线缓存、运行只读诊断与更新预演、使用更新历史回退，以及消费 `--json` 结构化输出。

## 固定项目版本

在项目目录中运行 `cjv pin lts`，会将解析后的完整发行版名称（例如 `lts-1.0.5`）写入当前目录的 [`cangjie-sdk.toml`](toolchain-file.md)，同时保留文件中原有的 `components`、`targets` 和其他自定义键值（注释和排版会重新生成）。省略名称直接运行 `cjv pin` 时，会固定当前激活的工具链选择；如果本机已安装对应的跟踪通道，`cjv pin` 会优先使用本机已安装的发行版，不会触发下载。

将生成的 `cangjie-sdk.toml` 提交到版本控制，即可让所有项目成员和 CI 选择完全相同的 SDK 版本：

```bash
cjv pin lts
cjv pin --check
```

`cjv pin --check` 只检查当前目录的 `cangjie-sdk.toml` 是否已经固定了具体发行版，不修改文件也不发起网络请求，非常适合作为 CI 的检查步骤。由于版本固定与制品 SHA-256 校验分别由 `cangjie-sdk.toml` 和分发清单承担，项目不需要维护额外的 lockfile。

## 配置与网络

### 隔离 CI 目录与设置文件

在 CI 任务中，通常建议为作业指定独立的数据目录和用户设置文件，避免读取或污染构建机宿主的主目录：

```bash
export CJV_HOME="$PWD/.cjv-data"
export CJV_CONFIG_FILE="$PWD/ci-settings.toml"
cjv install lts
cjv --offline --no-auto-install exec cjc --version
```

`CJV_HOME` 必须为绝对路径。请注意，`--config` 和 `CJV_CONFIG_FILE` 只替换用户设置文件，系统后备配置、项目选择与其他[环境变量](environment-variables.md)仍然生效；如果需要同时隔离或覆盖系统后备配置，请另行设置 `CJV_FALLBACK_SETTINGS`。

### 离线缓存与网络控制标志

要在离线环境中使用 `--offline`（或 `CJV_OFFLINE=1`）执行安装，需要事先在相同分发源配置下下载好对应的版本清单、SDK 归档和组件归档。联网安装成功后，`cjv` 会将已验证 SHA-256 的归档保留在 `<CJV_HOME>/cache/` 中（无校验和的归档不会保留在可复用缓存中）。你可以通过 `cjv cache list --json` 检查缓存内容，并将填充好的 `cache/` 目录与项目工具链文件一起提供给离线环境。

如果缓存目录中存在损坏记录、孤立归档或未完成的临时文件，`cjv cache list --json` 会通过可选的 `problem` 字段标出问题条目，同时正常列出其余有效条目。运行 `cjv cache clean` 可以清理这些由 `cjv` 管理的缓存文件，`cjv cache prune --max-age <duration>` 则按年龄筛选清理。如果联网查询成功但无法写入可选缓存，`cjv` 会向 `stderr` 发出警告，查询结果本身仍可正常使用。

- **`--no-auto-install`** — 禁止代理命令或 `cjv exec` 隐式自动补装缺失项；显式执行的 `cjv install` 仍可正常运行。
- **`--offline`** — 约束 `cjv` 自身的网络请求，但不会限制子进程（通过 `run`、`exec` 或代理命令启动的程序）自身的联网行为。

## 检查与修复

在自动化脚本或故障排查中，可以使用以下命令执行只读诊断、更新预演或重装修复：

```bash
cjv doctor --json
cjv update --dry-run --json
cjv install lts --reinstall
```

- **`cjv doctor`** — 只读检查设置文件、当前选中的 SDK、`<CJV_HOME>/bin` 代理入口、安装记录以及关键 SDK 文件是否存在，并在发现问题时给出具体修复建议和返回非零退出码（`1`）。它不会对整个 SDK 逐文件计算哈希摘要。
- **`cjv update --dry-run`** — 解析兼容版本与更新阻塞原因，不下载归档、不修改安装，也不刷新缓存。
- **`cjv install <name> --reinstall`** — 重新准备并安装同名 SDK 的当前发行版，同时保留已选择的组件和交叉编译目标。与之相对，`--force` 仍表示允许跳过并移除不可用的可选组件或目标，不会代替 `--reinstall` 重装未发生变化的 SDK。

## 更新历史与回退

当跟踪通道（`lts`、`sts`、`nightly`）的宿主 SDK 或既有交叉目标更新到不同的发行版或摘要时，`cjv` 会在 `<CJV_HOME>/history/` 下保存更新前的完整工具链组快照（包含宿主 SDK、跟踪目标、离线文档和 `stdx`）。

每个跟踪通道默认自动保留最近 **2** 份快照。首次安装、同一发行版的 `--reinstall` 修复，以及单独补装目标或组件不会生成快照。固定版本安装（如 `lts-1.0.5`）独立保留，不参与通道更新历史。

```bash
cjv toolchain history lts
cjv toolchain rollback lts
cjv toolchain history prune lts --keep 1
```

- **`cjv toolchain rollback <channel> [snapshot]`** — 省略快照 ID 时默认选择最新一份快照，也可传入 `<snapshot>` ID 指定恢复某份历史记录。回退前会在本地完整校验整组快照内容，并将回退前的当前组保存为快照，以便再次撤销回退。
- **`cjv toolchain history prune [channel] [--keep N]`** — 清理历史快照。由于每份快照都是独立文件副本（已链接组件的目标内容也会复制保存），会占用额外磁盘空间，且 `cjv cache clean` 不会删除历史快照。运行 `history prune --keep 0` 可清空历史并保留当前安装。`--keep` 只控制当次清理操作，不改变后续更新时默认自动保留 2 份快照的规则。历史记录由 `cjv` 内部维护，请勿直接手动修改 `<CJV_HOME>/history/` 中的文件。

### 损坏快照的处理规则

`cjv toolchain history` 在 `snapshots` 数组中保留可读快照记录，并通过可选的 `invalid_snapshots` 数组报告损坏记录及其原因。

当 `history prune` 使用非零 `--keep` 参数时，`cjv` 会保留指定数量的可读快照，并留下损坏记录供排查；传入 `--keep 0` 则会一并清理损坏记录。如果最新的一份快照已损坏，省略快照 ID 的 `cjv toolchain rollback <channel>` 会要求你显式指定一个有效的 `<snapshot>` ID，而不会静默退到更早的版本。任何被选中的快照在恢复前都必须通过完整内容校验。

## JSON 兼容规则

支持 `--json` 的命令会将单个结构化 JSON 文档写入 `stdout`，并将日志与警告写入 `stderr`。`run`、`exec` 和 `init` 不支持 `--json`，因为子进程的标准输出和退出码需要按原样透传。

编写解析 `--json` 输出的客户端时，请遵循以下兼容约定：

- **按命令定义成功字段**：成功对象的字段按各命令独立定义；客户端应忽略未知字段和不依赖的枚举值。
- **带 `schema_version` 的对象**：在同一 `schema_version` 版本内保留现有字段的含义和类型。新增可选字段不改变版本号；不兼容变更必须递增 `schema_version`。
- **无版本号的既有命令**：保留原有输出结构，请勿假设其与其他命令拥有相同的顶层字段。
- **集合与可选字段**：必选集合始终使用数组（空集合为 `[]`）；标为可选的字段可以缺省或为 `null`，消费方应按具体命令兼容处理。
- **机器可读错误处理**：自动化脚本应使用进程退出码和错误 `code` 判断结果，不要匹配可能随语言变化的 `message` 文本。

例如，空缓存时 `cjv cache list --json` 的输出为：

```json
{"schema_version":1,"path":"/home/user/.cjv/cache","entries":[]}
```

下面的 JSON Schema 描述了 `cjv cache list --json` 的稳定字段（允许后续新增可选字段）：

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["schema_version", "path", "entries"],
  "properties": {
    "schema_version": {"const": 1},
    "path": {"type": "string"},
    "entries": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["key", "kind", "url", "size", "updated"],
        "properties": {
          "key": {"type": "string"},
          "kind": {"type": "string", "examples": ["metadata", "artifact"]},
          "url": {"type": "string"},
          "sha256": {"type": "string"},
          "size": {"type": "integer", "minimum": 0},
          "updated": {"type": "string", "format": "date-time"}
        }
      }
    }
  }
}
```

### 错误对象与自动补装失败原因

命令执行失败时，`stdout` 返回 `{"error":{"code":"...","message":"...","details":{}}}`，且进程退出码非零。后续版本可能新增错误码，客户端应保留未知错误码的处理分支；`details` 内的字段取决于具体的错误码。

`cjv doctor --json` 是特殊情况：当发现诊断问题时，它仍返回包含 `checks` 的正常诊断报告对象，将 `ok` 设为 `false` 并以退出码 `1` 结束，不会外包一层 `error` 对象。

当自动补装失败时，顶层错误码仍保持为 `TOOLCHAIN_NOT_INSTALLED` 或 `COMPONENT_NOT_INSTALLED`，而 `details.cause` 保留底层实际失败的 `code`、`message` 和 `details`：

- **`OFFLINE_CACHE_MISS`** — `--offline` 模式下所需的版本清单或归档不在缓存中。
- **`CHECKSUM_MISMATCH`** — 下载或缓存的归档未通过 SHA-256 校验。
- **`HTTP_ERROR`** 与 **`NETWORK_ERROR`** — 分别表示 HTTP 响应状态码异常和底层网络传输错误。
- **`CANCELED`** 或 **`DEADLINE_EXCEEDED`** — 表示操作被信号取消或下载超时。

当关闭了自动补装（`--no-auto-install`、`CJV_AUTO_INSTALL=0` 或 `auto_install = false`）时，缺失安装错误没有 `details.cause`，因为并未尝试执行自动安装。
