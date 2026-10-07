# 企业与离线部署

允许终端访问上游时，配置[网络代理](../network-proxies.md)即可。终端只能访问内网时，将 SDK、组件和版本清单放入[内部分发源](distribution-server.md)。完全离线时使用本地归档或目录。

`dist_server` 控制 SDK 和组件的分发，不控制 cjv 本体升级，也不配置 `cjpm` 的项目依赖仓库。后两者需要分别安排。

## 部署客户端

通过企业软件分发系统安装 cjv，或托管安装脚本及发布归档：

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` 只选择安装脚本的下载地址。该目录需要提供平台归档和 `checksums.txt`。`-NoModifyPath` 适合由终端管理系统统一设置 PATH 的环境。

先安装 cjv，再下发设置，最后安装 SDK，可分别检查每一步的退出码。

## 系统后备配置

| 系统 | 默认文件 |
| --- | --- |
| Windows | `C:\ProgramData\cjv\settings.toml` |
| Linux / macOS | `/etc/cjv/settings.toml` |

`CJV_FALLBACK_SETTINGS` 可指定其他文件。示例：

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

将示例版本替换为批准版本。关闭自动安装后，项目缺少 SDK 或组件时会报错，由部署流程补齐。为每个用户分配独立的 `CJV_HOME`。

后备设置提供默认值，用户文件可以逐字段覆盖，环境变量也能改变分发源。强制来源和版本策略需由网络 ACL、终端权限及软件分发流程实施。

## 安装批准版本

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

项目的 `cangjie-sdk.toml` 使用同一完整版本名。nightly 也应固定完整版本，并在分发源中保留该版本及其组件。

## 完全离线

```bash
cjv toolchain link corp-sdk ./cangjie-sdk.zip
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

有批准的哈希时，给归档安装传 `--sha256 <approved-sha256>`。也可直接链接已有 SDK 目录，但这种方式由外部 SDK 所有者维护组件。项目文件使用 `corp-sdk` 这样的自定义名称，构建依赖也需预先准备。

归档格式见[自定义 SDK](../install-from-url.md)。`docs` 和 `stdx-docs` 不支持本地链接，可在断网前通过分发源安装。

## 发布和验收

先上传并校验制品，再原子替换清单和推进 `latest`。nightly 的 `latest` 位于 `nightly.json` 顶层；客户端可能根据组件和目标需求选择历史兼容版本。保留被项目固定的所有版本。

验收时使用普通用户和实际 CI 服务账户，检查：

- LTS/STS 和 nightly 分别读取 `versions.json`、`nightly.json`，制品 URL 都在批准范围内。
- 批准版本及组件能安装，`cjv which cjc` 和 `cjc --version` 指向预期 SDK。
- 断网后，已准备好工具链及依赖的项目仍能构建。
- 企业 CA、代理和 `NO_PROXY` 生效，缺失项按预期报错。

关闭自动自更新后，由企业软件分发系统升级 cjv 本体。显式执行 `cjv self update` 仍会使用该构建的上游更新源；网络限制应与部署策略一致。
