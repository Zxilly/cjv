# 测试与检查

请根据改动涉及的范围选择对应的验证命令。CLI 参数测试验证标志解析和输出格式；安装与更新行为通过 `lifecycle` 的真实生产入口进行测试，使用本地 `httptest.Server` 提供测试清单和归档。常规单元测试和集成测试不依赖线上分发源。

## Go

在仓库根目录运行常规构建、测试与静态检查套件：

```bash
go build ./...
go test -race -count=1 -timeout 300s ./...
go vet ./...
golangci-lint run
```

本地开发迭代时，可以把 `./...` 换成受影响的包路径。`-race` 启用数据竞态检测，`-count=1` 禁用测试缓存。修改后的 Go 文件请使用 `gofmt` 格式化，并用 `gofmt -l .` 检查是否有未格式化文件。静态检查规则定义在 `.golangci.yml` 中。

修改下载逻辑、自更新或 `config` / `selfupdate` 的共享接口时，请同时验证 `mirror` 构建：

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

CLI 端到端集成测试会编译并在隔离子进程中运行实际的 `cjv` 二进制：

```bash
go test -race -tags integration -count=1 ./tests/integration/
```

编写测试时，应始终隔离用户主目录、`CJV_HOME` 和系统后备设置（使用 `configtest`）。请注意，只修改 `CJV_HOME` **不会**隔离用户的 `~/.cjv/settings.toml`。`PATH` 测试在 Unix 上修改临时 shell 配置文件，在 Windows 上使用注册表守卫保存和恢复用户 `PATH`。

## 回归测试放在哪里

请将回归测试放在负责该行为的包中：

| 行为 | 测试位置与断言 |
| --- | --- |
| 参数、重复调用、JSON | `internal/cli` 及各 `*cmd` 命令包：真实命令树（`clitest.Root`）、独立 writer、单份结果或错误 |
| 安装、更新、回退、卸载 | `internal/lifecycle`：本地模拟分发源、最终目录内容、设置引用及失败恢复 |
| 主机与目标隔离 | `internal/toolchain`、`internal/lifecycle`：安装身份、宿主平台及发行版精确匹配 |
| 组件所有权 | `internal/component`：与 SDK 共享的文件、越界清单拒绝、符号链接及备份恢复 |
| 进程中断与日志 | `internal/fstx`、`lifecycle` 回归：子进程异常退出后的真实日志重放恢复 |
| 续传、取消、重试 | `internal/dist`：实际 HTTP 请求、`Range` 请求头、SHA-256 哈希及请求次数 |
| 环境与进程 | `internal/env`、`internal/process`：显式传入环境、真实子进程退出码和信号转发 |

测试应断言用户可观察的结果。进度事件请使用 `testutil.ProgressRecorder` 检查事件种类，不依赖可能随语言变化的翻译文本。测试恢复失败时检查备份文件仍在；测试拒绝危险路径时检查 `<CJV_HOME>` 外部文件未被改动。

## 安装脚本与前端

运行 Unix 安装脚本测试套件：

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

运行 PowerShell 安装脚本测试套件：

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"
./tests/install-scripts/install-ps1.ps1
```

CI 还会覆盖 `sh`、`zsh`、`fish` 和 Windows PowerShell 5.1。

前端测试在 `web/` 目录下运行：

```bash
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm build
pnpm coverage
pnpm test:built
```

Vitest 通过 Playwright 使用真实浏览器运行测试：

- `VITEST_BROWSER` 可选择 `chromium`、`firefox` 或 `webkit`。
- `pnpm test:integration` 在每种浏览器中遍历运行全部模拟平台与架构用例。
- `pnpm coverage` 合并运行单元测试并检查 `web/vitest.config.ts` 中的覆盖率阈值。
- 构建完成后运行 `pnpm test:built`，使用真实浏览器的平台信息验证安装入口、字体下载挂起时的内容可见性，以及桌面端和窄屏移动端的布局溢出（可通过 `PLAYWRIGHT_BROWSER` 选择浏览器）。

## 文档、工作流和依赖

提交 Pull Request 前，请在仓库根目录执行文档、工作流和依赖校验：

```bash
npx --yes markdownlint-cli2@0.22.1
python3 scripts/build-docs.py
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
go mod verify
```

修改依赖后请运行 `go mod tidy` 并审查 `go.mod`、`go.sum` 差异；CI 会再次执行 `go mod tidy` 并要求工作区无变化。漏洞扫描使用 `.github/workflows/ci.yml` 中固定的 `govulncheck` 版本并在当前稳定 Go 上运行；升级扫描器时请同时检查本项目支持的 Go 版本。

共享文档构建脚本 `scripts/build-docs.py` 会构建全部四份中英文手册，并自动检查所有内部页面、跨书 `/book/` 链接、静态资源和 `#anchor` 标题锚点；PR 检查和线上发布均使用同一入口，详见[文档站](documentation.md)。

