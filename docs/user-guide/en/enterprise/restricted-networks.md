# Proxy and fully offline environments

## Proxy-only deployment

If upstream services are allowed through a controlled proxy, an internal distribution source is not required. Set `HTTPS_PROXY` / `https_proxy` for the cjv process, and use `NO_PROXY` / `no_proxy` to bypass the proxy for internal services:

```powershell
$env:HTTPS_PROXY = "http://proxy.corp.example:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts-1.0.5
```

cjv does not read `ALL_PROXY`. If a TLS-inspecting proxy uses an enterprise CA, install that CA in the operating-system trust store first. See [Network proxies](../network-proxies.md) for the complete variable reference.

Slow links can increase the retry count and whole-request timeout:

```powershell
$env:CJV_MAX_RETRIES = "5"
$env:CJV_DOWNLOAD_TIMEOUT = "600"
```

When an internal distribution source is also configured, add its host to `NO_PROXY` so internal artifact traffic does not traverse the external proxy.

## Fully offline deployment

An SDK archive or extracted directory can be installed without a manifest:

```bash
# Materialize a local archive; always provide the approved SHA-256 when possible
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>

# Or reference an already extracted SDK directory
cjv toolchain link corp-sdk /opt/corp/cangjie-sdk

# stdx can be linked from a local directory
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

A local archive or directory creates a custom toolchain, so project `cangjie-sdk.toml` files must use the same custom name. `stdx` supports local linking; `docs` and `stdx-docs` currently do not, so install them through the internal source before the endpoint becomes air-gapped. See [Installing a Toolchain from a URL or Archive](../install-from-url.md) for archive layout details.

A local download cache is not a replacement for the distribution manifest. Remote queries and updates still need a reachable internal HTTPS source. A truly air-gapped endpoint should use a pinned local custom toolchain and avoid remote-query commands.
