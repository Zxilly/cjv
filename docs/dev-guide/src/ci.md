# 持续集成

cjv 的所有自动化都放在 `.github/workflows/` 下，一共四个工作流：

- `ci.yml`：每次 push 到 `master` 和每个 pull request 都跑，是日常开发的主关卡。
- `pages.yml`：把落地页和两本 book 部署到 GitHub Pages。
- `release.yml`：打 `v*` tag 时触发，跑 GoReleaser 并镜像到 GitCode。
- `smoke.yml`：每天定时跑一次，验证真实的组件下载没有挂掉。

下面逐个说明，以仓库里的实际文件为准。

## ci.yml

触发条件是 push 到 `master` 分支或任意 pull request：

```yaml
on:
  push:
    branches: [master]
  pull_request:
```

工作流里的 job 互相独立，并行跑。所有 job 用 SHA 钉死 action 版本，Go 统一用 `go-version: stable`，Node 用 22，pnpm 用 latest。

### go-test

在五个 runner 上构建并测试 Go 代码：`ubuntu-24.04`、`ubuntu-24.04-arm`、`macos-26`、`macos-26-intel`、`windows-2025`，全部带 `-race`。每个 runner 上依次跑：

```bash
go build ./...
go build -tags=mirror ./...
go test -race -count=1 -timeout 300s ./...
go test -tags=mirror -race -count=1 -timeout 300s ./internal/selfupdate/...
go vet ./...
go vet -tags=mirror ./...
```

`mirror` 这个 build tag 切换的是面向中国大陆镜像的下载源，它只影响下载相关的代码路径，所以 mirror 变体的测试只跑 `./internal/selfupdate/...`，而构建和 vet 仍覆盖全部包。`-count=1` 禁用测试缓存，保证每次都真跑。

### lint

在 `ubuntu-24.04` 上跑 `golangci-lint`，用官方的 `golangci/golangci-lint-action`，`version: latest`。lint 规则由仓库根目录的配置文件决定，具体见 [代码检查与格式化](linting.md)。

### markdown-lint

用 Node 跑 `npx --yes markdownlint-cli2@0.22.1`，检查仓库里的 Markdown(包括两本 book 的源文件)。版本钉死在 `0.22.1`。

### cross-build

确认交叉编译在所有目标平台都能过。矩阵是 `goos` 取 `linux`、`darwin`、`windows`，`goarch` 取 `amd64`、`arm64`，排除 `windows/arm64`，所以一共五个组合。每个组合只编译主程序，产物丢弃：

```bash
GOOS=... GOARCH=... go build -o /dev/null ./cmd/cjv/
```

这一步只验证能编过，不跑测试。

### mod-check

保证模块声明是干净的。先 `go mod tidy`，再用 `git diff --exit-code go.mod go.sum` 检查有没有改动，有改动就说明本地忘了 tidy。然后 `go mod verify` 校验依赖完整性。

### vuln-check

`go install` 最新的 `govulncheck`，然后 `govulncheck ./...` 扫描已知漏洞。

### web-build

落地页的构建检查。用 `pnpm/action-setup` 装 pnpm，Node 22 开启 pnpm 缓存，缓存键挂在 `web/pnpm-lock.yaml` 上。然后在 `web` 目录里：

```bash
pnpm install --frozen-lockfile
pnpm build
```

`--frozen-lockfile` 保证 lockfile 和 `package.json` 一致，不会在 CI 里偷偷改依赖。落地页本身见 [落地页](web.md)。

### web-test

跑落地页的单元和组件测试，`timeout-minutes: 15`。装好依赖后会先解析出 Playwright 的版本号，用它做缓存键缓存 `~/.cache/ms-playwright`，再 `playwright install --with-deps chromium`。测试在 Chromium 里跑：

```bash
pnpm exec vitest run --exclude src/App.platform.integration.test.tsx
```

这里显式排除了 `src/App.platform.integration.test.tsx`，因为那个文件是跨平台跨浏览器的集成测试，交给下面的 `web-browser-integration` 在真实 runner 矩阵上跑。

### web-coverage

和 `web-test` 几乎一样的准备步骤(Playwright 缓存、装 Chromium)，最后跑 `pnpm coverage`，即 `vitest run --coverage`，生成覆盖率。同样 `timeout-minutes: 15`。

### web-browser-integration

job 名是 `Web integration (...)`，这是落地页里平台/架构探测逻辑的端到端验证。落地页要根据浏览器报告的操作系统和 CPU 架构，决定推荐哪个安装包、或者标记为不支持，这个 job 就是把它放到真实机器上核对。

矩阵有两维：`browser` 取 `chromium`、`firefox`、`webkit`；`runner` 是一组带期望值的对象，覆盖 Windows arm64、macOS intel/arm、Windows amd64、Linux amd64/arm64，以及伪装成 iOS、Android 的 Linux runner。每个 runner 对象带一组 `expected_*` 字段，比如 `expected_state`(`ready` 或 `unsupported`)、`expected_label`、`expected_binary_goos`/`expected_binary_goarch`。这些期望值通过 `VITE_EXPECTED_*` 环境变量传给测试：

```yaml
env:
  VITEST_BROWSER: ${{ matrix.browser }}
  VITE_EXPECTED_STATE: ${{ matrix.runner.expected_state }}
  VITE_EXPECTED_LABEL: ${{ matrix.runner.expected_label }}
  # ...其余 VITE_EXPECTED_* 同理
run: pnpm test:integration
```

`pnpm test:integration` 就是单跑 `src/App.platform.integration.test.tsx`。`fail-fast: false`，所以一个组合挂了不会带掉其它组合，`timeout-minutes: 30`。Linux 上装浏览器带 `--with-deps`，其它平台不带。

