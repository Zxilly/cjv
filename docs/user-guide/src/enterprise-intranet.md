# 企业内网部署

企业网络可以让 cjv 通过统一代理访问上游，也可以把 cjv、工具链和组件镜像到内部 HTTPS 制品库。工具链安装完成后，日常执行 `cjc`、`cjpm` 等命令不需要网络；安装、更新、远程查询或自动补齐缺失内容时才会发起请求。

## 选择部署模式

| 网络条件 | 推荐模式 | 支持范围 |
| --- | --- | --- |
| 允许经企业代理访问公网 | 设置标准代理环境变量 | 全部通道和自更新，受上游可用性约束 |
| 终端只能访问内部制品库 | 内部 manifest + 内部 SDK / 组件镜像 | LTS / STS 完整可用 |
| 完全离线或气隙环境 | 本地归档或目录链接 | 已安装或本地提供的工具链可用 |
| 要求强制统一源和版本 | 内部镜像 + 网络 ACL + 企业软件分发 | cjv 的系统后备配置本身不构成强制策略 |

`mirror` 构建变体只把默认 manifest 和自更新后端从 GitHub 切换到 GitCode，适合 GitHub 访问不稳定的网络；它不是通用的企业内部镜像版。

## cjv 会访问哪些地址

| 操作 | 默认来源 | 内网替代方式 |
| --- | --- | --- |
| 安装 cjv 本体 | GitHub 或 GitCode Release | 在内部托管发布归档和 `checksums.txt`；安装脚本可设置 `CJV_UPDATE_ROOT` |
| 安装或查询 LTS / STS | 默认 manifest | 在 `settings.toml` 中设置 `manifest_url` |
| 下载 LTS / STS SDK 和组件 | manifest 中记录的 URL | 把 manifest 内的全部 URL 改为内部制品地址 |
| 安装 nightly | GitCode API 和 `nightly_build` Release | 当前不能通过 `manifest_url` 改写，需要保持 GitCode 出站访问或修改 cjv |
| `cjv self update` | 官方版使用 GitHub，mirror 版使用 GitCode | 建议禁用自更新，由企业软件分发系统升级 cjv |

`cjpm` 下载项目依赖时使用的仓库和凭据不由 cjv 管理。部署 cjv 只能解决 SDK 与组件分发，依赖仓库镜像需要单独配置。

## 推荐的内部镜像拓扑

```text
开发机 / CI 构建机
        |
        | HTTPS GET
        v
企业制品库
  ├── cjv/releases/       # cjv 发布归档和 checksums.txt
  ├── cjv/versions.json   # 工具链 manifest
  ├── cjv/sdk/            # LTS / STS SDK 归档
  └── cjv/components/     # stdx、docs、stdx-docs
```

当 manifest 及其中的所有 URL 都指向内部地址，同时禁用 nightly 和自更新时，终端不需要访问公网。

### 准备内部制品库

1. 镜像所需平台的 cjv 发布归档及同一 Release 下的 `checksums.txt`。
2. 镜像批准版本的 SDK 和组件归档，使用不可变、带版本号的 URL。
3. 复制上游 manifest，并把 SDK、`stdx`、`docs` 和 `stdx-docs` 的 URL 全部改为内部地址。
4. 保留并核对每个 SDK 条目的 SHA-256。
5. 先发布归档，确认可下载后再更新 manifest。

建议直接复制完整的上游 manifest，再改写其中的 URL。SDK 条目必须保留 `name`、`url` 和 `sha256`；组件没有校验和字段，因此内部组件地址应使用 HTTPS，并限制发布权限。

内部制品端点应允许 HTTPS GET。需要鉴权时，建议通过企业反向代理提供机器可访问的只读端点。

## 分发系统后备配置

cjv 会从下列系统级路径读取后备设置：

- Windows：`C:\ProgramData\cjv\settings.toml`
- Linux / macOS：`/etc/cjv/settings.toml`

