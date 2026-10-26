# 环境变量

在启动 cjv 前设置环境变量，可以对当前进程及其子进程的[配置](configuration.md)进行临时覆盖，而不会修改磁盘上的设置文件。本章汇总 cjv 读取的环境变量、为 SDK 工具注入的环境变量以及安装脚本使用的环境变量。

## 设置环境变量示例

在 Bash 或 Zsh 中：

```bash
CJV_LOG=debug cjv show active
```

在 Windows PowerShell 中：

```powershell
$env:CJV_LOG = "debug"
cjv show active
```

## `cjv` CLI 读取的环境变量

| 变量名 | 默认值 | 作用说明 |
| --- | --- | --- |
| `CJV_HOME` | `~/.cjv`，或设置中的 `home` | 指定数据目录（必须为绝对路径）；不改变用户 `settings.toml` 的位置 |
| `CJV_CONFIG_FILE` | `~/.cjv/settings.toml` | 指定用户设置文件（命令行 `--config` 优先）；仍保留系统后备配置与项目选择 |
| `CJV_FALLBACK_SETTINGS` | [系统后备配置路径](enterprise/index.md#系统后备配置) | 指定自定义的系统后备设置文件 |
| `CJV_TOOLCHAIN` | *(未设置)* | 指定工具链名称或 SDK 绝对路径；优先于本机目录覆盖、`cangjie-sdk.toml` 和默认工具链 |
| `CJV_AUTO_INSTALL` | 设置中的 `auto_install` | `0` 禁止、`1` 允许自动补装缺失项；不写入 `settings.toml` |
| `CJV_OFFLINE` | *(未设置)* | 设为 `1` 时禁止 cjv 发起网络请求，仅读取本地安装与[可复用缓存](configuration.md#可复用离线缓存cache) |
| `CJV_DIST_SERVER` | *(未设置)* | 覆盖[工具链分发源根地址](enterprise/distribution-server.md) |
| `CJV_ALLOW_INSECURE_MANIFEST` | *(未设置)* | 设为 `1` 时允许非本地回环主机通过明文 `http://` 提供版本清单 |
| `CJV_MAX_RETRIES` | `3` | 下载失败后的最大重试次数（非负整数） |
| `CJV_DOWNLOAD_TIMEOUT` | `180` | HTTP 下载超时秒数（正整数） |
| `CJV_NO_PATH_SETUP` | *(未设置)* | 设为 `1` 时在 `init` 或首次安装时跳过自动 `PATH` 配置 |
| `CJV_LOG` | `warn` | 设置输出到 `stderr` 的日志级别：`debug`、`info`、`warn`、`error`（未知值回退为 `warn`） |
| `CJV_LANG` | 系统区域设置 | 覆盖 CLI 界面语言，例如 `zh`、`en`、`ja` |

如果 `CJV_MAX_RETRIES` 或 `CJV_DOWNLOAD_TIMEOUT` 的值非法，cjv 会自动回退到默认值。由于版本清单同时提供下载 URL 与 SHA-256 校验和，默认强制要求使用 `https://`（本地回环地址 `localhost` 和 `127.0.0.1` 的测试服务器除外）。

## `cjv` 为 SDK 工具注入的环境变量

通过代理命令、`cjv run`、`cjv exec` 或 `cjv envsetup` 启动工具时，cjv 会自动设置 `CANGJIE_HOME`、`PATH` 以及对应操作系统的动态库搜索路径（Linux 为 `LD_LIBRARY_PATH`，macOS 为 `DYLD_LIBRARY_PATH`，Windows 为 `PATH`）。

如果当前选中的工具链已安装 `stdx` [组件](concepts/components.md)，cjv 还会自动注入以下两个变量：

- `CANGJIE_STDX_PATH_DYNAMIC`：指向该工具链 `stdx` 组件的 `dynamic/` 目录。
- `CANGJIE_STDX_PATH_STATIC`：指向该工具链 `stdx` 组件的 `static/` 目录。

上述 `CANGJIE_*` 变量均由 cjv 自动维护，无需手动配置。详见[运行命令与配置环境](runtime-environment.md)。

## 网络代理与安装脚本变量

- **网络代理变量**：cjv 读取 `HTTPS_PROXY`、`HTTP_PROXY` 和 `NO_PROXY`（及其小写形式）。代理地址格式与企业 TLS 证书配置见[网络代理](network-proxies.md)。
- **安装脚本版本选择**：`install.sh` 和 `install.ps1` 均会读取 `CJV_VERSION` 来选择 cjv 本体的发布标签（命令行 `--version` / `-Version` 优先）。
- **安装脚本下载源**：`CJV_UPDATE_ROOT`、`CJV_GITHUB_ROOT` 和 `CJV_GITCODE_ROOT` 用于覆盖首次安装 cjv 本体时的下载根地址，且优先于版本参数生效；注意 `CJV_UPDATE_ROOT` 不会改变已安装 cjv 的自更新源。具体示例见[安装 cjv](installation/index.md#固定-cjv-版本)与[企业与离线部署](enterprise/index.md#部署客户端)。
