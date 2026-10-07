# 代码架构

这一章讲 cjv 的代码怎么组织：仓库里有哪些目录、Go CLI 的 `internal/` 拆成了哪些包、各包负责什么，以及一条命令从进程启动到落地是怎么流过这些包的。模块路径是 `github.com/Zxilly/cjv`，Go 版本以 `go.mod` 的 `go` 指令为准（当前 1.26.0）。

## 仓库布局

仓库根目录的主要构成在[简介](introduction.md)里已经过了一遍，这里只补一句：版本化追踪的就是 `cmd/`、`internal/`、`web/`、`docs/`、`tests/`、`scripts/` 这几块，加上根目录的若干配置文件（`go.mod`、`.goreleaser.yml`、`.golangci.yml` 等）。本章往下都是讲 Go CLI 这块。

```text
cmd/cjv/        二进制入口（main 包）
internal/       全部实现，按子系统拆包
scripts/        构建期辅助脚本（代码生成、CI 用）
tests/          跨包的集成测试与冒烟测试
web/            落地页（见“落地页”一章）
docs/           两本 mdBook（见“文档站”一章）
```

`scripts/` 下是两段辅助脚本，不参与 CLI 编译：`gen-platform-surfaces.go` 是 `internal/target` 的 `go:generate` 目标，从平台清单生成代码；`extract-init-binaries.sh` 给发布流程用。`tests/` 下 `integration/` 是端到端集成测试，`smoke/` 验证真实下载，`install-scripts/` 测安装脚本。单元测试按 Go 惯例和被测代码同目录，`_test.go` 紧挨着源文件，所以 `internal/` 各包里看到的 `*_test.go` 都是本包的单元测试。

## 入口：`cmd/cjv/main.go`

`cmd/cjv/main.go` 是唯一的 `main` 包，很薄。它做几件进程级的事，然后把控制权交给 `internal/`。

`version` 和 `updateURL` 两个变量在构建时由链接器注入（详见[从源码构建](building.md)），未注入时 `version` 是 `"dev"`。

`main` 调 `run`，`run` 先初始化日志（`logging.Init`）、记下版本号，再从 `os.Args[0]` 取出被调用的程序名（`proxy.ExtractToolName`），据此分三条路：

- 程序名是某个已知 SDK 工具（`cjc`、`cjpm` 等，由 `sdktools.IsProxyTool` 判定），走代理路径 `proxy.Run`，把参数透传给真正的工具。
- 程序名以 `cjv-init` / `cjv-setup` 开头，把它当安装器，改写 `os.Args` 为 `cjv init` 再继续。
- 否则就是普通的 `cjv ...` 调用，交给 `cli.Execute(version, updateURL)`。

普通 CLI 调用的错误由 `cli.Execute` 输出一次：文本模式写 stderr，JSON 模式写 stdout 信封。`main` 只把 `*cjverr.ExitCodeError` 解包成进程退出码，其余错误返回 1；代理路径不经过 CLI，其错误仍由 `main` 处理。Windows 控制台的 UTF-8 切换、双击运行时的暂停提示也都在 `main` 这层处理（`console_windows.go`），因为它们是进程级的关切；`main` 还把链接器注入的版本号交给 `dist.AppVersion`，供 HTTP 请求的 User-Agent 使用。

## `internal/` 各包职责

`internal/` 下每个目录是一个包，按子系统划分。下面按它们在一条命令里大致的依赖方向，从上层往下层列。

### `cli`：命令定义

`internal/cli` 是 cobra 命令树。每次 `Execute` 都创建一个新的 `application`，由它持有本次调用的命令树、标志值、版本与更新地址、`output.Renderer`。`root.go` 注册全局 `--json`、`--quiet/-q`、`--verbose` 标志，并提取开头的 `+toolchain` 选择器，再挂上各子命令，不复用上次调用的命令或输出模式。每个子命令一个文件，`install.go`、`uninstall.go`、`toolchain.go`、`run.go`、`exec.go`、`which.go`、`show.go`、`check.go`、`update.go`、`component.go` 等，文件名基本能对上命令名。