也可以用 `CJV_FALLBACK_SETTINGS` 指定其他路径。一个适合受管终端的示例是：

```toml
version = 1
manifest_url = "https://artifacts.corp.example/cjv/versions.json"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

请把示例版本替换为企业批准的确切版本。建议在受限网络中关闭 `auto_install`：如果项目声明的工具链或组件尚未部署，`cjc` / `cjpm` 会立即报错，而不会在编译过程中隐式等待网络超时。

系统文件只提供后备值，用户在 `~/.cjv/settings.toml` 中显式设置的字段优先。严格禁止公网访问时，还必须通过防火墙、DNS 或代理白名单实施网络策略。

建议每个用户使用独立的 `CJV_HOME`，不要让多个用户共享同一个可写目录。

## 部署客户端

推荐把“安装 cjv”和“安装默认工具链”拆成两个可分别检查退出码的步骤。

### 1. 安装 cjv 本体

可以直接通过企业软件分发系统下发并校验 cjv 归档，也可以在内部托管安装脚本。Windows PowerShell 示例：

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` 只影响安装脚本首次下载 cjv，不会改变已安装二进制的 manifest 或自更新来源。`-NoModifyPath` 让企业通过 GPO、终端管理工具或构建镜像统一设置 `PATH`；如果允许 cjv 修改用户环境，可省略该参数。

自动化部署建议使用 `-DefaultToolchain none`，再单独执行 `cjv install`，以便分别检查 cjv 与 SDK 的安装结果。

### 2. 安装并固定批准版本

确认系统后备配置已经就位，再安装确切版本：

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

把 `1.0.5` 替换为内部 manifest 中批准的版本。项目还应提交 `cangjie-sdk.toml`，避免不同开发机随通道更新到不同版本：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

## 仅使用企业代理

如果允许通过统一代理访问上游，不必建设 manifest 镜像。为 cjv 进程设置 `HTTPS_PROXY` / `https_proxy`，并用 `NO_PROXY` / `no_proxy` 排除内部服务：

```powershell
$env:HTTPS_PROXY = "http://proxy.corp.example:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts-1.0.5
```

cjv 不读取 `ALL_PROXY`。TLS 解密代理使用企业 CA 时，需要先把 CA 安装进操作系统信任库。完整变量说明见[网络代理](network-proxies.md)。

慢速链路可以调整下载重试和整个请求的超时：

```powershell
$env:CJV_MAX_RETRIES = "5"
$env:CJV_DOWNLOAD_TIMEOUT = "600"
```

## 完全离线部署

已经拿到 SDK 归档或解压目录时，不需要 manifest：

```bash
# 从本地归档物化安装；推荐始终提供批准的 SHA-256
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>

# 或引用已经解压的 SDK 目录
cjv toolchain link corp-sdk /opt/corp/cangjie-sdk

# stdx 可以从本地目录链接
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

本地归档或目录会创建 custom 工具链，项目中的 `cangjie-sdk.toml` 应使用相同的自定义名称。`stdx` 支持本地链接；`docs` 和 `stdx-docs` 当前不支持本地目录链接，应在终端进入隔离区前通过内部镜像安装好。更多归档格式说明见[从 URL 或本地归档安装工具链](install-from-url.md)。

## 上线验收清单

- 从普通用户会话执行 `cjv toolchain list-remote --channel lts`，确认只访问内部 manifest。
- 安装批准版本和所需组件后，执行 `cjv which cjc` 与 `cjc --version`。
- 断开网络后再次执行编译，确认已安装工具链不产生下载请求。
- 确认 `auto_install = false`、`auto_self_update = "disable"`，并由企业流程负责升级。
- 验证企业 CA、代理变量和 `NO_PROXY` 在实际终端与 CI 服务账户下均生效。
- 用网络 ACL 验证终端无法绕过内部镜像访问 GitHub、GitCode 或其他上游下载地址。
