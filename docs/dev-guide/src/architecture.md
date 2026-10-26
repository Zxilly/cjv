# 代码架构

从整体上看，`cli` 负责解析命令行参数与渲染输出，`lifecycle` 负责编排工具链和组件的安装与更新，底层模块则分别管理分发清单、文件所有权和持久化事务。代理执行（如调用 `cjc`、`cjpm`）直接从同一套本地安装状态解析 SDK，绕过 Cobra CLI 命令树。

## 入口与模块职责

`cmd/cjv/main.go` 按可执行文件调用名称（`os.Args[0]`）进行分流：

- **SDK 工具名**（`cjc`、`cjpm` 等）直接进入 `proxy.Run`。
- **`cjv-init` / `cjv-setup` 前缀**进入交互式安装向导。
- **其余调用**进入 `cli.Execute`。

| 模块 | 职责 |
| --- | --- |
| `cli` | Cobra 命令树、参数校验、结果渲染与错误输出 |
| `lifecycle` | SDK、交叉目标、组件、自定义安装、历史回退和卸载的编排 |
| `toolchain` | 安装身份、版本解析、宿主与目标关联关系及本地恢复 |
| `resolve` | 活跃工具链准备与项目缺失项自动安装 |
| `component` | 组件目录、文件清单、本地 `stdx` 链接及批次编辑 |
| `dist` | 按需加载清单、断点续传下载、SHA-256 校验、解包及准备生命周期 |
| `target` | 平台 tuple、SDK/组件平台映射及发布平台清单 |
| `env`、`sdktools` | 环境合并、shell 脚本输出（`envsetup`）和 SDK 工具布局 |
| `proxy`、`process` | 工具代理转发、子进程执行、退出码和信号处理 |
| `config` | 设置来源与继承、项目选择规则和 `<CJV_HOME>` 数据目录布局 |
| `reachable`、`selfupdate` | 受管二进制、`<CJV_HOME>/bin` 代理入口、`PATH` 配置及 `cjv` 自更新 |
| `fstx`、`fsops` | 持久化事务、原子文件操作、目录树合并和平台重试 |
| `ohos` | 为 OpenHarmony 宿主准备原生 ELF 载荷并自签名 |

横切基础模块为全仓库提供共用能力：

- `progress` 定义进度事件类型和输出适配器。
- `i18n`、`cjverr`、`logging` 分别提供多语言消息、带错误码的结构化错误和分级日志。
- `retry` 是文件系统重试与 HTTP 下载重试策略共用的退避循环。
- `testutil` 提供测试分发源、进度记录器及平台测试辅助；`config/configtest` 用于隔离设置文件，`cli/clitest` 用于在单元测试中运行命令组。

## 命令状态与输出

每次调用 `cli.Execute` 都会创建独立的 `clictx.Context`（持有输出渲染器、全局 `+toolchain` 选择器和根命令）以及一棵全新的命令树。各命令组位于各自的子包中（`showcmd`、`componentcmd`、`runcmd`、`initcmd`、`toolchaincmd`、`settingscmd`、`selfcmd`），各自在结构体中保存自己的标志参数并注册到根命令。子命令将请求交给业务模块，业务模块通过 `progress.Sink` 报告进度事件并返回结果结构，而不直接打印最终文本：

- **文本模式**由 `progress.Text` 和结果对象的 `Text()` 方法渲染。
- **JSON 模式**丢弃进度事件，只向 `stdout` 输出单个结构化 JSON 文档或错误对象。
- **代理模式**将 `stdout` 完全留给被调用的 SDK 工具，自动补装进度统一写入 `stderr`。

错误由 CLI 层统一渲染一次，`main` 负责将其转换为进程退出码。修改 CLI 时请始终保持单次调用的状态隔离，不要把标志、`io.Writer` 或 JSON 模式存回包级全局变量。

## 安装与发布

`lifecycle.OpenDistribution` 仅读取设置、平台定义与远程版本清单，不恢复也不扫描本地安装目录；因此 `toolchain list-remote`、`check` 等远程查询不依赖本地恢复成功。写入操作则打开安装分发会话，在读取本地安装快照前先恢复中断事务并迁移旧版目录布局。

一次受管 SDK 安装按以下六个阶段执行：

1. `dist.Preparation` 获取全局安装锁（`downloads/.lock`），并创建本次操作的私有暂存区。
2. 在 `HomeLock` 锁内读取目标安装身份、已选组件及宿主依赖快照。
3. 释放 `HomeLock`（继续持有安装锁），解析清单版本、下载归档，并在暂存区准备宿主 SDK、已跟踪交叉目标和组件。
4. 重新获取 `HomeLock`，检查本地快照和依赖关系在锁释放期间是否发生变化。
5. `publication.go` 通过 `fstx.NewToolchainGroupTransaction` 在单个持久化事务中一起发布 SDK、`stdx` 和离线文档根目录。
6. 完成 `<CJV_HOME>/bin` 命令入口配置等收尾工作，标记准备阶段成功，并清理本次操作拥有的下载和暂存文件。

安装策略标志仅调整版本与组件解析规则：`--force` 允许跳过并移除缺失的可选组件或目标，不会强制重装未变化的 SDK；`--no-update` 保留已安装的宿主发行版；`nightly` 会根据所需的组件和目标集合在已发布历史中选择兼容版本，降级需显式传入 `--allow-downgrade`。

受管组件批次（`component add`、`component remove`）同样先在锁外暂存准备、再在 `HomeLock` 下复查，并通过单个事务发布组件内容和文件清单，期间 SDK 本体留在原处。固定版本与跟踪通道彼此独立；硬链接去重（`link_mode = "hardlink"`）只用于满足来源及 SHA-256 内容条件的普通 SDK 文件，元数据和组件从不跨工具链共享。

