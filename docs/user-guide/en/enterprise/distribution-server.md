# Internal distribution source

`dist_server` is the toolchain artifact root. For example:

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv reads:

```text
https://artifacts.corp.example/cjv/dist/versions.json
```

Relative SDK and component URLs are resolved against `dist_server`. Absolute URLs are honored exactly as written and may point to another path or host. cjv treats the manifest as trusted administrator-published configuration; it does not implement the organization's network-access policy.

## Recommended layout

```text
Enterprise artifact repository
└── cjv/
    ├── dist/                         # dist_server points here
    │   ├── versions.json
    │   ├── sdk/                      # LTS / STS SDKs
    │   ├── components/               # LTS / STS components
    │   └── nightly/                  # nightly SDKs and components
    └── releases/                     # cjv archives and checksums.txt
```

Managed endpoints must be able to read the source with HTTPS GET. If access control is required, use network allowlists, device identity, or an enterprise reverse proxy that exposes a machine-readable read-only endpoint; do not depend on an interactive login page.

## Manifest contract

`versions.json` must contain `lts` and `sts`. To use nightly in an isolated intranet, it must also contain `nightly`. Every SDK entry requires `name`, `url`, and `sha256`. Component entries accept an optional `sha256`, which enterprise mirrors should provide whenever possible.

```jsonc
{
  "channels": {
    "lts": {
      "latest": "1.0.5",
      "versions": {
        "1.0.5": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.0.5.tar.gz",
            "url": "sdk/cangjie-sdk-linux-x64-1.0.5.tar.gz",
            "sha256": "<64-character hexadecimal SHA-256>"
          }
        }
      }
    },
    "sts": {
      "latest": "1.1.0",
      "versions": {
        "1.1.0": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.1.0.tar.gz",
            "url": "sdk/cangjie-sdk-linux-x64-1.1.0.tar.gz",
            "sha256": "<64-character hexadecimal SHA-256>"
          }
        }
      }
    },
    "nightly": {
      "latest": "1.2.0-alpha.20260822010101",
      "versions": {
        "1.2.0-alpha.20260822010101": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
            "url": "nightly/20260822/cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
            "sha256": "<64-character hexadecimal SHA-256>",
            "release_tag": "1.1.0-alpha.20260822010101"
          }
        }
      },
      "components": {
        "1.2.0-alpha.20260822010101": {
          "docs": {
            "name": "cangjie-docs-html-1.2.0-alpha.20260822010101.tar.gz",
            "url": "nightly/20260822/cangjie-docs-html-1.2.0-alpha.20260822010101.tar.gz",
            "sha256": "<64-character hexadecimal SHA-256>"
          }
        }
      }
    }
  }
}
```

`release_tag` is optional. Set it only when the upstream Release tag differs from the SDK asset version. The installed toolchain name still uses the version key, while the manifest URL remains authoritative for downloads.

The component layout is shared across channels: `docs` and `stdx-docs` are single entries, while `stdx` is keyed by artifact-platform name. Keep every approved platform in the enterprise manifest.

## How nightly works

Under a unified distribution source, nightly uses the same manifest as LTS and STS:

- `cjv install nightly` installs `channels.nightly.latest`.
- `cjv install nightly-<version>` installs that exact manifest version.
- `cjv check`, `cjv update nightly`, and `cjv toolchain list-remote --channel nightly` read the same source.
- Nightly SDKs and components use manifest URLs and checksums, so no GitCode API token is required.
- A missing nightly channel, target, or component fails locally and never triggers a public fallback.

cjv does not backtrack day by day when the latest nightly lacks a component, as rustup does. Publish all approved platforms and components for one nightly, verify them, and only then advance `latest`. Projects should pin an exact nightly version instead of following a moving alias.

## Publishing order

1. Mirror and verify the approved SDK and component archives.
2. Compute each SDK SHA-256; compute and publish component hashes as well.
3. Publish all archives first and verify HTTPS GET from a standard endpoint account.
4. Generate `versions.json`. Intranet deployments normally use relative paths or approved internal absolute URLs.
5. Atomically replace the manifest; advance a channel's `latest` only after its artifacts are complete.
6. Retain every exact version still referenced by project toolchain files.

`CJV_DIST_SERVER` overrides `dist_server` from settings and is useful for temporarily selecting a staging source in CI. If neither is set, cjv uses the legacy `manifest_url` / GitCode nightly model.
