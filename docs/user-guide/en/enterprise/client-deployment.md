# Deploy managed clients

Install cjv, distribute system settings, and install the default toolchain as separate steps whose exit codes can be checked independently.

## 1. Install cjv itself

An enterprise software distribution system can deploy and verify the cjv archive directly, or the installer can be hosted internally. Windows PowerShell example:

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` is scoped to this installer download. `dist_server` controls toolchain distribution, while the enterprise software workflow controls upgrades of installed binaries. `-NoModifyPath` lets the organization configure `PATH` through GPO, endpoint management, or a build image; omit it when cjv may update the user's environment.

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

Replace the example version with the exact version approved by your organization. Restricted networks should set `auto_install = false`, so `cjc` or `cjpm` immediately returns a missing-content error for an undeployed toolchain or component.

Configuration precedence is `CJV_DIST_SERVER`, user `~/.cjv/settings.toml`, the system fallback file, and built-in defaults. Firewall, DNS, and proxy allowlists enforce network-access policy.

Assign a separate `CJV_HOME` to each user.

## 3. Install and pin an approved version

After the system configuration is present, install an exact version:

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

Projects can commit `cangjie-sdk.toml` so every workstation uses the same approved version:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

Nightly projects gain a reproducible configuration by pinning an exact version retained by the enterprise manifest:

```toml
[toolchain]
channel = "nightly-1.2.0-alpha.20260822010101"
```

See [The Toolchain File](../toolchain-file.md) for the complete schema.
