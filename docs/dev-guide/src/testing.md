# 测试与检查

请根据改动范围选择对应的验证命令。CLI 参数测试验证解析与输出格式，安装生命周期行为通过 `lifecycle` 的真实生产入口结合本地 `httptest.Server` 提供的清单与归档进行测试。常规测试不依赖公网分发源。各包职责边界见[代码架构](architecture.md)。

## Go 构建、测试与静态检查

### 常规 Go 检查

在仓库根目录执行：

```bash
go build ./...
go test -race -count=1 -timeout 300s ./...
go vet ./...
golangci-lint run
```

- 日常开发迭代时，可将 `./...` 替换为实际修改的包路径。
- `-race` 启用数据竞态检测，`-count=1` 禁用测试结果缓存。
- 修改的 Go 文件请使用 `gofmt` 格式化，并运行 `gofmt -l .` 检查是否有未格式化文件。静态检查规则定义在 `.golangci.yml` 中。

### 镜像构建（`mirror`）检查

修改下载逻辑、自更新或 `config` / `selfupdate` 的共享接口时，请同时验证 `-tags=mirror` 构建：

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

### CLI 端到端集成测试

端到端集成测试会编译实际的 `cjv` 二进制并在隔离子进程中运行：

```bash
go test -race -tags integration -count=1 ./tests/integration/
```

编写测试时，请始终隔离用户主目录、`CJV_HOME` 和系统后备配置（使用 `configtest` 辅助包）；注意仅修改 `CJV_HOME` 不会隔离用户的 `~/.cjv/settings.toml`。`PATH` 测试在 Unix 上使用临时 shell 配置文件，在 Windows 上通过注册表守卫保存并恢复用户 `PATH`。

## 回归测试放置位置

请将新增的回归测试放在负责该行为的模块中：

| 待测行为 | 测试包位置与断言要求 |
| --- | --- |
| 命令行参数、重复调用、JSON 输出 | `internal/cli` 及各 `*cmd` 子包：使用真实命令树（`clitest.Root`）与独立 `io.Writer`，断言单份结果或错误 |
| 安装、更新、回退、卸载 | `internal/lifecycle`：使用本地模拟分发源，断言最终目录内容、`settings.toml` 引用及故障恢复 |
| 宿主与交叉目标隔离 | `internal/toolchain`、`internal/lifecycle`：断言安装身份、宿主平台及精确发行版匹配 |
| 组件文件所有权 | `internal/component`：断言与 SDK 共享的文件、越界清单拒绝、符号链接处理及备份恢复 |
| 进程中断与事务日志 | `internal/fstx` 及 `lifecycle` 恢复测试：断言子进程异常退出后的真实日志重放恢复 |
| 断点续传、取消、重试 | `internal/dist`：断言实际 HTTP 请求、`Range` 请求头、SHA-256 校验与请求次数 |
| 运行时环境与子进程 | `internal/env`、`internal/process`：断言合并后的环境变量、真实子进程退出码与信号转发 |

测试应断言用户可观察的最终结果。进度事件请使用 `testutil.ProgressRecorder` 检查事件类型，不要匹配可能随语言变化的翻译文本。测试故障恢复时，需验证恢复受阻时备份文件仍然保留；测试危险路径拦截时，需验证 `<CJV_HOME>` 外部文件未被修改。

## 安装脚本与前端测试

### 安装脚本测试

在 Bash 中运行 Unix 安装脚本测试：

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

在 PowerShell 中运行 Windows 安装脚本测试：

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"
./tests/install-scripts/install-ps1.ps1
```

CI 还会覆盖 `sh`、`zsh`、`fish` 和 Windows PowerShell 5.1。

### 前端落地页测试

在 `web/` 目录下执行：

```bash
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm build
pnpm coverage
pnpm test:built
```

- Vitest 通过 Playwright 使用真实浏览器运行测试；可通过 `VITEST_BROWSER` 选择 `chromium`、`firefox` 或 `webkit`。
- `pnpm test:integration` 在每种浏览器中遍历测试全部模拟操作系统与架构组合。
- `pnpm coverage` 合并运行单元测试并检查 `web/vitest.config.ts` 中的覆盖率阈值。
- `pnpm test:built` 启动生产构建产物 `dist/`，使用真实浏览器验证安装入口、字体加载挂起时的内容可见性，以及桌面端和移动端窄屏的布局溢出；可通过 `PLAYWRIGHT_BROWSER` 切换浏览器。前端结构见[落地页](web.md)。

## 文档、工作流与依赖检查

提交 Pull Request 前，请在仓库根目录执行以下检查：

```bash
npx --yes markdownlint-cli2@0.22.1
python3 scripts/build-docs.py
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
go mod verify
```

- **文档与链接检查**：`markdownlint-cli2` 按 `.markdownlint-cli2.jsonc` 检查 Markdown 格式；`scripts/build-docs.py` 构建全部四份中英文手册，并校验所有内部页面、跨书 `/book/` 链接、静态资源和标题 `#anchor` 锚点（见[文档站](documentation.md)）。
- **Go 依赖与安全扫描**：修改依赖后请运行 `go mod tidy` 并审查 `go.mod` 与 `go.sum` 差异；CI 会再次执行 `go mod tidy` 并要求工作区无变化。漏洞扫描使用 `.github/workflows/ci.yml` 中固定版本的 `govulncheck`。
- **CI 范围与发布脚本夹具测试**：在装有 `bash`、`zip`、`unzip`、`tar` 和 `sha256sum` 的环境中运行：

