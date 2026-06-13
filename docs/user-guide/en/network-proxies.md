# Network proxies

Enterprise networks often don't allow direct outside access and require going through a proxy server. cjv reads the standard proxy environment variables, so on such a network **no cjv configuration is needed** — setting the environment variables is enough to make all of cjv's downloads (toolchains, components, manifests, self-update) go through the proxy.

## Setting a proxy

cjv's downloads all use HTTPS, so setting `https_proxy` is usually sufficient. The exact command differs between systems and shells:

- Linux / macOS (bash / zsh):

  ```bash
  export https_proxy=http://proxy.example.com:8080
  ```

- Windows Command Prompt (cmd):

  ```cmd
  set https_proxy=http://proxy.example.com:8080
  ```

- Windows PowerShell:

  ```powershell
  $env:https_proxy="http://proxy.example.com:8080"
  ```

The proxy URL supports the `http://`, `https://`, and `socks5://` schemes. To use a SOCKS5 proxy, replace the value above with `socks5://proxy.example.com:1080`.

## Excluding internal hosts

`no_proxy` lists the hosts that should **not** go through the proxy, which is handy for letting an internal mirror or a local service connect directly:

```bash
export no_proxy=localhost,127.0.0.1,mirror.corp.internal
```

For example, when you install toolchains through `--mirror` or a self-hosted mirror, you can add the mirror host to `no_proxy` so it bypasses the proxy and connects directly.

## Recognized variables

cjv recognizes the following variables (either case), consistent with most command-line tools:

|Variable|Effect|
|--------|------|
|`https_proxy` / `HTTPS_PROXY`|Proxy for HTTPS requests. cjv's downloads are all HTTPS, so this is the important one|
|`http_proxy` / `HTTP_PROXY`|Proxy for HTTP requests|
|`no_proxy` / `NO_PROXY`|Comma-separated list of hosts that bypass the proxy|

Note: cjv does **not** recognize `ALL_PROXY` / `all_proxy`. If you have only set `ALL_PROXY`, set `https_proxy` instead.

The proxy variables are read from the environment when cjv starts, so set them in your shell **before** running cjv (the per-invocation `https_proxy=… cjv …` prefix form also works).

Related: for the `CJV_*` variables cjv reads itself, see [Environment Variables](environment-variables.md).
