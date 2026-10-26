# Enterprise and offline deployment

Organizations can deploy `cjv` across developer workstations and CI runners under a variety of network policies:

- **Outbound access via proxy** — When client machines can reach upstream servers through a corporate proxy, configure [network proxy variables](../network-proxies.md) and your corporate CA certificate.
- **Internal-only network** — When clients can only reach internal hosts, publish approved SDKs, components, and manifests to an [internal distribution server](distribution-server.md) and point `dist_server` at it.
- **Fully air-gapped environments** — When clients have no network access at all, distribute local SDK archives or pre-staged directories and register them with `cjv toolchain link`.

Keep in mind that `dist_server` controls Cangjie SDK and component distribution only. Upgrades to the `cjv` binary itself and `cjpm` package dependency repositories are configured and managed separately.

## Deploy clients

You can distribute the `cjv` executable directly through your enterprise endpoint management system, or host the installer scripts (`install.sh`, `install.ps1`) and release archives on an internal HTTP(S) server:

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` directs the installer script to download `cjv` from your internal release directory (which must serve the platform archives alongside `checksums.txt`). Passing `-NoModifyPath` (or `--no-modify-path` on Unix) leaves shell profiles and registry `PATH` values untouched when your endpoint management software configures `PATH` centrally.

In automated rollouts, it is best practice to separate deployment into three distinct steps—install `cjv`, distribute the system fallback settings file, and then install the approved SDKs—so that each step reports its own exit status.

## System fallback settings

Before reading a user's personal `~/.cjv/settings.toml` file, `cjv` consults a system-wide fallback settings file:

| Operating system | Default fallback settings file |
| --- | --- |
| Windows | `C:\ProgramData\cjv\settings.toml` |
| Linux and macOS | `/etc/cjv/settings.toml` |

You can also point `cjv` at a custom fallback file using the `CJV_FALLBACK_SETTINGS` [environment variable](../environment-variables.md). A typical enterprise fallback configuration looks like this:

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

Replace `lts-1.0.5` with your organization's approved release. With `auto_install = false`, missing SDKs or components fail immediately with a clear error rather than triggering an unapproved download, ensuring that all toolchains are supplied through your managed deployment workflow. On multi-user machines, make sure each user account has its own separate `CJV_HOME`.

Note that fallback settings provide default values: user settings in `~/.cjv/settings.toml` can still override individual fields, and environment variables like `CJV_DIST_SERVER` take precedence at runtime (see [Configuration](../configuration.md#inheritance-and-temporary-overrides)). To enforce strict source and version policies, combine system fallback settings with network ACLs, filesystem permissions, and managed software distribution.

## Install approved versions

On internal clients connected to your mirror, install and verify the approved release:

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

Use the exact same full version name (`lts-1.0.5`) in each project's [`cangjie-sdk.toml`](../toolchain-file.md). If teams use `nightly` builds, pin full nightly version strings as well and retain their SDK and component archives in your internal distribution.

## Fully offline clients

On air-gapped workstations or build nodes without an internal HTTP server, register toolchains from local archives or pre-staged directories:

```bash
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

Pass `--sha256 <approved-sha256>` when linking an archive so `cjv` verifies the archive's integrity before extraction. Alternatively, you can link an unpacked read-only SDK directory (`cjv toolchain link corp-sdk /opt/corp/sdk`), in which case the owner of that directory maintains its components directly. In project repositories, reference the custom toolchain name (`channel = "corp-sdk"`) in `cangjie-sdk.toml`, and make sure any required `cjpm` project dependencies are pre-staged for offline builds.

See [Custom SDKs](../install-from-url.md) for supported archive formats. Because `docs` and `stdx-docs` do not support local directory links, install them from a distribution source before disconnecting the machine (or include them in a pre-staged `CJV_HOME`).

## Publication and validation

When publishing new releases to your [internal distribution server](distribution-server.md), always upload and verify all archive files _before_ atomically replacing `versions.json` or `nightly.json` and advancing `latest`. Note that in `nightly.json`, `latest` is a top-level field and clients may search historical versions in that file when resolving compatible components or cross targets. Always retain all versions that are pinned by active projects.

Before rolling out updates broadly, validate the setup using both an ordinary developer account and your actual CI service account:

- Confirm that LTS/STS operations read `versions.json` and nightly operations read `nightly.json`, and that all resolved artifact URLs stay within approved internal hosts.
- Confirm that approved releases and components install cleanly, and that `cjv which cjc` and `cjc --version` identify the intended SDK.
- Confirm that projects with prepared toolchains and dependencies build offline as expected.
- Confirm that corporate CA certificates, proxy settings, and `NO_PROXY` rules work properly, and that missing requirements produce the expected errors when `auto_install` is disabled.

With `auto_self_update = "disable"`, distribute `cjv` binary upgrades through your enterprise software management tool. Keep in mind that running `cjv self update` explicitly still contacts the binary's compiled-in upstream update source, so network firewall policies should restrict external update hosts where appropriate.
