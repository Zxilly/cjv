# 发布流程

cjv 的发布由 git tag 驱动。给某个提交打上稳定版本 tag 并推送，`.github/workflows/release.yml` 就会用 goreleaser 构建全平台产物、发布到 GitHub Releases、把仓库和产物镜像到 GitCode，最后重新部署落地页和文档站。本章讲清楚这条链路上每一步做了什么，以及 `cjv-init` 二进制是怎么从 release 产物落到落地页的下载按钮上的。

## 版本号与 tag 约定

发布只认形如 `vX.Y.Z` 的稳定 tag。`release.yml` 的第一步会校验 tag 名，不符合就直接失败：

```yaml
- name: Ensure stable tag
  run: |
    if ! [[ "${GITHUB_REF_NAME}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "Only stable tags trigger a release"
      exit 1
    fi
```

也就是说 `v1.2.3` 会发布，而 `v1.2.3-rc1`、`v1.2`、`1.2.3`(缺 `v` 前缀)这类都会被拒。workflow 的触发条件是 `tags: ["v*"]`，范围比这个正则宽，所以预发布 tag 仍会启动 workflow，只是会在第一步就停下。

版本号通过 ldflags 注入二进制。goreleaser 给两个 build 都加了：

```yaml
ldflags:
  - -s -w
  - -X main.version={{.Version}}
  - -X main.commit={{.Commit}}
  - -X main.date={{.Date}}
```

其中 `{{.Version}}` 是去掉 `v` 前缀的 tag(例如 tag `v1.2.3` 对应 `main.version=1.2.3`)。`cmd/cjv/main.go` 里 `version` 默认是 `"dev"`，从源码直接 `go build` 时就是这个值；只有走 goreleaser 才会被替换成真实版本。`main.commit` 和 `main.date` 目前在 `cmd/cjv` 里没有对应的包级变量，这两条 `-X` 是空操作，留着是为以后接上版本信息预备的，现在不影响产物。

## goreleaser 配置

`.goreleaser.yml`(`version: 2`)定义了两套并行的产物，官方版和镜像版。

两个 build 的 `main` 都是 `./cmd/cjv`，源码完全一样，区别在编译参数：

```yaml
builds:
  - id: cjv
    binary: cjv
    ldflags:
      - -X main.updateURL=https://github.com/Zxilly/cjv/releases
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64

  - id: cjv-mirror
    binary: cjv-mirror
    flags:
      - -tags=mirror
    ldflags:
      - -X main.updateURL=https://gitcode.com/Zxilly/cjv/releases
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64
```

`cjv-mirror` 多了 `-tags=mirror`。这个构建标签会切换两处编译期常量：`internal/config/manifest_mirror.go` 把默认 SDK 清单地址换成 GitCode 的镜像，`internal/selfupdate/update_mirror.go` 让自更新走 GitCode。配合 `main.updateURL` 指向 GitCode，镜像版从下载安装、查清单到自更新整条链路都不碰 GitHub，供 GitHub 访问不畅的环境使用。官方版(`!mirror`)对应的默认实现在 `manifest_default.go`，指向 GitHub。

平台矩阵是 `linux`、`darwin`、`windows` 三个 OS 乘 `amd64`、`arm64` 两个架构，再 `ignore` 掉 `windows/arm64`，每套各 5 个目标，两套共 10 个二进制。

### 产物命名

```yaml
archives:
  - id: cjv
    ids: [cjv]
    name_template: "cjv_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
  - id: cjv-mirror
    ids: [cjv-mirror]
    name_template: "cjv-mirror_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
```

归档名是 `cjv_<os>_<arch>` 和 `cjv-mirror_<os>_<arch>`。Windows 打成 `.zip`，其余平台是默认的 `.tar.gz`。所以最终的文件名形如 `cjv_linux_amd64.tar.gz`、`cjv_windows_amd64.zip`、`cjv-mirror_darwin_arm64.tar.gz`。归档里的可执行文件名按 `binary` 字段定，官方版是 `cjv`(Windows 上 `cjv.exe`)，镜像版是 `cjv-mirror`(`cjv-mirror.exe`)。

校验和单独出一个文件：

```yaml
checksum:
  name_template: "checksums.txt"
```

`checksums.txt` 覆盖全部归档。`install.sh` 下载归档后会拉这个文件来校验 sha256；文件缺失或本机没有 sha256 工具时降级为告警，只有"能拿到校验和但对不上"才会判为失败并中止安装。

## release workflow

`.github/workflows/release.yml` 在推送 `v*` tag 时触发。正式发布只接受稳定版本 `vX.Y.Z`，整个流程由 CI 自动执行：

1. `release` job 校验 tag，启用 TCP BBR，拉取完整历史并设置 Go。
2. 推送发布 tag 到 GitCode，同时传输它指向的提交；不会把旧发布提交推成 GitCode 的 `master`。
3. GoReleaser 创建 GitHub Release，上传官方版和镜像版的全部 10 个归档及 `checksums.txt`。
4. `gitcode` job 通过 `workflow_call` 自动调用 `sync-gitcode-release.yml`，明确传入 tag 和 GitCode secret。它先检查 GitHub Release 已发布且六个所需资产完整，再确保 GitCode tag 存在。
5. 六个资产分别上传，每个 job 启用 BBR、限时 15 分钟，最多两路并行。五个镜像归档上传前用 GitHub 的 `checksums.txt` 校验 SHA-256；上传后重新下载 GitCode 上的资产，与 GitHub 原件逐字节比对。
6. `pages` job 等待 `release` 和 `gitcode` 都成功后，再复用 `pages.yml` 部署站点。

