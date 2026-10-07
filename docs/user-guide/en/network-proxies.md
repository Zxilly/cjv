# Network proxies

cjv network requests follow standard proxy environment variables, including manifests, SDKs, components, and self-updates.

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

In cmd, use `set HTTPS_PROXY=http://proxy.example.com:8080`.

| Variable | Purpose |
| --- | --- |
| `HTTPS_PROXY` / `https_proxy` | HTTPS requests |
| `HTTP_PROXY` / `http_proxy` | HTTP requests |
| `NO_PROXY` / `no_proxy` | Comma-separated hosts to access directly |

Proxy URLs support `http://`, `https://`, and `socks5://`. cjv does not read `ALL_PROXY` / `all_proxy`; if that is your only proxy variable, use one from the table instead. Set variables before starting cjv.

For a corporate TLS proxy with a private CA, add the CA to the operating system's trust store. Add internal mirrors to `NO_PROXY` when they should be reached directly. For slow connections, adjust `CJV_MAX_RETRIES` and `CJV_DOWNLOAD_TIMEOUT`; see [environment variables](environment-variables.md).
