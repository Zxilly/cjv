# 从源码构建

Go 版本要求以 `go.mod` 为准。CLI 不依赖前端构建；已提交的平台生成文件和嵌入的本地化资源可直接参与编译。

```bash
go build ./...
go build -o cjv ./cmd/cjv
```

Windows 使用 `go build -o cjv.exe ./cmd/cjv`。第一条命令检查所有包，第二条生成可执行文件。要把当前工作区安装到 `GOBIN`，运行 `go install ./cmd/cjv`。

## 版本信息与镜像构建

普通构建的版本为 `dev`。发布时 GoReleaser 注入 `main.version` 和 `main.updateURL`；本地可显式指定：

```bash
go build -ldflags "-X main.version=0.0.0-local" -o cjv ./cmd/cjv
go build -tags=mirror -o cjv-mirror ./cmd/cjv
```

`mirror` 构建标签选择 GitCode 默认 manifest 和自更新实现。对应文件位于 `internal/config/manifest_*.go` 与 `internal/selfupdate/update_*.go`。修改共享接口时应验证普通和 mirror 两种构建。

## 交叉构建与生成文件

使用 Go 的平台变量交叉编译。Bash 示例：

```bash
GOOS=linux GOARCH=arm64 go build -o cjv ./cmd/cjv
```

发布平台来自 `internal/target/catalog.go`，包括 Linux 和 macOS 的 amd64/arm64，Windows amd64，以及 OpenHarmony amd64/arm64。修改平台清单后运行：

```bash
go generate ./...
```

检查生成的前端平台列表，并同步核对发布配置和 CI 矩阵。构建后的行为验证见[测试与检查](testing.md)。

## OpenHarmony

使用 [Go-HMOS go1.27.2-hmos.4](https://github.com/ZxillyFork/go-hmos-build/releases/tag/go1.27.2-hmos.4) 的 Linux/amd64 工具链，并设置 `CGO_ENABLED=0`。

在仓库根目录设置工具路径，用主机 Go 构建签名工具：

```bash
export GOENV=off GOTOOLCHAIN=local
export HMOS_GO=/absolute/path/to/go-hmos/bin/go
export HMOS_SIGN=/tmp/cjv-hmos-sign
go build -tags=hmos_sign -o "$HMOS_SIGN" ./scripts/harmonyos
GOOS=openharmony GOARCH=arm64 CGO_ENABLED=0 "$HMOS_GO" build -o cjv ./cmd/cjv
"$HMOS_SIGN" cjv cjv
```

构建 x64 版本时使用 `GOARCH=amd64`；mirror 版增加 `-tags=mirror`。

签名工具在主机上运行，复用 `internal/ohos/selfsign`，检查 ELF 和 Go 构建信息，添加并验证 `.codesign`。使用 `--verify FILE...` 检查已有产物。

发布通过 `.goreleaser.yml` 构建普通版与 mirror 版。鸿蒙产物在归档和校验和生成前签名。下载 SDK 的签名由 `internal/dist` 的系统钩子完成。
