# Project toolchains

`cjv` automatically determines which [toolchain](concepts/toolchains.md) to use whenever you run an installed proxy command like `cjc` or `cjpm`. In a team project, you can check a `cangjie-sdk.toml` file into the root of your repository so that every developer and CI runner automatically selects the right SDK version, components, and cross-compilation targets:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

The version `lts-1.0.5` above is an example; see [Toolchains and versions](concepts/toolchains.md) for all supported channel and version formats.

Instead of writing the version string by hand, you can run `cjv pin lts` in your project root to write the resolved concrete release name to `cangjie-sdk.toml` (or `cjv pin` with no arguments to pin whichever toolchain is currently active). Existing `components` and `targets` entries in the file are preserved. In CI pipelines, you can run `cjv pin --check` to verify that the project file pins a concrete release without modifying it; see [Automation and JSON interfaces](automation.md#pin-a-project-version).

## Selection order

There are several ways to control and override which toolchain is active. `cjv` chooses the toolchain in the following order, using the first one that is specified:

1. An explicit command-line selection, such as a global `+name` shorthand (`cjv +nightly show`) or a command's `--toolchain` option.
2. The `CJV_TOOLCHAIN` [environment variable](environment-variables.md).
3. A directory walk from the current working directory up toward the filesystem root. At each directory level, `cjv` checks for a [local directory override](#local-directory-overrides) before checking for `cangjie-sdk.toml`, stopping at the first match it finds.
4. The global default toolchain set with `cjv default`.

Notice how step 3 works when walking up the directory tree: both directory overrides and `cangjie-sdk.toml` files are preferred by their proximity to your current directory. Within the _same_ directory, a local directory override takes precedence over `cangjie-sdk.toml`; however, a `cangjie-sdk.toml` file in a child directory takes precedence over a directory override set on a parent directory. `cangjie-sdk.toml` files at different directory levels are never merged.

To verify which toolchain is active in your current directory and see which rule selected it, run:

```bash
cjv show active
```

Note that `cjv run <toolchain> <command>` always uses its explicit `<toolchain>` argument. Whenever an explicit command-line selection or `CJV_TOOLCHAIN` overrides `cangjie-sdk.toml`, the `components` and `targets` listed in `cangjie-sdk.toml` are not attached to that override.

## The `cangjie-sdk.toml` file

The `cangjie-sdk.toml` file uses the [TOML](https://toml.io/) format. All settings live under the `[toolchain]` table:

| Field | Type | Meaning |
| --- | --- | --- |
| `channel` | `string` | A release channel, version selector, or registered custom toolchain name |
| `path` | `string` | An absolute path to a local SDK directory; mutually exclusive with `channel` |
| `components` | `string[]` | Additional components to install (`stdx`, `docs`, `stdx-docs`); empty by default |
| `targets` | `string[]` | Cross-compilation target suffixes to install (such as `ohos`); empty by default |

### Toolchain file settings

#### `channel` and `path`

Every `cangjie-sdk.toml` file must specify either `channel` or `path`, but never both:

- **`channel`** names an official release channel (`lts`, `sts`, `nightly`), a pinned version selector (`lts-1.0.5`, `1.2`, `nightly-2026-10-04`), or a custom toolchain name registered via `cjv toolchain link`.
- **`path`** points directly to an unpacked SDK directory on the local filesystem using an absolute path:

```toml
[toolchain]
path = 'C:\SDKs\cangjie'
```

Because a `path` directive points directly to an external SDK directory whose contents are managed by its owner, `components` and `targets` do not trigger automatic installation when `path` is used.

#### `components` and `targets`

When `channel` selects a managed toolchain, you can list additional [components](concepts/components.md) and [cross-compilation target SDKs](cross-compilation.md) that your project requires:

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos", "android"]
```

In `targets`, always write target suffixes such as `ohos` or `android` rather than full host-target platform names like `linux-x64-ohos`. `cjv` normalizes letter case and underscores, expands comma-separated items, and removes duplicates automatically; empty target strings are rejected with an error.

When `cangjie-sdk.toml` selects the active toolchain, this declaration ensures the host SDK, the listed host components, and the listed target SDKs are prepared. See [Cross-compilation](cross-compilation.md) for how to load a target SDK's build environment when compiling.

## Automatic installation

By default, the `auto_install` setting is `true`. Whenever you run a proxy command such as `cjc` or `cjpm` inside a project, `cjv` automatically downloads and installs any missing SDK, component, or target declared in the active `cangjie-sdk.toml`.

If you prefer missing requirements to fail immediately instead of downloading automatically, you can disable `auto_install` in your user settings:

```bash
cjv set auto-install false
```

You can also disable automatic installation for a single command invocation without changing your saved configuration by passing `--no-auto-install` (`cjv --no-auto-install exec cjc --version`) or by setting `CJV_AUTO_INSTALL=0` in the environment for proxy commands. Explicit `cjv install` commands always run regardless of this setting.

## Invalid project files

If `cangjie-sdk.toml` contains unrecognized keys under `[toolchain]`, `cjv` prints a warning to `stderr` while still honoring all recognized fields. However, invalid TOML syntax, unknown component names, an empty file, or a missing/invalid `channel` or `path` will cause the command to fail with an error rather than silently falling back to another toolchain. If you want a directory to fall back to a parent directory's configuration or the global default, delete the `cangjie-sdk.toml` file.

## Local directory overrides

Directories can be assigned their own local toolchain override with `cjv override`. When a directory has an override, running `cjc` or `cjpm` inside that directory—or any of its subdirectories—uses the override toolchain without touching the project's repository files:

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
cjv override list
cjv override unset
cjv override unset --nonexistent
```

Per-directory overrides are saved in your user [`settings.toml`](configuration.md) file. Passing `--nonexistent` to `cjv override unset` cleans up stale overrides for directories that have been deleted from disk.
