# Publishing, upgrades, and acceptance

## Upgrading cjv itself

The toolchain `dist_server` does not control cjv self-update. Enterprise deployments should:

1. Retrieve the cjv release archives and matching `checksums.txt` in a controlled environment.
2. Verify and publish them through the enterprise artifact or software distribution system.
3. Set `auto_self_update = "disable"` on clients.
4. Upgrade cjv through GPO, endpoint management, a system package, or a base image.

`CJV_UPDATE_ROOT` can redirect the installer's initial download, but it is not persisted as the update source of the installed cjv binary.

## Nightly publication policy

Nightly is an ordinary channel in the unified manifest, but its publication policy should be stricter than LTS or STS:

- Mirror every approved platform, target SDK, and component for one version before advancing `channels.nightly.latest`.
- If the version selected by `latest` lacks a requested target or component, cjv fails instead of choosing an older nightly automatically.
- Retain every nightly version pinned by a `cangjie-sdk.toml` file.
- When an upstream Release tag differs from the SDK version, record `release_tag` in the SDK entry instead of relying on client-side filename guesses.
- Publish manifest updates atomically so clients never observe a partially written file.

## Deployment acceptance checklist

- Run `cjv toolchain list-remote --channel lts` and `--channel nightly` from a standard user session; confirm both only hit `<dist_server>/versions.json`.
- Install an approved nightly without configuring `CJV_GITCODE_API_KEY`; confirm it succeeds.
- Install the approved version and components, then run `cjv which cjc` and `cjc --version`.
- Review the host SDK, cross target, stdx, docs, and stdx-docs URLs in the manifest and confirm they match the organization's approved destinations.
- Temporarily remove nightly or a component from the manifest; confirm the command fails and does not reach GitHub, GitCode, or another upstream endpoint.
- Disconnect the network and compile again; installed toolchains must not trigger downloads.
- Confirm `auto_install = false` and `auto_self_update = "disable"`, with upgrades owned by the enterprise process.
- Verify enterprise CA, proxy variables, and `NO_PROXY` under real user and CI service accounts.
- Use network ACLs to confirm endpoints cannot bypass the internal distribution source.

System fallback settings provide consistent defaults. Enforcement comes from network ACLs, endpoint permissions, and the software distribution process together.
