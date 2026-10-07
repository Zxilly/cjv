# 代码架构

CLI 解析命令，`lifecycle` 编排安装与更新，底层模块负责分发、文件所有权和事务。代理执行从同一套安装状态解析 SDK，绕过 CLI 命令树。

## 入口与模块职责

`cmd/cjv/main.go` 按调用名称分流：SDK 工具名进入 `proxy.Run`；`cjv-init` / `cjv-setup` 前缀进入安装向导；其他调用进入 `cli.Execute`。

| 模块 | 职责 |
| --- | --- |
| `cli` | Cobra 命令树、参数校验、结果与错误输出 |
| `lifecycle` | SDK、目标、组件、自定义安装和卸载的编排 |
| `toolchain` | 安装身份、版本解析、宿主与目标关联、本地恢复 |
| `resolve` | 活跃工具链准备与缺失项自动安装 |
| `component` | 组件目录、文件清单、链接及批次编辑 |
| `dist` | 按需加载 manifest、下载、校验、解包及准备生命周期 |
| `target` | 平台 tuple、SDK/组件平台映射及发布平台清单 |
| `env`、`sdktools` | 环境合并、shell 输出和 SDK 工具布局 |
| `proxy`、`process` | 工具代理、子进程、退出码和信号处理 |
| `config` | 设置来源、项目选择规则和数据目录布局 |
| `reachable`、`selfupdate` | 受管二进制、代理入口、PATH 及 cjv 更新 |
| `fstx`、`fsops` | 持久化事务、原子文件操作和平台重试 |

`progress` 定义事件和输出适配器；`i18n`、`cjverr`、`logging` 分别提供消息、结构化错误和日志。`testutil` 提供测试分发源、进度记录器及平台测试辅助。

## 命令状态与输出

每次 `cli.Execute` 创建独立的 application、命令树、标志和 `output.Renderer`。子命令把请求交给业务模块，业务模块通过 `progress.Sink` 报告事件，不直接打印最终结果。

文本模式由 `progress.Text` 和结果的 `Text()` 方法渲染；JSON 模式丢弃进度，只输出结构化结果或错误。错误由 CLI 渲染一次，`main` 负责退出码。代理路径的 stdout 属于 SDK 工具，自动安装进度写入 stderr。

修改 CLI 时，保持每次调用的状态隔离，不把标志、writer 或 JSON 模式存回全局变量。

## 安装与发布

`lifecycle.OpenDistribution` 读取设置、平台与清单，不恢复或扫描本地安装，远程查询因此不依赖本地恢复成功。写入操作使用安装分发会话，先恢复中断事务和迁移旧布局，再读取安装快照。

一次受管 SDK 安装遵循以下顺序：

1. `dist.Preparation` 获取安装锁，创建本次操作的私有暂存区。
2. 在 home 锁内读取安装身份、组件选择及宿主依赖快照。
3. 释放 home 锁，解析版本、下载并准备宿主、跟踪目标和组件。安装锁继续持有。
4. 重新获取 home 锁，检查快照和依赖是否变化。
5. `publication.go` 通过 `fstx.NewToolchainGroupTransaction` 一起发布 SDK、stdx 和文档。
6. 完成命令入口等收尾工作，标记准备成功并清理本次拥有的下载和暂存文件。

`--force` 对应允许缺失的可选项，不强制重装相同 SDK。`--no-update` 保留已安装发行版；nightly 根据组件和目标集合选择已发布的兼容版本，降级需显式允许。

受管组件批次也先准备再复查，通过单个事务发布内容和清单。SDK 本体留在原处。固定版本与跟踪通道独立；硬链接去重只用于满足来源及内容条件的普通 SDK 文件，元数据和组件不共享。

URL 和本地归档安装走 `resolved_install.go`。随包 stdx 在 SDK 提交后处理，失败会保留已提交 SDK 并报告部分成功。这与受管通道的整组发布语义不同。

## 恢复与文件所有权

`fstx` 将提交状态、受管路径和备份写入日志。进程中断后的恢复使用同一提交或回滚决策；恢复受阻时保留日志和备份，并返回 `RecoveryError`。安装、删除和活跃工具链准备要求先完成恢复，不能把需要恢复的备份当作普通残留清除。

`component` 在创建快照及修改文件前校验组件根、完整文件清单和父目录链接。共享文件保留，受管叶链接可以移除且不删除源目录，归档合并不能穿过未跟踪链接。外部 SDK 目录链接不能作为组件编辑根，回滚同样检查目标根。

受管组件批次有持久化发布日志；直接组件链接和归档编辑使用备份及普通错误回滚，恢复失败时保留备份。不要把两类操作都描述为可跨进程恢复的事务。

数据路径从 `config/layout.go` 派生，树复制、合并和平台重试交给 `fsops`。归档与树操作拒绝逃出所拥有目录的路径或链接。

## 活跃工具链与交叉目标

`toolchain.SelectActive` 处理显式选择、环境变量、逐层目录覆盖及项目文件、默认设置。`PrepareActive` 恢复并定位安装，只有确实缺少官方 SDK 时才调用安装回调；无效选择、损坏记录和文件系统错误直接返回。

`resolve.Active` 根据 `auto_install` 补齐项目要求。`run --install` 显式允许安装，状态查询不安装。`toolchain.ReadHostTargets` 统一读取宿主身份、实际发行版和平台；`target` 命令、`component --target`、`resolve.ActiveTarget` 及代理补齐都使用它，避免关联到其他宿主或版本的交叉 SDK。

`env.Runtime` 从显式传入的基础环境合并 SDK 路径与组件变量，再生成子进程环境或 shell 脚本。`sdktools` 统一工具路径。Unix 代理替换当前进程，Windows 代理与 `run`、`exec` 共用 `process.Run` 处理子进程。

## 分发与下载

`dist.Source` 按需读取并缓存 LTS/STS 的 `versions.json` 与独立 `nightly.json`，组件由同一通道数据解析。cjv 消费静态清单；上游发现和清单生成由独立的 `cangjie-version-manifest` 仓库负责。

`Preparation` 拥有安装锁、暂存目录和下载归档，持续到发布与清理结束。失败保留可复用下载，成功仅清理自己拥有的文件；用户提供的本地归档和恢复日志不归它清理。

归档和 nightly SHA-256 sidecar 共用可取消重试。408、429 和服务器错误可重试，永久 HTTP 错误及格式错误不重试；sidecar 404 表示尚未发布。带 SHA-256 的下载可跨命令续传并校验完整文件，无哈希下载只在单次操作内续传。

## 设置与 cjv 自管理

`config.SettingsFile` 保留字段是否由用户设置的信息，`Update` 只写明确选择的字段，`Save` 可恢复值及字段存在性。环境覆盖保持临时生效；写入前校验，发布成功后更新缓存。

`reachable.Ensure` 统一安装受管二进制、代理入口和环境脚本，按调用方策略配置 PATH。`selfupdate.Check` 只查版本，`Update` 校验并替换二进制后返回状态，由 CLI 渲染。默认与 mirror 构建分别实现 GitHub 和 GitCode 版本发现。
