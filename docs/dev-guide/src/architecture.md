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

普通 CLI 调用的错误由 `cli.Execute` 输出一次：文本模式写 stderr，JSON 模式写 stdout 信封。`main` 只把 `*cjverr.ExitCodeError` 解包成进程退出码，其余错误返回 1；代理路径不经过 CLI，其错误仍由 `main` 处理。Windows 控制台的 UTF-8 切换、双击运行时的暂停提示也都在 `main` 这层处理，因为它们是进程级的关切。

## `internal/` 各包职责

`internal/` 下每个目录是一个包，按子系统划分。下面按它们在一条命令里大致的依赖方向，从上层往下层列。

### `cli`：命令定义

`internal/cli` 是 cobra 命令树。每次 `Execute` 都创建一个新的 `application`，由它持有本次调用的命令树、标志值、版本与更新地址、`output.Renderer`。`root.go` 注册根级 `--json` 标志并挂上各子命令，不复用上次调用的命令或输出模式。每个子命令一个文件，`install.go`、`uninstall.go`、`toolchain.go`、`run.go`、`exec.go`、`which.go`、`show.go`、`check.go`、`update.go`、`component.go` 等，文件名基本能对上命令名。

`cli` 自己不实现业务逻辑，它做的是参数解析、调用下层包、把结果交给渲染层。命令不自己加载设置、构造分发源或计算 host tuple：`install.go` 把标志装进 `lifecycle.InstallRequest` 交给 `lifecycle.Install`；`check.go`、`toolchain_list_remote.go` 通过 `lifecycle.OpenDistribution` 拿到同一份设置、分发源和 host tuple；`component.go` 的 `add` 调 `lifecycle.InstallComponents`；`toolchain.go` 的 `link` 按参数是 URL、归档文件还是目录，分别调 `InstallToolchainFromURL`、`InstallToolchainFromZip`、`LinkToolchainDir`，自己只做名字校验、标志互斥和结果渲染。读设置的唯一入口是 `config.LoadDefaultSettings`。几个子包分担横切关注点：

- `cli/output` 的 `Renderer` 保存本次调用的 JSON 模式。命令各自定义实现 `Result` 接口（一个 `Text()` 方法）的结构体，由 renderer 输出文本或 JSON。错误的 JSON 信封也在这里组装，它认得 `cjverr` 的 `Coded` 接口来填机器可读的错误码。
- `cli/settings` 构造 `set`、`default`、`override` 配置子命令。每次注册都创建新命令，目录路径和清理标志由各自的 closure 持有。
- `cli/selfmgmt` 构造 `cjv self` 命令，接收本次调用的 renderer，并将卸载确认标志保存在命令内。显式与自动自更新共同调用 `UpdateManaged`，由它准备受管二进制、执行更新，再经 `reachable.Ensure` 刷新代理链接和 env 脚本；调用方决定输出方式和错误是否致命。`cjv self uninstall` 用 `reachable.RemovePath` 撤销 PATH 配置。

### `lifecycle`：安装与内容生命周期

`internal/lifecycle` 把下载、解压、校验、组件、可达性和事务替换串成一条安装流程。对外的安装入口只有一个：`Install(ctx, InstallRequest{Toolchain, Targets, Components, Force}, opts)`，设置、分发源和 host tuple 都在里面解析；`InstallComponents` 给已装工具链加组件；`InstallToolchainFromURL`、`InstallToolchainFromZip`、`LinkToolchainDir` 是 `toolchain link` 的三种形式；`UpdateInstalled`、`UpdateAll`、`RemoveToolchain` 管更新与卸载。一次操作要读的"设置 + 分发源 + host tuple"由 `OpenDistribution` 一次性打开成 `Distribution`，`check`、`list-remote` 也用它，所以各命令看到的 manifest、`dist_server` 根和平台一致；`Distribution.Resolve` 把通道/版本请求解析成 `ResolvedToolchain`，manifest 进度提示每次操作只报一次。校验由它直接调用；工具链落盘后经 `reachable.Ensure` 建立受管二进制和代理链接，首次发布默认工具链时按 `Options.ConfigurePath` 决定是否再调 `reachable.ConfigurePath`。`Options` 只剩 `Report`、`ConfigurePath`、`ComponentInstall`：前者与后者服务表现层和测试，`ConfigurePath` 是个普通布尔值，`cjv install` 置为真，`cjv init` 与代理自动安装置为假，没有函数字段承载 PATH 策略。依赖方向从 `cli` 指向 `lifecycle`，再从 `lifecycle` 指向 `reachable` 与 `sdktools`。`Report` 只报告进度，未设置时保持静默；输出格式由 CLI 决定。同一流程服务 `cli install` 与代理自动安装：两者装出的工具链都带受管二进制和代理链接。已有 SDK 的重复安装在文本与 JSON 模式下都成功，`component add` 的 JSON 结果也由 CLI 统一渲染。

