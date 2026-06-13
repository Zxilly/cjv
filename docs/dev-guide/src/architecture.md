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

- 程序名是某个已知 SDK 工具（`cjc`、`cjpm` 等，由 `proxy.IsProxyTool` 判定），走代理路径 `proxy.Run`，把参数透传给真正的工具。
- 程序名以 `cjv-init` / `cjv-setup` 开头，把它当安装器，改写 `os.Args` 为 `cjv init` 再继续。
- 否则就是普通的 `cjv ...` 调用，交给 `cli.Execute(version, updateURL)`。

错误处理也集中在这里：实现层返回的 `*cjverr.ExitCodeError` 被解包成进程退出码，其余错误统一打到 stderr 并返回 1（JSON 模式下信封已由 `cli.Execute` 写到 stdout，stderr 保持干净）。Windows 控制台的 UTF-8 切换、双击运行时的暂停提示也都在 `main` 这层处理，因为它们是进程级的关切，不该渗进业务逻辑。

## `internal/` 各包职责

`internal/` 下每个目录是一个包，按子系统划分。下面按它们在一条命令里大致的依赖方向，从上层往下层列。

### `cli`：命令定义

`internal/cli` 是 cobra 命令树。`root.go` 定义根命令 `cjv` 和 `Execute` 入口：注册全局 `--json` 标志、把版本号塞给 cobra、挂上各子命令，然后 `rootCmd.Execute()`。每个子命令一个文件，`install.go`、`uninstall.go`、`toolchain.go`、`run.go`、`exec.go`、`which.go`、`show.go`、`check.go`、`update.go`、`component.go` 等，文件名基本能对上命令名。

`cli` 自己不实现业务逻辑，它做的是参数解析、调用下层包、把结果交给渲染层。几个子包分担横切关注点：

- `cli/output` 渲染命令结果。命令各自定义一个实现 `Result` 接口（一个 `Text()` 方法）的结构体，`output` 根据全局 `--json` 决定是调 `Text()` 输出人读文本，还是把结构体直接 marshal 成 JSON。错误的 JSON 信封也在这里组装，它认得 `cjverr` 的 `Coded` 接口来填机器可读的错误码。
- `cli/settings` 是 `cjv settings` 那组配置子命令（`set`、`default`、`override` 等）。
- `cli/selfmgmt` 是 `cjv self` 那组自我管理子命令（更新、卸载），以及提权安全检查。

### `lifecycle`：安装编排

`internal/lifecycle` 把一次工具链安装编排起来：下载、解压、校验、装组件、配 PATH、建代理链接，按顺序串成一条流程。它刻意不依赖 `cli`，而是通过一个 `Options` 结构体接收回调（`IsJSON`、`ComponentInstall`、`CreateProxyLinks`、`ValidateInstallation` 等），把展示和具体实现留在外面。这样同一套安装流程既能被 `cli install` 调用，也能被代理路径的自动安装复用，`cli` 在 `lifecycleOptions()` 里把这些回调接到 `output`、`component`、`proxy`、`selfupdate` 上。

### `resolve`：活动工具链解析

`internal/resolve` 回答“现在该用哪个工具链”。`Active` 综合命令行的 `+toolchain` 覆盖、`CJV_TOOLCHAIN` 环境变量、目录级与全局的 override、默认设置，定出活动工具链的名字和目录，连同它的目标平台和组件一起返回成 `ActiveToolchain`。解析过程中如果工具链没装，它能通过 `AutoInstallFunc` 这个测试缝触发自动安装；生产环境里这个缝默认接到 `lifecycle`，这样 `resolve` 不必反向依赖 `cli`。

### `toolchain` 与 `component`：已装内容的模型

`internal/toolchain` 管已安装的 SDK：列出已装工具链（`ListInstalled`）、解析活动工具链目录、清理 staging 与备份残留目录。它定义了 staging（`.staging`）、备份（`.old`）、事务（`.fstx-`）这些目录后缀约定，以及工具链名字的解析与版本比较。

`internal/component` 管工具链的附加组件：`stdx`、`docs`、`stdx-docs`。每个组件是单独下载的归档，解压后的文件通过逐组件的清单（manifest）记录，从而能独立卸载。`component` 还定义了组件装到哪（`InstallLocation`：有的落进工具链目录树，有的作为纯数据放到 `<CJV_HOME>/docs/<tc>/`）以及组件要注入哪些环境变量。

### `dist`：下载与解包

`internal/dist` 负责把 SDK 和组件从网上弄下来。`manifest.go` 解析版本清单（LTS / STS 两个通道，version -> platform -> 下载信息的嵌套结构）；`download.go` 做带进度条、重试和 SHA256 校验的下载；`install.go` 把归档解包到目标目录（`ExtractFlattened` 处理单层顶级目录的剥离）；`nightly.go` 处理 nightly 构建；`platform.go` 把 `(GOOS, GOARCH)` 和目标 tuple 映射到清单索引键与 nightly 文件名，底层委托给 `target`。

### `target`：平台身份

`internal/target` 是平台与目标 tuple 的单一事实源。它解析目标 tuple（host 部分加可选的交叉编译环境后缀），给出清单索引键、nightly 归档命名、stdx 平台 token 等结构化视图，免得每个调用方各自去切字符串。`catalog.go` 列出 cjv 出 host 二进制的全部 `(GOOS, GOARCH)` 组合，是发布产物和落地页下载入口的源头；它带 `go:generate` 指令，跑 `scripts/gen-platform-surfaces.go` 生成。

