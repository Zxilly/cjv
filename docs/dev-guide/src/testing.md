# 测试

cjv 的测试分成几层：Go 单元测试覆盖 `internal/` 下的各个包，Go 集成测试在 `tests/integration/` 编译出真正的 `cjv` 二进制并端到端地跑命令，安装脚本集成测试在多种 shell 下执行 `install.sh` 和 `install.ps1`，落地页用 vitest 的浏览器模式跑前端测试，smoke 测试每天定时去真实分发源拉一遍组件。每一层都有对应的 CI job，本章逐层说明怎么在本地跑、它们各自验证什么。

CI 的全部 job 定义在 `.github/workflows/ci.yml`，smoke 单独在 `.github/workflows/smoke.yml`。开发时不必把整套都跑一遍，本地通常只跑单元测试和你改动到的那一层。

## 单元测试

单元测试和被测代码放在同一个包里，文件名以 `_test.go` 结尾，分布在 `internal/` 各处，目前一百多个文件。它们用 `github.com/stretchr/testify` 的 `assert` 和 `require` 做断言，不依赖网络：需要分发源时用 `internal/testutil` 里的 mock 服务器（`MockDistServer`、`ValidMockServer`、`MockServerWithTargetSDKs`、`SplitNightlyMockServer` 等）起一个 `httptest.Server`，返回构造好的 manifest 和打包好的 mock SDK；`cli` 与 `lifecycle` 的测试共用这一套。

分发源回归测试也使用真实 `httptest.Server`，覆盖 `versions.json` 与按需加载的 `nightly.json`、相对与绝对 URL、nightly SDK/组件安装、SHA256 sidecar，以及旧聚合清单的迁移兼容。测试会断言 LTS 操作不会请求 nightly 文件。

跨职责的回归放在拥有对应生产流程的包里，重点包括：