`cli` 自己不实现业务逻辑，它做的是参数解析、调用下层包、把结果交给渲染层。命令不自己加载设置、构造分发源或计算 host tuple：`install.go` 把标志装进 `lifecycle.InstallRequest` 交给 `lifecycle.Install`；`check.go`、`toolchain_list_remote.go` 通过 `lifecycle.OpenDistribution` 拿到同一份设置、分发源和 host tuple；`component.go` 的 `add` 调 `lifecycle.InstallComponents`；`toolchain.go` 的 `link` 按参数是 URL、归档文件还是目录，分别调 `InstallToolchainFromURL`、`InstallToolchainFromZip`、`LinkToolchainDir`，自己只做名字校验、标志互斥和结果渲染。读设置的唯一入口是 `config.LoadDefaultSettings`。几个子包分担横切关注点：

- `cli/output` 的 `Renderer` 保存本次调用的 JSON 模式，输出模式只在这里判断一次。命令各自定义实现 `Result` 接口（一个 `Text()` 方法）的结构体，由 renderer 输出文本或 JSON；文本输出已经由进度事件说完的结果（`install`、`component add`、URL 与归档形式的 `toolchain link`）内嵌 `output.ProgressDriven`，文本模式下不再输出。`Renderer.Progress` 按模式选出操作要汇报到的进度适配器：文本模式是写到命令输出的 `progress.Text`，JSON 模式是 `progress.Discard`。`RenderOutcome` 渲染部分成功再返回错误，JSON 模式下只在无错误时渲染，保证 stdout 只有一份文档；`RenderErrorTo` 在两种模式下各输出一次错误：JSON 信封写到 stdout，文本写到 stderr；`Note` 把给人看的附注写到 stderr，JSON 模式下不输出（`show` 的"未配置默认工具链"提示、`doc` 的打开浏览器提示）。信封认得 `cjverr` 的 `Coded` 接口来填机器可读的错误码。命令代码里剩下的 `IsJSON()` 判断只用于改变控制流：不支持 JSON 的命令、不能与 JSON 组合的交互确认、`envsetup` 的两条代码路径、`doc` 是否打开浏览器。命令的全部输出（包括 `init` 向导的对话和 stderr 上的提示）都写到 cobra 命令的 out/err writer，测试用缓冲区接住；直接写 `os.Stdout`/`os.Stderr` 的只剩子进程的标准流透传和 `progress.Text` 的下载进度条。
- `cli/settings` 构造 `set`、`default`、`override` 配置子命令。每次注册都创建新命令，目录路径和清理标志由各自的 closure 持有。
- `cli/selfmgmt` 构造 `cjv self` 命令，接收本次调用的 renderer，并将卸载确认标志保存在命令内。显式与自动自更新共同调用 `UpdateManaged`，由它准备受管二进制、执行更新，再经 `reachable.Ensure` 刷新代理链接和 env 脚本；调用方决定输出方式和错误是否致命。`cjv self uninstall` 用 `reachable.RemovePath` 撤销 PATH 配置。

### `lifecycle`：安装与内容生命周期

`internal/lifecycle` 管理官方 SDK、交叉 SDK、组件、自定义安装及卸载。`OpenDistribution` 读取设置、主机平台和分发源，不锁定、恢复或扫描本地安装，因此本地发布或恢复受阻时仍可读取远程目录。`Distribution.Resolve` 解析通道、具体版本、minor 和日期选择器。安装操作打开 `installationDistribution`：在可响应上下文取消的 home 锁内恢复中断操作、迁移旧布局，并取得设置和安装记录快照；发布前再次检查快照，避免覆盖已经变化的内容。`Install(ctx, InstallRequest{Toolchain, Targets, Components, Force, NoUpdate}, opts)` 服务 CLI 与代理自动安装。`Force` 统一映射为 `Options.AllowMissing`，不再触发同版本重装；`Options.AllowDowngrade` 控制 nightly 降级，`Progress` 和 `ConfigurePath` 分别控制展示和首次安装的 PATH 配置。