包内按职责分文件：`install.go` 是 `Install` 的编排（host 工具链、目标平台变体、组件），`source.go` 是 `Distribution`（打开设置与分发源、解析 `ResolvedToolchain`），`component_install.go` 编排组件批量安装并交给 `component.ApplyChanges` 负责回滚，`resolved_install.go` 是落盘流水线 `placeToolchain`：恢复、staging、校验、事务替换、`finalizeInstalledToolchain`，只有 SDK 归档怎么到达 staging 这一步（`acquisition` 的 `fetch`/`extract`）随来源变化——manifest 发布版下载后解包，URL 或本地归档先解开 CI 外层包再定位内层 SDK/stdx；`link.go` 是这三种链接形式，其中目录链接不经 staging 和事务（它只是一个符号链接项），但同样先恢复、校验编译器、走同一个 finalize，finalize 失败则撤掉链接；`update.go` 编排更新流程，`upgrade.go` 是它替换单个工具链的步骤，`downloads_purge.go` 在全量更新结束后清空下载暂存区。工具链替换的回归测试直接调用这些生产安装入口，对三种来源验证最终步骤失败后的旧安装恢复和回滚错误传播。

`UpdateInstalled` 与 `UpdateAll` 是更新入口：`UpdateInstalled` 接收一个已解析的工具链名，通道名更新该通道最新的已安装宿主版本，目标平台变体名更新该变体，明确版本缺失时安装、已存在时不动；`UpdateAll` 遍历所有已安装工具链，跳过自定义和链接的，逐个失败不中断，最后清空下载暂存区。两者都在包内查找已安装版本、加载设置、构建分发源并解析通道头，再交给包内的升级步骤，结果以 `UpdateOutcome`（已更新、已是最新、已跳过、固定版本、失败）返回，由 CLI 渲染为文本或 JSON。升级步骤与 `RemoveToolchain` 统一处理 SDK、外置的 stdx/docs 内容及默认工具链、目录 override 引用。升级会为下载组件获取新版本对应的制品，为链接组件保留原始来源；替代工具链已存在时保留其组件选择，只补齐缺少的组件。失败时撤回本次新建且未被引用的替代安装，恢复受阻时保留仍需使用的内容并报告错误。同名强制重装保留已有组件清单和外置内容。URL 安装中的附带 stdx 仍在 SDK 安装后处理，可能出现 SDK 成功、stdx 失败的部分成功。

### `resolve`：活动工具链解析

`internal/resolve` 回答“现在该用哪个工具链”。`Active` 综合命令行的 `+toolchain` 覆盖、`CJV_TOOLCHAIN` 环境变量、目录级与全局的 override、默认设置，定出活动工具链的名字和目录，连同它的目标平台和组件一起返回成 `ActiveToolchain`。解析过程中如果工具链没装，它能通过 `AutoInstallFunc` 这个测试缝触发自动安装；生产环境里这个缝默认接到 `lifecycle`，这样 `resolve` 不必反向依赖 `cli`。

### `toolchain` 与 `component`：已装内容的模型

`internal/toolchain` 管已安装的 SDK：列出已装工具链（`ListInstalled`）、解析活动工具链目录，以及工具链名字的解析与版本比较。`RecoverHome` 是 CJV_HOME 唯一的恢复入口：先让 `fstx` 恢复 `toolchains/` 下未完成的事务，再删除废弃的 staging 树、把原目录已缺失的旧式备份放回去。恢复受阻时它原样返回 `fstx.RecoveryError`，不碰任何残留；需要恢复的备份不会作为普通残留直接删除。安装、升级、删除在改动文件前调用它，代理解析与 `update` 在启动时调用它并把受阻的恢复记为警告。