URL 和本地归档安装走 `resolved_install.go`：先提交 SDK 本体，再处理随包附带的 `stdx`。如果随包 `stdx` 安装失败，会保留已提交的 SDK 并报告部分成功，这与受管通道整组原子发布的语义不同。

## 恢复与文件所有权

`fstx` 在移动文件前先将提交状态、受管路径和备份位置写入磁盘日志。进程中断后的恢复会重放日志并采用同一提交或回滚决策；当文件系统错误导致恢复受阻时，会保留日志和备份并返回 `RecoveryError`。安装、删除和活跃工具链准备均要求先完成恢复，绝不能把恢复所需的备份目录当作普通临时残留清除。

### 并发读写锁（`HomeLock` 与 `WithStableReads`）

`HomeLock` 始终先获取全局写锁，再获取活跃工具链查询门禁锁（gate）。在执行历史快照复制、校验及历史清理等不修改已安装 SDK 目录的耗时阶段，写进程通过 `WithStableReads` 暂时释放查询门禁锁，同时继续持有全局写锁。

由于这些阶段不改动已安装的根目录，并发的读进程（如运行 `cjc`、`cjv show active`、`cjv which`）可以直接解析现有 SDK 而不必阻塞等待：

- 读进程在目录定位期间持有查询门禁锁，并以非阻塞方式探测全局写锁。
- 如果确认仍有存活写进程持有全局写锁，读进程直接使用现有安装目录；如果没有存活写进程持有全局写锁（例如写进程已崩溃退出），则执行正常的事务恢复。
- 写进程在真正发布修改前，必须重新获取查询门禁锁并原子清除只读阶段标记。
- 读进程**绝不能**在持有查询门禁锁的同时阻塞等待全局写锁，以避免锁顺序反转死锁。进程崩溃后遗留的阶段标记不会跳过恢复，且在首次旧布局迁移完成前不启用此快速路径。

### 组件文件清单与路径安全

`component` 在创建快照及修改文件前会校验组件根目录、完整文件清单和父目录符号链接。与基础 SDK 共享的文件在移除组件时予以保留；受管的叶节点符号链接可以直接移除而不删除其指向的源目录；归档合并拒绝穿过未跟踪的符号链接。外部 SDK 目录链接不能作为组件编辑根，回退操作同样会检查目标根。

请注意，受管组件批次拥有可跨进程恢复的持久化发布日志，而直接 `component link` 和自定义归档组件编辑使用进程内备份与普通错误回滚（恢复失败时保留备份）——不要把两类操作都假设为具有相同的跨进程事务恢复保证。

数据路径统一从 `config/layout.go` 派生，目录树复制、合并和平台重试统一交给 `fsops`。所有归档解包与目录树操作均拒绝逃出受管目录的路径或链接。

## 活跃工具链与交叉目标

`toolchain.SelectActive` 按顺序处理显式 CLI 选择（`+name`、`--toolchain`）、`CJV_TOOLCHAIN` 环境变量、逐层目录覆盖与 `cangjie-sdk.toml` 项目文件以及全局默认设置。`PrepareActive` 执行待定恢复并定位本地安装，只有确实缺少官方 SDK 时才调用自动安装回调；无效选择、损坏记录和文件系统错误均直接返回。

`resolve.Active` 根据 `auto_install` 补齐项目声明的依赖项（`run --install` 显式允许安装，只读状态查询绝不安装）。`toolchain.ReadHostTargets` 统一绑定宿主身份、实际发行版和平台；`target` 命令、`component --target`、`resolve.ActiveTarget` 及代理目标补齐都使用它，避免错误关联到其他宿主或不同发行版的交叉 SDK。

`env.Runtime` 从显式传入的基础环境变量合并 SDK 路径与组件变量，生成子进程环境或 `envsetup` 脚本（`env` 本身不负责选择工具链；`resolve.ActiveRuntime` 将当前选择与对应的 `env.Runtime` 组合，`sdktools` 统一工具路径）。Unix 代理直接替换当前进程，Windows 代理与 `cjv run`、`cjv exec` 共用 `process.Run` 处理子进程、退出码与信号。

## 分发与下载

`dist.Source` 按需读取并缓存 LTS/STS 的 `versions.json` 与独立的 `nightly.json`，组件由同一通道数据解析。`cjv` 消费静态清单；上游发现和清单生成由独立的 `cangjie-version-manifest` 仓库负责。

`Preparation` 持有安装锁、暂存目录和已下载归档，持续到发布与清理结束。失败时保留可续传下载，成功时仅清理本次操作拥有的文件，用户提供的本地归档和恢复日志不归它清理。

归档和 nightly SHA-256 sidecar 请求共用可取消的退避重试。HTTP `408`、`429` 和服务器 `5xx` 错误可重试，永久性 `4xx` 错误及格式错误不重试（sidecar 返回 `404` 表示校验和尚未发布）。带 SHA-256 的下载支持跨命令断点续传并校验完整文件，无哈希下载只在单次操作内续传。

## 设置与 `cjv` 自管理

`config.SettingsFile` 保留每个字段是否由用户显式设置的信息：`Update` 只写入明确选择的字段，使未设置字段继续继承系统后备配置；`Save` 则可从快照完整恢复值及字段存在性。环境覆盖保持临时生效；写入前先校验，发布成功后更新内存缓存。

`reachable.Ensure` 统一安装受管二进制、代理入口和环境脚本，并按调用方策略配置 `PATH`。`selfupdate.Check` 只查远程版本，而 `selfupdate.Update` 校验并替换二进制后返回状态结构，交由 CLI 渲染。默认构建与 `mirror` 构建分别实现 GitHub 和 GitCode 的版本发现。
