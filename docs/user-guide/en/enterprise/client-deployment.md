# Deploy managed clients

Install cjv, distribute system settings, and install the default toolchain as separate steps whose exit codes can be checked independently.

## 1. Install cjv itself

An enterprise software distribution system can deploy and verify the cjv archive directly, or the installer can be hosted internally. Windows PowerShell example:

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` affects only this installer download. It is not persisted and does not alter the toolchain distribution source or the installed binary's self-update source. `-NoModifyPath` lets the organization configure `PATH` through GPO, endpoint management, or a build image. Omit it when cjv may update the user's environment.

Automated deployments should use `-DefaultToolchain none`, then run `cjv install` separately so cjv and SDK installation results can be checked independently.

## 2. Distribute system fallback settings

cjv reads fallback settings from:

- Windows: `C:\ProgramData\cjv\settings.toml`
- Linux / macOS: `/etc/cjv/settings.toml`

`CJV_FALLBACK_SETTINGS` can select a different path. A managed-endpoint example is:

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

Replace the example version with the exact version approved by your organization. Disabling `auto_install` is recommended on restricted networks: if a project requests content that has not been deployed, `cjc` or `cjpm` fails immediately instead of waiting for an implicit network request during a build.

The system file supplies fallback values only. Fields explicitly set in `~/.cjv/settings.toml` take precedence, and `CJV_DIST_SERVER` takes precedence over settings files. Enforce a strict no-public-egress policy with firewall, DNS, or proxy allowlists as well.

Use a separate `CJV_HOME` for each user rather than one writable directory shared by multiple users.

## 3. Install and pin an approved version

After the system configuration is present, install an exact version:

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

Projects should commit `cangjie-sdk.toml` so workstations do not move to different versions when a channel advances:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

Projects that need nightly should likewise pin an exact version retained by the enterprise manifest:

```toml
[toolchain]
channel = "nightly-1.2.0-alpha.20260822010101"
```

See [The Toolchain File](../toolchain-file.md) for the complete schema.