`internal/component` 管工具链的附加组件：`stdx`、`docs`、`stdx-docs`。每个组件是单独下载的归档，解压后的文件通过逐组件的清单（manifest）记录，从而能独立卸载。`component` 还定义了组件装到哪（`InstallLocation`：有的落进工具链目录树，有的作为纯数据放到 `<CJV_HOME>/docs/<tc>/`）以及组件要注入哪些环境变量。

`ApplyChanges` 管理一次组件修改或一批修改的备份、失败恢复和清理；归档安装与本地链接共用替换流程。备份包含组件文件和清单，恢复失败时保留备份并在错误中返回位置，供后续恢复，调用方不再自行管理快照寿命。

### `dist`：下载与解包

`internal/dist` 负责分发源与网络制品。`source.go` 是统一入口：LTS/STS 按需缓存 `versions.json`，nightly 按需缓存同目录的 `nightly.json`。显式 `dist_server` 时两者位于分发根下。相对 URL 以 manifest 所在目录解析，绝对 URL 原样使用。组件制品也由它按通道、版本和 stdx 平台解析（`ResolveComponent`），`component.InstallFromSource` 是唯一的组件下载路径。`manifest.go` 解析并校验通道数据；`download.go` 做进度、重试和 SHA256 校验；`install.go` 解包归档；`nightly.go` 读取 nightly 资产的 SHA256 sidecar。host 与目标 tuple 的计算在 `target`（`CurrentHostTuple`、`CurrentTargetTuple`），`dist` 不再转发。

### `target`：平台身份

`internal/target` 是平台与目标 tuple 的单一事实源。它解析目标 tuple（host 部分加可选的交叉编译环境后缀），给出清单索引键、stdx 平台 token 等结构化视图，免得每个调用方各自去切字符串。`catalog.go` 列出 cjv 出 host 二进制的全部 `(GOOS, GOARCH)` 组合，是发布产物和落地页下载入口的源头；它带 `go:generate` 指令，跑 `scripts/gen-platform-surfaces.go` 生成。

### `env`：运行时环境

`internal/env` 组装运行仓颉工具所需的环境。`Runtime` 持有活动工具链和私有的 SDK 环境配置，调用方不再读取或修改内部配置。组件变量由 `component.ApplyEnv` 直接叠加，工具链内的工具路径经 `sdktools` 定位。`ProxyEnv`、`ToolchainEnv` 都显式接收基础环境，按同一规则合并 PATH、动态库路径、`SDKROOT` 和组件变量；不会在合并时偷读进程中的另一份环境。`Contributions` 返回不含继承值的 SDK 贡献，`ShellScript` 根据相同合并结果生成 shell 差异脚本。平台变量名、路径次序与大小写规则集中在环境 module 中，shell 检测及格式化仍由 `shelldetect.go`、`shellformat.go` 负责。修改 shell 配置文件、注册表 PATH 和写 env 脚本不在这里，它们属于 `reachable`。

### `proxy`：透明代理

`internal/proxy` 实现透明代理：当二进制以 `cjc`、`cjpm` 等工具名被调用时，`Run` 解析活动工具链（经 `env.ResolveRuntime`）、经 `sdktools` 在工具链目录里定位真正的工具二进制、组装代理环境并透传参数。Unix 通过 `syscall.Exec` 替换当前进程，Windows 通过 `process.Run` 启动并等待子进程。它带一个递归计数器（`CJV_RECURSION_COUNT`），防止代理无限自调。包里只有这条运行路径，工具布局本身不在这里。

### `sdktools`：SDK 工具布局

