# Project toolchains

Commit a `cangjie-sdk.toml` in the project root to share its toolchain requirements:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

The version is an example. See [toolchains and versions](concepts/toolchains.md) for name syntax.

## Selection order

1. An explicit command selection, such as global `+name` or a supported `--toolchain` option.
2. The `CJV_TOOLCHAIN` environment variable.
3. A search from the current directory to the filesystem root. At each level, cjv checks the directory override before `cangjie-sdk.toml` and stops at the first selection.
4. The default set by `cjv default`.

An override takes precedence over a project file in the same directory. A project file in a child directory takes precedence over a parent directory's override. Files from different levels are not merged. `cjv show active` reports the selected toolchain and its source.

`cjv run <toolchain> <command>` always uses its toolchain argument. When an explicit selection or environment variable takes precedence, the project file's component and target requirements are not attached to it.

## Fields

| Field | Type | Meaning |
| --- | --- | --- |
| `channel` | string | A channel, version selector, or registered custom name |
| `path` | string | An absolute SDK path; mutually exclusive with `channel` |
| `components` | string[] | Required `stdx`, `docs`, or `stdx-docs`; empty by default |
| `targets` | string[] | Required cross SDK suffixes, such as `ohos`; empty by default |

Specify exactly one of `channel` and `path`. For a direct SDK path, the SDK owner manages its components; `components` and `targets` do not trigger automatic installation:

```toml
[toolchain]
path = 'C:\SDKs\cangjie'
```

`targets` accepts suffixes, not full platform names such as `linux-x64-ohos`. Case and underscores are normalized, comma-separated entries are expanded, and duplicates are removed. Empty targets are errors.

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos", "android"]
```

This declaration prepares the host SDK, host components, and target SDKs. See [cross-compilation](cross-compilation.md) for selecting the target build environment.

## Automatic installation and errors

`auto_install` is enabled by default. When a proxy command such as `cjc` or `cjpm` runs, cjv installs missing requirements from the selected project file. With this setting disabled, missing requirements cause an error:

```bash
cjv set auto-install false
```

Unknown keys produce warnings while recognized fields are still parsed. Invalid TOML, unknown components, and a missing valid `channel` or `path` cause errors. An empty file also fails; delete the file to fall back to a parent directory or the default selection.

## Local directory overrides

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
cjv override list
cjv override unset
cjv override unset --nonexistent
```

Overrides are stored in user settings and apply to a directory and its descendants. `--nonexistent` removes overrides for directories that no longer exist.
