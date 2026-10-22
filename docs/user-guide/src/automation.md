# 自动化与 JSON 接口

## 固定项目版本

用 `cjv pin lts` 将解析后的完整发行名写入当前目录的 `cangjie-sdk.toml`。已有文件中的组件、目标和其他键值会保留，注释和排版会重新生成。省略名称时使用当前选择；已安装的跟踪通道优先使用本机发行版。提交这个文件即可让项目成员选择相同 SDK 版本。

```bash
cjv pin lts
cjv pin --check
```

`--check` 只检查当前目录的项目文件是否固定了具体发行版，不写文件。版本固定与制品校验分别由项目工具链文件和分发清单承担，不需要额外的 lockfile。

## 配置与网络

为 CI 选择专用设置文件和数据目录：

```bash
export CJV_HOME="$PWD/.cjv-data"
export CJV_CONFIG_FILE="$PWD/ci-settings.toml"
cjv install lts
cjv --offline --no-auto-install exec cjc --version
```

`CJV_HOME` 必须为绝对路径。`--config` 和 `CJV_CONFIG_FILE` 只替换用户设置文件，系统后备配置、项目选择与其他环境变量仍生效。需要控制后备配置时，另行设置 `CJV_FALLBACK_SETTINGS`。

离线安装需要事先下载对应来源的清单、SDK 和组件。成功安装会保留已验证的归档；可通过 `cjv cache list --json` 检查缓存。将安装时使用的 `cache/` 与项目工具链文件一起提供给离线环境，并保持分发源配置一致。无校验和的归档不会被保留为可复用缓存。

`--no-auto-install` 禁止隐式补装；显式 `install` 仍可执行。`--offline` 约束 cjv 自身的请求，不限制子进程的联网行为。

## 检查与修复

```bash
cjv doctor --json
cjv update --dry-run --json
cjv install lts --reinstall
```

`doctor` 只读检查配置、当前 SDK、命令入口、安装记录和关键文件是否存在，并给出修复建议；它不对整个 SDK 逐文件校验摘要。发现问题时返回非零退出码。`update --dry-run` 解析兼容版本和阻塞原因，不下载归档、不安装，也不刷新缓存。

`--reinstall` 重新准备同名 SDK 并保留已选组件和目标。`--force` 仍表示允许移除不可用的可选组件或目标，不代替重新安装。

## 更新历史与回退

跟踪通道的宿主或既有目标更新到不同发行版或摘要时，cjv 保存更新前的完整工具链组：宿主 SDK、跟踪目标、文档和 stdx。每组默认保留最近两份快照。首次安装、同发行版修复和单独补目标或组件不会生成快照。

```bash
cjv toolchain history lts
cjv toolchain rollback lts
cjv toolchain history prune lts --keep 1
```

`rollback` 省略快照 ID 时选择最新一份，也可用 `cjv toolchain rollback lts <snapshot>` 指定历史记录。回退在本地校验整组内容后一起恢复，并保存回退前的当前组，以便再次撤销。固定版本安装独立保留，不参与通道历史。

快照是独立副本，已链接的组件内容也会复制，因此会占用额外磁盘空间。缓存清理不会删除快照；`history prune --keep 0` 可删除历史，保留当前安装。`--keep` 只控制本次清理，不改变更新时自动保留两份的规则。历史记录位于 `<CJV_HOME>/history/`，不要直接修改其中的文件。

## JSON 兼容规则

支持 JSON 的命令将一个 JSON 文档写入 stdout，日志写入 stderr。`run`、`exec` 和 `init` 不支持 `--json`；子进程的标准输出和退出码按原样传递。

- 成功对象按命令定义字段；客户端应忽略未知字段和不依赖的枚举值。
- 带 `schema_version` 的对象在同一版本内保留现有字段的含义和类型。增加可选字段不改变版本；不兼容变更必须提高版本。
- 既有未带版本号的命令保留原输出结构，不应假设其与其他命令拥有相同字段。
- 必选集合使用数组，空集合为 `[]`；标为可选的字段可以缺省。既有接口的 `null` 和缺省行为保持兼容，消费方应按具体命令处理。
- 自动化使用错误 `code` 和退出码判断结果，不匹配可能随语言变化的 `message`。

例如空缓存的输出为：

```json
{"schema_version":1,"path":"/home/user/.cjv/cache","entries":[]}
```

下面的 JSON Schema 描述 `cache list --json` 的稳定字段；其他字段允许后续添加：

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

执行失败时 stdout 返回 `{"error":{"code":"…","message":"…","details":{}}}`，进程退出码非零。错误码可以新增，客户端应保留未知错误的处理分支；`details` 的字段取决于错误码。普通成功命令退出码为零。`doctor` 发现诊断问题时仍返回包含 `checks` 的诊断报告，设置 `ok: false` 并以退出码 1 结束，不另附错误对象。