`internal/sdktools` 描述 SDK 的工具布局：工具链带哪些工具、每个工具在工具链目录里的相对路径（`toolPathMap`）、cjv 二进制和各工具在各平台上的文件名（`CjvBinaryName`、`PlatformBinaryName`）、在 `CJV_HOME/bin` 下建出代理链接（`CreateAllProxyLinks`），以及按工具链目录和目标 tuple 定位已装工具二进制（`ResolveInstalledToolBinary`、`ResolveInstalledToolBinaryForTuple`）。它只依赖 `config`、`target`、`utils` 和 `cjverr`，位于 `lifecycle`、`env`、`selfupdate` 和 `proxy` 之下，让安装校验、代理链接、`cjv run`/`which` 的工具查找和受管二进制路径读的是同一份布局。

### `reachable`：让 cjv 可达

`internal/reachable` 负责让装好的 cjv 能从用户的 shell 里被调用。这件事由四步组成：`CJV_HOME/bin` 下的受管二进制、旁边的代理链接、`CJV_HOME` 下的 env 脚本，以及写进 shell 配置文件（`.profile`、`.bashrc`、`.zshrc`、`.zprofile`、fish）或 Windows 用户注册表的 PATH 项。所有安装路径通过一个 `Policy` 请求同一操作 `Ensure`：零值只建立缺失的受管二进制和代理链接，`ForceManagedBinary` 用当前可执行文件覆盖受管二进制，`EnvScripts` 重写 env 脚本，`ConfigurePath` 追加 PATH。`cjv init` 传 `{ForceManagedBinary, EnvScripts, ConfigurePath: 是否修改 PATH}`，`lifecycle` 落盘后传零值，`cjv toolchain link <目录>` 传零值，`cjv self update` 传 `{EnvScripts}`。PATH 策略集中在 `ConfigurePath`：`CJV_NO_PATH_SETUP=1` 跳过、按平台选 shell 配置或注册表、幂等、失败只记日志并在 stderr 给一次提示；`RemovePath` 是它的逆操作，供 `cjv self uninstall` 使用。`ShellConfigPaths` 列出会被修改的 shell 配置文件，`cjv init` 用它向用户展示。包只依赖 `config`、`i18n`、`sdktools`、`selfupdate` 和 `utils`，位于 `lifecycle` 与 `cli` 之下。

### `process`：子进程执行

`internal/process.Run` 接收调用方配置好的 `exec.Cmd`，统一启动、等待和终止信号处理，并将子进程非零退出转换为 `cjverr.ExitCodeError`。`cjv run`、`cjv exec` 与 Windows 代理共用它；命令查找、参数、环境和标准流仍由调用方配置。Unix 的 SIGTERM 转发及超时升级、各平台避免父进程先于子进程响应 Ctrl+C 退出的处理都留在此包内。

### `config`：配置与路径

`internal/config` 是配置层。它定义所有 `CJV_*` 环境变量名（包括 `CJV_DIST_SERVER`）、解析 `CJV_HOME`、读写用户与系统后备设置、工具链文件和目录级 override。`layout.go` 是 CJV_HOME 布局的唯一出处：`toolchains/`、`stdx/`、`docs/`、`downloads/`、`bin/` 这些子目录名及各工具链在其中的目录（`ToolchainDirFor`、`StdxDirFor`、`DocsDirFor`），以及安装残留的命名规则——staging 树由 `StagingDir(dest)` 给出，`IsScratchName` 判断一个目录名是 staging、旧式备份还是 `fstx` 事务目录。其他包从这里派生路径，不自己拼后缀。`manifest_url` 提供正式通道清单并确定 nightly 文件目录，`dist_server` 选择包含两份清单的企业分发根；`mirror` 构建标记选择默认地址。

`SettingsFile.Load` 返回生效配置的副本，同时保留用户字段的来源。`Update(SettingsUpdate)` 只保存明确选择的字段，未指定字段继续继承系统或内置默认值；显式选择与继承值相同的值仍可将其固定，`false` 和空字符串也不会被当成未设置。`Save` 支持恢复已加载快照的值与字段存在性。写入前完成配置校验和缓存准备，文件发布成功后不会因重新读取失败而误报保存失败。环境覆盖保持临时生效，不会落盘。

### `selfupdate`：自我更新

