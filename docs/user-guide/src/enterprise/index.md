# 企业与离线部署

企业可以在不同网络隔离策略下，将 cjv 部署到开发终端和 CI 构建机。本章介绍如何选择部署模式、下发客户端与系统后备配置、安装经批准的 SDK 版本，以及对完全离线环境进行发布和验收。

## 选择部署模式

请根据终端机器的网络访问权限选择对应的部署方式：

| 网络环境 | 推荐方案 | 对应指南 |
| --- | --- | --- |
| 终端可通过代理访问公网上游 | 配置 `HTTPS_PROXY`、`NO_PROXY` 与企业根证书 | [网络代理](../network-proxies.md) |
| 终端仅能访问企业内网 | 在内网 HTTPS 服务上托管清单和归档，配置 `dist_server` | [内部分发源](distribution-server.md) |
| 终端完全无网络连接（物理隔离） | 下发本地 SDK 归档与 `stdx` 目录，通过 `link` 命令注册 | [自定义 SDK](../install-from-url.md) |

注意：`dist_server` 仅控制仓颉 SDK 和组件的分发，不控制 cjv 本体升级，也不配置 `cjpm` 的项目依赖仓库；后两者需在企业环境中分别安排。

## 部署客户端

你可以通过企业软件分发系统直接下发 `cjv` 可执行文件，也可以在内网托管安装脚本和发布归档：

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

- `CJV_UPDATE_ROOT`：指定安装脚本下载 cjv 本体的内网目录。该目录必须包含对应平台的归档文件和 `checksums.txt`。
- `-NoModifyPath`（Unix 脚本为 `--no-modify-path`）：当终端管理系统已统一配置 `PATH` 时，跳过脚本自身的 `PATH` 修改。

在自动化批量部署时，建议按以下三步分步执行，以便分别检查每一步的退出码：

1. 安装 `cjv` 本体程序。
2. 下发系统后备配置文件 `settings.toml`。
3. 安装经批准的 SDK 与组件版本。

## 系统后备配置

cjv 在读取用户主目录的 `~/.cjv/settings.toml` 之前，会先读取系统级后备配置文件：

| 操作系统 | 默认系统后备配置文件路径 |
| --- | --- |
| Windows | `C:\ProgramData\cjv\settings.toml` |
| Linux 与 macOS | `/etc/cjv/settings.toml` |

也可以通过 `CJV_FALLBACK_SETTINGS` [环境变量](../environment-variables.md)指定其他路径。典型的企业后备配置如下：

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

请将示例中的 `lts-1.0.5` 替换为企业批准的发行版。关闭 `auto_install` 后，当项目缺少所需的 SDK 或组件时，命令会立即报错退出，由统一部署流程负责补齐。请为操作系统中的每个用户账户分配独立的 `CJV_HOME` 目录。

系统后备配置仅提供默认初始值，用户仍可在自己的 `settings.toml` 中按字段覆盖，或通过环境变量修改分发源。如需强制执行来源和版本合规策略，请结合网络 ACL、终端文件权限和受管软件分发实施。字段继承规则见[配置](../configuration.md#配置继承与优先级)。

## 安装批准版本

在可访问内网分发源的终端上，安装并验证企业批准的固定版本：

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

请在项目的 [cangjie-sdk.toml](../toolchain-file.md) 中写入完全相同的完整版本名（如 `lts-1.0.5`）。如果团队使用 `nightly`，也应固定完整的 nightly 版本号，并在内部分发源中长期保留该版本及其组件归档。

## 完全离线客户端

在没有任何 HTTP 服务的完全离线终端上，使用本地归档文件或预置目录完成注册：

```bash
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

1. 传入 `--sha256 <approved-sha256>`，让 cjv 在解包前校验归档文件的 SHA-256 摘要。
2. 也可以直接链接已解压的只读 SDK 目录（`cjv toolchain link corp-sdk /opt/corp/sdk`），此时组件由该外部目录的维护者直接管理。
3. 在项目的 `cangjie-sdk.toml` 中设置 `channel = "corp-sdk"`，并提前准备好离线构建所需的 `cjpm` 项目依赖包。

支持的归档结构见[自定义 SDK](../install-from-url.md)。注意 `docs` 和 `stdx-docs` 组件不支持本地目录链接；如需离线文档，请在断网前通过分发源安装，或直接随预置的 `CJV_HOME` 目录一起下发。

## 发布与验收清单

更新[内部分发源](distribution-server.md)时，请务必先上传并校验所有制品归档，再原子替换 `versions.json` 或 `nightly.json` 并推进 `latest`。请保留所有被现有项目固定的历史版本。

向研发人员全面推广前，请分别使用普通终端用户账户和实际 CI 服务账户验证以下事项：

- [ ] LTS/STS 和 nightly 操作分别读取 `versions.json` 与 `nightly.json`，且所有制品下载 URL 均在批准的内网范围内。
- [ ] 批准的发行版及组件能够正常安装，`cjv which cjc` 和 `cjc --version` 均指向预期 SDK。
- [ ] 断网状态下，已预置好工具链、缓存和依赖的项目可通过 `--offline --no-auto-install` 正常构建。
- [ ] 企业私有 CA、代理配置和 `NO_PROXY` 均正常生效，缺失未批准组件时按预期报错。

关闭 `auto_self_update` 后，请通过企业软件分发系统统一升级 cjv 本体。由于显式运行 `cjv self update` 仍会访问编译时注入的上游更新地址，网络边界策略应与企业部署规范保持一致。
