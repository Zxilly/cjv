# Internal distribution source

The `dist_server` setting in [settings.toml](../configuration.md) (or the `CJV_DIST_SERVER` [environment variable](../environment-variables.md)) sets the root URL for toolchain manifests and archives. This chapter defines the directory layout, the JSON schemas for `versions.json` and `nightly.json`, and the safe publication sequence for an internal mirror.

## How cjv resolves manifests and URLs

Suppose you configure this `dist_server` URL:

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv fetches two manifest files under that root URL:

```text
https://artifacts.corp.example/cjv/dist/versions.json  # LTS and STS channels
https://artifacts.corp.example/cjv/dist/nightly.json   # nightly channel
```

- LTS and STS commands read `versions.json` only.
- Nightly commands read `nightly.json` only.
- `cjv check` and `cjv update` load only the manifest files for your installed channels.
- If an archive `url` inside a manifest is a relative path, cjv resolves it relative to `dist_server`. If `url` is an absolute `https://` URL, cjv uses that exact URL.

Precedence for the distribution source is `CJV_DIST_SERVER`, then `dist_server` in merged settings, then `manifest_url`. CI pipelines can set `CJV_DIST_SERVER` to test a staging mirror before production rollout.

## Recommended repository layout

Organize your internal artifact repository so `dist/` serves SDKs and manifests while `releases/` serves the `cjv` binary itself:

```text
Enterprise artifact repository
└── cjv/
    ├── dist/                         # Point dist_server here
    │   ├── versions.json             # LTS and STS channels and components
    │   ├── nightly.json              # nightly channel and components
    │   ├── sdk/                      # Stable SDK archives
    │   ├── components/               # Component archives (stdx, docs, stdx-docs)
    │   └── nightly/                  # Nightly SDK and component archives
    └── releases/                     # cjv binary archives and checksums.txt
```

Configure the HTTPS server to give client machines read-only `GET` access. You can control access with network allowlists, device certificates, or an enterprise reverse proxy.

## `versions.json` schema

The `versions.json` file wraps the `lts` and `sts` channel objects inside a top-level `channels` object:

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

## `nightly.json` schema

Unlike `versions.json`, `nightly.json` represents the `nightly` channel at the top level with `latest`, `versions`, and optional `components`. Do not wrap `nightly.json` inside `channels.nightly`:

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

Every SDK entry contains `name`, `url`, and `sha256`. On upstream nightly builds, `sha256` may be empty briefly while the build finishes, in which case cjv fetches `<url>.sha256`. On an internal enterprise server, always populate the 64-character `sha256` field directly in the JSON.

The version key in `versions` defines the toolchain version string, while `url` gives the archive path. When a client installs `nightly`, cjv starts at `latest` and searches earlier versions in `nightly.json` if the client needs specific components or cross targets. By default, cjv does not downgrade an installed nightly toolchain.

## Component and cross-target entries

Inside each channel object (`channels.lts`, `channels.sts`, or the root of `nightly.json`), the `components` object indexes component archives by version:

```jsonc
{
  "components": {
    "1.0.5": {
      "stdx": {
        "linux-x64": {
          "name": "stdx-linux-x64.tar.gz",
          "url": "components/stdx-linux-x64.tar.gz",
          "sha256": "<64-character hexadecimal SHA-256>"
        }
      },
      "docs": { "name": "docs.tar.gz", "url": "components/docs.tar.gz" },
      "stdx-docs": { "name": "stdx-docs.tar.gz", "url": "components/stdx-docs.tar.gz" }
    }
  }
}
```

- `docs` and `stdx-docs` are platform-independent and map directly to one archive object per version.
- `stdx` groups its archive objects by component platform name (such as `linux-x64` for a host or `ohos-aarch64` for a cross target).
- [Cross-compilation](../cross-compilation.md) SDK archives live under `versions[version][platform]` using host-target keys such as `linux-x64-ohos`.
- Component entries accept an optional `sha256` field. Populate `sha256` for all component archives on internal mirrors so clients check integrity and cache the archives for `--offline` use.

## Safe publication procedure

Obey this sequence whenever you publish new releases to your internal distribution server:

1. Download and check all approved SDK, cross-target, and component archives.
2. Compute the 64-character SHA-256 hash for every SDK and component archive.
3. Upload all archive files to their target paths first, and test `HTTPS GET` access from a standard client account.
4. Generate the updated `versions.json` and `nightly.json` files with all existing pinned versions kept in place.
5. Atomically replace `versions.json` and `nightly.json` on the server, advancing `latest` only after every archive for that release is reachable.

Read [Enterprise and offline deployment](index.md#publication-and-validation-checklist) for the client validation checklist.
