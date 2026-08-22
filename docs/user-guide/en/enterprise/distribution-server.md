# Internal distribution source

`dist_server` is the toolchain artifact root. For example:

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv reads:

```text
https://artifacts.corp.example/cjv/dist/versions.json
```

Relative SDK and component URLs are resolved against `dist_server`. Absolute URLs are honored exactly as written and may point to another path or host.

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

The distribution endpoint gives managed clients machine-readable, read-only access through HTTPS GET. Network allowlists, device identity, or an enterprise reverse proxy can enforce access control.

## Manifest contract

`versions.json` contains `lts` and `sts`; enterprise nightly adds the `nightly` channel. Every SDK entry contains `name`, `url`, and `sha256`. Component entries accept an optional `sha256`, which enterprise mirrors should normally provide.

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

Set the optional `release_tag` field when the upstream Release tag differs from the SDK asset version. The installed toolchain name uses the version key, while the manifest URL remains authoritative for downloads.

The component layout is shared across channels: `docs` and `stdx-docs` are single entries, while `stdx` is keyed by artifact-platform name. Keep every approved platform in the enterprise manifest.

## How nightly works

Under a unified distribution source, nightly uses the same manifest as LTS and STS:

- `cjv install nightly` installs `channels.nightly.latest`.
- `cjv install nightly-<version>` installs that exact manifest version.
- `cjv check`, `cjv update nightly`, and `cjv toolchain list-remote --channel nightly` read the same source.
- Nightly SDKs and components use manifest URLs and checksums, with version resolution supplied directly by the enterprise manifest.
- A missing nightly channel, target, or component returns the corresponding missing-content error.

`channels.nightly.latest` selects one exact version, whose targets and components form a publication invariant. Publish and verify all approved artifacts for that nightly before advancing `latest`. Projects gain reproducible builds by pinning an exact nightly version.

## Publishing order

1. Mirror and verify the approved SDK and component archives.
2. Compute each SDK SHA-256; compute and publish component hashes as well.
3. Publish all archives first and verify HTTPS GET from a standard endpoint account.
4. Generate `versions.json`. Intranet deployments normally use relative paths or approved internal absolute URLs.
5. Atomically replace the manifest; advance a channel's `latest` only after its artifacts are complete.
6. Retain every exact version referenced by project toolchain files.

Source precedence is `CJV_DIST_SERVER`, `dist_server` from settings, then the `manifest_url` / GitCode nightly compatibility model. CI can use the environment variable to select a staging source.