### `env`：运行时环境

`internal/env` 组装运行仓颉工具所需的环境。`Runtime` 把活动工具链和派生出的 SDK 环境包在一起，对外暴露几种窄视图：给代理子进程的环境、直接执行工具链用的环境、写进 shell 的环境。它处理 `LD_LIBRARY_PATH` / `PATH` 拼装（按平台分 `ldpath_unix.go` / `ldpath_windows.go`）、`SDKROOT`、shell 检测与各 shell 的脚本格式（`shelldetect.go`、`shell_*.go`、`shellformat.go`），是 `cjv env` 和代理执行共用的底座。

### `proxy`：透明代理

`internal/proxy` 实现透明代理：当二进制以 `cjc`、`cjpm` 等工具名被调用时，`Run` 解析活动工具链（经 `env.ResolveRuntime`）、在工具链目录里定位真正的工具二进制（`tools.go` 里 `toolPathMap` 是工具名到相对路径的映射）、组装代理环境、然后 `exec` 那个二进制把参数透传过去。它带一个递归计数器（`CJV_RECURSION_COUNT`），防止代理无限自调。`link.go` 负责在安装时建出这些代理链接（`CreateAllProxyLinks`）。

### `config`：配置与路径

`internal/config` 是配置层。它定义所有 `CJV_*` 环境变量名（`EnvHome`、`EnvToolchain`、`EnvLog` 等）、解析 `CJV_HOME`（区分来自环境变量、来自 `settings.toml`、还是默认 `<user-home>/.cjv`）、读写 `settings.toml` 与工具链文件、管理目录级 override。清单 URL 也在这里按 `mirror` 构建标记切换（`manifest_default.go` 走 GitHub，`manifest_mirror.go` 走镜像）。

### `selfupdate`：自我更新

`internal/selfupdate` 实现 `cjv self update`。具体走 GitHub 还是 GitCode 由 `mirror` 构建标记在编译期选定（`update_default.go` / `update_mirror.go`）。它还管理把当前二进制确立为受管可执行文件、以及更新时替换正在运行的二进制（Windows 与其他平台分 `replace_windows.go` / `replace_other.go`）。

### 支撑包

剩下几个是被各层共用的支撑包：

- `i18n` 国际化。消息存在 `locales/en.toml` 和 `locales/zh-CN.toml` 里并嵌进二进制，`i18n.T` 按消息 ID 取串。所有面向用户的文本都走它，错误信息也是。
- `cjverr` 错误类型。定义带稳定机器码（`ErrorCode`）的结构化错误，`Error()` 方法通过 `i18n` 产出人读信息，`Coded` 接口让 `output` 能在 JSON 模式下输出错误码。`ExitCodeError` 携带进程退出码。
- `fstx` 文件系统事务。把一组文件增删改包成可回滚的事务，工具链替换这类操作靠它保证失败时不留半成品。
- `utils` 杂项工具：原子写、文件操作、Windows junction、重试、控制台 UTF-8、打开浏览器、版本号解析等，多数按平台分文件。
- `logging` 用 `CJV_LOG` 环境变量配 `slog` 全局 logger（默认 `warn`）。
- `testutil` 测试辅助：mock 下载服务器、Windows 注册表守卫。它带 `_test.go` 之外的源文件，供其他包的测试导入。

## 一条命令的流向

把上面串起来，看 `cjv install <toolchain>` 大致怎么走。

进程从 `cmd/cjv/main.go` 的 `run` 起步：`logging.Init` 配好日志，程序名是 `cjv` 不是某个工具名，于是走 `cli.Execute`。cobra 把 `install` 子命令路由到 `internal/cli/install.go` 的 `runInstall`。`runInstall` 收集 `--target`、`--component`、`--force` 等标志，组好 `lifecycle.Options`（把 `output`、`component`、`proxy`、`selfupdate` 的实现接进去），调进 `internal/lifecycle`。

`lifecycle` 编排其余步骤：经 `config` / `target` 把请求的版本和平台解析成清单键，让 `dist` 去下载和校验归档、解包到 staging 目录，让 `component` 装上请求的组件，让 `proxy` 建出代理链接、相关的 PATH 配置就位，整个落地过程借 `fstx` 事务完成以便失败回滚。一路上的进度和结果通过 `output` 渲染（受 `--json` 控制），文本来自 `i18n`，出错则是 `cjverr` 的类型化错误，最终在 `main` 那层被翻译成退出码。

代理路径是另一条主线。运行 `cjc build` 时，被调用的其实是名为 `cjc` 的 cjv 链接，`main` 认出工具名走 `proxy.Run`：`proxy` 经 `env.ResolveRuntime` 让 `resolve` 定出活动工具链、在工具链目录里找到真正的 `cjc`、组装好运行环境，然后 `exec` 过去。这条线不碰 `cli`，也不渲染 cjv 自己的输出，纯粹把工具透传出去。

想深入某一块，从这几处入手最快：命令定义看 `internal/cli/root.go` 起，安装编排看 `internal/lifecycle/install.go`，代理看 `internal/proxy/proxy.go`。测试怎么组织见[测试](testing.md)。
