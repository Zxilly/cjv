# 代码架构

`cjv` 将命令行解析、安装生命周期编排以及底层文件系统和网络操作拆分为 `internal/` 下的独立模块。代理执行（如调用 `cjc`、`cjpm`）直接基于同一套本地安装状态解析 SDK，不经过 Cobra CLI 命令树。本章说明模块职责划分、命令状态隔离、事务性安装与发布、并发锁机制以及运行时解析流程。

## 入口分流与模块职责

`cmd/cjv/main.go` 根据进程调用名称（`os.Args[0]`）进行分流：

- **SDK 工具名**（`cjc`、`cjpm` 等）：直接进入 `proxy.Run`。
- **`cjv-init` 或 `cjv-setup` 前缀**：进入交互式安装向导。
- **`cjv` 或其他名称**：进入 `cli.Execute`。

| 模块 | 核心职责 |
| --- | --- |
| `cli` | Cobra 命令树组装、参数校验、文本与 JSON 输出渲染及错误展示 |
| `lifecycle` | SDK、交叉目标、组件、自定义安装、历史回退和卸载的高层编排 |
| `toolchain` | 安装身份标识、版本解析、宿主与目标关联关系及本地状态恢复 |
| `resolve` | 活跃工具链准备与项目缺失依赖项的自动补装 |
| `component` | 组件目录根、文件清单、本地 `stdx` 链接及批次编辑 |
| `dist` | 按需加载版本清单、断点续传下载、SHA-256 校验、解包及暂存区生命周期 |
| `target` | 平台四元组转换、SDK 与组件平台名称映射及发布平台清单 |
| `env`、`sdktools` | 环境变量合并、shell 脚本生成（`envsetup`）与 SDK 工具路径查找 |
| `proxy`、`process` | 工具代理转发、子进程启动、退出码传递与系统信号处理 |
| `config` | 设置文件继承、项目工具链查找规则与 `<CJV_HOME>` 目录布局 |
| `reachable`、`selfupdate` | 受管 `cjv` 二进制安装、`<CJV_HOME>/bin` 代理入口、`PATH` 配置与自更新 |
| `fstx`、`fsops` | 持久化文件事务、原子重命名、目录树合并与平台文件重试 |
| `ohos` | 为 OpenHarmony 宿主准备原生 ELF 载荷并执行自签名 |

横切支撑包提供通用基础设施：

- `progress`：定义进度事件模型与终端/JSON 输出适配器。
- `i18n`、`cjverr`、`logging`：分别提供多语言消息、带错误码的结构化错误和分级日志。
- `retry`：文件系统重试与 HTTP 下载重试共用的退避循环。
- `testutil`、`config/configtest`、`cli/clitest`：提供测试用本地分发源、进度记录器、隔离配置与命令组测试夹具（见[测试与检查](testing.md)）。

## 命令状态与输出渲染

每次调用 `cli.Execute` 都会创建独立的 `clictx.Context`（包含输出渲染器、全局 `+toolchain` 选择器和根命令）及全新命令树。各命令组分别位于独立的子包中（`showcmd`、`componentcmd`、`runcmd`、`initcmd`、`toolchaincmd`、`settingscmd`、`selfcmd`），各自在结构体中保存标志参数并注册到根命令。

子命令将请求参数组装后传给业务模块（`lifecycle`、`resolve`、`toolchain`、`config`），业务模块通过 `progress.Sink` 报告进度事件并返回结构化结果，不直接向标准输出打印最终文本：

- **文本模式**：由 `progress.Text` 渲染实时进度，并调用结果对象的 `Text()` 方法输出。
- **JSON 模式**：丢弃进度事件，向 `stdout` 输出单个结构化 JSON 文档或错误对象。
- **代理模式**：将 `stdout` 完全留给被调用的 SDK 工具，自动补装进度统一写入 `stderr`。

每个错误由 CLI 层渲染一次，再由 `main` 转换为进程退出码。修改 CLI 时请保持每次调用的状态隔离，禁止将命令标志、`io.Writer` 或 JSON 模式存回包级全局变量。

## 事务性安装与发布

