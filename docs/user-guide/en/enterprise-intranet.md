# Enterprise intranet deployment

An enterprise network can let cjv reach upstream services through a controlled proxy or mirror cjv, toolchains, and components into an internal HTTPS artifact repository. After a toolchain is installed, normal `cjc` and `cjpm` execution does not need network access; requests occur only during installation, updates, remote queries, or automatic installation of missing content.

## Choose a deployment mode

| Network condition | Recommended mode | Supported scope |
| --- | --- | --- |
| Public services are reachable through an enterprise proxy | Set the standard proxy environment variables | All channels and self-update, subject to upstream availability |
| Endpoints can reach only an internal artifact repository | Internal manifest plus internal SDK and component mirrors | Full LTS / STS support |
| Fully offline or air-gapped | Local archives or directory links | Installed or locally supplied toolchains |
| Sources and versions must be centrally enforced | Internal mirror plus network ACLs and enterprise software distribution | The cjv fallback settings are not an enforcement mechanism by themselves |

The `mirror` build variant only switches the default manifest and self-update backend from GitHub to GitCode. It is useful where GitHub is unreliable, but it is not a generic enterprise-mirror build.

## Network destinations

| Operation | Default source | Intranet replacement |
| --- | --- | --- |
| Install cjv itself | A GitHub or GitCode Release | Host release archives and `checksums.txt` internally; the installer accepts `CJV_UPDATE_ROOT` |
| Install or query LTS / STS | The default manifest | Set `manifest_url` in `settings.toml` |
| Download LTS / STS SDKs and components | URLs recorded in the manifest | Rewrite every manifest URL to an internal artifact URL |
| Install nightly | The GitCode API and `nightly_build` Releases | Cannot currently be rewritten through `manifest_url`; retain GitCode egress or modify cjv |
| Run `cjv self update` | GitHub for the official build, GitCode for the mirror build | Disable self-update and upgrade cjv through enterprise software distribution |

The package repositories and credentials used by `cjpm` to download project dependencies are outside cjv's scope. Deploying cjv solves SDK and component distribution; dependency mirrors must be configured separately.

## Recommended internal-mirror topology

```text
Developer workstation / CI runner
               |
               | HTTPS GET
               v
Enterprise artifact repository
  ├── cjv/releases/       # cjv release archives and checksums.txt
  ├── cjv/versions.json   # toolchain manifest
  ├── cjv/sdk/            # LTS / STS SDK archives
  └── cjv/components/     # stdx, docs, and stdx-docs
```

When the manifest and every URL in it point to internal services, and nightly and self-update are disabled, an endpoint does not need public network access.

### Prepare the internal artifact repository

1. Mirror the cjv release archives for the required platforms and the matching `checksums.txt`.
2. Mirror the approved SDK and component archives at immutable, versioned URLs.
3. Copy the upstream manifest and rewrite every SDK, `stdx`, `docs`, and `stdx-docs` URL to an internal address.
4. Preserve and verify the SHA-256 for every SDK entry.
5. Publish the archives first, verify that they are downloadable, and then update the manifest.

Copying the complete upstream manifest and rewriting its URLs is recommended. SDK entries must retain `name`, `url`, and `sha256`. Component entries have no checksum field, so internal component URLs should use HTTPS and publication permissions should be tightly controlled.

The internal endpoint must allow HTTPS GET requests. If authentication is required, expose a machine-readable, read-only endpoint through an enterprise reverse proxy.

## Distribute system fallback settings

cjv reads fallback settings from the following system-level locations:

- Windows: `C:\ProgramData\cjv\settings.toml`
- Linux / macOS: `/etc/cjv/settings.toml`

`CJV_FALLBACK_SETTINGS` can select a different path. A suitable managed-endpoint example is:

```toml
version = 1
manifest_url = "https://artifacts.corp.example/cjv/versions.json"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

Replace the example version with the exact version approved by your organization. Disabling `auto_install` is recommended on restricted networks: if a project requests a toolchain or component that has not been deployed, `cjc` or `cjpm` fails immediately instead of implicitly waiting for a network timeout during a build.

The system file supplies fallback values only. A field explicitly set in the user's `~/.cjv/settings.toml` takes precedence. A strict no-public-egress policy must also be enforced by firewall, DNS, or proxy allowlists.

Use a separate `CJV_HOME` for each user rather than one writable directory shared by multiple users.

## Deploy clients

Install cjv and the default toolchain as separate steps whose exit codes can be checked independently.

### 1. Install cjv itself

An enterprise software distribution system can deploy and verify the cjv archive directly, or the installer can be hosted internally. Windows PowerShell example:

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` affects only the installer's initial cjv download; it does not change the installed binary's manifest or self-update source. `-NoModifyPath` lets the organization configure `PATH` through GPO, endpoint management, or a build image. Omit it when cjv is allowed to update the user's environment.

Automated deployments should use `-DefaultToolchain none`, then run `cjv install` separately so the cjv and SDK installation results can be checked independently.

### 2. Install and pin an approved version

After the fallback settings are present, install an exact version:

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

Replace `1.0.5` with a version approved in the internal manifest. Projects should also commit `cangjie-sdk.toml` so different workstations do not move to different versions when a channel advances:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

## Proxy-only deployment

If upstream services are allowed through a controlled proxy, a manifest mirror is not required. Set `HTTPS_PROXY` / `https_proxy` for the cjv process, and use `NO_PROXY` / `no_proxy` to bypass the proxy for internal services:

```powershell
$env:HTTPS_PROXY = "http://proxy.corp.example:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts-1.0.5
```

cjv does not read `ALL_PROXY`. If a TLS-inspecting proxy uses an enterprise CA, install that CA in the operating-system trust store first. See [Network proxies](network-proxies.md) for the complete variable reference.

Slow links can increase the download retry count and whole-request timeout:

```powershell
$env:CJV_MAX_RETRIES = "5"
$env:CJV_DOWNLOAD_TIMEOUT = "600"
```

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

A local archive or directory creates a custom toolchain, so project `cangjie-sdk.toml` files must use the same custom name. `stdx` supports local linking; `docs` and `stdx-docs` currently do not, so install them through the internal mirror before the endpoint enters the air-gapped environment. See [Installing a Toolchain from a URL or Archive](install-from-url.md) for archive layout details.

## Deployment acceptance checklist

- Run `cjv toolchain list-remote --channel lts` from a standard user session and confirm that only the internal manifest is accessed.
- Install the approved version and required components, then run `cjv which cjc` and `cjc --version`.
- Disconnect the network and compile again to confirm that installed toolchains do not trigger downloads.
- Confirm `auto_install = false` and `auto_self_update = "disable"`, with upgrades owned by the enterprise process.
- Verify the enterprise CA, proxy variables, and `NO_PROXY` under both real user accounts and CI service accounts.
- Use network ACLs to confirm endpoints cannot bypass the internal mirror and reach GitHub, GitCode, or other upstream download services.
