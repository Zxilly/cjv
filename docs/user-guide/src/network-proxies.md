# 网络代理

cjv 的所有网络请求（包括获取版本清单、下载 SDK 与组件归档以及执行自更新）均遵循标准的代理环境变量。本章说明如何配置 HTTP、HTTPS 和 SOCKS5 代理，以及如何处理企业 TLS 证书与内网直连。

## 设置代理环境变量

请在启动 cjv 之前在终端中设置代理环境变量。

在 Bash 或 Zsh 中：

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

在 Windows PowerShell 中：

```powershell
$env:HTTPS_PROXY = "http://proxy.example.com:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts
```

在 Windows 命令提示符（`cmd.exe`）中：

```cmd
set HTTPS_PROXY=http://proxy.example.com:8080
set NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

## 支持的代理变量与协议

| 变量名 | 用途 |
| --- | --- |
| `HTTPS_PROXY` 或 `https_proxy` | 用于 `https://` 请求的代理服务器 |
| `HTTP_PROXY` 或 `http_proxy` | 用于 `http://` 请求的代理服务器 |
| `NO_PROXY` 或 `no_proxy` | 逗号分隔的直连主机名、域名或 IP 列表（绕过代理） |

代理 URL 支持 `http://`、`https://` 和 `socks5://` 三种协议。

> **注意：** cjv 不会读取 `ALL_PROXY` 或 `all_proxy`。如果当前环境仅配置了 `ALL_PROXY`，请在运行 cjv 前显式设置 `HTTPS_PROXY` 和 `HTTP_PROXY`。

## 企业 TLS 代理与慢速网络调优

- **私有 CA 证书**：如果企业安全代理使用自签名或私有 CA 拦截 TLS 流量，请将该 CA 根证书导入操作系统的受信任证书存储区。
- **内网镜像直连**：如果同时配置了外网代理与企业[内部分发源](enterprise/distribution-server.md)，请将内部分发源的域名或 IP 加入 `NO_PROXY`，确保内网流量直接连接。
- **超时与重试次数**：在慢速或不稳定的网络环境下，可以调大 `CJV_MAX_RETRIES` 和 `CJV_DOWNLOAD_TIMEOUT`；变量说明见[环境变量](environment-variables.md)。
