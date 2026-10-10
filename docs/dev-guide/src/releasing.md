# 发布流程

推送 `vX.Y.Z` 稳定 tag 触发 `.github/workflows/release.yml`。其他 `v*` tag 也会启动工作流，但在格式检查处失败。

## 发布顺序

1. 校验 tag，将发布提交及 tag 推送到 GitCode。
2. GoReleaser 构建普通版与 mirror 版，上传 GitHub Release。
3. 调用 `sync-gitcode-release.yml` 同步 GitCode Release 制品。
4. 两边发布完成后调用 Pages，更新安装器、落地页和文档。

普通分支与 tag 同步由 `mirror.yml` 处理。制品同步失败可使用 `sync-gitcode-release.yml` 的手动入口恢复，具体输入与凭据见工作流文件。

## 产物

`.goreleaser.yml` 定义 `cjv` 和带 `-tags=mirror` 的 `cjv-mirror`。每套发布 Linux/macOS/OpenHarmony 的 amd64、arm64，以及 Windows amd64。

| 项目 | 命名 |
| --- | --- |
| 普通归档 | `cjv_<os>_<arch>.tar.gz`，Windows 为 `.zip` |
| 镜像归档 | `cjv-mirror_<os>_<arch>.tar.gz`，Windows 为 `.zip` |
| 校验和 | `checksums.txt` |
| 二进制 | `cjv` 或 `cjv-mirror`，Windows 加 `.exe` |

发布通过 ldflags 注入版本与更新地址。mirror 切换默认 manifest 和自更新后端；清单中的绝对制品 URL 仍按清单使用。

修改归档命名或平台时，同时检查安装脚本、`scripts/extract-init-binaries.sh`、前端生成平台数据和 CI 矩阵。

## OpenHarmony 构建与签名

GoReleaser 使用 `HMOS_GO` 构建 `openharmony_arm64` 和 `openharmony_amd64`，在归档前调用 `HMOS_SIGN`。`.github/actions/setup-hmos` 安装 `go1.27.2-hmos.5` 并构建主机签名工具。归档参与 GitCode 同步和 Pages 提取。

本地快照先按[构建文档](building.md#openharmony)设置 `HMOS_GO`、`HMOS_SIGN` 并编译主机签名工具。

## 网页安装器

Pages 从最新 Release 下载归档，运行：

```bash
ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

脚本提取二进制并重命名为 `cjv-init[.exe]`，放到 `dl/{official,mirror}/{os}_{arch}/`。程序识别这个名称后进入 `init`；名称前缀匹配也兼容浏览器添加的下载副本后缀。

`install.sh` 和 `install.ps1` 直接下载发布归档并读取 `checksums.txt`。因此网页单文件安装器与脚本安装都依赖同一发布，但使用不同的下载文件。

## 本地预演与确认

```bash
goreleaser release --snapshot --clean
```

快照构建把产物写到 `dist/`，不发布 Release。发布前运行相关[测试与检查](testing.md)，检查归档内容与校验和。发布后确认 GitHub、GitCode 的 tag 和附件，以及 Pages 的平台下载路径；工作流成功之外还需验证安装器能下载对应版本。
