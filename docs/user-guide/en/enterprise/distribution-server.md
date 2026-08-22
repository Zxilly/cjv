# Internal distribution source

`dist_server` is the toolchain artifact root. For example:

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv reads two channel-specific files:

```text
https://artifacts.corp.example/cjv/dist/versions.json  # LTS / STS
https://artifacts.corp.example/cjv/dist/nightly.json   # nightly
```

LTS/STS operations read only `versions.json`; nightly operations read only `nightly.json`. `cjv check` and `cjv update` load the files selected by installed channels. Relative SDK and component URLs resolve against the distribution root, while absolute URLs are honored exactly as written.

## Recommended layout

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

The distribution endpoint gives managed clients machine-readable, read-only access through HTTPS GET. Network allowlists, device identity, or an enterprise reverse proxy can enforce access control.

## `versions.json` contract

The top-level `channels` object contains `lts` and `sts`:

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

`nightly.json` directly represents one channel with `latest`, `versions`, and optional `components`:

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

Every SDK entry contains `name`, `url`, and `sha256`. A nightly `sha256` may temporarily be empty, in which case cjv reads `<url>.sha256`; enterprise mirrors should normally populate it directly. Component entries accept an optional `sha256`. The version key determines the toolchain name, while the URL identifies the exact asset.

`cjv install nightly` installs `latest` from `nightly.json`. Pinned nightly installs, `check`, `update`, `list-remote`, and component installation all read that same file. Projects gain reproducible builds by pinning an exact nightly version.

## Publishing order

1. Mirror and verify the approved SDK and component archives.
2. Compute each SDK SHA-256; compute and publish component hashes as well.
3. Publish all archives first and verify HTTPS GET from a standard endpoint account.
4. Generate `versions.json` and `nightly.json` independently.
5. Atomically replace the corresponding file; advance its `latest` after artifacts are complete.
6. Retain every exact version referenced by project toolchain files.

Source precedence is `CJV_DIST_SERVER`, `dist_server` from settings, then `manifest_url`. CI can use the environment variable to select a staging source.
