# Environment variables

You can set environment variables before starting `cjv` to override [configuration](configuration.md) settings for that process and its children without modifying `settings.toml`:

```bash
CJV_LOG=debug cjv show active
```

In Windows PowerShell:

```powershell
$env:CJV_LOG = "debug"
cjv show active
```

## Quick reference

| Variable | Default | Purpose |
| --- | --- | --- |
| `CJV_HOME` | `~/.cjv`, or the `home` setting | Absolute data directory; does not move user `settings.toml` |
| `CJV_CONFIG_FILE` | `~/.cjv/settings.toml` | User settings file (`--config` takes precedence); system fallback and project selection still apply |
| `CJV_FALLBACK_SETTINGS` | [System settings path](enterprise/index.md#system-fallback-settings) | Select an alternate system fallback settings file |
| `CJV_TOOLCHAIN` | Unset | Toolchain name or absolute SDK path; overrides directory overrides, `cangjie-sdk.toml`, and the default toolchain |
| `CJV_AUTO_INSTALL` | The `auto_install` setting | `0` disables or `1` enables automatic installation without saving settings |
| `CJV_OFFLINE` | Unset | Disable `cjv` network requests when set to `1`; cached files remain available |
| `CJV_DIST_SERVER` | Unset | Override the [toolchain distribution root](enterprise/distribution-server.md) |
| `CJV_ALLOW_INSECURE_MANIFEST` | Unset | Allow plain `http://` manifests from non-loopback hosts when set to `1` |
| `CJV_MAX_RETRIES` | `3` | Maximum retries after a failed download (non-negative integer) |
| `CJV_DOWNLOAD_TIMEOUT` | `180` | HTTP download timeout in seconds (positive integer) |
| `CJV_NO_PATH_SETUP` | Unset | Skip automatic `PATH` setup when set to `1` |
| `CJV_LOG` | `warn` | Log level on `stderr`: `debug`, `info`, `warn`, or `error` (unknown values fall back to `warn`) |
| `CJV_LANG` | System locale | Override the CLI interface language, such as `zh`, `en`, or `ja` |

## Detailed behavior

- **Paths and configuration (`CJV_HOME`, `CJV_CONFIG_FILE`, `CJV_FALLBACK_SETTINGS`)** — `CJV_HOME` selects an absolute data directory for toolchains, components, caches, and rollback history. It does not move the user `settings.toml` file. To isolate CI settings, also set `CJV_CONFIG_FILE` or pass `--config <path>`. See [Configuration](configuration.md#inheritance-and-temporary-overrides) for precedence rules.
- **Toolchain selection and auto-install (`CJV_TOOLCHAIN`, `CJV_AUTO_INSTALL`)** — If `CJV_TOOLCHAIN` is set, it overrides directory overrides, `cangjie-sdk.toml`, and the default toolchain (only an explicit CLI flag like `+name` or `--toolchain` ranks higher). `CJV_AUTO_INSTALL=0` or `1` temporarily overrides `auto_install` for the current process tree.
- **Networking, retries, and HTTPS enforcement (`CJV_OFFLINE`, `CJV_DIST_SERVER`, `CJV_MAX_RETRIES`, `CJV_DOWNLOAD_TIMEOUT`, `CJV_ALLOW_INSECURE_MANIFEST`)** — If `CJV_MAX_RETRIES` or `CJV_DOWNLOAD_TIMEOUT` is set to an unparseable or out-of-range value, `cjv` falls back to its default (`3` retries and `180` seconds). Because version manifests supply both download URLs and SHA-256 checksums, `cjv` requires `https://` for remote manifests by default; loopback test servers (`localhost`, `127.0.0.1`, `::1`) are exempt, and setting `CJV_ALLOW_INSECURE_MANIFEST=1` permits plain `http://` on non-loopback hosts when testing internal infrastructure.
- **Variables injected into SDK environments (`CANGJIE_HOME`, `CANGJIE_STDX_PATH_*`)** — When `stdx` is installed for the selected toolchain, `cjv` automatically injects `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` into the SDK's runtime environment, pointing to the component's `dynamic/` and `static/` directories. These variables are managed by `cjv` and normally need no manual configuration; see [Running commands and setting the environment](runtime-environment.md).
- **Proxy and installer script variables (`HTTPS_PROXY`, `CJV_VERSION`, `CJV_UPDATE_ROOT`)** — See [Network proxies](network-proxies.md) for HTTP(S) and SOCKS5 proxy variables. During initial script installation, `install.sh` and `install.ps1` read `CJV_VERSION` to select a `cjv` release tag (command-line `--version` / `-Version` takes precedence). `CJV_UPDATE_ROOT` (along with `CJV_GITHUB_ROOT` and `CJV_GITCODE_ROOT`) selects the initial download base URL for the installer scripts and takes precedence over version selection, though it does not override `cjv self update` on an already-installed binary. See [Install cjv](installation/index.md#pinning-the-cjv-version) for examples.
