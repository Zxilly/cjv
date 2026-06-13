# Targets and Overrides

Each time a proxied SDK tool (such as `cjc`, `cjpm`) runs, cjv has to answer two questions: which toolchain to use, and which cross-compilation targets to carry on that toolchain. The former is decided by a prioritized resolution chain; the latter by targets, an additional install dimension.

## Toolchain resolution priority

cjv resolves the active toolchain in the following priority order, from highest to lowest, taking the first source that matches:

1. the `CJV_TOOLCHAIN` environment variable
1. Directory override (set via `cjv override set`)
1. Toolchain file (a `cangjie-sdk.toml` in the current directory or some ancestor directory)
1. Default toolchain (set via `cjv default`)

Proxying, `cjv exec`, `cjv envsetup`, and most commands that take a `[toolchain]` argument all use the same resolution chain. Which toolchain gets used in any given directory is consistent and predictable.

### 1. The `CJV_TOOLCHAIN` environment variable

Setting `CJV_TOOLCHAIN` unconditionally overrides all other sources, which suits CI, containers, or ad-hoc verification scenarios:

```bash
CJV_TOOLCHAIN=nightly cjc --version
```

It has the highest priority; directory overrides, the toolchain file, and the default toolchain are all ignored. See [Environment variables](../environment-variables.md).

### 2. Directory override

A directory override binds a specific directory (and its subdirectories) to a toolchain. This is stored in the global `settings.toml`, not in the project. It suits cases where you do not want to, or cannot, put a `cangjie-sdk.toml` in the project, for example when the repository is not yours, or when you only want to switch locally for a while.

### 3. The toolchain file `cangjie-sdk.toml`

cjv searches upward from the current directory for `cangjie-sdk.toml`, using the first file it finds as the project's toolchain declaration. This is the project-level toolchain anchor that is committed with the repository and takes effect for collaborators. See [The toolchain file](../toolchain-file.md) for the full field reference.

### 4. Default toolchain

When none of the three above match, cjv falls back to the global default toolchain set via `cjv default`:

```bash
cjv default lts
```

If not even a default toolchain is set, cjv reports an error indicating that no toolchain has been configured yet.

### How overrides and toolchain files interleave along the directory tree

Levels 2 and 3 do not scan all overrides first and then all toolchain files. Instead, walking up the directory tree level by level, at each level it checks both the directory override and `cangjie-sdk.toml`:

- At the same directory level, the directory override takes precedence over that level's `cangjie-sdk.toml`;
- A toolchain file closer to the current directory takes precedence over a directory override higher up.

In other words, priority depends on both source type and distance. If a directory override is set on `~/work` while `~/work/proj` has a `cangjie-sdk.toml`, then when working under `~/work/proj` the closer toolchain file wins; the override only takes effect when some level has no closer file and matches the override. This way a project's own declaration is not accidentally shadowed by a broad override on an ancestor directory.

 >
 > Tip: when `cangjie-sdk.toml` exists but `channel` is empty, cjv does not silently fall back; it reports an error to remind you to complete the declaration. Only when the file does not exist at all does the search continue upward.

## Managing directory overrides

### Setting an override

```bash
# Set an override for the current directory
cjv override set nightly

# Set one for a specified directory (no need to cd into it first)
cjv override set lts --path /path/to/project
```

`cjv override set` validates the toolchain name first. Standard names (such as `lts`, `sts`, `nightly`, or a specific version) are normalized before being stored, while custom toolchain names are accepted as written. The directory path is normalized to an absolute path before being written, with symbolic links resolved and the drive letter case unified on Windows, so different spellings of the same directory do not produce duplicate entries.

### Removing an override

```bash
# Remove the override for the current directory
cjv override unset

# Remove the override for a specified directory
cjv override unset --path /path/to/project

# Clean up all overrides pointing at "directories that no longer exist"
cjv override unset --nonexistent
```

`--nonexistent` is suited to periodic cleanup. After a project directory is deleted, its override entry stays behind in `settings.toml`. This command removes, in one pass, every override whose target directory no longer exists.

### Listing overrides

```bash
cjv override list
```

The output is sorted by directory path, with each line formatted as `directory → toolchain`. When there are no overrides at all, a corresponding message is shown.

## Cross-compilation targets

A target is a dimension orthogonal to toolchain resolution. It answers not which toolchain to use, but which cross-compilation SDKs this toolchain should additionally carry.

A target SDK is an additional install on top of the host toolchain; it does not change the currently active toolchain. When proxying `cjc` or `cjpm`, the host SDK is still used. Installing a target only readies the corresponding cross-compilation SDK so that you can produce binaries for that target when needed.

### Fill in the suffix only

Whether on the command line or in `cangjie-sdk.toml`, `targets` takes only the target suffix, such as `ohos`, `android`, or `ohos-arm32`; do not write a full platform key (such as `linux-x64-ohos`). cjv appends the suffix to the current host tuple to form the full target automatically. The suffix must match `^[a-z0-9]+(?:-[a-z0-9]+)*$`, and must not itself be a full platform tuple.

### Adding targets at install time

```bash
# Install the host STS SDK, and additionally install the OHOS cross SDK matching the current host
cjv install sts -t ohos

# target supports repetition or comma separation
cjv install sts -t ohos -t android
cjv install sts --target ohos,android
```

### Declaring targets in a project

A project can declare additional `targets` in `cangjie-sdk.toml`. When `auto_install` is enabled, proxy execution will automatically fill in the missing target SDKs:

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

Duplicate suffixes are deduplicated automatically. `targets` is unrelated to toolchain resolution; it only adds cross-compilation capability on top of the toolchain you have selected. For how to drive a cross-compilation build, and for the runtime environment of a target SDK, see [Cross-compilation](../cross-compilation.md).