`group_install.go` 在安装锁下先取得安装记录、组件选择及宿主依赖的快照，准备期间释放 home 锁。主机操作准备自身及所有同平台跟踪 targets；直接指定交叉 SDK 时使用相同流程且不发布默认工具链。未变化的 SDK 复用原内容，只准备新增组件。nightly 根据已发布历史解析完整的组件和目标集合，缺失项默认阻止升级或触发兼容版本回溯；AllowMissing 允许跳过并移除缺失项，校验或网络错误仍使操作失败。

准备完成后重新获取 home 锁，检查依赖和快照，再由 `publication.go` 通过 `fstx.NewToolchainGroupTransaction` 一起替换 SDK、stdx 与文档根。事务包含所有成员的路径约束；失败及进程中断恢复使用同一提交或回滚决策。固定版本与其他主机独立保留，SDK 去重仅作用于符合来源与内容要求的文件。代理补齐目标保留宿主版本和默认选择。

受管组件安装采用相同的准备和再次校验顺序。`component_publication.go` 暂存所选组件根及其清单，再通过一个 `fstx.NewToolchainTransaction` 发布整批组件。内容和元数据共用持久化提交决策，普通错误与进程中断都由 home 恢复流程处理。已有 SDK 内容留在原处，成功事件只在整批提交后发出。

`UpdateInstalled` 更新指定通道，`UpdateAll` 先恢复本地状态再判断是否存在安装，随后遍历跟踪通道、跳过固定和自定义安装，交叉 SDK 由对应主机更新。独立目标入口也使用 group 策略。批量更新继续处理单项失败并返回聚合错误，仅全部成功时清理下载暂存区，失败时保留可续传分段。`InstallComponents`、目录链接、URL 和本地归档安装维持各自的组件替换或同名覆盖参数；自定义安装仍由 `resolved_install.go` 的放置流程负责。随包 stdx 在 SDK 发布后、home 锁释放前安装，准备生命周期覆盖这两步；stdx 失败仍独立于已提交的 SDK 记录部分成功。

### `resolve`：活动工具链解析

`internal/resolve` 回答“现在该用哪个工具链”。`Active` 通过 `toolchain.SelectActive` 共用命令行 `+toolchain`、`CJV_TOOLCHAIN`、目录级与全局 override、默认设置的优先级规则。随后 `toolchain.PrepareActive` 恢复本地状态、校验所选宿主并定位安装目录。`Active`、`cjv run` 和状态检查共用这套准备策略：仅当官方工具链确实缺失时调用可选的安装函数，安装后再用相同查找流程校验结果。无效的宿主选择、损坏的记录和其他文件系统错误直接返回，不会触发安装。是否允许安装由调用方决定：`run --install` 显式启用，`Active` 遵循 `auto_install`，状态检查不传安装函数。

`Active` 还会补齐项目要求的 targets 和组件，并返回包含解析目录及配置来源的 `ActiveToolchain`。`AutoInstallFunc` 与 `AutoInstallComponentsFunc` 保留为测试缝，生产实现默认调用 `lifecycle`，因此 `resolve` 不依赖 `cli`。自动安装的进度（包括它自己的“正在自动安装”与失败提示）和 `cjv install` 一样发到一个 `progress.Sink`：代理路径在任何 renderer 之前运行，stdout 属于被代理的工具，所以它用写到 stderr 的 `progress.Text`。

### `toolchain` 与 `component`：已装内容的模型