> 关于浏览器为什么读不准 macOS 的 CPU 架构、落地页又是怎么绕过这个限制的，见用户手册的[安装说明](https://cjv.zxilly.dev/book/user-guide/zh-CN/)。

### install-script-integration

job 名是 `Installer (...)`，验证 `install.sh` 和 `install.ps1` 两个安装脚本在各种 shell 下都能正常工作。矩阵按 `label` 展开，Unix 侧覆盖 Linux 的 `sh`/`bash`/`zsh`/`fish`、Linux arm64 的 `sh`、macOS arm 和 intel 的 `bash`/`zsh`/`fish`；Windows 侧覆盖老的 `WindowsPowerShell` 和新的 `pwsh`。缺的 shell 在 Linux 上用 `apt-get`、在 macOS 上用 `brew` 临时装。

测试入口是仓库里的两个脚本：

```bash
# kind == unix
sh tests/install-scripts/install-sh.sh
# kind == powershell
./tests/install-scripts/install-ps1.ps1
```

矩阵里的 `shell_name`、`test_mode`、`default_toolchain` 通过 `CJV_INSTALL_TEST_*` 环境变量传进去，让同一个脚本在不同 shell 和模式下复用。`fail-fast: false`，`timeout-minutes: 30`。

### go-integration

跑 Go 的集成测试，runner 矩阵和 `go-test` 一样(五个平台，全带 `-race`)。和单元测试分开，是因为它用 `integration` build tag 圈起来：

```bash
go test -race -v -tags integration -count=1 ./tests/integration/
```

`tests/integration/` 下是实打实拉起 cjv、操作真实文件系统和 PATH 的测试。测试分层的细节见 [测试](testing.md)。

## pages.yml

把落地页和两本 book 一起部署到 GitHub Pages。触发条件有三个：push 到 `master`、`workflow_call`(被 `release.yml` 调用)、`workflow_dispatch`(手动)。并发组 `pages` 且 `cancel-in-progress: false`，部署不会被新提交打断。权限上需要 `pages: write` 和 `id-token: write`。

只有一个 `deploy` job，环境是 `github-pages`，按顺序做这几件事：

1. 构建落地页：`pnpm install --frozen-lockfile` 后 `pnpm build`，产物在 `web/dist`。
2. 下载最新 release 的归档：用 `gh release download` 按 `cjv_*` 和 `cjv-mirror_*` 两个 pattern 拉下发布产物。
3. 抽取 init 二进制：跑 `scripts/extract-init-binaries.sh`，把各平台各变体的 `cjv-init` 二进制解到 `web/dist/dl`，供落地页直接分发。
4. 装 mdBook(`MDBOOK_VERSION: 0.5.3`，直接下预编译二进制)。
5. 构建两本 book，各出中英两版，都落到 `web/dist/book/<book>/<lang>/`：

```bash
# docs/user-guide
MDBOOK_OUTPUT__HTML__SITE_URL=/book/user-guide/zh-CN/ mdbook build -d ../../web/dist/book/user-guide/zh-CN
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/user-guide/en/ mdbook build -d ../../web/dist/book/user-guide/en
# docs/dev-guide
MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/zh-CN/ mdbook build -d ../../web/dist/book/dev-guide/zh-CN
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/en/ mdbook build -d ../../web/dist/book/dev-guide/en
```

book 必须在 `pnpm build` 之后再构建，因为 `pnpm build` 会清空 `web/dist`。最后把整个 `web/dist` 作为 Pages artifact 上传并部署。book 的多语言机制见 [文档站](documentation.md)。

## release.yml

打 `v*` tag 时触发，做正式发布。第一步 `Ensure stable tag` 用正则把 tag 限制为 `vX.Y.Z` 这样的稳定版本，带后缀的预发布 tag 会直接退出。权限是 `contents: write`，用于创建 GitHub Release。

`release` job 依次：

1. 用 `Zxilly/actions-bbr` 打开 TCP BBR，提升上传带宽。
2. `actions/checkout` 带 `fetch-depth: 0` 拿全部历史，GoReleaser 需要完整 tag 历史生成 changelog。
3. 把仓库镜像推到 GitCode(`git push gitcode`)，给中国大陆用户一份镜像源。
4. 跑 GoReleaser(`args: release --clean`)，配置见仓库根目录的 `.goreleaser.yml`，它负责交叉编译、打包、生成 checksum、创建 GitHub Release。
5. 用 `Zxilly/upload-gitcode-release` 把 mirror 变体的产物(`dist/cjv-mirror_*` 和 `dist/checksums.txt`)单独传到 GitCode release。

`release` 之后有一个 `pages` job，`needs: release`，通过 `uses: ./.github/workflows/pages.yml` 复用上面的 Pages 工作流，在发布完成后重新部署一次落地页，让它指向最新的 release 产物。发布的完整流程见 [发布流程](releasing.md)。

## smoke.yml

定时冒烟测试，确认真实的组件下载链路没有断。触发条件是 cron(每天 08:00 UTC)加手动 `workflow_dispatch`，并发组按 ref 分组且 `cancel-in-progress: true`。

只有一个 `real-component-downloads` job，在 `ubuntu-latest` 和 `ubuntu-24.04-arm` 上跑，`fail-fast: false`。它打开 BBR 后，用 `smoke` build tag 跑真实下载测试：

```bash
go test -v -tags smoke -run TestSmokeRealComponentDownloads_LTSSTS -count=1 -timeout 45m ./tests/smoke/
```

环境变量 `CJV_DOWNLOAD_TIMEOUT: "900"` 给单次下载 900 秒上限，整个 job 的超时放到 45 分钟。这个测试会去拉真实的 LTS/STS 组件，所以放在定时任务里而不是 PR 关卡上，避免上游波动影响日常开发，也避免每个 PR 都消耗大流量。
