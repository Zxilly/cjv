# 部署受管客户端

推荐把“安装 cjv”“下发系统配置”和“安装默认工具链”拆成可分别检查退出码的步骤。

## 1. 安装 cjv 本体

可以直接通过企业软件分发系统下发并校验 cjv 归档，也可以在内部托管安装脚本。Windows PowerShell 示例：

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` 的作用域是安装脚本此次下载。工具链分发由 `dist_server` 控制，已安装二进制的升级由企业软件分发流程控制。`-NoModifyPath` 让企业通过 GPO、终端管理工具或构建镜像统一设置 `PATH`；允许 cjv 修改用户环境时可省略该参数。

自动化部署建议使用 `-DefaultToolchain none`，再单独执行 `cjv install`，以便分别检查 cjv 与 SDK 的安装结果。

## 2. 下发系统后备配置

cjv 会从下列系统级路径读取后备设置：

- Windows：`C:\ProgramData\cjv\settings.toml`
- Linux / macOS：`/etc/cjv/settings.toml`

也可以用 `CJV_FALLBACK_SETTINGS` 指定其他路径。受管终端示例：

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

请把示例版本替换为企业批准的确切版本。受限网络建议设置 `auto_install = false`：项目请求尚未部署的工具链或组件时，`cjc` / `cjpm` 会立即返回缺失错误。

配置优先级为 `CJV_DIST_SERVER`、用户 `~/.cjv/settings.toml`、系统后备文件和内置默认值。防火墙、DNS 与代理白名单实施网络访问策略。

建议为每个用户分配独立的 `CJV_HOME`。

## 3. 安装并固定批准版本

确认系统配置已经就位，再安装确切版本：

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

项目提交 `cangjie-sdk.toml` 后，各开发机使用同一批准版本：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

nightly 项目固定 manifest 中保留的确切版本即可获得可复现配置：

```toml
[toolchain]
channel = "nightly-1.2.0-alpha.20260822010101"
```

工具链文件的完整语义见[工具链文件](../toolchain-file.md)。