`internal/toolchain` 管已安装的 SDK：列出已装工具链（`ListInstalled`）、解析活动工具链目录，以及工具链名字的解析与版本比较。`RecoverHomeContext` 以可取消的方式获取 home 锁，再执行恢复和旧布局迁移；已经持锁的调用方使用锁上的 `Recover` 与 `MigrateLegacy` 方法。恢复先让 `fstx` 处理 `toolchains/` 下未完成的事务，再删除废弃的 staging 树、把原目录已缺失的旧式备份放回去。恢复受阻时原样返回 `fstx.RecoveryError`，不碰任何残留；需要恢复的备份不会作为普通残留直接删除。安装、升级、删除在改动文件前要求恢复成功；活动工具链准备也先完成恢复，恢复失败会阻止使用不完整安装。

`internal/component` 管工具链的附加组件：`stdx`、`docs`、`stdx-docs`。每个组件是单独下载的归档，解压后的文件通过逐组件的清单（manifest）记录，从而能独立卸载。`component` 还定义了组件装到哪（`InstallLocation`：有的落进工具链目录树，有的作为纯数据放到 `<CJV_HOME>/docs/<tc>/`）以及组件要注入哪些环境变量。

`StagePreparedBatch` 将受影响的根和组件索引复制到私有暂存目录，在其中应用整批选择，再由 `lifecycle` 负责受管安装的持久化发布。`ApplyChanges` 继续为本地链接和直接归档操作提供备份与普通错误回滚；备份包含组件文件和清单，恢复失败时保留备份并在错误中返回位置。这些本地操作及自定义 SDK 的随包 stdx 保持各自原有的恢复语义。

### `dist`：下载与解包

`internal/dist` 负责分发源与网络制品。`source.go` 是统一入口：LTS/STS 按需缓存 `versions.json`，nightly 按需缓存同目录的 `nightly.json`。显式 `dist_server` 时两者位于分发根下。相对 URL 以 manifest 所在目录解析，绝对 URL 原样使用。组件制品也由它按通道、版本和 stdx 平台解析（`ResolveComponent`）；`component.PrepareFromSource` 在调用方的下载生命周期内准备组件，`InstallFromSource` 为独立调用方开启该生命周期。`manifest.go` 解析并校验通道数据；`download.go` 做重试、断点续传和 SHA256 校验，并把传输的开始、字节进展和结束作为 `progress` 事件发给调用方传入的 sink，自己不画进度条；`install.go` 解包归档，解出的树由 `fsops.MoveTree` 落到目标目录；`nightly.go` 持有共享 HTTP 客户端并读取 nightly 资产的 SHA256 sidecar。host 与目标 tuple 的计算在 `target`（`CurrentHostTuple`、`CurrentTargetTuple`），`dist` 不再转发。

`Preparation` 统一拥有一次 SDK 或组件操作的安装锁、私有暂存目录和下载归档。`BeginPreparation` 获取安装锁，随后由调用方获取 home 锁。安装锁持续到发布和清理结束，home 锁可在下载、解压期间释放。SDK 安装组和组件批次中的所有制品共用一次准备生命周期。`Complete` 标记发布成功；`Close` 总是清理私有暂存目录，仅在成功后删除自己拥有的已校验归档，失败时保留已下载归档及可续传分段，最后释放安装锁。用户提供的本地归档不归它所有；持久化日志和恢复备份放在其暂存区之外。下载清理也使用同一把锁，避免删除正在使用的文件。

### `target`：平台身份

`internal/target` 是平台与目标 tuple 的单一事实源。它解析目标 tuple（host 部分加可选的交叉编译环境后缀），给出清单索引键、stdx 平台 token 等结构化视图，免得每个调用方各自去切字符串。`catalog.go` 列出 cjv 出 host 二进制的全部 `(GOOS, GOARCH)` 组合，是发布产物和落地页下载入口的源头；它带 `go:generate` 指令，跑 `scripts/gen-platform-surfaces.go` 生成。

### `env`：运行时环境

