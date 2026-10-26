# Enterprise and offline deployment

Organizations can deploy cjv across developer workstations and CI runners under varying network policies. This chapter explains how to choose a deployment mode, roll out managed clients, configure system fallback settings, install approved SDKs, and check air-gapped environments.

## Choose a deployment mode

Match your setup to the network access available on your client machines:

| Network environment | Recommended setup | Related guide |
| --- | --- | --- |
| Clients can reach upstream servers through a proxy | Configure `HTTPS_PROXY`, `NO_PROXY`, and corporate CA trust | [Network proxies](../network-proxies.md) |
| Clients can reach internal servers only | Host manifests and archives on an internal HTTPS server and set `dist_server` | [Internal distribution](distribution-server.md) |
| Clients have no network access (air-gapped) | Distribute SDK archives and `stdx` directories locally and register them with `link` | [Custom SDKs](../install-from-url.md) |

Note that `dist_server` controls Cangjie SDK and component downloads only. You must manage cjv binary upgrades and `cjpm` package dependency repositories separately.

## Deploy clients

You can push the `cjv` binary through your enterprise software management tool, or host the installer scripts and release archives on an internal server:

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

- `CJV_UPDATE_ROOT` sets the download URL for the installer script. That directory must serve the platform archives and `checksums.txt`.
- `-NoModifyPath` (or `--no-modify-path` on Unix) leaves `PATH` unchanged when your endpoint management system configures `PATH` centrally.

In automated rollouts, run these three steps as separate commands so you can check the exit code of each step:

1. Install the `cjv` executable.
2. Deploy the system fallback `settings.toml` file.
3. Install the approved SDKs and components.

## System fallback settings

cjv reads a system-wide fallback settings file before it reads the user's `~/.cjv/settings.toml` file:

| Operating system | Default system fallback path |
| --- | --- |
| Windows | `C:\ProgramData\cjv\settings.toml` |
| Linux and macOS | `/etc/cjv/settings.toml` |

You can override this path with the `CJV_FALLBACK_SETTINGS` [environment variable](../environment-variables.md). An enterprise fallback `settings.toml` typically looks like this:

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

Replace `lts-1.0.5` with your organization's approved release. When `auto_install` is `false`, missing SDKs or components cause an immediate error instead of an unapproved download. Give each OS user account its own `CJV_HOME` directory.

System fallback settings supply default values only. A user can still override individual fields in `~/.cjv/settings.toml` or set `CJV_DIST_SERVER` in the environment. Enforce strict source and version policies through firewall rules, filesystem permissions, and managed software deployment. Read [Configuration](../configuration.md#inheritance-and-precedence) for field inheritance rules.

## Install approved versions

On connected internal clients, install and check the approved release from your [internal distribution](distribution-server.md):

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

Write the exact same full version string (`lts-1.0.5`) in each project's [cangjie-sdk.toml](../toolchain-file.md). If your teams use `nightly`, pin the full nightly version string as well and keep that version's SDK and component archives on your internal server.

## Fully offline clients

On air-gapped machines without an internal HTTP server, install from local archives or pre-staged directories:

```bash
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

1. Pass `--sha256 <approved-sha256>` when you link the archive so cjv checks the file hash before extraction.
2. Alternatively, link an unpacked read-only SDK directory (`cjv toolchain link corp-sdk /opt/corp/sdk`). In that mode, the directory administrator maintains its components directly.
3. Set `channel = "corp-sdk"` in the project's `cangjie-sdk.toml` and pre-stage any `cjpm` project dependencies needed for offline builds.

Read [Custom SDKs](../install-from-url.md) for supported archive layouts. Note that `cjv component link` does not support `docs` or `stdx-docs` directories; install them from a distribution server before disconnecting the machine, or bundle them into your pre-staged `CJV_HOME`.

## Publication and validation checklist

When you update your [internal distribution](distribution-server.md), upload and check all archive files before you atomically replace `versions.json` or `nightly.json` and advance `latest`. Keep all historical versions that active projects pin in `cangjie-sdk.toml`.

Before you roll out a change to developers, do a check with both a standard user account and your CI service account:

- [ ] LTS/STS commands read `versions.json`, nightly commands read `nightly.json`, and every artifact URL points to an approved internal host.
- [ ] Approved SDKs and components install cleanly, and `cjv which cjc` and `cjc --version` report the expected SDK.
- [ ] Projects with pre-installed toolchains and dependencies build with `--offline --no-auto-install`.
- [ ] Corporate CA certificates, proxy variables, and `NO_PROXY` rules work, and missing unapproved toolchains fail with clear errors.

When `auto_self_update` is `disable`, distribute new cjv binaries through your enterprise software management tool. Because an explicit `cjv self update` command still queries the upstream release URL compiled into the binary, block external update hosts at the network boundary when your security policy demands it.