`internal/selfupdate` 负责发现、校验和安装 cjv 更新。具体走 GitHub 还是 GitCode 由 `mirror` 构建标记在编译期选定（`update_default.go` / `update_mirror.go`）。`Update` 返回状态（`skipped`、`dev`、`up-to-date`、`updated`）及版本，供 `cli/selfmgmt` 渲染，不自行向 stdout 打印结果。它还管理把当前二进制确立为受管可执行文件（`EnsureManagedExecutable`、`ForceUpdateManagedExecutable`，由 `reachable.Ensure` 调用）、以及更新时替换正在运行的二进制（Windows 与其他平台分 `replace_windows.go` / `replace_other.go`）。受管二进制的文件名取自 `sdktools`。

### 支撑包

剩下几个是被各层共用的支撑包：

- `i18n` 国际化。消息存在 `locales/en.toml` 和 `locales/zh-CN.toml` 里并嵌进二进制，`i18n.T` 按消息 ID 取串。所有面向用户的文本都走它，错误信息也是。
- `cjverr` 错误类型。定义带稳定机器码（`ErrorCode`）的结构化错误，`Error()` 方法通过 `i18n` 产出人读信息，`Coded` 接口让 `output` 能在 JSON 模式下输出错误码。`ExitCodeError` 携带进程退出码。
- `fstx` 文件系统事务。落盘日志记录受管路径、备份和事务状态，读取时限制日志大小与路径范围；事务目录名、staging 树和工具链事务覆盖的 `toolchains/`、`stdx/`、`docs/` 条目都取自 `config` 的布局。`toolchain.RecoverHome` 在启动清理及安装、删除重试时先调用 `Recover` 恢复未完成操作；提交及准备发布的状态保留已就绪内容，再清理备份。恢复受阻时保留日志和备份并报告位置，后续可以重试，而不是依赖进程内的 undo 闭包。
- `utils` 杂项工具：原子写、文件操作、Windows junction、重试、控制台 UTF-8、打开浏览器、版本号解析等，多数按平台分文件。
- `logging` 用 `CJV_LOG` 环境变量配 `slog` 全局 logger（默认 `warn`）。
- `testutil` 测试辅助：mock 下载服务器、Windows 注册表守卫。它带 `_test.go` 之外的源文件，供其他包的测试导入。

## 一条命令的流向

把上面串起来，看 `cjv install <toolchain>` 大致怎么走。

进程从 `cmd/cjv/main.go` 的 `run` 起步：`logging.Init` 配好日志，程序名是 `cjv` 不是某个工具名，于是走 `cli.Execute`。cobra 把 `install` 子命令路由到 `internal/cli/install.go` 的 `runInstall`。`runInstall` 把 `--target`、`--component`、`--force` 装进 `lifecycle.InstallRequest`，组好 `lifecycle.Options`（进度报告，以及首次安装时配置 PATH 的选择），调 `lifecycle.Install`。

`lifecycle` 编排其余步骤：`OpenDistribution` 读设置、选分发源、定 host tuple，`Distribution.Resolve` 让 `dist.Source` 从 manifest 解析通道、版本和平台，`placeToolchain` 把归档下载、解包到 staging 目录并校验，最后由 `fstx` 事务替换、`reachable` 建立受管二进制与代理链接、`component` 装上请求的组件。所有通道共用这条安装路径。CLI 按输出模式选择进度报告方式，并用本次调用的 renderer 渲染命令结果；错误由 `cli.Execute` 输出，再由 `main` 翻译成退出码。

代理路径是另一条主线。运行 `cjc build` 时，被调用的其实是名为 `cjc` 的 cjv 链接，`main` 认出工具名走 `proxy.Run`：`proxy` 经 `env.ResolveRuntime` 让 `resolve` 定出活动工具链、经 `sdktools` 在工具链目录里找到真正的 `cjc`、组装好运行环境，然后按平台替换当前进程或运行子进程。这条线绕过 cobra 命令树，保留工具的标准流和退出语义。

想深入某一块，从这几处入手最快：命令定义看 `internal/cli/root.go`，安装编排看 `internal/lifecycle/install.go`，分发源看 `internal/dist/source.go`，落盘事务看 `internal/lifecycle/resolved_install.go`，代理看 `internal/proxy/proxy.go`。测试怎么组织见[测试](testing.md)。
