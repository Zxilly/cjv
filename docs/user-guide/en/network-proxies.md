# Network proxies

cjv obeys standard proxy environment variables for all network requests, including version manifests, SDK archives, components, and self-updates. This chapter shows how to configure HTTP, HTTPS, and SOCKS5 proxies and how to handle corporate TLS certificates.

## Set proxy variables

Set the proxy variables in your shell before you start cjv.

In Bash or Zsh:

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

In Windows PowerShell:

```powershell
$env:HTTPS_PROXY = "http://proxy.example.com:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts
```

In Windows Command Prompt (`cmd.exe`):

```cmd
set HTTPS_PROXY=http://proxy.example.com:8080
set NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

## Supported variables and URL schemes

| Variable | Effect |
| --- | --- |
| `HTTPS_PROXY` or `https_proxy` | Proxy server for `https://` requests |
| `HTTP_PROXY` or `http_proxy` | Proxy server for `http://` requests |
| `NO_PROXY` or `no_proxy` | Comma-separated hostnames, domains, or IPs that bypass the proxy |

Proxy URLs support the `http://`, `https://`, and `socks5://` schemes.

> **CAUTION:** cjv does not read `ALL_PROXY` or `all_proxy`. If your environment sets `ALL_PROXY` only, set `HTTPS_PROXY` and `HTTP_PROXY` explicitly before you run cjv.

## Corporate TLS inspection and slow links

- **Private CA certificates**: If your corporate proxy intercepts TLS traffic with a private certificate authority (CA), add that CA certificate to your operating system's trust store.
- **Internal mirrors**: If you use an [internal distribution server](enterprise/distribution-server.md), add its hostname to `NO_PROXY` so clients connect to the internal server directly.
- **Timeouts and retries**: On slow or unstable network links, increase `CJV_MAX_RETRIES` and `CJV_DOWNLOAD_TIMEOUT`. Read [Environment variables](environment-variables.md) for their default values and syntax.