### 只读查询会话与写入安装会话

`lifecycle.OpenDistribution` 仅读取合并后的设置、平台定义与远程版本清单，不扫描也不恢复本地安装目录；因此即使本地状态存在待恢复事务，`list-remote` 和 `check` 等远程查询仍能正常执行。

写入类操作则打开安装分发会话：在读取本地安装快照之前，先恢复上次中断的持久化事务并完成旧版目录布局迁移。

### 受管 SDK 安装的六个阶段

一次受管通道或版本的 SDK 安装按以下六个步骤执行：

1. `dist.Preparation` 获取全局安装锁（`downloads/.lock`），并创建本次操作独享的私有暂存区。
2. 在 `HomeLock` 锁内读取目标安装身份、已选组件集合及宿主与交叉目标的依赖快照。
3. 释放 `HomeLock`（继续持有安装锁），解析清单版本、下载归档，并在暂存区解包准备宿主 SDK、已跟踪交叉目标和组件。
4. 重新获取 `HomeLock`，校验本地安装快照和依赖关系在锁释放期间是否发生变化。
5. `publication.go` 通过单个持久化事务（`fstx.NewToolchainGroupTransaction`）将 SDK、`stdx` 和离线文档根目录原子发布到位。
6. 更新 `<CJV_HOME>/bin` 代理入口，标记准备阶段完成，并清理本次操作拥有的暂存文件和临时下载。

安装策略标志仅调整版本与组件解析规则，不削弱事务完整性：

- `--force` 允许跳过并移除缺失的可选组件或目标，不会强制重装未变化的 SDK。
- `--no-update` 保留已安装的宿主发行版，仅补齐新增的目标或组件。
- `nightly` 解析会结合所需组件和交叉目标在已发布历史中选择兼容版本；除非显式传入 `--allow-downgrade`，否则不会降级。

### 组件批次、文件复用与自定义归档

受管组件批次操作（`component add`、`component remove`）同样先在锁外暂存准备，再在 `HomeLock` 下复查并通过单个 `fstx` 事务发布组件内容与文件清单，期间保持 SDK 本体目录不动。

硬链接文件复用（`link_mode = "hardlink"`）仅对具有相同发行版、平台和 SHA-256 的普通 SDK 文件生效；安装元数据和组件目录始终保持独立。

URL 和本地归档安装（`resolved_install.go`）先提交 SDK 本体，再处理随包附带的 `stdx`。如果随包 `stdx` 安装失败，cjv 会保留已提交的 SDK 并报告部分成功，这与受管通道整组原子发布的语义不同。

## 崩溃恢复、并发锁与文件所有权

### 持久化日志恢复（`fstx`）

`fstx` 在移动文件前先将事务状态、受管目标路径和备份路径写入磁盘日志。如果进程中途崩溃或被终止，下一次执行写入命令或准备活跃工具链时，会重放日志并按原提交或回滚决策完成恢复。

如果文件系统错误导致恢复受阻，`fstx` 会保留日志和备份目录并返回 `RecoveryError`。任何时候都不得将恢复所需的备份目录当作普通临时残留清除。

### 读写并发控制（`HomeLock` 与 `WithStableReads`）

`HomeLock` 先获取全局写锁，再获取活跃工具链查询门禁锁（gate）。在执行复制历史快照、校验快照或清理历史记录等不修改已安装 SDK 目录的耗时阶段，写进程通过 `WithStableReads` 暂时释放查询门禁锁，同时继续持有全局写锁。

这一机制允许并发的读进程（如运行 `cjc`、`cjv show active`、`cjv which`）在不等待历史快照操作的情况下直接解析并使用现有 SDK：

- 读进程在定位目录期间持有查询门禁锁，并以非阻塞方式探测全局写锁。
- 如果探测到有存活的写进程持有全局写锁且处于稳定读阶段，读进程直接使用当前已安装的根目录。
- 如果没有存活写进程持有全局写锁（例如写进程已崩溃退出），读进程会先执行正常的事务恢复，再解析 SDK。
- 写进程在真正修改已安装目录前，必须重新获取查询门禁锁并原子清除只读阶段标记。
- 读进程绝对不能在持有查询门禁锁的同时阻塞等待全局写锁，从而杜绝锁顺序反转死锁。进程崩溃遗留的阶段标记不会跳过恢复，且在首次旧布局迁移完成前不启用该快速路径。

