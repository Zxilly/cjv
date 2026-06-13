# The Toolchain File

`cangjie-sdk.toml` is a toolchain declaration file placed in a project. With it, a project pins to a particular toolchain, and can also declare the cross-compilation targets and components the project needs. When anyone runs `cjc`, `cjpm`, or similar commands in this directory (or a subdirectory), cjv switches to the declared toolchain automatically, with no need to run `cjv default` or set environment variables by hand.

For its position in the toolchain resolution chain, see [Targets and overrides](concepts/targets-overrides.md). The `CJV_TOOLCHAIN` environment variable and directory overrides have higher priority, and the default toolchain has lower priority.

```toml
[toolchain]
channel = "lts"                  # required
components = ["stdx", "docs"]    # optional
targets = ["ohos", "android"]    # optional
```

## File location and lookup

Starting from the current working directory, cjv searches up the directory tree for a file named `cangjie-sdk.toml`, uses the first file it finds, and stops there; it does not merge files from multiple levels. A toolchain file in a subdirectory therefore overrides one in a parent directory.

For example, given the following directory structure:

```text
~/work/
  cangjie-sdk.toml        # channel = "lts"
  project/
    cangjie-sdk.toml      # channel = "sts"
    src/
```

When running a command in `~/work/project/src/`, the first file cjv finds going upward is `~/work/project/cangjie-sdk.toml`, so it uses `sts`, and the `lts` file in `~/work/` is shadowed. The search continues upward to the filesystem root.

 >
 > Priority at the same level: if a level has both a directory override (`cjv override set`) and a `cangjie-sdk.toml`, the directory override at that level wins. But a toolchain file closer to the current directory still beats a directory override higher up.

## Field reference

All fields live under the `[toolchain]` table. The table name must be exactly `toolchain`.

### `channel`

|Item|Value|
|----|-----|
|Type|string|
|Required|Yes|
|Default|None|

The toolchain name, that is, the identifier you normally pass to `cjv install`. It can be a channel name (`lts`, `sts`, `nightly`), or a precise name with a version (such as `lts-1.0.5` or `nightly-1.1.0-alpha.20260306010001`). For how channels and versions are written, see [Channels](concepts/channels.md) and [Toolchains](concepts/toolchains.md).

```toml
[toolchain]
channel = "lts"
```

```toml
[toolchain]
channel = "lts-1.0.5"
```

