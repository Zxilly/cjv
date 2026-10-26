# 自动化与 JSON 接口

本章介绍如何在 CI 流水线、自动化脚本和离线构建环境中使用 cjv，涵盖项目版本固定、CI 目录隔离、离线缓存准备、环境诊断、更新历史回退以及 JSON 输出兼容约定。

## 固定项目版本

在团队项目或 CI 仓库中，运行 `cjv pin` 可将解析后的完整 SDK 发行版名称写入当前目录的 [cangjie-sdk.toml](toolchain-file.md)：

```bash
cjv pin lts
cjv pin --check
```

- `cjv pin lts`：将 `lts` 解析为完整发行版名称并写入 `cangjie-sdk.toml`，保留原有的 `components`、`targets` 和其他自定义键值（注释和排版会重新生成）。
- `cjv pin`（省略名称）：使用当前激活的工具链选择；如果本机已安装对应的跟踪通道，优先采用本机已安装的发行版，不会触发下载。
- `cjv pin --check`：只检查当前目录的 `cangjie-sdk.toml` 是否已经固定为具体发行版，不修改文件也不访问网络。

请将生成的 `cangjie-sdk.toml` 提交到版本控制。由于 `cangjie-sdk.toml` 锁定了精确版本号，而分发清单提供了制品的 SHA-256 校验和，项目不需要额维护单独的 lockfile。

## CI 隔离与离线缓存

### 隔离 CI 设置与数据目录

在 CI 任务中，建议指定专用的数据目录和用户设置文件，避免读取或污染构建机宿主的主目录：

```bash
export CJV_HOME="$PWD/.cjv-data"
export CJV_CONFIG_FILE="$PWD/ci-settings.toml"
cjv install lts
cjv --offline --no-auto-install exec cjc --version
```

`CJV_HOME` 必须为绝对路径。`--config` 和 `CJV_CONFIG_FILE` 只替换用户设置文件，系统后备配置、项目选择与其他[环境变量](environment-variables.md)仍然生效；如果需要同时隔离系统后备配置，请另行设置 `CJV_FALLBACK_SETTINGS`。

### 准备与检查离线缓存

要在离线环境中使用 `--offline` 执行安装，必须提前在相同分发源配置下下载好版本清单、SDK 归档和组件归档。联网安装成功后，cjv 会将带 SHA-256 的已校验归档保存在 `<CJV_HOME>/cache/` 中（无校验和的归档不会保留在可复用缓存中）。

将准备好的 `cache/` 目录与 `cangjie-sdk.toml` 一起提供给离线环境，并保持分发源配置一致即可。你可以运行 `cjv cache list --json` 检查缓存内容。

当缓存中存在损坏记录、孤立归档或未完成的临时文件时，`cjv cache list --json` 会通过可选的 `problem` 字段标出问题条目，同时正常列出其他有效条目。运行 `cjv cache clean` 可以清理这些由 cjv 管理的缓存文件，`cjv cache prune --max-age <duration>` 则按条目年龄清理。如果联网查询成功但写入可选缓存失败，cjv 会打印警告并继续返回查询结果。

### 控制联网与隐式安装行为

- `--no-auto-install`：禁止代理命令或 `cjv exec` 隐式自动补装缺失项；显式 `cjv install` 仍可正常执行。
- `--offline`：禁止 cjv 自身发起网络请求，但不会限制 `run`、`exec` 或代理命令启动的子进程自身的联网行为。

## 诊断、预演与修复

在自动化脚本中使用以下命令检查环境健康度、预演更新或修复安装：

```bash
cjv doctor --json
cjv update --dry-run --json
cjv install lts --reinstall
```

- `cjv doctor`：只读检查设置文件、当前选中的 SDK、代理命令入口、安装元数据和关键文件是否存在，并给出修复建议（不会对整个 SDK 逐文件计算哈希）。发现问题时返回非零退出码。
- `cjv update --dry-run`：解析兼容的目标版本与更新阻塞原因，不下载归档、不修改安装，也不刷新缓存。
- `cjv install <name> --reinstall`：重新准备并安装同一发行版的 SDK，同时保留已选择的组件和交叉编译目标。与之不同，`--force` 表示允许跳过并移除未发布的可选组件或目标，不会重装未发生变化的 SDK。

## 更新历史与回退

### 快照保存机制

当跟踪通道的宿主 SDK 或已安装交叉目标更新到不同的发行版或摘要时，cjv 会在 `<CJV_HOME>/history/` 下保存更新前的完整工具链组快照（包含宿主 SDK、已跟踪交叉目标、离线文档和 `stdx`）。

每个跟踪通道默认保留最近 **2** 份快照。首次安装、同一发行版的 `--reinstall` 修复，以及单独执行 `target add` 或 `component add` 都不会生成快照。固定版本（如 `lts-1.0.5`）独立管理，不生成通道更新历史。

