# Environment variables

Set environment variables before you start cjv to override [configuration](configuration.md) settings for the current process and its child processes. This chapter lists the environment variables that cjv reads, the variables that cjv sets for SDK tools, and the variables used by the installer scripts.

## Set an environment override

In Bash or Zsh, prefix the command or `export` the variable:

```bash
CJV_LOG=debug cjv show active
```

In Windows PowerShell, set `$env:` before you run the command:

```powershell
$env:CJV_LOG = "debug"
cjv show active
```

## Variables read by the `cjv` CLI

| Variable | Default value | Effect |
| --- | --- | --- |
| `CJV_HOME` | `~/.cjv`, or `home` in settings | Sets the data directory (must be an absolute path); does not move `settings.toml` |
| `CJV_CONFIG_FILE` | `~/.cjv/settings.toml` | Selects the user settings file (`--config` takes precedence); system fallback and project files still apply |
| `CJV_FALLBACK_SETTINGS` | [System fallback path](enterprise/index.md#system-fallback-settings) | Selects an alternative system fallback settings file |
| `CJV_TOOLCHAIN` | *(unset)* | Selects a toolchain name or absolute SDK path; overrides directory overrides, `cangjie-sdk.toml`, and the default toolchain |
| `CJV_AUTO_INSTALL` | `auto_install` in settings | `0` disables or `1` enables automatic installation without writing to `settings.toml` |
| `CJV_OFFLINE` | *(unset)* | When set to `1`, disables cjv network requests and uses local installations and [cache](configuration.md#reusable-offline-cache) only |
| `CJV_DIST_SERVER` | *(unset)* | Overrides the [toolchain distribution server root](enterprise/distribution-server.md) |
| `CJV_ALLOW_INSECURE_MANIFEST` | *(unset)* | When set to `1`, allows plain `http://` manifest URLs on non-loopback hosts |
| `CJV_MAX_RETRIES` | `3` | Sets the maximum retry count after a failed download (non-negative integer) |
| `CJV_DOWNLOAD_TIMEOUT` | `180` | Sets the HTTP download timeout in seconds (positive integer) |
| `CJV_NO_PATH_SETUP` | *(unset)* | When set to `1`, skips automatic `PATH` configuration during `init` or first install |
| `CJV_LOG` | `warn` | Sets the log level on `stderr`: `debug`, `info`, `warn`, or `error` (unknown values use `warn`) |
| `CJV_LANG` | System locale | Overrides the CLI interface language, such as `zh`, `en`, or `ja` |

If you set an invalid value for `CJV_MAX_RETRIES` or `CJV_DOWNLOAD_TIMEOUT`, cjv falls back to the default number. Because version manifests supply both download URLs and SHA-256 checksums, cjv enforces `https://` for remote manifests unless you set `CJV_ALLOW_INSECURE_MANIFEST=1`. Local loopback addresses (`localhost` and `127.0.0.1`) always allow `http://` for local testing.

## Variables set by `cjv` for SDK tools

When you run a command through a proxy binary, `cjv run`, `cjv exec`, or `cjv envsetup`, cjv sets `CANGJIE_HOME`, `PATH`, and platform dynamic library variables (`LD_LIBRARY_PATH` on Linux, `DYLD_LIBRARY_PATH` on macOS, or `PATH` on Windows).

When the active SDK has the `stdx` [component](concepts/components.md), cjv also sets these variables automatically:

- `CANGJIE_STDX_PATH_DYNAMIC`: points to the `dynamic/` library directory of `stdx`.
- `CANGJIE_STDX_PATH_STATIC`: points to the `static/` library directory of `stdx`.

You do not need to set these `CANGJIE_*` variables manually. Read [Run commands and set the environment](runtime-environment.md) for details.

## Proxy and installer variables

- **Network proxies**: cjv reads `HTTPS_PROXY`, `HTTP_PROXY`, and `NO_PROXY` (and their lowercase forms). Read [Network proxies](network-proxies.md) for proxy URL formats and TLS certificate configuration.
- **Installer version selection**: both `install.sh` and `install.ps1` read `CJV_VERSION` to install a specific cjv release tag when `--version` or `-Version` is not passed on the command line.
- **Installer download source**: `CJV_UPDATE_ROOT`, `CJV_GITHUB_ROOT`, and `CJV_GITCODE_ROOT` override the download base URL during initial script installation and take precedence over `CJV_VERSION`. Note that `CJV_UPDATE_ROOT` does not change the self-update URL of an already installed cjv binary. Read [Install cjv](installation/index.md#pin-the-cjv-version) and [Enterprise and offline deployment](enterprise/index.md#deploy-clients) for examples.
