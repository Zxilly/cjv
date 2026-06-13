# 从源码构建

cjv 是一个标准的 Go 模块，没有代码生成步骤，也不依赖外部构建系统。装好 Go 工具链、克隆仓库，就能直接构建。

## 前置条件

go.mod 中声明的 Go 版本是构建 cjv 的下限：

```toml
module github.com/Zxilly/cjv

go 1.26.0
```

用 `go version` 确认本机版本不低于这里写的版本。CI 在 `actions/setup-go` 里用的是 `go-version: stable`，跟着最新的稳定版走，所以本地用当前稳定版即可。

落地页（`web/`）是一个独立的前端项目，不会被编译进 cjv 二进制，构建 Go 部分时不需要它。本地化文案（`internal/i18n/locales/*.toml`）通过 `go:embed` 嵌入，已随仓库提交，构建前无需额外生成。落地页的开发见 [落地页](web.md)。

## 构建所有包

在仓库根目录编译全部包，确认整个代码库能通过编译：

```bash
go build ./...
```

这条命令不产出可执行文件，只检查每个包是否构建成功，等同于 CI 里 `go-test` job 的 `Build all packages` 步骤。

## 构建二进制

二进制入口是 `cmd/cjv`，是仓库里唯一的 `main` 包。构建本机平台的可执行文件：

```bash
go build -o cjv ./cmd/cjv
```

Windows 上输出名带 `.exe`：

```bash
go build -o cjv.exe ./cmd/cjv
```

不带 `-o` 时，`go build ./cmd/cjv` 会在当前目录生成名为 `cjv`（或 `cjv.exe`）的文件。

直接安装到 `GOBIN`（默认 `$(go env GOPATH)/bin`）：

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

`@latest` 拉取最新打过 tag 的版本；想装某个具体版本就把它换成对应的 tag，想装当前工作区的代码就在仓库内执行 `go install ./cmd/cjv`。这样装出来的二进制是「裸」的，不带版本信息，原因见下一节。

## 版本信息

`cmd/cjv/main.go` 里有两个包级变量，靠链接期注入：

```go
var (
	version   = "dev"
	updateURL string
)
```

普通 `go build` / `go install` 不会设置它们，所以 `version` 保持 `dev`，`cjv --version` 会显示 `dev`。正式版本由 goreleaser 在发布时通过 `-ldflags -X` 注入，规则写在 `.goreleaser.yml`：

```yaml
ldflags:
  - -s -w
  - -X main.version={{.Version}}
  - -X main.commit={{.Commit}}
  - -X main.date={{.Date}}
  - -X main.updateURL=https://github.com/Zxilly/cjv/releases
```

`updateURL` 决定 `cjv update` 自更新时去哪个 release 页面找新版本，默认指向 GitHub。本地手动注入版本号也是一样的写法：

```bash
go build -ldflags "-X main.version=$(git describe --tags)" -o cjv ./cmd/cjv
```

发布流程的完整说明见 [发布流程](releasing.md)。

## mirror 变体

cjv 有一个 mirror 构建变体，给 GitHub 访问不稳定的环境用：它把默认数据源换成 GitCode 镜像。变体通过 `mirror` 构建标签切换，源码里成对的文件用构建约束区分：

```go
//go:build !mirror
```

```go
//go:build mirror
```

目前受标签影响的有两处。一是 SDK 清单地址 `internal/config/manifest_default.go`（GitHub raw）与 `internal/config/manifest_mirror.go`（GitCode raw）里的 `DefaultManifestURL`。二是自更新逻辑 `internal/selfupdate/update_default.go`（走 go-selfupdate 的 GitHub source）与 `internal/selfupdate/update_mirror.go`（自己实现的 GitCode source，通过跟随 `releases/latest` 的重定向来探测最新 tag）。

加上 `-tags=mirror` 构建 mirror 变体：

```bash
go build -tags=mirror ./...
```

```bash
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

CI 在 `go-test` job 里专门有 `Build mirror variant` 步骤跑 `go build -tags=mirror ./...`，并对 mirror 变体单独跑测试和 `go vet`，确保两套构建约束都能编译、不退化。goreleaser 也把它作为独立产物 `cjv-mirror` 发布，`updateURL` 指向 GitCode。

写带构建标签的代码时记住：默认构建用的是 `!mirror` 那一份，IDE 和 `go build ./...` 默认都不带 `mirror` 标签，加了 `mirror` 标签的文件平时不参与编译，容易在重构时被漏掉。改动其中一侧后，跑一遍 `go build -tags=mirror ./...` 验证另一侧也没坏。

## 交叉编译

cjv 是纯 Go、不依赖 cgo，交叉编译只需设置 `GOOS` 和 `GOARCH`：

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

CI 的 `cross-build` job 用一个矩阵覆盖所有发布目标，把产物丢到 `/dev/null` 只验证可编译：

```bash
GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build -o /dev/null ./cmd/cjv/
```

矩阵是 `goos` 取 `linux`、`darwin`、`windows`，`goarch` 取 `amd64`、`arm64`，并排除 `windows/arm64`。这五个组合就是 cjv 实际发布的平台集合，和 `.goreleaser.yml` 里 `goos` / `goarch` / `ignore` 的配置一致。本地想确认某次改动不会破坏交叉编译，照着这几组跑一遍 `go build` 即可。

## 验证

构建只是第一步。提交前还要跑测试、`go vet` 和 linter，对应的命令见 [测试](testing.md) 和 [代码检查与格式化](linting.md)。CI 会做什么、各个 job 怎么组织，见 [持续集成](ci.md)。