## 真实下载与 CI

### 鸿蒙模拟器测试

独立的 `.github/workflows/hmos.yml` 流程在 x86_64 Linux KVM runner 上启动官方 HarmonyOS 6.1.1（API 24）PC 模拟器。它先交叉编译独立的自签名单元测试二进制，分别以未签名和签名形式执行，再通过 `go test -exec` 在模拟器中运行 `ohos`、`config`、`env`、`fsops`、`target`、`cjverr`、`i18n` 和 `retry` 包的完整测试套件。其余包（包括 HTTP 服务测试）继续由主机 CI 覆盖。该流程仅执行 x86_64 二进制，不执行 ARM64 二进制。

`scripts/harmonyos/` 下的两个工具负责协调模拟器运行：

- **`scripts/harmonyos/ci.py`** 负责流程的三个阶段：`build` 编译主机工具，`install` 安装固定版本的模拟器和 Ubuntu 依赖，`test` 执行未签名、签名、退出码及包测试。它通过 Python 的 `subprocess` 直接调用程序，不使用 Bash 编排。完成 `build` 后，运行 `python3 scripts/harmonyos/emulator.py python3 scripts/harmonyos/ci.py test` 即可在本地复现 CI 的测试序列。
- **`scripts/harmonyos/exec`** 是供 `go test -exec` 使用的主机 Go 程序：它通过 HDC 传入测试二进制，将包目录映射到模拟器内的源码目录，为每次运行分配独立的 `HOME` 和 `TMPDIR`，并返回模拟器内的退出码。运行时需设置 `HDC`、`HDC_TARGET`、`HMOS_TEST_SOURCE` 和 `HMOS_TEST_ROOT`（后者必须位于 `/data/local/tmp` 下）。设置 `HMOS_SIGN` 时会在传输前签名私有副本，留空则测试未签名执行。`scripts/harmonyos/emulator.py` 负责准备模拟器内的源码与环境变量，运行传入的命令，并在结束后停止模拟器。

将官方 CLI 解压到 `HMOS_EMULATOR_HOME`，并按[从源码构建](building.md#openharmony)准备好工具后，可在本地运行模拟器包测试：

```bash
go build -o /tmp/cjv-hmos-exec ./scripts/harmonyos/exec
python3 scripts/harmonyos/emulator.py env GOOS=openharmony GOARCH=amd64 CGO_ENABLED=0 \
  "$HMOS_GO" test -exec /tmp/cjv-hmos-exec -p 1 -count=1 ./internal/ohos/...
```

在该模拟器镜像中，HDC shell 使用 `uid 2000` 且 SELinux 为 `Enforcing`：它允许运行未签名程序和符号链接，但拒绝创建硬链接。因此只有必须成功创建硬链接的测试会在确认权限错误（`EPERM`/`EACCES`）后跳过，而文件去重测试仍会验证复制回退行为。由于模拟器 HDC shell 的结果不能证明真机 HiShell 的签名策略相同，正式发布和 SDK 解包准备仍保留仅在鸿蒙启用的自签名。

### 在线下载与 CI 范围筛选

线上组件下载测试独立于常规测试，会从上游下载真实制品：

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

`.github/workflows/smoke.yml` 定时执行该冒烟测试。主工作流 `.github/workflows/ci.yml` 在每次 PR 和 `master` 推送时触发，通过 `scripts/ci-scope.py` 按变更路径选择较重的任务；工作流文件或范围规则变更会运行全部检查。同一 PR 的连续更新会自动取消旧运行，`master` 推送则不会取消。

前端常规 CI 执行三个浏览器的模拟平台用例，以及 Linux/Windows Chromium 和 macOS WebKit 的真实构建页面检查。手动触发或每周定时 CI 会运行全部任务，并补充原生 Linux ARM64、Windows ARM64 和 macOS Intel 的真实页面检查。HarmonyOS 在其独立工作流中使用相同的路径分类规则和 PR 取消策略。

发布辅助脚本的隔离夹具测试可在装有 `bash`、`zip`、`unzip`、`tar` 和 `sha256sum` 的环境中运行：

```bash
python3 scripts/test_ci_scope.py
bash tests/release-scripts/release-scripts.sh
```