`channel` must not be empty. A toolchain file that is found but whose `channel` is empty (an empty file, `channel = ""`, or only unrecognized keys) is treated as an incomplete configuration and reports an error directly; cjv does not skip it and continue resolving the next level. See [Empty file and empty channel](#empty-file-and-empty-channel) below.

### `components`

|Item|Value|
|----|-----|
|Type|string\[\]|
|Required|No|
|Default|`[]` (empty)|

Declares the [components](concepts/components.md) that the project needs to have ready alongside the toolchain, such as the extension library `stdx` and the offline docs `docs` and `stdx-docs`.

```toml
[toolchain]
channel = "lts"
components = ["stdx", "docs"]
```

When the auto-install conditions are met (see [Relationship to `auto_install`](#relationship-with-auto_install) below), cjv installs any component listed here but not yet present before proxying. When the conditions are not met, a missing component causes the command to terminate with a "component not installed" error, prompting you to run `cjv component add` by hand.

Component names are validated; an unknown component name causes an error. For the list of available components, see [Components](concepts/components.md).

### `targets`

|Item|Value|
|----|-----|
|Type|string\[\]|
|Required|No|
|Default|`[]` (empty)|

Declares the [cross-compilation targets](cross-compilation.md) the project needs; each entry is just the target suffix, such as `ohos`, `android`, or `ohos-arm32`.

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

The following rules apply:

- Write only the suffix, not a full platform key. `ohos` is correct; a full SDK target tuple like `linux-x64-ohos` is rejected with an error.
- Case and underscores are normalized: `OHOS` and `ohos_arm32` are normalized to `ohos` and `ohos-arm32` respectively.
- A single string may use comma separation, which is equivalent to writing multiple entries: `targets = ["ohos,android"]` is equivalent to `targets = ["ohos", "android"]`.
- Empty targets are not allowed: `targets = ["ohos", ""]` or `targets = [","]` will report an error.
- Duplicate entries are automatically deduplicated.

As with `components`, when the auto-install conditions are met, cjv installs any target SDK declared here but missing before proxying; otherwise it prompts you with a "toolchain not installed" error to run `cjv install <toolchain> --target <suffix>` by hand.

A target SDK is an additional install on the host toolchain; it does not change the currently active toolchain, and `cjc` and `cjpm` still run with the host SDK. For the full description, see [Cross-compilation](cross-compilation.md).

## Unrecognized keys

An unrecognized key does not fail parsing; it is reported only at the `warn` log level (the log level can be controlled with `CJV_LOG`, see [Environment variables](environment-variables.md)), and the other recognized fields take effect as usual. A common cause is a typo:

```toml
[toolchian]        # table name misspelled, should be [toolchain]
channal = "lts"    # key name misspelled, should be channel
```

In the example above no recognized field is read, so `channel` is effectively empty, which in turn triggers the [empty channel error](#empty-file-and-empty-channel). When a toolchain file appears to be configured but has no effect, first check for warn logs of this kind.

 >
 > Note: a misspelled key name only warns and does not error, but a TOML syntax error does. For example, `[toolchain` missing its closing bracket causes the whole parse to fail and terminates the command.

## Empty file and empty channel

As soon as `cangjie-sdk.toml` is found, cjv assumes you intend to declare a toolchain here. If the file exists but `channel` is empty after parsing, cjv reports an error instead of quietly falling back. All of the following cases count as an empty `channel`:

- a completely empty file;
- `channel = ""` is written;
- only unrecognized keys are written (such as the typo in the previous section), leaving no valid `channel`.

All of these cases produce an error similar to the following, pointing out the specific file path:

```text
…/cangjie-sdk.toml: toolchain.channel is empty; please specify a channel (e.g. lts, sts, nightly)
```

This is designed to avoid a hard-to-notice problem: thinking you have switched to some toolchain while in fact the default toolchain is quietly used. If you really want a directory to fall back to the level above or to the default toolchain, delete the file rather than emptying it.

## Relationship with `auto_install`

`channel` decides which toolchain to use; `targets` and `components` decide which targets and components are readied alongside it. Whether the latter two are filled in automatically depends on `auto_install` in the user settings:

- When `auto_install` is on (the default, corresponding to `cjv set auto-install true`), proxying (directly calling `cjc`, `cjpm`, and so on) automatically installs, before running, the target SDKs and components declared in the toolchain file but still missing on this machine.
- When `auto_install` is off (`cjv set auto-install false`), cjv does not install automatically; a missing target or component causes the command to terminate with the corresponding "not installed" error and prompts you to install it by hand.

For the meaning of `auto_install` and how to set it, see [Proxies](concepts/proxies.md) and [Configuration](configuration.md).

`targets` and `components` take effect only when the toolchain is resolved from `cangjie-sdk.toml`. If the currently active toolchain comes from a higher-priority source, such as the `CJV_TOOLCHAIN` environment variable or a `+toolchain` given explicitly to `cjv run`/`cjv exec`, then the `targets` and `components` in the toolchain file are not applied (and in that case the file may not even be read). They are part of the project's toolchain declaration and come from the same file as `channel`.

## Complete example

A project that cross-compiles for OpenHarmony and needs the extension library and offline documentation:

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos"]
```

With auto-install enabled, the first time a team member runs `cjpm build` in this directory, cjv automatically installs the `sts` toolchain, the `ohos` target SDK, and the `stdx` and `docs` components, with no extra steps required:

```bash
cjv set auto-install true
cd my-ohos-project
cjpm build        # missing toolchains / targets / components are auto-installed
```

## Related chapters

- [Targets and overrides](concepts/targets-overrides.md): the position of the toolchain file in the full resolution chain
- [Toolchains](concepts/toolchains.md) and [Channels](concepts/channels.md): the values `channel` can take
- [Components](concepts/components.md): the values `components` can take
- [Cross-compilation](cross-compilation.md): the meaning of `targets` and the target SDK model
- [Configuration](configuration.md) and [Proxies](concepts/proxies.md): the setting and behavior of `auto_install`
- [Environment variables](environment-variables.md): `CJV_TOOLCHAIN`, `CJV_LOG`, and others
