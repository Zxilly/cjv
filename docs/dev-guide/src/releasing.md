# 发布流程

向仓库推送符合 `vX.Y.Z` 格式的稳定标签会触发 `.github/workflows/release.yml` 发布工作流。其他以 `v*` 开头的标签虽然也会启动工作流，但会在第一步标签格式检查处直接失败。

## 发布顺序

推送 `vX.Y.Z` 标签后，发布工作流按以下四个阶段依次执行：

1. 校验 `vX.Y.Z` 标签格式，并将发布提交及标签推送到 GitCode。
2. 使用 GoReleaser 构建所有平台的普通版（`cjv`）与镜像版（`cjv-mirror`），并上传到 GitHub Release。
3. 调用 `sync-gitcode-release.yml` 同步上传 GitCode Release 制品。
4. 两边 Release 都发布完成后，调用 `.github/workflows/pages.yml` 更新网站 `/dl/` 安装器、[落地页](web.md)和[文档站](documentation.md)。

日常分支与标签同步由 `.github/workflows/mirror.yml` 处理。如果第 3 步因临时网络问题导致制品同步失败，可使用 `sync-gitcode-release.yml` 的手动入口（`workflow_dispatch`）重试恢复，具体输入参数与凭据见工作流文件。

## 产物

`.goreleaser.yml` 定义了 `cjv` 和带 `-tags=mirror` 编译的 `cjv-mirror` 两套构建，每套发布 Linux、macOS、OpenHarmony 的 `amd64` 与 `arm64`，以及 Windows `amd64`：

| 项目 | 命名 |
| --- | --- |
| 普通归档 | `cjv_<os>_<arch>.tar.gz`，Windows 为 `.zip` |
| 镜像归档 | `cjv-mirror_<os>_<arch>.tar.gz`，Windows 为 `.zip` |
| 校验和 | `checksums.txt` |
| 归档内二进制 | `cjv` 或 `cjv-mirror`，Windows 加 `.exe` |

正式发布通过 `-ldflags` 注入版本号与自更新地址。`mirror` 构建切换默认的清单地址和自更新后端，而清单中显式写出的绝对制品 URL 仍按原样使用。

修改归档命名规则或增减支持平台时，请同步检查 `.goreleaser.yml`、安装脚本（`install.sh`、`install.ps1`）、`scripts/extract-init-binaries.sh`、前端生成平台数据（`go generate ./...`）和 CI 矩阵。

## OpenHarmony 构建与签名

发布构建时，`.github/actions/setup-hmos` 会安装 [`go1.27.2-hmos.5`](building.md#openharmony) 工具链（`HMOS_GO`）并构建主机签名工具（`HMOS_SIGN`）。GoReleaser 使用 `HMOS_GO` 构建 `openharmony_arm64` 和 `openharmony_amd64`，并在打包归档前调用 `HMOS_SIGN` 完成签名。生成的鸿蒙归档一并参与 GitCode 同步和 Pages 提取。

在本地运行 GoReleaser 快照预演前，请先按[从源码构建](building.md#openharmony)设置 `HMOS_GO`、`HMOS_SIGN` 并编译好主机签名工具。

## 网页安装器

发布工作流向 Pages 传入当前发布标签；普通网站部署则在下载前解析一次最新稳定标签（手动运行 Pages 也可指定 `release_tag`）。随后 `scripts/download-release-archives.sh` 从同一标签下载全部 14 个归档和 `checksums.txt`，并运行：

```bash
ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

该脚本会先对照 `checksums.txt` 校验全部 14 个归档的 SHA-256——任何缺失、重复、无效的校验项或内容不匹配都会在解包前直接报错失败。校验通过后，脚本提取二进制并重命名为 `cjv-init`（Windows 为 `cjv-init.exe`），输出到 `web/dist/dl/{official,mirror}/{os}_{arch}/`。程序启动时识别到 `cjv-init` 前缀后会自动进入 `init` 安装向导（名称前缀匹配同样兼容浏览器重复下载时自动添加的后缀，如 `cjv-init (1).exe`）。

与此同时，`install.sh` 和 `install.ps1` 直接下载发布归档并核对 `checksums.txt`。因此网页单文件安装器与脚本安装都绑定到同一次发布，只是下载的文件形式不同。

## 本地预演与确认

正式推送标签前，请先运行相关[测试与检查](testing.md)，并在本地执行快照预演：

```bash
goreleaser release --snapshot --clean
```

快照构建把产物写到 `dist/` 目录，不会真正发布 Release。发布前请检查归档内容与校验和；正式发布后，请确认 GitHub 和 GitCode 的标签与附件、运行 `node deployment/verify-site.mjs https://cjv.zxilly.dev/ https://cjv.zxilly.com/` 检查线上下载路径，并实测验证安装器能够下载对应的新版本。
