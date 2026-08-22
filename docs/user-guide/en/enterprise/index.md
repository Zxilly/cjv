# Enterprise deployment overview

An enterprise network can let cjv reach upstream services through a controlled proxy or publish toolchains and components to an internal HTTPS artifact repository. After a toolchain is installed, normal `cjc` and `cjpm` execution does not need network access; requests occur only during installation, updates, remote queries, or automatic installation of missing content.

## Choose a deployment mode

| Network condition | Recommended mode | Supported scope |
| --- | --- | --- |
| Public services are reachable through an enterprise proxy | Set the standard proxy environment variables | All channels and self-update, subject to upstream availability |
| Endpoints can reach only an internal artifact repository | Configure a unified `dist_server` | LTS, STS, nightly, SDKs, and components can all stay internal |
| Fully offline or air-gapped | Use local archives or directory links | Installed or locally supplied toolchains |
| Sources and versions must be centrally enforced | Internal source plus network ACLs and enterprise software distribution | cjv fallback settings are not an enforcement mechanism by themselves |

The `mirror` build variant only switches the default LTS/STS manifest and cjv self-update backend from GitHub to GitCode. It is useful where GitHub is unreliable, but it is not a generic enterprise-mirror build.

## Toolchain distribution and cjv updates are separate

cjv separates two kinds of artifacts:

- **Toolchain distribution**: SDKs, stdx, docs, stdx-docs, and the LTS, STS, and nightly metadata. Enterprise deployments control this with `dist_server` or `CJV_DIST_SERVER`.
- **cjv itself**: `CJV_UPDATE_ROOT` can redirect the installer's initial download. Upgrades of an installed cjv should normally be owned by enterprise software distribution, with cjv self-update disabled.

With `dist_server` configured, cjv reads `<dist_server>/versions.json`, and that manifest must describe every channel and component in use. A missing nightly channel or artifact is an error; cjv does not fall back to GitCode or another public endpoint.

Without `dist_server`, compatibility behavior remains unchanged: LTS/STS use `manifest_url`, while nightly uses GitCode `nightly_build` Releases.

## Network destinations

| Operation | Default source | Enterprise replacement |
| --- | --- | --- |
| Install cjv itself | A GitHub or GitCode Release | Host release archives and `checksums.txt` internally; set `CJV_UPDATE_ROOT` for the installer |
| Install, query, or update toolchains | LTS/STS manifest; GitCode API for nightly | Configure one `dist_server` |
| Download SDKs and components | Manifest URLs or nightly Releases | Declare approved relative or absolute URLs in `<dist_server>/versions.json` |
| Run `cjv self update` | GitHub for the official build, GitCode for the mirror build | Disable self-update and upgrade cjv through enterprise software distribution |

The package repositories and credentials used by `cjpm` are outside cjv's scope. Deploying cjv solves SDK and component distribution; dependency mirrors must be configured separately.

Proceed through these sections:

1. [Build an internal distribution source](distribution-server.md)
2. [Deploy managed clients](client-deployment.md)
3. [Configure proxy-only or fully offline endpoints](restricted-networks.md)
4. [Define publishing, upgrade, and acceptance procedures](operations.md)
