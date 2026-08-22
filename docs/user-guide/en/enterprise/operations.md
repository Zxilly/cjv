# Publishing, upgrades, and acceptance

## Upgrading cjv itself

`dist_server` distributes toolchains, while the enterprise software system upgrades cjv itself. A recommended flow is:

1. Retrieve the cjv release archives and matching `checksums.txt` in a controlled environment.
2. Verify and publish them through the enterprise artifact or software distribution system.
3. Set `auto_self_update = "disable"` on clients.
4. Upgrade cjv through GPO, endpoint management, a system package, or a base image.

`CJV_UPDATE_ROOT` selects the installer's initial download location; later versions are published through the enterprise upgrade flow above.

## Nightly publication policy

Nightly is an ordinary channel in the unified manifest, but its publication policy should be stricter than LTS or STS:

- Mirror every approved platform, target SDK, and component for one version before advancing `channels.nightly.latest`.
- `latest` selects one exact version; a missing target or component returns its corresponding error.
- Retain every nightly version pinned by a `cangjie-sdk.toml` file.
- When an upstream Release tag differs from the SDK version, record `release_tag` as the authoritative mapping.
- Publish manifest updates atomically so clients always observe a complete file.

## Deployment acceptance checklist

- Run `cjv toolchain list-remote --channel lts` and `--channel nightly` from a standard user session; confirm both only hit `<dist_server>/versions.json`.
- Install an approved nightly with only `dist_server` configured; confirm it succeeds.
- Install the approved version and components, then run `cjv which cjc` and `cjc --version`.
- Review the host SDK, cross target, stdx, docs, and stdx-docs URLs in the manifest and confirm they match the organization's approved destinations.
- Temporarily remove nightly or a component from the manifest; confirm the command returns the explicit missing-content error and inspect the distribution access log.
- Disconnect the network and compile again; confirm the installed toolchain completes a local build.
- Confirm `auto_install = false` and `auto_self_update = "disable"`, with upgrades owned by the enterprise process.
- Verify enterprise CA, proxy variables, and `NO_PROXY` under real user and CI service accounts.
- Use network ACLs to restrict endpoints to approved download destinations.

System fallback settings provide consistent defaults. Enforcement comes from network ACLs, endpoint permissions, and the software distribution process together.