- `cli`、`cli/settings`、`cli/selfmgmt`：每次构造独立命令，实际解析参数，验证重复执行时 JSON 模式、目录标志和卸载确认不会相互影响；已有 SDK 的重复安装在文本与 JSON 模式下都成功，组件添加输出完整 JSON 结果。`install` 与 `toolchain link` 在 `cli` 里只测标志解析、参数校验和结果渲染；安装本身的行为在 `lifecycle` 测。
- `config` 与 `cli/settings`：通过真实设置命令验证无关修改不会固定系统后备值、相同生效值可被显式固定，以及 `false`、空字符串和快照恢复的语义；无效的待保存配置内容会在发布前被拒绝，原文件和缓存保持不变。
- `env`：向 `Runtime` 传入与进程环境不同的基础环境，验证子进程、贡献视图和 shell 脚本遵守相同路径合并规则，并覆盖平台变量名、大小写和重复构造时的隔离。
- `lifecycle`：`install_test.go` 通过 `Install` 覆盖通道、裸版本、明确版本的解析，目标平台变体及其版本钉住、组件装到目标变体、manifest 只取一次、首次安装设默认与受管二进制、缺编译器时校验失败，以及 `Distribution.Resolve` 按 tuple 选可用版本；`resolved_install_test.go` 对 manifest、URL、本地归档三种来源验证同一条落盘流水线在最终步骤失败后的旧安装恢复与回滚错误传播；`link_test.go` 只测每种来源多出来的部分：CI 外层包与内层 stdx、跨 OS 拒绝、本地归档不被删除、目录链接原地引用且 finalize 失败即撤销；`component_install_test.go` 用提供组件制品的 `httptest` 服务器走生产安装路径，测组件批量安装的输入校验、目标 tuple 选择、回滚，以及代理路径按通道名解析已装工具链并把进度发给调用方的 sink。进度断言用 `testutil.ProgressRecorder` 记录的事件种类，不比较消息文本。升级与卸载测试覆盖 SDK/stdx/docs 及设置引用、匹配新版本的组件、链接来源、已有替代版本的选择、失败后的撤回与恢复错误；同名重装后组件仍可正确卸载。更新测试位于 `update_test.go`，覆盖通道名、目标平台变体和明确版本三种更新方式、已是最新与跳过自定义工具链的结果、全量更新中单个失败不中断以及下载暂存区清理；`cli` 只测试 `update` 命令的参数处理、自更新决定和 JSON 渲染。`cli` 里的 `init` 与 `component add` 测试同样走 mock 服务器的生产路径，没有组件安装的替身。
- `progress`：验证 `Text` 适配器按事件顺序输出 i18n 文本、非终端上不画下载进度条，以及 `Discard`；`dist` 的续传测试断言下载事件从续传偏移开始并累计到完整大小；`cli/output` 测试覆盖两种模式下的错误输出、部分成功的渲染和进度适配器的选择。
- `component`：验证归档安装、本地链接与组件批量修改失败后，文件和清单恢复；恢复本身失败时，备份仍可用于后续恢复。组件下载走生产路径 `InstallFromSource`，测试用 `httptest` 提供 manifest 并通过真实的 `dist.Source` 解析，断言的是实际请求到的制品链接。
- `fstx`：验证中断后的日志恢复和重复恢复、提交及准备发布状态、失败恢复后的备份保留，并检查异常日志和越界路径不会触发未授权的文件修改。
- `reachable`：单元测试覆盖 PATH 策略与可达性操作：在类 Unix 系统把 `HOME` 指向临时目录，验证 `ConfigurePath` 对每个 shell 配置只写一次标记块、`CJV_NO_PATH_SETUP=1` 时不写、`RemovePath` 能撤销、`Ensure` 只在策略要求时配置 PATH；Windows 用 `testutil.SaveRegistryPath` 守护后对注册表做同样的验证，只在 CI 上运行。`Ensure` 本身的测试覆盖受管二进制的保留与强制覆盖、代理链接和按需写 env 脚本。`lifecycle` 另验证首次发布默认工具链时按 `Options.ConfigurePath` 配置 PATH、后续安装不再改动。
- `selfupdate`：通过 `Update` 验证两种构建的版本发现、无需更新、校验失败及实际文件替换，保留运行中二进制替换测试；`cli/selfmgmt` 检查跳过更新的真实 JSON 状态及 stdout 无文本混入。
- `fsops`：树操作 `MoveTree`/`CopyTree` 在临时目录上验证嵌套目录、已有目标条目的覆盖与合并、内部相对符号链接的保留、逃出源树的符号链接被拒绝且不删除已有目标，以及 Linux 上借 `/dev/shm` 触发的跨文件系统复制回退；Windows 专有测试覆盖 junction 和瞬时错误的分类与删除重试。
- `process`：启动真实子进程，验证标准流、环境、启动错误、退出码和取消；Unix 专有测试另验证终止信号转发、超时升级及不重复转发终端中断。

本地跑全部单元测试：

```bash
go test -race -count=1 ./...
```

`-race` 打开竞态检测器，并发相关的 bug 大多在这里暴露。`-count=1` 关掉测试结果缓存，保证真的重新跑了一遍，而不是把上次的缓存结果打印出来。CI 的 `go-test` job 加了 `-timeout 300s`，在 Linux x86_64、Linux arm64、macOS arm64、macOS Intel、Windows 五个平台的矩阵上各跑一遍。`go-test` 还顺带跑 `go build ./...` 和 `go vet ./...`，确保所有包都能编译、`vet` 干净。

只跑某个包，把 `./...` 换成包路径：

```bash
go test -race -count=1 ./internal/config/
```

### mirror 变体

部分代码受 `mirror` build tag 控制(`internal/config/manifest_mirror.go`、`internal/selfupdate/update_mirror.go` 等)，给中国大陆镜像分发提供另一套实现。这条路径默认编译不进去，需要显式带 tag。CI 在 `go-test` 里单独验证 mirror 变体能编译、`selfupdate` 的 mirror 测试能过、`vet` 干净：

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