`internal/env` 组装运行仓颉工具所需的环境。`Runtime` 持有活动工具链和私有的 SDK 环境配置，调用方不再读取或修改内部配置。组件变量由 `component.ApplyEnv` 直接叠加，工具链内的工具路径经 `sdktools` 定位。`ProxyEnv`、`ToolchainEnv` 都显式接收基础环境，按同一规则合并 PATH、动态库路径、`SDKROOT` 和组件变量；不会在合并时偷读进程中的另一份环境。`Contributions` 返回不含继承值的 SDK 贡献，`ShellScript` 根据相同合并结果生成 shell 差异脚本。平台变量名、路径次序与大小写规则集中在环境 module 中，shell 检测及格式化仍由 `shelldetect.go`、`shellformat.go` 负责。修改 shell 配置文件、注册表 PATH 和写 env 脚本不在这里，它们属于 `reachable`。

### `proxy`：透明代理

`internal/proxy` 实现透明代理：当二进制以 `cjc`、`cjpm` 等工具名被调用时，`Run` 解析活动工具链（经 `env.ResolveRuntime`）、经 `sdktools` 在工具链目录里定位真正的工具二进制、组装代理环境并透传参数。Unix 通过 `syscall.Exec` 替换当前进程，Windows 通过 `process.Run` 启动并等待子进程。它带一个递归计数器（`CJV_RECURSION_COUNT`），防止代理无限自调。包里只有这条运行路径，工具布局本身不在这里。

### `sdktools`：SDK 工具布局

`internal/sdktools` 描述 SDK 的工具布局：工具链带哪些工具、每个工具在工具链目录里的相对路径（`toolPathMap`）、cjv 二进制和各工具在各平台上的文件名（`CjvBinaryName`、`PlatformBinaryName`）、在 `CJV_HOME/bin` 下建出代理链接（`CreateAllProxyLinks`），以及按工具链目录和目标 tuple 定位已装工具二进制（`ResolveInstalledToolBinary`、`ResolveInstalledToolBinaryForTuple`）。它只依赖 `config`、`target`、`fsops` 和 `cjverr`，位于 `lifecycle`、`env`、`selfupdate` 和 `proxy` 之下，让安装校验、代理链接、`cjv run`/`which` 的工具查找和受管二进制路径读的是同一份布局。

### `reachable`：让 cjv 可达

`internal/reachable` 负责让装好的 cjv 能从用户的 shell 里被调用。这件事由四步组成：`CJV_HOME/bin` 下的受管二进制、旁边的代理链接、`CJV_HOME` 下的 env 脚本，以及写进 shell 配置文件（`.profile`、`.bashrc`、`.zshrc`、`.zprofile`、fish）或 Windows 用户注册表的 PATH 项。所有安装路径通过一个 `Policy` 请求同一操作 `Ensure`：零值只建立缺失的受管二进制和代理链接，`ForceManagedBinary` 用当前可执行文件覆盖受管二进制，`EnvScripts` 重写 env 脚本，`ConfigurePath` 追加 PATH。`cjv init` 传 `{ForceManagedBinary, EnvScripts, ConfigurePath: 是否修改 PATH}`，`lifecycle` 落盘后传零值，`cjv toolchain link <目录>` 传零值，`cjv self update` 传 `{EnvScripts}`。PATH 策略集中在 `ConfigurePath`：`CJV_NO_PATH_SETUP=1` 跳过、按平台选 shell 配置或注册表、幂等、失败只记日志并在 stderr 给一次提示；`RemovePath` 是它的逆操作，供 `cjv self uninstall` 使用。`ShellConfigPaths` 列出会被修改的 shell 配置文件，`cjv init` 用它向用户展示。包只依赖 `config`、`i18n`、`sdktools`、`selfupdate` 和 `fsops`，位于 `lifecycle` 与 `cli` 之下。

### `process`：子进程执行

`internal/process.Run` 接收调用方配置好的 `exec.Cmd`，统一启动、等待和终止信号处理，并将子进程非零退出转换为 `cjverr.ExitCodeError`。`cjv run`、`cjv exec` 与 Windows 代理共用它；命令查找、参数、环境和标准流仍由调用方配置。Unix 的 SIGTERM 转发及超时升级、各平台避免父进程先于子进程响应 Ctrl+C 退出的处理都留在此包内。

