# Project toolchains

You can commit a `cangjie-sdk.toml` file in the root directory of a project to share its SDK, component, and target settings with your team. This chapter explains how to create and pin project files, how cjv selects the active toolchain, and how to use local directory overrides.

## Declare a project toolchain

Create `cangjie-sdk.toml` in the project root directory:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

The version `lts-1.0.5` is an example. Read [Toolchains and versions](concepts/toolchains.md) for all supported version and channel formats.

To write a concrete release name automatically, run `cjv pin`:

```bash
cjv pin lts
cjv pin --check
```

- `cjv pin lts` writes the resolved release name for `lts` into `cangjie-sdk.toml` and keeps existing component and target lists.
- `cjv pin` without a name pins the currently active toolchain.
- `cjv pin --check` checks that `cangjie-sdk.toml` pins a concrete release without changing the file. Use this flag in CI; read [Automation and JSON interfaces](automation.md#pin-a-project-version) for pipeline examples.

## Toolchain selection order

When you run a proxy command or a cjv subcommand, cjv selects the active toolchain in this sequence:

1. An explicit command-line selection, such as a global `+name` selector or a `--toolchain` flag.
2. The `CJV_TOOLCHAIN` environment variable; read [Environment variables](environment-variables.md).
3. A directory walk from the current working directory up to the filesystem root. At each directory level, cjv checks for a local directory override before `cangjie-sdk.toml`, and stops at the first match.
4. The global default toolchain set by `cjv default`.

Because cjv stops at the first directory level that has a match, a directory override takes precedence over `cangjie-sdk.toml` in the same directory. However, a `cangjie-sdk.toml` file in a child directory takes precedence over a directory override in a parent directory. cjv never merges `cangjie-sdk.toml` files from different directories. Run `cjv show active` at any time to see the selected toolchain and its source.

The `cjv run <toolchain> <command>` command always uses its `<toolchain>` argument. When an explicit CLI selector or `CJV_TOOLCHAIN` takes precedence over `cangjie-sdk.toml`, cjv does not attach the `components` or `targets` from the project file to that selection.

## Configuration fields

Place all fields under the `[toolchain]` table in `cangjie-sdk.toml`:

| Field | Type | Description |
| --- | --- | --- |
| `channel` | `string` | A channel name, version selector, or registered custom toolchain name |
| `path` | `string` | An absolute path to a local SDK directory; mutually exclusive with `channel` |
| `components` | `string[]` | Optional components to install: `stdx`, `docs`, or `stdx-docs` |
| `targets` | `string[]` | Optional cross SDK suffixes to install, such as `ohos` or `android` |

### Channel vs. direct SDK path

You must set either `channel` or `path`, but never both. When you set `path`, cjv uses that external SDK directory directly:

```toml
[toolchain]
path = 'C:\SDKs\cangjie'
```

The owner of an external SDK directory manages its components. When `path` is set, cjv ignores `components` and `targets` during automatic installation.

### Components and target SDKs

In the `targets` list, write target suffixes such as `ohos` or `android` rather than full platform names such as `linux-x64-ohos`. cjv normalizes letter case and underscores, expands comma-separated items, and removes duplicates. An empty target string causes an error.

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos", "android"]
```

When `cangjie-sdk.toml` selects the active toolchain, this declaration prepares the host SDK, the host [components](concepts/components.md), and the [cross-compilation](cross-compilation.md) target SDKs. To compile code with a target SDK environment, read [Run commands and set the environment](runtime-environment.md#load-a-cross-compilation-environment).

### Validation and errors

If `cangjie-sdk.toml` contains unknown keys, cjv prints a warning and parses the known fields. If the file has invalid TOML syntax, an unknown component name, an empty body, or no valid `channel` or `path`, cjv stops with an error instead of falling back to the default toolchain. Delete the file if you want to restore selection from a parent directory or the global default.

## Automatic installation

The `auto_install` setting is `true` by default. When you run a proxy command such as `cjc` or `cjpm`, cjv automatically installs any missing SDK, component, or target listed in the active `cangjie-sdk.toml`.

If you disable `auto_install`, missing items cause an immediate error instead of a download:

```bash
cjv set auto-install false
```

To disable automatic installation for a single command without changing your saved [configuration](configuration.md), pass `--no-auto-install` or set `CJV_AUTO_INSTALL=0`:

```bash
cjv --no-auto-install exec cjc --version
```

Explicit `cjv install` commands always run regardless of `auto_install`.

## Local directory overrides

Use `cjv override` when you want to test a directory with another toolchain on your machine without editing `cangjie-sdk.toml`:

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
cjv override list
cjv override unset
cjv override unset --nonexistent
```

cjv stores directory overrides in your user `settings.toml` file, and each override applies to that directory and its subdirectories. Pass `--nonexistent` to clean up overrides for directories that you deleted from disk. Read [Command reference](command-reference.md#override) for all flags.