### 组件文件所有权与路径安全

`component` 在创建快照或修改文件前，会校验受管组件根目录、完整文件清单和父目录符号链接：

- 卸载组件时保留与基础 SDK 共享的文件。
- 受管的叶节点符号链接可以直接移除而不删除其指向的源目录；归档合并拒绝穿过未跟踪的符号链接。
- 通过 `cjv toolchain link` 创建的外部 SDK 目录链接不能作为组件编辑根目录；回退操作同样会校验目标根目录。
- 受管组件批次拥有可跨进程恢复的持久化发布日志；而直接 `component link` 和自定义归档组件编辑使用进程内备份与普通错误回滚（若回滚失败则保留备份）。

所有 `<CJV_HOME>` 子路径统一从 `config/layout.go` 派生；目录树复制、合并和平台重试统一调用 `fsops`。所有归档解包与目录树操作均拒绝任何逃出受管根目录的路径或链接。

## 活跃工具链解析与运行环境执行

1. `toolchain.SelectActive` 按顺序评估显式 CLI 选择器（`+name`、`--toolchain`）、`CJV_TOOLCHAIN` 环境变量、逐层目录覆盖与 `cangjie-sdk.toml` 以及全局 `default_toolchain`。
2. `toolchain.PrepareActive` 执行待定事务恢复并定位本地安装；仅当确认缺少有效的官方 SDK 时才调用自动安装回调，无效名称、损坏记录和文件系统错误均直接返回。
3. `resolve.Active` 根据 `auto_install` 设置（或 `run --install` 标志）自动补齐项目声明的组件与交叉编译目标；`show`、`which` 等只读状态查询绝不触发安装。
4. `toolchain.ReadHostTargets` 统一绑定宿主安装身份、实际发行版和主机平台；`target` 子命令、`component --target`、`resolve.ActiveTarget` 以及代理目标补齐均调用该函数，防止错误关联到其他宿主或不同版本的交叉 SDK。
5. `env.Runtime` 基于显式传入的基础环境变量合并 SDK 路径与组件变量，生成子进程环境或 `envsetup` 脚本。`resolve.ActiveRuntime` 将活跃工具链解析与 `env.Runtime` 组合，`sdktools` 统一解析工具路径。Unix 代理直接通过 `execve` 替换当前进程；Windows 代理以及 `cjv run`、`cjv exec` 则共用 `process.Run` 启动子进程并传递信号与退出码。

## 分发清单、下载重试与自管理

- **按需加载清单（`dist.Source`）**：按需读取并缓存 LTS/STS 的 `versions.json` 与独立的 `nightly.json`，组件数据从同一通道对象解析。cjv 只消费静态清单，上游发现与清单生成由独立的 `cangjie-version-manifest` 仓库负责。
- **下载与暂存生命周期（`dist.Preparation`）**：统一持有安装锁、暂存目录和已下载归档直至发布与清理结束。失败时保留支持断点续传的下载文件；成功时仅清理本次操作拥有的文件，不触碰用户传入的本地归档和恢复日志。
- **重试与续传策略**：归档下载和 nightly `.sha256` sidecar 请求共用可取消的退避重试。HTTP `408`、`429` 和 `5xx` 错误会自动重试，永久性 `4xx` 错误和格式损坏立即失败（sidecar 返回 `404` 表示校验和尚未发布）。带 SHA-256 的下载支持跨命令断点续传并校验完整文件；无哈希下载仅在单次操作内续传。
- **设置与自更新（`config`、`reachable`、`selfupdate`）**：`config.SettingsFile` 记录每个字段是否由用户显式设置，`Update` 仅写入被修改的字段。`reachable.Ensure` 统一安装受管二进制、代理入口与环境脚本并按策略配置 `PATH`。`selfupdate.Check` 仅查询版本，`selfupdate.Update` 校验并替换二进制后返回状态供 CLI 渲染。