### `config`：配置与路径

`internal/config` 是配置层。它定义所有 `CJV_*` 环境变量名（包括 `CJV_DIST_SERVER`）、解析 `CJV_HOME`、读写用户与系统后备设置、工具链文件和目录级 override。`layout.go` 是 CJV_HOME 布局的唯一出处：`toolchains/`、`stdx/`、`docs/`、`downloads/`、`bin/` 这些子目录名及各工具链在其中的目录（`ToolchainDirFor`、`StdxDirFor`、`DocsDirFor`），以及安装残留的命名规则——staging 树由 `StagingDir(dest)` 给出，`IsScratchName` 判断一个目录名是 staging、旧式备份还是 `fstx` 事务目录。其他包从这里派生路径，不自己拼后缀。`manifest_url` 提供正式通道清单并确定 nightly 文件目录，`dist_server` 选择包含两份清单的企业分发根；`mirror` 构建标记选择默认地址。

`SettingsFile.Load` 返回生效配置的副本，同时保留用户字段的来源。`Update(SettingsUpdate)` 只保存明确选择的字段，未指定字段继续继承系统或内置默认值；显式选择与继承值相同的值仍可将其固定，`false` 和空字符串也不会被当成未设置。`Save` 支持恢复已加载快照的值与字段存在性。写入前完成配置校验和缓存准备，文件发布成功后不会因重新读取失败而误报保存失败。环境覆盖保持临时生效，不会落盘。

### `selfupdate`：自我更新

`Check` 只获取发布信息并返回 `available` 等状态，不下载或替换二进制，供 `cjv check` 和 auto-self-update 的 check 模式使用。

`internal/selfupdate` 负责发现、校验和安装 cjv 更新。具体走 GitHub 还是 GitCode 由 `mirror` 构建标记在编译期选定（`update_default.go` / `update_mirror.go`）。`Update` 返回状态（`skipped`、`dev`、`up-to-date`、`updated`）及版本，供 `cli/selfmgmt` 渲染，不自行向 stdout 打印结果。它还管理把当前二进制确立为受管可执行文件（`EnsureManagedExecutable`、`ForceUpdateManagedExecutable`，由 `reachable.Ensure` 调用）、以及更新时替换正在运行的二进制（Windows 与其他平台分 `replace_windows.go` / `replace_other.go`）。受管二进制的文件名取自 `sdktools`。

### 支撑包

剩下几个是被各层共用的支撑包：

