# 从源码构建

本地编译 `cjv` 需要安装满足仓库 `go.mod` 要求的 Go 版本。由于平台生成文件和嵌入的本地化资源已提交到仓库，Go CLI 可以独立编译，无需先构建 `web/` 前端。本章介绍标准构建、版本注入、镜像版构建、交叉编译以及 OpenHarmony 签名流程。

## 编译标准 CLI

在仓库根目录运行以下命令：

```bash
go build ./...
go build -o cjv ./cmd/cjv
```

- 在 Windows 上，请使用 `go build -o cjv.exe ./cmd/cjv`。
- 第一条命令检查所有 Go 包是否能正常编译，第二条命令生成单文件可执行程序。
- 如需将当前工作区直接安装到 `GOBIN`，请运行 `go install ./cmd/cjv`。

## 版本信息注入与镜像构建

本地默认构建的版本号显示为 `dev`，并在运行时自动跳过自更新版本对比。正式发布时，GoReleaser 会通过 `-ldflags` 注入 `main.version` 和 `main.updateURL`。你也可以在本地显式指定版本号，或编译使用 GitCode 源的 `mirror` 变体：

```bash
go build -ldflags "-X main.version=0.0.0-local" -o cjv ./cmd/cjv
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

`mirror` 构建标签会选择 `internal/config/manifest_*.go` 和 `internal/selfupdate/update_*.go` 中的 GitCode 默认清单地址与自更新实现。修改 `config` 或 `selfupdate` 的共享接口时，请务必同时验证默认构建和 `-tags=mirror` 构建。

## 交叉编译与生成文件同步

使用 Go 标准环境变量（`GOOS` 与 `GOARCH`）即可跨平台交叉编译。例如在 Bash 中：

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

发布支持的平台清单定义在 `internal/target/catalog.go` 中：

- **Linux**：`amd64` 与 `arm64`
- **macOS（`darwin`）**：`amd64` 与 `arm64`
- **Windows**：`amd64`
- **OpenHarmony（`openharmony`）**：`amd64` 与 `arm64`

修改 `internal/target/catalog.go` 后，请运行以下命令更新派生文件：

```bash
go generate ./...
```

生成完成后，请检查 `web/src/generated/platforms.ts` 中的前端平台列表，并同步核对 `.goreleaser.yml` 发布配置与 CI 矩阵。构建后的测试验证命令见[测试与检查](testing.md)。

## OpenHarmony

编译 OpenHarmony 目标程序（`GOOS=openharmony`）时，请使用 [Go-HMOS go1.27.2-hmos.5](https://github.com/ZxillyFork/go-hmos-build/releases/tag/go1.27.2-hmos.5) 提供的 Linux `amd64` 工具链，并设置 `CGO_ENABLED=0`。

在 Linux x86_64 主机的仓库根目录下，先用主机 Go 构建签名工具，再交叉编译并签名鸿蒙可执行文件：

```bash
export GOENV=off GOTOOLCHAIN=local
export HMOS_GO=/absolute/path/to/go-hmos/bin/go
export HMOS_SIGN=/tmp/cjv-hmos-sign
go build -tags=hmos_sign -o "$HMOS_SIGN" ./scripts/harmonyos
GOOS=openharmony GOARCH=arm64 CGO_ENABLED=0 "$HMOS_GO" build -o cjv ./cmd/cjv
"$HMOS_SIGN" cjv cjv
```

- 编译 x86_64 鸿蒙版本时将 `GOARCH` 设为 `amd64`；编译镜像版时加上 `-tags=mirror`。
- 主机签名工具（`scripts/harmonyos`）复用 `internal/ohos/selfsign`，负责检查 ELF 结构与 Go 构建信息，写入并验证 `.codesign` 代码签名段。传入 `--verify FILE...` 可以校验已有的二进制文件。
- 在[发布流程](releasing.md)中，`.goreleaser.yml` 会在打包归档和生成校验和之前对鸿蒙产物签名。而在鸿蒙本机下载安装仓颉 SDK 时，则由 `internal/dist` 的平台钩子自动完成自签名。
