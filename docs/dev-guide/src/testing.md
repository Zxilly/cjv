# 测试与检查

根据改动选择验证范围。CLI 参数测试验证解析和输出；安装行为在 `lifecycle` 的真实生产入口测试；下载使用 `httptest.Server` 提供清单和归档。普通测试不依赖线上分发源。

## Go

在仓库根目录运行：

```bash
go build ./...
go test -race -count=1 -timeout 300s ./...
go vet ./...
golangci-lint run
```

开发时可把 `./...` 换成受影响包。`-race` 检测竞态，`-count=1` 禁用测试缓存。格式化改动文件使用 `gofmt`；`gofmt -l .` 检查未格式化文件。lint 规则在 `.golangci.yml`。

修改下载、自更新或共享接口时，同时验证 mirror：

```bash
go build -tags=mirror ./...
go test -tags=mirror -race -count=1 ./internal/selfupdate/...
go vet -tags=mirror ./...
```

CLI 端到端测试会编译并运行实际二进制：

```bash
go test -race -tags integration -count=1 ./tests/integration/
```

测试应隔离用户主目录、`CJV_HOME` 和后备设置。只改 `CJV_HOME` 不会隔离用户 `settings.toml`。PATH 测试修改临时 shell 文件；Windows 用注册表守卫保存和恢复 PATH。

## 回归测试放在哪里

| 行为 | 测试位置与断言 |
| --- | --- |
| 参数、重复调用、JSON | `internal/cli`：真实命令树、独立 writer、单份结果或错误 |
| 安装、更新、卸载 | `internal/lifecycle`：本地分发源、最终内容、设置引用及失败恢复 |
| 主机与目标隔离 | `internal/toolchain`、`internal/lifecycle`：安装身份、平台及发行版匹配 |
| 组件所有权 | `internal/component`：共享文件、越界清单、链接及备份恢复 |
| 进程中断 | `internal/fstx`、lifecycle 回归：子进程退出后的真实日志恢复 |
| 续传、取消、重试 | `internal/dist`：实际 HTTP 请求、Range、哈希及请求次数 |
| 环境与进程 | `internal/env`、`internal/process`：传入环境、真实子进程退出码和信号 |

测试应断言用户可观察的结果。进度用 `testutil.ProgressRecorder` 检查事件种类，不依赖翻译文本。恢复失败时检查备份仍在；拒绝危险路径时检查外部文件未改变。

## 安装脚本与前端

Unix 安装脚本：

```bash
CJV_INSTALL_TEST_SHELL=bash sh tests/install-scripts/install-sh.sh
```

PowerShell 安装脚本：

```powershell
$env:CJV_INSTALL_TEST_POWERSHELL = "pwsh"
./tests/install-scripts/install-ps1.ps1
```

CI 还覆盖 sh、zsh、fish 和 Windows PowerShell 5.1。前端在 `web/` 运行：

```bash
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm build
pnpm test
pnpm coverage
```

Vitest 使用真实浏览器，`VITEST_BROWSER` 可选择 chromium、firefox 或 webkit。平台集成测试用 `pnpm test:integration`，CI 提供各 runner 的 `VITE_EXPECTED_*` 值。覆盖率阈值由 `web/vitest.config.ts` 定义。

## 文档、工作流和依赖

```bash
npx --yes markdownlint-cli2@0.22.1
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
go mod verify
```

修改依赖后运行 `go mod tidy` 并审查 `go.mod`、`go.sum` 差异。CI 会再次 tidy 并要求无变化。漏洞检查的工具版本以 `.github/workflows/ci.yml` 为准；当前固定 `govulncheck@v1.3.0` 以避开 Go 1.26 兼容问题。

文档还需构建两个语言版本，见[文档站](documentation.md)。

## 真实下载与 CI

线上组件下载测试独立于常规测试，会下载真实制品：

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

`.github/workflows/smoke.yml` 定时执行该测试。主 `ci.yml` 在 PR 和 master 推送时检查 Go、mirror、五个平台的测试与构建、前端浏览器矩阵、安装脚本、lint 和依赖。平台和工具版本以工作流文件为准；站点与发布流程分别见[文档站](documentation.md)和[发布](releasing.md)。