> **注意：** 每份历史快照都是独立文件副本（已链接组件的目标内容也会复制保存），因此会占用额外磁盘空间。请勿直接手动修改 `<CJV_HOME>/history/` 中的任何文件。

### 执行回退与清理历史

使用以下命令查看历史快照、回退更新或清理旧快照：

```bash
cjv toolchain history lts
cjv toolchain rollback lts
cjv toolchain history prune lts --keep 1
```

- `cjv toolchain rollback lts`：省略快照 ID 时默认恢复最新一份快照；也可传入 `<snapshot>` ID 指定恢复某份历史快照。回退前会在本地完整校验快照内容，并将当前安装组保存为新快照，以便随时再次撤销回退。
- `cjv toolchain history prune lts --keep 1`：保留指定数量的可读快照并清理更早的历史；传入 `--keep 0` 可清空全部历史快照并保留当前安装。`--keep` 仅控制当次清理操作，后续更新仍默认自动保留 2 份快照。注意 `cjv cache clean` 不会删除历史快照。

### 损坏快照的处理规则

`cjv toolchain history` 在 `snapshots` 数组中列出可读快照，并通过可选的 `invalid_snapshots` 数组报告损坏记录及原因。

当 `history prune` 的 `--keep` 参数为非零值时，cjv 会保留指定数量的可读快照，并留下损坏记录供排查；传入 `--keep 0` 则会一并清理损坏记录。如果最新的一份快照已损坏，不带快照 ID 的 `cjv toolchain rollback <channel>` 会拒绝自动回退到更早版本，并要求你显式指定一个有效的 `<snapshot>` ID。

## JSON 输出兼容约定

### 输出流与兼容性保证

支持 `--json` 的命令会将单个 JSON 文档写入 `stdout`，并将日志与警告写入 `stderr`。`run`、`exec` 和 `init` 不支持 `--json`，因为它们需要原样透传子进程的标准输出和退出码。

编写解析 `--json` 输出的自动化脚本时，请遵循以下规则：

- **按命令定义字段**：各命令定义各自的成功返回字段；客户端应忽略未知字段和未使用的枚举值。
- **带 `schema_version` 的对象**：在同一 `schema_version` 版本号内保留现有字段的名称、类型和语义。新增可选字段不会改变 `schema_version`；只有不兼容变更才会递增 `schema_version`。
- **无版本号的既有命令**：保留原有输出结构，请勿假设不同命令之间共享相同的顶层字段。
- **集合与可选字段**：必选集合始终输出为 JSON 数组（空集合为 `[]`）；可选字段可能缺省或为 `null`，消费方应按具体命令兼容处理。
- **错误判断依据**：请始终根据进程退出码和结构化的 `error.code` 判断结果，不要做基于本地化文本 `error.message` 的字符串匹配。

### 示例：`cache list --json` 的 Schema

空缓存时的输出示例：

```json
{"schema_version":1,"path":"/home/user/.cjv/cache","entries":[]}
```

以下 JSON Schema 描述了 `cjv cache list --json` 的稳定字段（允许后续新增可选字段）：

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

### 错误对象与诊断退出码

命令执行失败时，`stdout` 返回 `{"error":{"code":"...","message":"...","details":{}}}`，且进程退出码非零。后续版本可能新增错误码，`details` 内的字段取决于具体的 `error.code`。

`cjv doctor --json` 是特殊情况：当发现环境问题时，它仍返回包含 `checks` 的正常诊断报告对象，将 `ok` 设为 `false` 并以退出码 `1` 结束，不会外包一层 `error` 对象。

当运行代理命令或 `exec` 触发自动补装并失败时，顶层错误码保持为 `TOOLCHAIN_NOT_INSTALLED` 或 `COMPONENT_NOT_INSTALLED`，而 `details.cause` 保留底层实际失败的 `code`、`message` 和 `details`：

| `details.cause.code` | 含义 |
| --- | --- |
| `OFFLINE_CACHE_MISS` | `--offline` 模式下所需的版本清单或归档不在 `<CJV_HOME>/cache/` 中 |
| `CHECKSUM_MISMATCH` | 下载或缓存的归档未通过 SHA-256 校验 |
| `HTTP_ERROR` | 服务器返回了异常的 HTTP 状态码 |
| `NETWORK_ERROR` | 在收到完整 HTTP 响应前发生连接或传输错误 |
| `CANCELED` | 操作被系统信号或调用方上下文取消 |
| `DEADLINE_EXCEEDED` | 下载或请求超过设定超时时间 |

如果关闭了自动补装，错误对象中不会包含 `details.cause`，表示并未尝试自动安装。完整命令语法见[命令参考](command-reference.md)。
