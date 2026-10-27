# 从源码构建

构建所需的 Go 版本以 `go.mod` 为准。Go CLI 可以独立于 `web/` 前端构建；生成的平台清单和嵌入的多语言资源已提交到仓库。

```bash
go build ./...
go build -o cjv ./cmd/cjv
```

在 Windows 上请使用 `go build -o cjv.exe ./cmd/cjv`。第一条命令检查所有 Go 包是否正常编译，第二条命令生成独立可执行文件。如果要把当前工作区的代码直接安装到 `GOBIN`，可以运行 `go install ./cmd/cjv`。

在本地交互式调试改动时，建议将 `CJV_HOME` 和 `CJV_CONFIG_FILE` 指向一个临时目录，这样既能完整测试 `cjv init` 和工具链安装，又不会影响本机已有的 `cjv` 环境：

```bash
mkdir -p /tmp/cjv-dev
CJV_HOME=/tmp/cjv-dev CJV_CONFIG_FILE=/tmp/cjv-dev/settings.toml ./cjv init --no-modify-path -y
```

## 版本信息与镜像构建

普通本地构建的版本号显示为 `dev`，并在运行时自动跳过自更新版本比较。正式发布时，GoReleaser 会通过 `-ldflags` 注入 `main.version` 和 `main.updateURL`；你在本地也可以显式注入版本号，或者使用 `-tags=mirror` 构建 GitCode 镜像版：

```bash
go build -ldflags "-X main.version=0.0.0-local" -o cjv ./cmd/cjv
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

`mirror` 构建标签会选择 `internal/config/manifest_*.go` 与 `internal/selfupdate/update_*.go` 中的 GitCode 默认版本清单地址和自更新实现。修改 `config` 或 `selfupdate` 的共享接口时，请务必同时验证普通版和 `mirror` 版两种构建。

## 交叉构建与生成文件

使用 Go 的标准平台环境变量（`GOOS` 与 `GOARCH`）即可交叉编译。以 Bash 为例：

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

发布平台清单统一定义在 `internal/target/catalog.go` 中，包括：

- **Linux**：`amd64`、`arm64`
- **macOS（`darwin`）**：`amd64`、`arm64`
- **Windows**：`amd64`
- **OpenHarmony（`openharmony`）**：`amd64`、`arm64`

修改平台清单后，请在仓库根目录运行：

```bash
go generate ./...
```

随后检查生成的 `web/src/generated/platforms.ts` 前端平台列表，并同步核对 `.goreleaser.yml` 发布配置和 CI 矩阵的一致性。构建后的行为验证见[测试与检查](testing.md)。

## OpenHarmony

编译 OpenHarmony 目标程序（`GOOS=openharmony`）需使用 [Go-HMOS `go1.27.2-hmos.5`](https://github.com/ZxillyFork/go-hmos-build/releases/tag/go1.27.2-hmos.5) 提供的 Linux/amd64 工具链，并设置 `CGO_ENABLED=0`。

在仓库根目录设置工具路径，先用主机 Go 构建代码签名工具，再交叉编译并签名鸿蒙可执行文件：

```bash
export GOENV=off GOTOOLCHAIN=local
export HMOS_GO=/absolute/path/to/go-hmos/bin/go
export HMOS_SIGN=/tmp/cjv-hmos-sign
go build -tags=hmos_sign -o "$HMOS_SIGN" ./scripts/harmonyos
GOOS=openharmony GOARCH=arm64 CGO_ENABLED=0 "$HMOS_GO" build -o cjv ./cmd/cjv
"$HMOS_SIGN" cjv cjv
```

构建 x64 版本时使用 `GOARCH=amd64`；构建镜像版时增加 `-tags=mirror`。

主机签名工具（`scripts/harmonyos`）复用 `internal/ohos/selfsign`，负责检查 ELF 结构与 Go 构建信息，写入并校验 `.codesign` 签名段。传入 `--verify FILE...` 可单独检查已有二进制产物。

正式[发布](releasing.md)通过 `.goreleaser.yml` 构建普通版与 `mirror` 版，鸿蒙产物在打包归档和生成校验和之前完成签名；而在鸿蒙宿主上运行 `cjv` 下载安装仓颉 SDK 时，则由 `internal/dist` 的系统钩子自动对解压出的原生可执行文件完成自签名。