```bash
python3 scripts/test_ci_scope.py
bash tests/release-scripts/release-scripts.sh
```

## 鸿蒙模拟器与在线下载测试

### 鸿蒙模拟器 CI 与本地复现

独立的 `.github/workflows/hmos.yml` 工作流在 x86_64 Linux KVM runner 上启动官方 HarmonyOS 6.1.1（API 24）PC 模拟器。它先交叉编译独立的自签名测试程序，分别以未签名和签名形式执行验证，再通过 `go test -exec` 在模拟器内运行 `ohos`、`config`、`env`、`fsops`、`target`、`cjverr`、`i18n` 和 `retry` 包的完整测试套件。其余包（包括 HTTP 服务测试）继续由主机 CI 覆盖。该模拟器流程仅执行 x86_64 二进制，不执行 ARM64 二进制。

`scripts/harmonyos/` 下的两个工具负责驱动模拟器测试：

- `scripts/harmonyos/ci.py` 通过 Python `subprocess` 直接管理三个步骤：`build`（编译主机工具）、`install`（安装固定版本的模拟器和 Ubuntu 依赖）和 `test`（执行未签名、签名、退出码及包测试）。完成 `build` 后，运行 `python3 scripts/harmonyos/emulator.py python3 scripts/harmonyos/ci.py test` 即可在本地复现 CI 测试序列。
- `scripts/harmonyos/exec` 是供 `go test -exec` 调用的主机 Go 程序：它通过 HDC 将测试二进制传入模拟器，将宿主包目录映射到模拟器内的源码目录，在 `/data/local/tmp` 下为每次运行分配独立的 `HOME` 和 `TMPDIR`，并透传模拟器内的退出码。运行时需设置 `HDC`、`HDC_TARGET`、`HMOS_TEST_SOURCE` 和 `HMOS_TEST_ROOT`（必须位于 `/data/local/tmp` 下）；设置 `HMOS_SIGN` 时会在传输前对私有副本签名，留空则测试未签名执行。

将官方 CLI 解压到 `HMOS_EMULATOR_HOME` 并按[从源码构建](building.md#openharmony)准备好工具后，可在本地运行单个包的模拟器测试：

```bash
go build -o /tmp/cjv-hmos-exec ./scripts/harmonyos/exec
python3 scripts/harmonyos/emulator.py env GOOS=openharmony GOARCH=amd64 CGO_ENABLED=0 \
  "$HMOS_GO" test -exec /tmp/cjv-hmos-exec -p 1 -count=1 ./internal/ohos/...
```

在该模拟器镜像中，HDC shell 以 `uid 2000` 运行且 SELinux 为 `Enforcing`：它允许运行未签名程序和创建符号链接，但拒绝创建硬链接。因此只有必须成功创建硬链接的测试会在确认权限错误（`EPERM` / `EACCES`）后跳过，而文件去重测试仍会验证复制回退行为。由于模拟器中 HDC shell 的策略不能替代真机 HiShell 的签名要求，正式发布和 SDK 解包准备阶段仍保留鸿蒙自签名。

### 在线组件下载冒烟测试与 CI 范围筛选

线上组件下载测试独立于常规单元测试，会从真实上游下载制品：

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

- `.github/workflows/smoke.yml` 按计划定时执行该冒烟测试。
- 主工作流 `.github/workflows/ci.yml` 在每次 PR 和 `master` 推送时触发，并通过 `scripts/ci-scope.py` 按变更文件路径选择需要运行的重型任务；修改工作流文件或范围规则本身会运行全部检查。
- 同一 PR 的连续推送会自动取消旧的运行，而 `master` 分支推送不会取消正在执行的任务。
- 手动触发或每周定时任务会运行完整矩阵，并额外增加原生 Linux ARM64、Windows ARM64 和 macOS Intel 的真实页面检查。发布自动化详见[发布流程](releasing.md)。
