# 网络代理

cjv 的网络请求遵循标准代理环境变量，涵盖清单、SDK、组件和自更新。

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

```powershell
$env:HTTPS_PROXY = "http://proxy.example.com:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts
```

cmd 使用 `set HTTPS_PROXY=http://proxy.example.com:8080`。

| 变量 | 用途 |
| --- | --- |
| `HTTPS_PROXY` / `https_proxy` | HTTPS 请求 |
| `HTTP_PROXY` / `http_proxy` | HTTP 请求 |
| `NO_PROXY` / `no_proxy` | 逗号分隔的直连主机列表 |

代理地址支持 `http://`、`https://` 和 `socks5://`。cjv 不读取 `ALL_PROXY` / `all_proxy`，只配置该变量时需改为上表中的变量。请在启动 cjv 前设置代理。

企业 TLS 代理使用自签 CA 时，将 CA 加入操作系统信任库。内部镜像需要直连时，将其主机加入 `NO_PROXY`。慢速连接可调整 `CJV_MAX_RETRIES` 和 `CJV_DOWNLOAD_TIMEOUT`，见[环境变量](environment-variables.md)。