改动到 `*_mirror.go` 或它们依赖的接口时，记得带 `-tags=mirror` 再跑一遍，普通 `go test ./...` 不会编译这些文件。

## Go 集成测试

集成测试在 `tests/integration/`，所有文件开头都有 `//go:build integration`，所以默认的 `go test ./...` 看不到它们。这是有意为之：集成测试会现编译一个 `cjv` 二进制再反复调用，比单元测试慢得多，不该掺进日常的快速回路。

跑集成测试要显式带上 tag：

```bash
go test -race -v -tags integration -count=1 ./tests/integration/
```

这正是 CI `go-integration` job 在同样五平台矩阵上跑的命令。`TestMain`(在 `helpers_test.go`)会在所有用例之前用 `go build -o ... ./cmd/cjv` 把二进制编译一次并缓存，后续每个用例复用同一个产物，不会反复编译。每个用例用 `t.TempDir()` 拿到隔离的 `CJV_HOME`，把 mock server 的 `manifest_url` 写进 `settings.toml`，再把 cjv 二进制预先放进 `bin/`，然后真正执行 `cjv install`、`cjv default`、`cjv which`、`cjv completions` 这些命令并断言输出和落盘结果。调用时统一带 `CJV_NO_PATH_SETUP=1`，避免测试改到真实环境的 PATH。

平台相关的用例用组合 build tag 拆分。PATH 注入在类 Unix 系统改 shell 配置文件，在 Windows 改注册表，于是分成两个文件：

```go
//go:build integration && !windows
```

```go
//go:build integration && windows
```

Windows 上集成测试会读写注册表里的用户 PATH，`testmain_windows_test.go` 的 `runWithRegistryGuard` 在跑测试前用 `testutil.SaveRegistryPath` 存下当前值，跑完再 `Restore`，防止测试污染开发机的注册表。非 Windows 平台的同名函数是空壳，直接 `m.Run()`。

## 安装脚本集成测试

落地页提供的两个安装脚本 `web/public/install.sh` 和 `web/public/install.ps1` 有各自的集成测试，放在 `tests/install-scripts/`。这两个测试本身就是 shell 脚本，不是 Go 测试：它们在临时目录里真正跑一遍安装流程，然后断言文件、目录、PATH 配置都落到了预期位置。

Unix 侧是 `install-sh.sh`，通过环境变量选择用哪个 shell 跑被测脚本：

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

CI 的 `install-script-integration` job 用矩阵在 `sh`、`bash`、`zsh`、`fish` 下分别跑一遍，覆盖 Linux x86_64、Linux arm64、macOS arm64、macOS Intel；`zsh`、`fish` 这类不是默认装好的 shell 由 job 先用 `apt-get` 或 `brew` 装上。`CJV_INSTALL_TEST_MODE` 控制断言哪种安装模式。

