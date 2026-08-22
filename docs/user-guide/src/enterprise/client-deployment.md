# 部署受管客户端

推荐把“安装 cjv”“下发系统配置”和“安装默认工具链”拆成可分别检查退出码的步骤。

## 1. 安装 cjv 本体

可以直接通过企业软件分发系统下发并校验 cjv 归档，也可以在内部托管安装脚本。Windows PowerShell 示例：

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` 只影响安装脚本此次下载 cjv，不会持久化，也不会改变工具链分发源或已安装二进制的自更新来源。`-NoModifyPath` 让企业通过 GPO、终端管理工具或构建镜像统一设置 `PATH`；如果允许 cjv 修改用户环境，可省略该参数。

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

请把示例版本替换为企业批准的确切版本。受限网络建议关闭 `auto_install`：项目请求尚未部署的工具链或组件时，`cjc` / `cjpm` 会立即报错，而不是在编译过程中隐式等待网络。

系统文件只提供后备值，用户在 `~/.cjv/settings.toml` 中显式设置的字段优先；`CJV_DIST_SERVER` 环境变量又高于设置文件。需要严格禁止公网访问时，仍必须通过防火墙、DNS 或代理白名单实施网络策略。

建议每个用户使用独立的 `CJV_HOME`，不要让多个用户共享同一个可写目录。

## 3. 安装并固定批准版本

确认系统配置已经就位，再安装确切版本：

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

项目还应提交 `cangjie-sdk.toml`，避免不同开发机随通道更新到不同版本：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

需要 nightly 的项目同样应固定 manifest 中保留的确切版本，而不是提交浮动的 `nightly`：

```toml
[toolchain]
channel = "nightly-1.2.0-alpha.20260822010101"
```

工具链文件的完整语义见[工具链文件](../toolchain-file.md)。