- `i18n` 国际化。消息存在 `locales/en.toml` 和 `locales/zh-CN.toml` 里并嵌进二进制，`i18n.T` 按消息 ID 取串。所有面向用户的文本都走它，错误信息也是。
- `cjverr` 错误类型。定义带稳定机器码（`ErrorCode`）的结构化错误，`Error()` 方法通过 `i18n` 产出人读信息，`Coded` 接口让 `output` 能在 JSON 模式下输出错误码。`ExitCodeError` 携带进程退出码。
- `fstx` 文件系统事务。落盘日志记录受管路径、备份和事务状态，读取时限制日志大小与路径范围；事务目录名、staging 树和工具链事务覆盖的 `toolchains/`、`stdx/`、`docs/` 条目都取自 `config` 的布局。`toolchain.RecoverHome` 在启动清理及安装、删除重试时先调用 `Recover` 恢复未完成操作；提交及准备发布的状态保留已就绪内容，再清理备份。恢复受阻时保留日志和备份并报告位置，后续可以重试，而不是依赖进程内的 undo 闭包。
- `fsops` 文件系统操作。所有改动 CJV_HOME 的包都经它落盘：`RemoveAllRetry`、`RenameRetry` 和给 `os.Root` 用的 `Retry` 吸收 Windows 上杀毒软件与索引器造成的瞬时锁；`WriteFileAtomic` 原子写文件；`SymlinkOrJunction` 在符号链接需要特权时退回目录 junction；`CreateLink` 按符号链接、硬链接、复制三级退化建代理链接；`IsPathUnder` 判断路径归属。树操作只有一份实现：`MoveTree` 把解压好的树逐项合并进目标目录——目录合并、文件与符号链接覆盖已有条目、重命名失败时跨卷复制、拒绝绝对或逃出源树的符号链接、返回放置的文件清单；`CopyTree` 原样备份和恢复组件根，符号链接保留原目标、目录模式在填满后再套用。`dist` 解包、`component` 的暂存与快照、`lifecycle` 的 staging、`fstx`、`selfupdate`、`reachable`、`toolchain`、`config` 都调用它。
- `retry` 斐波那契退避的重试引擎 `retry.Do`：`fsops` 用它重试瞬时文件错误，`dist` 用它重试网络请求。
- `logging` 用 `CJV_LOG` 环境变量配 `slog` 全局 logger（默认 `warn`）。
- `progress` 进度缝。`Event` 是带 `Kind` 与所需数据（工具链、组件、下载字节数等）的类型化进度事件，`Sink` 是接收方。两个适配器：`Text` 把事件渲染成 i18n 文本并在终端上用 mpb 画下载进度条，`Discard` 什么都不输出。`lifecycle`、`resolve`、`dist`、`component`、`selfupdate` 只发事件，不认识 i18n 消息 ID。
- `testutil` 测试辅助：mock 下载服务器、Windows 注册表守卫、记录进度事件的 `ProgressRecorder`。它带 `_test.go` 之外的源文件，供其他包的测试导入。

## 一条命令的流向

把上面串起来，看 `cjv install <toolchain>` 大致怎么走。

进程从 `cmd/cjv/main.go` 的 `run` 起步：`logging.Init` 配好日志，程序名是 `cjv` 不是某个工具名，于是走 `cli.Execute`。cobra 把 `install` 子命令路由到 `internal/cli/install.go` 的 `runInstall`。`runInstall` 把 `--target`、`--component`、`--force` 装进 `lifecycle.InstallRequest`，组好 `lifecycle.Options`（renderer 按输出模式选出的进度适配器，以及首次安装时配置 PATH 的选择），调 `lifecycle.Install`。

`lifecycle` 编排其余步骤：`openInstallationDistribution` 恢复本地状态，取得分发设置与安装记录快照；`Distribution.Resolve` 让 `dist.Source` 从 manifest 解析通道、版本和平台；`installGroup` 用一次 `dist.Preparation` 在私有暂存目录下载、解压 SDK，准备全部所需组件并再次校验整个安装组，再由 `fstx` 一起发布所有根目录。`reachable` 在最终确认阶段建立托管二进制和代理链接。所有通道共用这条安装路径。进度事件一路发到 CLI 选好的适配器（文本模式下消息写到命令输出、下载进度条画在 stderr），命令结果由本次调用的 renderer 渲染；错误由 `cli.Execute` 经同一个 renderer 输出，再由 `main` 翻译成退出码。

代理路径是另一条主线。运行 `cjc build` 时，被调用的其实是名为 `cjc` 的 cjv 链接，`main` 认出工具名走 `proxy.Run`：`proxy` 经 `env.ResolveRuntime` 让 `resolve` 定出活动工具链、经 `sdktools` 在工具链目录里找到真正的 `cjc`、组装好运行环境，然后按平台替换当前进程或运行子进程。这条线绕过 cobra 命令树，保留工具的标准流和退出语义。

想深入某一块，从这几处入手最快：命令定义看 `internal/cli/root.go`，安装编排看 `internal/lifecycle/install.go`，分发源看 `internal/dist/source.go`，官方 SDK 落盘事务看 `internal/lifecycle/group_install.go` 和 `publication.go`，代理看 `internal/proxy/proxy.go`。测试怎么组织见[测试](testing.md)。