Windows 侧是 `install-ps1.ps1`，用 `CJV_INSTALL_TEST_POWERSHELL` 选 PowerShell 宿主：

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"; ./tests/install-scripts/install-ps1.ps1
```

CI 在 Windows 上同时跑系统自带的旧版 `powershell.exe`(Windows PowerShell 5.1)和新版 `pwsh`(PowerShell 7+)，因为两者行为有差异；`CJV_INSTALL_TEST_DEFAULT_TOOLCHAIN` 控制默认安装哪些工具链。

## 落地页测试

落地页在 `web/`，用 vitest 跑，测试文件命名为 `src/**/*.test.{ts,tsx}`。和大多数前端项目不同，这里跑在 vitest 的浏览器模式而不是 jsdom：`web/vitest.config.ts` 里 `browser.enabled` 为 `true`，provider 是 `@vitest/browser-playwright`，用 Playwright 驱动真实浏览器。这样平台探测、`navigator.userAgentData` 这类只有真实浏览器才有的 API 才能被如实测到。浏览器由 `VITEST_BROWSER` 环境变量选择(`chromium`、`firefox`、`webkit`)，默认 `chromium`。

前端测试都在 `web/` 目录下，用 pnpm 跑。先装依赖和 Playwright 浏览器：

```bash
cd web
pnpm install
pnpm exec playwright install chromium
```

然后跑测试：

```bash
pnpm test
```

`pnpm test` 对应 `vitest run`，跑全部用例。它会按 `src/test-setup.ts` 做初始化：挂上 `@testing-library/jest-dom` 的断言、桩掉 `navigator.clipboard`、把 i18n 语言固定到中文(用例断言的是中文源串)。

平台探测有一组单独的集成测试 `src/App.platform.integration.test.tsx`，针对不同操作系统和架构验证落地页给出的下载建议是否正确。它通过 `pnpm test:integration` 跑：

```bash
pnpm test:integration
```

CI 把这组用例和其余用例分开：`web-test` job 用 `--exclude src/App.platform.integration.test.tsx` 跑其余测试，平台集成测试由 `web-browser-integration` job 在 chromium、firefox、webkit 三种浏览器乘以一整列真实 runner(各种 OS 加架构)的矩阵上跑，每个组合通过一组 `VITE_EXPECTED_*` 环境变量把该 runner 应当被识别成的平台、架构、下载标签注入进去，再由测试断言探测结果与之一致。这样能确认在真实的 Windows arm64、macOS Intel、Linux arm64 等环境里，平台识别和下载建议都对。

### 覆盖率

覆盖率用 vitest 的 v8 provider 收集：

```bash
cd web
pnpm coverage
```

`web/vitest.config.ts` 设了四项阈值，低于任一项即失败：

```ts
thresholds: {
  lines: 80,
  functions: 80,
  branches: 80,
  statements: 80,
}
```

覆盖率统计 `src/**/*.{ts,tsx}`，但排除测试文件本身、`test-setup.ts`、`test-utils.tsx`、`src/components/ui/**`(第三方 UI 组件)、`src/locales/**`、`src/main.tsx`、`src/vite-env.d.ts`。CI 的 `web-coverage` job 跑 `pnpm coverage`，阈值不达标 job 直接红。给落地页加逻辑时记得补上对应测试，否则可能把覆盖率压到阈值以下。

## Smoke 测试

smoke 测试在 `tests/smoke/`，带 `//go:build smoke` tag，默认不参与任何常规测试。它和集成测试的关键区别是：集成测试用 mock server，smoke 测试真的去线上分发源把 LTS 和 STS 的全部已知组件下载下来，验证清单 URL、文件哈希、解包流程在真实数据上仍然成立。这类测试慢且依赖外部服务，不适合放进 PR 回路。

它由 `.github/workflows/smoke.yml` 每天 UTC 08:00 定时跑一次，也可手动 `workflow_dispatch`。需要本地复现时：

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

下载量大，超时设到 45 分钟，CI 里还通过 `CJV_DOWNLOAD_TIMEOUT` 放宽单次下载超时、用 BBR 改善吞吐。日常开发一般不需要跑它，改动到组件下载、解包、清单解析这类直接面向真实分发源的代码时，跑一遍能提前发现线上数据格式的变化。

## 改了什么，跑什么

日常改 Go 代码，先跑 `go test -race -count=1 ./...`；碰了 `*_mirror.go` 就再带 `-tags=mirror` 跑一遍；改了命令行端到端行为或 PATH/proxy link 逻辑，跑一遍 `-tags integration` 的集成测试；动了 `install.sh` 或 `install.ps1`，在本地至少用一种 shell 跑对应的安装脚本测试；改了落地页，进 `web/` 跑 `pnpm test`，涉及平台探测再跑 `pnpm test:integration`，加了逻辑顺手 `pnpm coverage` 看阈值。CI 会把所有层在完整的平台矩阵上替你再跑一遍，但本地先过一遍能省下来回等 CI 的时间。各 job 与触发条件的全貌见 [持续集成](ci.md)。
