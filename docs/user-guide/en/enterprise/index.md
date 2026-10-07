# Enterprise and offline deployment

When clients can reach upstream servers, configure a [network proxy](../network-proxies.md). For internal-only clients, publish SDKs, components, and manifests to an [internal distribution](distribution-server.md). Fully offline clients can use local archives or directories.

`dist_server` controls SDK and component distribution. cjv binary upgrades and `cjpm` project dependency repositories require separate configuration.

## Deploy clients

Install cjv through enterprise software distribution, or host the installer scripts and release archives internally:

```powershell
$env:CJV_UPDATE_ROOT = "https://artifacts.corp.example/cjv/releases/latest/download"
& ([scriptblock]::Create((irm https://artifacts.corp.example/cjv/install.ps1))) `
  -Yes -DefaultToolchain none -NoModifyPath
```

`CJV_UPDATE_ROOT` selects the installer download location only. The directory must provide platform archives and `checksums.txt`. Use `-NoModifyPath` when endpoint management sets PATH.

Install cjv, distribute settings, then install SDKs so each step has a separate exit status.

## System fallback settings

| System | Default file |
| --- | --- |
| Windows | `C:\ProgramData\cjv\settings.toml` |
| Linux / macOS | `/etc/cjv/settings.toml` |

`CJV_FALLBACK_SETTINGS` selects another file. Example:

```toml
version = 1
dist_server = "https://artifacts.corp.example/cjv/dist"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
```

Replace the example version with an approved release. With automatic installation disabled, missing SDKs or components cause errors and must be supplied by deployment. Give each user a separate `CJV_HOME`.

Fallback settings provide defaults. User settings can override individual fields, and environment variables can change the distribution source. Enforce source and version policies through network ACLs, endpoint permissions, and software distribution.

## Install approved versions

```bash
cjv install lts-1.0.5 -c stdx
cjv default lts-1.0.5
cjv which cjc
cjc --version
```

Use the same full version name in the project's `cangjie-sdk.toml`. Pin full nightly versions as well, and retain their SDKs and components in the distribution.

## Fully offline clients

```bash
cjv toolchain link corp-sdk ./cangjie-sdk.zip
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

Pass `--sha256 <approved-sha256>` when an approved hash is available. You can also link an existing SDK directory, whose owner then maintains its components. Project files use the custom name, such as `corp-sdk`; build dependencies must also be prepared in advance.

See [custom SDKs](../install-from-url.md) for archive formats. `docs` and `stdx-docs` cannot be linked locally; install them through a distribution before disconnecting.

## Publication and validation

Upload and verify artifacts before atomically replacing manifests and advancing `latest`. Nightly's `latest` is a top-level field in `nightly.json`; clients may choose a compatible historical release for their component and target requirements. Retain all versions pinned by projects.

Validate with ordinary user accounts and the actual CI service account:

- LTS/STS and nightly read `versions.json` and `nightly.json` respectively, and all artifact URLs stay within approved locations.
- Approved releases and components install, and `cjv which cjc` and `cjc --version` identify the intended SDK.
- Projects with prepared toolchains and dependencies still build offline.
- Corporate CAs, proxies, and `NO_PROXY` work, and missing requirements produce the expected errors.

With automatic self-updates disabled, distribute cjv upgrades through enterprise software management. An explicit `cjv self update` still uses the build's upstream update source; network restrictions should match deployment policy.
