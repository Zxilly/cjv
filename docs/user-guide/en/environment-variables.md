# Environment variables

Set environment variables before starting cjv to override settings for that process:

```bash
CJV_LOG=debug cjv show active
```

```powershell
$env:CJV_LOG = "debug"
cjv show active
```

| Variable | Default | Purpose |
| --- | --- | --- |
| `CJV_HOME` | `~/.cjv`, or the `home` setting | Absolute data directory; does not move user settings |
| `CJV_TOOLCHAIN` | Unset | Toolchain name or absolute SDK path; overrides directory and default settings |
| `CJV_DIST_SERVER` | Unset | Override the [toolchain distribution root](enterprise/distribution-server.md) |
| `CJV_LOG` | `warn` | `debug`, `info`, `warn`, or `error`; unknown values use `warn`; logs go to stderr |
| `CJV_MAX_RETRIES` | `3` | Maximum retries after a failed download; nonnegative integer |
| `CJV_DOWNLOAD_TIMEOUT` | `180` | HTTP download timeout in seconds; positive integer |
| `CJV_NO_PATH_SETUP` | Unset | Skip automatic PATH setup when exactly `1` |
| `CJV_LANG` | System locale | Override the interface language, such as `zh`, `en`, or `ja` |
| `CJV_FALLBACK_SETTINGS` | [System settings path](enterprise/index.md) | Select a fallback settings file |
| `CJV_ALLOW_INSECURE_MANIFEST` | Unset | Allow plain HTTP manifests from non-loopback hosts when `1` |

Invalid retry counts and timeouts fall back to defaults. Manifests supply both artifact URLs and checksums, so HTTPS is required by default. Loopback test servers are exempt.

When stdx is installed, cjv injects `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` into the selected SDK's environment, pointing to its component's `dynamic/` and `static/` directories. They normally need no manual configuration.

See [network proxies](network-proxies.md) for HTTP(S) proxy variables. `CJV_UPDATE_ROOT` belongs to the installer scripts and selects the initial cjv download location; it does not override self-updates of an installed cjv binary.
