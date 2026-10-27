# Network proxies

cjv uses standard proxy environment variables for network access. Proxy settings apply to version manifests, SDK archives, components, and self-update requests.

## Configuring a proxy in your shell

Set the proxy environment variables in your shell before running `cjv`:

- On Linux, macOS, or HarmonyOS HiShell (**Bash** or **Zsh**):

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

- On Windows **PowerShell**:

```powershell
$env:HTTPS_PROXY = "http://proxy.example.com:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts
```

- On Windows **Command Prompt (`cmd.exe`)**:

```cmd
set HTTPS_PROXY=http://proxy.example.com:8080
set NO_PROXY=localhost,127.0.0.1,artifacts.corp.example
cjv install lts
```

## Supported variables and URL schemes

| Variable | Purpose |
| --- | --- |
| `HTTPS_PROXY` / `https_proxy` | Proxy server for `https://` requests |
| `HTTP_PROXY` / `http_proxy` | Proxy server for `http://` requests |
| `NO_PROXY` / `no_proxy` | Comma-separated list of hostnames, domains, or IP addresses to contact directly |

Proxy URLs support the `http://`, `https://`, and `socks5://` schemes.

> **Note:** `cjv` uses Go's standard HTTP proxy resolution and does **not** read `ALL_PROXY` or `all_proxy`. If your shell environment only defines `ALL_PROXY`, set `HTTPS_PROXY` (and `HTTP_PROXY` if needed) explicitly before running `cjv`.

## Corporate TLS inspection and slow connections

- **Private CA certificates** — If your organization uses a TLS-inspecting proxy with a private Certificate Authority (CA), add the organization's root CA certificate to your operating system's trust store so `cjv` can verify HTTPS connections.
- **Bypassing the proxy for internal mirrors** — If you use an [internal distribution server](enterprise/distribution-server.md) alongside an outbound internet proxy, add your internal mirror's hostname or domain to `NO_PROXY` so `cjv` connects to it directly.
- **Tuning retries and timeouts** — On slow or high-latency connections, you can increase `CJV_MAX_RETRIES` (default `3`) and `CJV_DOWNLOAD_TIMEOUT` (default `180` seconds); see [Environment variables](environment-variables.md) for details.
