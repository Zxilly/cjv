# Toolchains

A toolchain is the basic unit cjv manages, a complete, self-contained Cangjie SDK installation. A toolchain contains at least the compiler `cjc`, the package manager `cjpm`, and the accompanying runtime libraries. On top of it you can also mount [components](components.md) (such as `stdx`, `docs`) and [cross-compilation targets](targets-overrides.md).

cjv can install multiple toolchains at the same time and switch between them. Each toolchain has a unique name, which you use in almost every command to refer to it:

```bash
cjv install lts          # Install a toolchain named lts
cjv default sts          # Set the default toolchain to sts
cjv run nightly cjc -V   # Run cjc with the nightly toolchain
cjv uninstall lts        # Uninstall lts
```

All installed toolchains live under `<CJV_HOME>/toolchains/<name>/`, where `CJV_HOME` defaults to `~/.cjv`.

## Forms of a toolchain name

A toolchain name can take one of the following forms. The first few are recognized by cjv directly and downloaded from official sources; the last one, `custom`, is created explicitly by you.

|Form|Example|Description|
|----|-------|-----------|
|Channel name|`lts`, `sts`, `nightly`|Resolves to the current latest version of that [channel](channels.md)|
|Channel name + version|`lts-1.0.5`, `sts-1.1.0-beta.23`|A specific version within that channel|
|Bare version number|`1.0.5`|A version number without a channel prefix, looked up across all channels|
|custom|`my-sdk`, `local-build`|Created with `cjv toolchain link`, see below|

### Channel name

`lts`, `sts`, and `nightly` are three channels. Used on their own, a channel name is resolved by cjv to that channel's current latest version. Channel names are case-insensitive, so `LTS`, `Lts`, and `lts` are equivalent.

```bash
cjv install lts      # Install the latest LTS
cjv install nightly  # Install the latest nightly
```

For the semantics, update cadence, and download sources of each channel, see [Channels](channels.md).

### Channel name + version

Append a `-` and a version number after a channel name to pin a specific version within that channel. The version number may include a pre-release suffix:

```bash
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23
cjv install nightly-1.1.0-alpha.20260306010001
```

A toolchain installed this way takes the full string you typed as its name, for example `lts-1.0.5`, and later commands refer to it by that name too:

```bash
cjv default lts-1.0.5
cjv uninstall sts-1.1.0-beta.23
```

### Bare version number

If you give only a version number (starting with a digit, without a channel prefix), cjv looks across all channels for an installed toolchain matching that version:

```bash
cjv run 1.0.5 cjc --version
```

Use it to refer to a specific installed version without remembering which channel it belongs to.

### custom: custom toolchains

Any name that does not match the forms above (not a channel name, not `channel-version`, and not starting with a digit) is treated as a custom toolchain. These toolchains do not come from the official source; you create them explicitly with `cjv toolchain link`:

```bash
cjv toolchain link my-sdk /path/to/local/sdk
```

Custom toolchains come from two sources, differing in who owns the data. See [Custom toolchains](#custom-toolchains) below.

## Naming rules

Whatever the form, a toolchain name must satisfy the following constraints, or the command reports an error:

- must not be empty;
- must not contain the path separators `/` or `\` (to prevent escaping outside the `toolchains/` directory);
- must not be `.` or `..`;
- It cannot start with a `+` prefix. `+` is the toolchain selection syntax in commands like `cjv exec` and `cjv envsetup`; just write the name directly;
- trailing `/` or `\` are stripped automatically.

In addition, in `cjv toolchain link`, a custom toolchain name cannot conflict with the reserved channel names: `lts`, `sts`, and `nightly` are taken by the official channels and cannot be used as link names.

## Custom toolchains

Custom toolchains let you bring SDKs from outside the official source under cjv's management, for example a locally built SDK, an internally distributed build, or an archive you downloaded temporarily. There are two ways to create one, differing in whether cjv owns the data.

### Source 1: link a local directory (cjv does not own the data)

The first way links in an existing local SDK directory. cjv creates a symlink under `<CJV_HOME>/toolchains/<name>/` pointing to your directory (on Windows it falls back to a directory junction), and does not copy any files:

```bash
cjv toolchain link my-sdk /path/to/local/sdk
```

The linked directory must be a real Cangjie SDK. cjv checks that `bin/cjc` exists in it, and refuses to link otherwise.

Because it is only a link, the original data still belongs to you. Any change you make in the source directory takes effect immediately through cjv; `cjv toolchain uninstall my-sdk` (as well as `cjv uninstall my-sdk`) only removes the link and does not touch your original directory. This way suits debugging a self-built SDK, or sharing one installation across several tools.

### Source 2: install from a URL (cjv owns the data)

When the second argument to `cjv toolchain link` is an `http://` or `https://` URL, cjv downloads the archive, extracts it, and materializes the contents under `<CJV_HOME>/toolchains/<name>/`, producing an installation fully owned by cjv:

```bash
cjv toolchain link my-sdk https://example.com/cangjie-sdk.tar.gz
```

In contrast to a local link, this toolchain's data is managed by cjv: `cjv toolchain uninstall my-sdk` actually deletes this directory and its components.

URL installation supports a few extra options. They apply only to a URL source and are rejected when used with a local path:

- `--sha256 <hash>`: verify the SHA-256 of the downloaded archive;
- `--force`: overwrite an already-installed toolchain of the same name;
- `--no-stdx`: skip auto-detecting and installing the bundled stdx.

For the full URL format conventions, archive layout requirements, verification behavior, and examples, see [Installing toolchains from a URL](../install-from-url.md).

### Attaching stdx to a custom toolchain

A custom toolchain created by linking a local directory has no corresponding official release asset, so `cjv component add stdx` does not work for it. When needed, use `cjv component link stdx` instead to attach a local stdx directory:

```bash
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

See [Components](components.md) for details.

## Viewing and managing toolchains

List all installed toolchains; custom toolchains also appear in the list:

```bash
cjv toolchain list
# Equivalent to
cjv show installed
```

View the currently active toolchain and the overall status:

```bash
cjv show
cjv show active
```

Setting the default toolchain, setting a directory override, and selecting a toolchain through an environment variable or `cangjie-sdk.toml` are the mechanisms that decide which toolchain is active in a given context. The priority rules for this are detailed in [Targets and overrides](targets-overrides.md) and [The toolchain file](../toolchain-file.md).
