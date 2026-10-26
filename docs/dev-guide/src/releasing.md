# 发布流程

向仓库推送符合 `vX.Y.Z` 格式的稳定版本标签会触发 `.github/workflows/release.yml` 发布工作流；其他以 `v` 开头的标签虽然也会启动该工作流，但会在第一步格式校验处直接失败。本章介绍自动发布顺序、发布产物规格、OpenHarmony 构建与签名、网页独立安装器提取以及本地预演验证。

## 自动发布顺序

推送 `vX.Y.Z` 标签后，`.github/workflows/release.yml` 按以下四个阶段顺序执行：

1. **校验并同步标签**：校验 `vX.Y.Z` 标签格式，并将发布提交与标签推送到 GitCode 镜像仓库。
2. **构建并发布 GitHub Release**：运行 GoReleaser（`.goreleaser.yml`）构建所有平台的普通版（`cjv`）和镜像版（`cjv-mirror`），对鸿蒙二进制执行自签名，并将归档与 `checksums.txt` 上传到 GitHub Releases。
3. **同步 GitCode Release 制品**：调用 `sync-gitcode-release.yml` 将全部发布归档和 `checksums.txt` 同步上传到 GitCode Releases。
4. **更新网站安装器与文档**：在 GitHub 和 GitCode 两端发布均完成后，调用 `.github/workflows/pages.yml` 更新网站 `/dl/` 安装器、[落地页](web.md)和[文档站](documentation.md)。

日常分支与标签同步由 `.github/workflows/mirror.yml` 处理。如果第 3 步因临时网络抖动失败，可以通过 `sync-gitcode-release.yml` 的手动触发入口（`workflow_dispatch`）单独重试制品同步。

## 发布产物规格

`.goreleaser.yml` 为每个受支持平台（Linux、macOS、OpenHarmony 的 `amd64` 与 `arm64`，以及 Windows `amd64`）同时构建两套二进制：

- **普通版（`cjv`）**：默认从 GitHub 获取版本清单和自更新。
- **镜像版（`cjv-mirror`）**：带 `-tags=mirror` 编译，默认从 GitCode 获取版本清单和自更新（清单中显式写出的绝对制品 URL 仍按原样访问）。

| 产物类型 | 命名格式 |
| --- | --- |
| 普通版归档 | `cjv_<os>_<arch>.tar.gz`（Windows 为 `.zip`） |
| 镜像版归档 | `cjv-mirror_<os>_<arch>.tar.gz`（Windows 为 `.zip`） |
| 校验和文件 | `checksums.txt`（包含全部 14 个平台归档的 SHA-256） |
| 归档内可执行文件 | `cjv` 或 `cjv-mirror`（Windows 附加 `.exe`） |

发布构建通过 `-ldflags` 注入 `main.version` 和 `main.updateURL`。如果需要修改归档命名规则或增减发布平台，必须同步检查 `.goreleaser.yml`、`internal/target/catalog.go`（运行 `go generate ./...`）、`web/public/install.sh`、`web/public/install.ps1`、`scripts/extract-init-binaries.sh` 以及 CI 工作流。

## OpenHarmony 构建与签名

发布构建期间，`.github/actions/setup-hmos` 会安装 [Go-HMOS `go1.27.2-hmos.5`](building.md#openharmony) 工具链（`HMOS_GO`）并编译主机签名工具（`HMOS_SIGN`）。GoReleaser 使用 `HMOS_GO` 交叉编译 `openharmony_arm64` 和 `openharmony_amd64`，并在打包归档和计算 `checksums.txt` 之前调用 `HMOS_SIGN` 完成签名。

签名后的鸿蒙归档会一并同步到 GitCode Releases，并由 Pages 工作流提取为网页安装器。

## 网页独立安装器（`/dl/`）

发布工作流会将当前标签传给 Pages 工作流；普通网站部署则在下载前解析一次最新稳定标签（手动运行 Pages 时也可传入 `release_tag`）。随后，`scripts/download-release-archives.sh` 从该标签下载全部 14 个归档和 `checksums.txt`，并执行：

```bash
ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

`extract-init-binaries.sh` 依次执行以下两步：

1. 对照 `checksums.txt` 严格校验全部 14 个归档的 SHA-256；如果存在缺失、重复、格式错误或哈希不匹配，脚本会在解包前直接报错终止。
2. 提取各归档中的可执行文件，统一重命名为 `cjv-init`（Windows 为 `cjv-init.exe`），输出到 `web/dist/dl/{official,mirror}/{os}_{arch}/`。

用户下载并运行 `cjv-init` 时，`cmd/cjv/main.go` 识别到 `cjv-init` 前缀后会自动进入 `cjv init` 安装向导（前缀匹配同样兼容浏览器重复下载时自动追加的后缀，如 `cjv-init (1).exe`）。与此同时，`install.sh` 和 `install.ps1` 直接下载发布归档并核对 `checksums.txt`。因此网页单文件安装器与脚本安装始终对应同一个发布版本。

## 本地预演与发布后确认

正式推送发布标签前，请先运行相关[测试与检查](testing.md)，并在本地执行 GoReleaser 快照预演：

```bash
goreleaser release --snapshot --clean
```

> **提示：** 本地快照预演包含 OpenHarmony 构建，请先按[从源码构建](building.md#openharmony)设置 `HMOS_GO` 与 `HMOS_SIGN` 并编译好主机签名工具。

快照构建会将产物写入 `dist/` 目录而不真正发布。推送正式 `vX.Y.Z` 标签并等待工作流全部成功后，请核对以下事项：

- [ ] GitHub Releases 与 GitCode Releases 均已生成新标签，且包含全部 14 个归档与 `checksums.txt`。
- [ ] 运行 `node deployment/verify-site.mjs https://cjv.zxilly.dev/ https://cjv.zxilly.com/` 验证落地页、双语手册、安装脚本与 `/dl/` 下载入口全部正常。
- [ ] 实际运行 `install.sh` 和 `install.ps1` 验证安装器能下载并安装最新的 `vX.Y.Z` 版本。
