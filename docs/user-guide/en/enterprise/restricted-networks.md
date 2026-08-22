# Proxy and fully offline environments

## Proxy-only deployment

Proxy mode reaches upstream services through `HTTPS_PROXY` / `https_proxy` and routes internal services directly through `NO_PROXY` / `no_proxy`:

```powershell
$env:HTTPS_PROXY = "http://proxy.corp.example:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts-1.0.5
```

cjv supports the standard `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` variables. Install an enterprise CA in the operating-system trust store before using a TLS-inspecting proxy. See [Network proxies](../network-proxies.md) for the complete variable reference.

Slow links can increase the retry count and whole-request timeout:

```powershell
$env:CJV_MAX_RETRIES = "5"
$env:CJV_DOWNLOAD_TIMEOUT = "600"
```

When an internal distribution source is also configured, add its host to `NO_PROXY` so internal artifact traffic reaches the repository directly.

## Fully offline deployment

A local SDK archive or extracted directory can be installed directly:

```bash
# Materialize a local archive; always provide the approved SHA-256 when possible
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>

# Or reference an already extracted SDK directory
cjv toolchain link corp-sdk /opt/corp/cangjie-sdk

# stdx can be linked from a local directory
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

A local archive or directory creates a custom toolchain, and project `cangjie-sdk.toml` files use the same custom name. `stdx` supports local linking; provision `docs` and `stdx-docs` through the internal source before the endpoint becomes air-gapped. See [Installing a Toolchain from a URL or Archive](../install-from-url.md) for archive layout details.

Remote queries and updates use a reachable internal HTTPS manifest. A fully air-gapped endpoint pins a local custom toolchain and follows a local build workflow.
