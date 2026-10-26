# 环境变量

在启动 `cjv` 前设置环境变量，可以对当前进程及其子进程的[配置](configuration.md)进行临时覆盖，而不会修改磁盘上的设置文件：

```bash
CJV_LOG=debug cjv show active
```

在 Windows PowerShell 中：

```powershell
$env:CJV_LOG = "debug"
cjv show active
```

## 变量速查表

| 变量 | 默认值 | 作用 |
| --- | --- | --- |
| `CJV_HOME` | `~/.cjv`，或设置中的 `home` | 数据目录（必须为绝对路径）；不改变用户设置文件位置 |
| `CJV_CONFIG_FILE` | `~/.cjv/settings.toml` | 用户设置文件（`--config` 优先）；保留系统后备配置和项目选择 |
| `CJV_FALLBACK_SETTINGS` | [系统配置路径](enterprise/index.md#系统后备配置) | 指定自定义的系统后备设置文件 |
| `CJV_TOOLCHAIN` | 无 | 指定工具链名或 SDK 绝对路径；优先于目录覆盖、项目文件和默认设置 |
| `CJV_AUTO_INSTALL` | 设置中的 `auto_install` | `0` 禁止、`1` 允许自动补装；不写入设置文件 |
| `CJV_OFFLINE` | 无 | 恰好为 `1` 时禁止 `cjv` 发起网络请求，允许读取本地安装与[缓存](configuration.md#下载缓存) |
| `CJV_DIST_SERVER` | 无 | 覆盖[工具链分发根地址](enterprise/distribution-server.md) |
| `CJV_ALLOW_INSECURE_MANIFEST` | 无 | 为 `1` 时允许非回环主机通过明文 `http://` 提供清单 |
| `CJV_MAX_RETRIES` | `3` | 下载失败后的最大重试次数（非负整数） |
| `CJV_DOWNLOAD_TIMEOUT` | `180` | HTTP 下载超时秒数（正整数） |
| `CJV_NO_PATH_SETUP` | 无 | 恰好为 `1` 时跳过自动 `PATH` 配置 |
| `CJV_LOG` | `warn` | 输出到 `stderr` 的日志级别：`debug`、`info`、`warn`、`error`（未知值回退为 `warn`） |
| `CJV_LANG` | 系统区域设置 | 覆盖 CLI 界面语言，如 `zh`、`en`、`ja` |

## 详细行为说明

- **路径与配置文件隔离（`CJV_HOME`、`CJV_CONFIG_FILE`、`CJV_FALLBACK_SETTINGS`）** — `CJV_HOME` 用于指定工具链、组件、缓存和历史快照的数据目录，必须为绝对路径。注意仅修改 `CJV_HOME` 不会移动用户的 `settings.toml`；在 CI 中做完全隔离时，请同时设置 `CJV_CONFIG_FILE`（或传入 `--config <path>`）。完整继承与优先级规则见[配置](configuration.md#继承与临时覆盖)。
- **工具链选择与自动补装（`CJV_TOOLCHAIN`、`CJV_AUTO_INSTALL`）** — 设置 `CJV_TOOLCHAIN` 会覆盖本机目录覆盖、`cangjie-sdk.toml` 和全局默认工具链（仅次于命令行显式 `+name` 或 `--toolchain`）。`CJV_AUTO_INSTALL=0` 或 `1` 用于在当前进程树中临时关闭或开启自动补装。
- **网络、重试与 HTTPS 限制（`CJV_OFFLINE`、`CJV_DIST_SERVER`、`CJV_MAX_RETRIES`、`CJV_DOWNLOAD_TIMEOUT`、`CJV_ALLOW_INSECURE_MANIFEST`）** — 如果 `CJV_MAX_RETRIES` 或 `CJV_DOWNLOAD_TIMEOUT` 设置了非法值，会自动回退到默认值（`3` 次和 `180` 秒）。由于版本清单同时提供制品 URL 与 SHA-256 校验和，默认强制要求使用 `https://`；本地回环测试服务器（`localhost`、`127.0.0.1`、`::1`）不受该限制，在测试内网明文源时可设置 `CJV_ALLOW_INSECURE_MANIFEST=1`。
- **注入 SDK 运行环境的变量（`CANGJIE_HOME`、`CANGJIE_STDX_PATH_*`）** — 安装 `stdx` 组件后，`cjv` 会为所选 SDK 的运行环境自动注入 `CANGJIE_STDX_PATH_DYNAMIC` 和 `CANGJIE_STDX_PATH_STATIC`，分别指向组件的 `dynamic/` 与 `static/` 目录，通常无需手动设置。详见[运行命令与配置环境](runtime-environment.md)。
- **网络代理与安装脚本变量（`HTTPS_PROXY`、`CJV_VERSION`、`CJV_UPDATE_ROOT`）** — HTTP(S) 和 SOCKS5 代理变量见[网络代理](network-proxies.md)。两个安装脚本（`install.sh` 与 `install.ps1`）会读取 `CJV_VERSION` 来选择 `cjv` 本体的发布标签（命令行 `--version` / `-Version` 优先）。`CJV_UPDATE_ROOT`（以及 `CJV_GITHUB_ROOT`、`CJV_GITCODE_ROOT`）用于选择 `cjv` 本体的首次下载地址，优先于版本选择，但它不会覆盖已安装 `cjv` 的自更新源。具体用法见[安装 cjv](installation/index.md#固定-cjv-本体版本)。
