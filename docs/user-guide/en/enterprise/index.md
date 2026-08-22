# Enterprise deployment overview

An enterprise network can let cjv reach upstream services through a controlled proxy or publish toolchains and components to an internal HTTPS artifact repository. Network requests are concentrated in installation, updates, remote queries, and automatic acquisition; daily commands for installed toolchains run locally.

## Choose a deployment mode

| Network condition | Recommended mode | Supported scope |
| --- | --- | --- |
| Public services are reachable through an enterprise proxy | Set the standard proxy environment variables | All channels and self-update, subject to upstream availability |
| Endpoints can reach only an internal artifact repository | Configure a unified `dist_server` | LTS, STS, nightly, SDKs, and components can all stay internal |
| Fully offline or air-gapped | Use local archives or directory links | Installed or locally supplied toolchains |
| Sources and versions must be centrally enforced | Internal source plus network ACLs and enterprise software distribution | Fallback settings provide defaults; network ACLs enforce access policy |

The `mirror` build variant provides GitCode default endpoints for networks where GitHub is unreliable. Enterprise mirrors use `dist_server`.

## Toolchain distribution and cjv updates are separate

cjv separates two kinds of artifacts:

- **Toolchain distribution**: SDKs, stdx, docs, stdx-docs, and the LTS, STS, and nightly metadata. Enterprise deployments control this with `dist_server` or `CJV_DIST_SERVER`.
- **cjv itself**: `CJV_UPDATE_ROOT` selects the installer's initial download location. Enterprise software distribution owns upgrades of installed cjv binaries, with clients configured as `auto_self_update = "disable"`.

With `dist_server` configured, cjv reads `<dist_server>/versions.json` as the authoritative description of every channel and component. A missing nightly channel or artifact produces an explicit error while the configured manifest remains authoritative.

Compatibility mode resolves LTS/STS through `manifest_url` and nightly through GitCode `nightly_build` Releases.

## Network destinations

| Operation | Default source | Enterprise replacement |
| --- | --- | --- |
| Install cjv itself | A GitHub or GitCode Release | Host release archives and `checksums.txt` internally; set `CJV_UPDATE_ROOT` for the installer |
| Install, query, or update toolchains | LTS/STS manifest; GitCode API for nightly | Configure one `dist_server` |
| Download SDKs and components | Manifest URLs or nightly Releases | Declare approved relative or absolute URLs in `<dist_server>/versions.json` |
| Run `cjv self update` | GitHub for the official build, GitCode for the mirror build | Disable self-update and upgrade cjv through enterprise software distribution |

The project-specific configuration of `cjpm` supplies dependency repositories and credentials. Enterprise environments configure those dependency mirrors separately, while cjv distributes SDKs and components.

Proceed through these sections:

1. [Build an internal distribution source](distribution-server.md)
2. [Deploy managed clients](client-deployment.md)
3. [Configure proxy-only or fully offline endpoints](restricted-networks.md)
4. [Define publishing, upgrade, and acceptance procedures](operations.md)