GitHub Release 上挂全部产物（官方版和镜像版），GitCode Release 上只挂五个镜像版归档及校验和。GitCode 的正文指向 GitHub Release 的完整 changelog。

独立的 `mirror.yml` 在 `master` 提交或任何 tag 推送时自动同步当前 `master` 和全部 tags。它始终检出 `master`，因此同步旧 tag 不会把镜像分支退回旧发布提交。

正常发布无需手动运行同步工作流。`sync-gitcode-release.yml` 的 `workflow_dispatch` 仅用于故障恢复或历史版本补传，填写稳定版 tag 后复用同一套校验及上传流程。失败的资产 job 可以独立重跑，无需重建二进制。

发布流程直接调用同步工作流，不依赖 GitHub Release 事件再次触发工作流，因为使用 `GITHUB_TOKEN` 创建的 Release 不会触发这类后续事件，详见 [GitHub 触发规则](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow)。

## cjv-init 如何落地

落地页上点"下载"拿到的不是某个单独构建的安装器，而就是 `cjv` 二进制本身换了个名字。`cmd/cjv/main.go` 在启动时看自己被以什么名字调用：文件名以 `cjv-init`(或 `cjv-setup`)开头时，它把自己当安装器跑，自动在参数前插入 `init` 子命令，并在独立控制台窗口里运行完后暂停等回车，方便从资源管理器双击使用。

```go
func isInitInvocation(toolName string) bool {
	return strings.HasPrefix(toolName, "cjv-init") || strings.HasPrefix(toolName, "cjv-setup")
}
```

前缀匹配是为了容忍浏览器重命名的副本，比如 `cjv-init(1)` 或 `cjv-init-2`。

所以发布产物里没有名叫 `cjv-init` 的东西，它是部署落地页时从 release 归档里现取现命名的。`pages.yml` 里这两步负责：

```yaml
- name: Download latest release archives
  run: |
    mkdir -p archives
    gh release download --repo "${GITHUB_REPOSITORY}" --dir archives \
      --pattern 'cjv_*' --pattern 'cjv-mirror_*'
- name: Extract cjv-init binaries by platform and variant
  run: |
    ARCHIVES_DIR=archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

`scripts/extract-init-binaries.sh` 遍历 5 个平台的官方版和镜像版归档，解压出里面的可执行文件，按平台和变体重命名摆放到：

```text
web/dist/dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
```

官方版归档里的 `cjv`(`cjv.exe`)被改名成 `cjv-init`(`cjv-init.exe`)放进 `official/`，镜像版归档里的 `cjv-mirror` 改名成同样的 `cjv-init` 放进 `mirror/`。这些文件随 Pages 一起发布，落地页的下载链接直接指向它们，形如 `/dl/official/windows_amd64/cjv-init.exe`、`/dl/mirror/darwin_arm64/cjv-init`(URL 在 `web/src/hooks/use-platform.ts` 里拼)。用户下载下来双击，二进制识别出自己叫 `cjv-init` 就进入安装流程。

这里有个时序前提：`pages.yml` 用 `gh release download ... releases/latest` 取的是最新 Release。在 `release.yml` 里 `pages` job `needs: [release, gitcode]`，所以总是先发完 Release 再部署页面，下载到的就是这次新发的版本。

## 安装脚本走的另一条路

落地页的下载按钮服务的是单文件 `cjv-init`，而 `install.sh`(以及 PowerShell 版)走的是完整 release 归档。它从 `releases/latest/download` 直接拉对应平台的 `.tar.gz`：

```sh
CJV_GITHUB_ROOT="${CJV_GITHUB_ROOT:-https://github.com/Zxilly/cjv/releases/latest/download}"
CJV_GITCODE_ROOT="${CJV_GITCODE_ROOT:-https://gitcode.com/Zxilly/cjv/releases/latest/download}"
```

默认拉官方版 `cjv_<platform>_<arch>.tar.gz`；带 `--mirror`(或环境变量 `CJV_MIRROR=1`)时改拉 `cjv-mirror_<platform>_<arch>.tar.gz` 并切到 GitCode root。下载后用 `checksums.txt` 校验，再解压执行 `cjv init`。注意脚本只处理 `.tar.gz`，因此 `install.sh` 服务的是 Linux 和 macOS；Windows 用户走 `install.ps1` 或直接下载 `cjv-init.exe`。这些命名和 `releases/latest/download` 路径都依赖 goreleaser 的归档命名，改 `.goreleaser.yml` 里的 `name_template` 时要同步检查安装脚本和 `extract-init-binaries.sh`。

## 本地预演

不想真发布、只想看看 goreleaser 会产出什么，可以在仓库根目录跑：

```bash
goreleaser release --snapshot --clean
```

`--snapshot` 不打 tag、不上传，把全部归档生成到 `dist/`，适合检查产物命名和构建是否通过。要单独验证 init 落地逻辑，可以手动把归档放到一个目录，再跑提取脚本：

```bash
ARCHIVES_DIR=./archives OUT_DIR=web/dist/dl bash scripts/extract-init-binaries.sh
```

这正是 `pages.yml` 用的同一个脚本和同一组环境变量。
