# 网络代理

cjv 通过标准代理环境变量配置网络访问。代理设置适用于版本清单、SDK 归档、组件和自更新请求。

## 在终端中配置代理

请在启动 `cjv` 前设置对应的环境变量：

- 在 Linux、macOS 或鸿蒙 HiShell（**Bash** 或 **Zsh**）中：

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

- 在 Windows **PowerShell** 中：

```powershell
$env:HTTPS_PROXY = "http://proxy.example.com:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts
```

- 在 Windows **命令提示符（`cmd.exe`）** 中：

```cmd
set HTTPS_PROXY=http://proxy.example.com:8080
set NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

## 支持的代理变量与协议

| 变量 | 用途 |
| --- | --- |
| `HTTPS_PROXY` / `https_proxy` | `https://` 请求使用的代理服务器 |
| `HTTP_PROXY` / `http_proxy` | `http://` 请求使用的代理服务器 |
| `NO_PROXY` / `no_proxy` | 逗号分隔的直连主机名、域名或 IP 列表 |

代理地址支持 `http://`、`https://` 和 `socks5://` 协议。

> **注意：** `cjv` 不读取 `ALL_PROXY` / `all_proxy`。如果你的终端环境只配置了 `ALL_PROXY`，请在启动 `cjv` 前改为显式设置上表中的 `HTTPS_PROXY` 或 `HTTP_PROXY`。

## 企业 TLS 证书与慢速连接

- **企业私有 CA 证书** — 当企业 TLS 代理使用自签名或私有 CA 拦截 HTTPS 流量时，请将该 CA 根证书加入操作系统的受信任证书存储区。
- **内网镜像直连** — 当内网同时部署了[内部分发源](enterprise/distribution-server.md)并且需要直连访问时，请将内部分发源的主机名加入 `NO_PROXY`。
- **调整重试与超时** — 在慢速或高延迟连接下，可以调大 `CJV_MAX_RETRIES` 和 `CJV_DOWNLOAD_TIMEOUT`，详见[环境变量](environment-variables.md)。
