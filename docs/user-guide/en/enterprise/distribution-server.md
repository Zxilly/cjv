# Internal distribution source

The `dist_server` setting in [`settings.toml`](../configuration.md) (or the `CJV_DIST_SERVER` [environment variable](../environment-variables.md)) configures the root URL from which `cjv` fetches version manifests and toolchain archives. For example:

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

Under that root URL, `cjv` reads two channel-specific manifest files:

```text
https://artifacts.corp.example/cjv/dist/versions.json  # LTS and STS channels
https://artifacts.corp.example/cjv/dist/nightly.json   # nightly channel
```

- LTS and STS operations read `versions.json` only.
- Nightly operations read `nightly.json` only.
- `cjv check` and `cjv update` load only the manifest files required by your locally installed channels.
- Relative SDK and component `url` fields inside a manifest are resolved against `dist_server`, whereas absolute `https://` URLs are used exactly as written.

Distribution source precedence is `CJV_DIST_SERVER`, then `dist_server` from merged settings, and finally `manifest_url`. In CI pipelines, you can set `CJV_DIST_SERVER` to test a staging mirror before promoting manifests to production.

## Recommended layout

We recommend organizing your internal artifact repository so `dist/` serves toolchains and manifests while `releases/` serves `cjv` binary releases:

```text
Enterprise artifact repository
└── cjv/
    ├── dist/                         # dist_server points here
    │   ├── versions.json             # LTS / STS and their components
    │   ├── nightly.json              # nightly and its components
    │   ├── sdk/
    │   ├── components/
    │   └── nightly/
    └── releases/                     # cjv archives and checksums.txt
```

The distribution endpoint simply needs to serve machine-readable, read-only files over `HTTPS GET`. You can enforce access control using network allowlists, device certificates, or an enterprise reverse proxy.

## `versions.json` contract

In `versions.json`, the top-level `channels` object contains the `lts` and `sts` channel objects:

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
    }
  }
}
```

## `nightly.json` contract

Unlike `versions.json`, `nightly.json` directly represents a single channel at the top level with `latest`, `versions`, and optional `components` (do **not** wrap `nightly.json` inside `channels.nightly`):

```jsonc
{
  "latest": "1.2.0-alpha.20260822010101",
  "versions": {
    "1.2.0-alpha.20260822010101": {
      "linux-x64": {
        "name": "cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
        "url": "nightly/20260822/cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
        "sha256": "<64-character hexadecimal SHA-256>"
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
```

Every SDK entry contains `name`, `url`, and `sha256`. On upstream nightly builds, `sha256` may temporarily be empty while artifacts finish publishing, in which case `cjv` fetches `<url>.sha256`; on an internal enterprise mirror, you should populate `sha256` directly in the manifest. Component entries also accept an optional `sha256` field—populating it ensures clients verify component archives and retain them in the reusable offline cache.

The version key under `versions` determines the toolchain version name, while `url` identifies the archive file to download. When installing or updating the tracked `nightly` channel, `cjv` starts from `latest` and searches published history in `nightly.json` if required components or cross targets are missing from the newest build (without downgrading an existing installation unless `--allow-downgrade` is passed). A full pinned nightly name selects that exact release directly.

## Component format

Within each channel object (`channels.lts`, `channels.sts`, or the top level of `nightly.json`), `components` is indexed by version. `docs` and `stdx-docs` each point to a single archive per version, whereas `stdx` is further indexed by component platform:

```jsonc
{
  "components": {
    "1.0.5": {
      "stdx": {
        "linux-x64": {
          "name": "stdx-linux-x64.tar.gz",
          "url": "components/stdx-linux-x64.tar.gz",
          "sha256": "<64-digit hexadecimal SHA-256>"
        }
      },
      "docs": { "name": "docs.tar.gz", "url": "components/docs.tar.gz" },
      "stdx-docs": { "name": "stdx-docs.tar.gz", "url": "components/stdx-docs.tar.gz" }
    }
  }
}
```

Combine this `components` fragment with `latest` and `versions` inside the channel object. [Cross-compilation](../cross-compilation.md) target SDKs are stored under `versions[version][platform]` using host-target keys such as `linux-x64-ohos`, while their corresponding `stdx` archives use component platform names such as `ohos-aarch64`.

## Publishing order

When updating your internal mirror, follow this sequence so clients never see a manifest pointing to missing files:

1. Mirror and verify all approved SDK, cross-target, and component archives.
2. Compute the 64-character SHA-256 digest for each SDK archive and component archive.
3. Upload all archives to their target paths first and verify `HTTPS GET` access from a standard client account.
4. Generate `versions.json` and `nightly.json` independently, retaining every exact version still referenced by project `cangjie-sdk.toml` files.
5. Atomically replace the manifest files on the server, advancing `latest` only after all artifacts for that release are in place.

See [Publication and validation](index.md#publication-and-validation) for the end-to-end client verification checklist.
