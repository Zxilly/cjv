# 企业与离线部署

企业可以根据开发终端和 CI 构建机的网络边界策略，选择最合适的部署方案：

- **允许通过代理访问上游** — 配置[网络代理](../network-proxies.md)与企业根证书即可。
- **仅允许访问企业内网** — 将经批准的 SDK、组件和版本清单放入[内部分发源](distribution-server.md)，并将 `dist_server` 指向该内网地址。
- **完全离线（物理隔离）** — 使用本地归档文件或预置目录，通过 `cjv toolchain link` 完成注册。

请注意，`dist_server` 仅控制仓颉 SDK 和组件的分发，不控制 `cjv` 本体升级，也不配置 `cjpm` 的项目依赖仓库；后两者需要分别安排。

## 部署客户端

可以通过企业软件分发系统直接安装 `cjv`，或者在内网托管安装脚本与发布归档：

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` 只选择安装脚本下载 `cjv` 本体的地址，该目录需要提供对应平台的归档和 `checksums.txt`。`-NoModifyPath`（Unix 为 `--no-modify-path`）适合由终端管理系统统一设置 `PATH` 的环境。

在自动化部署中，建议按“先安装 `cjv`、再下发系统后备设置、最后安装 SDK”三步分别执行，以便独立检查每一步的退出码。

## 系统后备配置

`cjv` 在读取用户个人的 `~/.cjv/settings.toml` 之前，会先查询系统级后备配置文件：

| 系统 | 默认文件 |
| --- | --- |
| Windows | `C:\ProgramData\cjv\settings.toml` |
| Linux / macOS | `/etc/cjv/settings.toml` |

也可以通过 `CJV_FALLBACK_SETTINGS` [环境变量](../environment-variables.md)指定其他文件。典型的系统后备配置如下：

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

请将示例版本替换为企业批准的发行版。关闭自动安装（`auto_install = false`）后，项目缺少 SDK 或组件时会立即报错，统一由受管部署流程补齐。多用户机器上请为每个用户账户分配独立的 `CJV_HOME`。

后备设置仅提供默认初始值，用户设置文件仍可逐字段覆盖，环境变量也能在运行时改变分发源（见[配置](../configuration.md#继承与临时覆盖)）。如需强制执行来源和版本合规策略，请结合网络 ACL、终端文件权限及软件分发流程共同实施。

## 安装批准版本

在连接内部分发源的受管终端上，安装并验证批准版本：

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

项目的 [`cangjie-sdk.toml`](../toolchain-file.md) 应使用同一个完整版本名（`lts-1.0.5`）。如果使用 `nightly`，同样应固定完整的 nightly 版本号，并在内部分发源中保留该版本及其组件。

## 完全离线

在无内网 HTTP 服务的完全离线终端上，使用本地归档或目录完成注册：

```bash
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

如果有批准的 SHA-256 哈希，请给归档安装传入 `--sha256 <approved-sha256>`。你也可以直接链接已有的只读 SDK 目录（`cjv toolchain link corp-sdk /opt/corp/sdk`），但这种方式下组件由外部 SDK 所有者直接维护。项目文件使用 `corp-sdk` 这样的自定义名称，项目构建依赖也需预先准备。

支持的归档格式见[自定义 SDK](../install-from-url.md)。`docs` 和 `stdx-docs` 不支持本地目录链接，可在断网前通过分发源安装（或随预置 `CJV_HOME` 一起下发）。

## 发布和验收

更新[内部分发源](distribution-server.md)时，请务必先上传并校验所有制品归档，再原子替换清单文件并推进 `latest`。`nightly` 的 `latest` 位于 `nightly.json` 顶层，客户端可能根据组件和交叉目标需求在清单中选择历史兼容版本。请保留被项目固定的所有历史版本。

验收时请分别使用普通研发用户账户和实际 CI 服务账户检查以下事项：

- LTS/STS 和 nightly 分别读取 `versions.json`、`nightly.json`，且所有制品 URL 均在批准的内网范围内。
- 批准版本及组件能正常安装，`cjv which cjc` 和 `cjc --version` 指向预期 SDK。
- 断网后，已准备好工具链及依赖的项目仍能正常离线构建。
- 企业私有 CA、代理和 `NO_PROXY` 正常生效，缺失项在关闭自动补装时按预期报错。

关闭自动自更新（`auto_self_update = "disable"`）后，由企业软件分发系统统一升级 `cjv` 本体。显式执行 `cjv self update` 仍会使用编译时注入的上游更新源，网络边界限制应与企业部署策略保持一致。
