# Installing a Toolchain from a URL or Archive

The `<path>` in `cjv toolchain link <name> <path>` can take three forms: a local directory, a local archive file (`.zip` / `.tar.gz`), or an `http(s)://` URL. When given a directory, cjv only creates a link to it; when given an archive — whether a local file or a URL — cjv extracts it and materializes it into a real toolchain owned by cjv.

## Two behaviors: reference and materialize

A local directory uses reference mode; a local archive or HTTP(S) URL uses materialize mode.

|Aspect|Reference mode (local directory)|Materialize mode (local archive / URL)|
|------|--------------------------------|--------------------------------------|
|`<path>` form|Local directory, e.g. `/path/to/sdk`|Local archive `sdk.zip`, or `https://...`|
|`toolchains/<name>` contents|References the source directory|An installation managed by cjv|
|Data ownership|cjv does not own it, only references it|Owned by cjv|
|Bundled stdx|Not applicable|Optional, auto-installed (see below)|
|Uninstall behavior|Only the link is deleted; the original directory is kept|Deletes the entire directory (including stdx)|
|Whether the default toolchain is changed|No|No|

A local archive is read in place and is never moved or deleted. For local-directory references, see [Toolchains](concepts/toolchains.md) and [Components](concepts/components.md).

```bash
# Materialize: download from a URL, extract, and materialize a real directory owned by cjv
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip

# Materialize: extract from a local archive into a real directory owned by cjv (the source file is kept)
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip

# Reference (for contrast): just create a link to the local directory
cjv toolchain link mysdk /path/to/local/sdk
```

## The name must be a custom name

`<name>` must be a custom name. It cannot collide with the reserved channel names `lts`, `sts`, `nightly`, contain path separators or a `+` prefix, or be empty, `.`, or `..`:

```bash
# Error: lts is a reserved channel name
cjv toolchain link lts https://example.com/sdk.zip
```

## Flags

Materialize mode supports three flags, applying equally to a local archive and a URL:

|Flags|Effect|
|-----|------|
|`--sha256 <hex>`|Verify the SHA-256 of the archive. When omitted, only the archive format is checked|
|`--force`|Overwrites and reinstalls when `toolchains/<name>` already exists|
|`--no-stdx`|Does not install the bundled stdx even if the archive contains stdx|

These three flags apply only in materialize mode (a local archive or a URL). If you pass any of them together with a local directory, cjv reports an error stating that the flag does not apply when linking a local directory, rather than silently ignoring it.

```bash
# Verify the SHA-256 of the archive (works for a local archive too)
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip \
  --sha256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

# Overwrite an existing toolchain of the same name
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.1.0.zip --force

# Install only the SDK, skipping the bundled stdx
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip --no-stdx
```

Without `--sha256`, a URL relies on TLS and a local archive is treated as trusted. Always provide SHA-256 when content integrity must be verified.

## Expected archive format

Materialize mode supports `cangjie-build` CI artifacts and bare SDK archives.

### CI build artifact (nested layout)

The `cangjie-<target>-<version>` build artifact downloaded from GitHub Actions is an outer ZIP that can contain these archives:

```text
<outer .zip>
├── cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>            (inner SDK, required)
└── cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip> (inner stdx, optional)
```

- The outer archive is a `.zip`.
- The inner archive is normally `.tar.gz` on Linux and `.zip` on Windows.
- The inner SDK extracts to a single top-level directory `cangjie/`, containing `bin/`, `lib/`, `tools/`, `runtime/`, and so on.
- The inner stdx extracts to a single top-level directory `<platform>_cjnative/` (such as `linux_x86_64_cjnative`, `windows_x86_64_cjnative`), containing `dynamic/` and `static/`.

### Bare SDK archive

An archive may instead contain a single top-level SDK directory. That directory must have a complete SDK layout; this form has no bundled stdx.

## Automatic installation of bundled stdx

When the archive contains `cangjie-stdx-*` and `--no-stdx` is not given, cjv installs the stdx component together with the SDK.

```bash
# archive contains stdx -> SDK and stdx are installed together
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip

# verify stdx is in place
cjv component list --toolchain mysdk
```

See [Components](concepts/components.md) for stdx management.

## Only the current system is supported

A materialize install supports only an SDK that matches the current operating system. Installing an SDK for another system reports an error.

```bash
# Trying to install a Windows SDK on Linux -> errors out before materializing; toolchains/<name> is not created
cjv toolchain link winsdk https://example.com/cangjie-windows-x64-1.0.0.zip
```

If you need to prepare an SDK for another platform, run the installation on that platform, or use the target SDK mechanism of [Cross-compilation](cross-compilation.md).

## Uninstall: cjv owns it, truly deleted

A materialized toolchain is owned by cjv, and uninstalling it actually deletes the materialized directory, including the bundled stdx:

```bash
cjv toolchain uninstall mysdk
```

This deletes `toolchains/mysdk/`, `stdx/mysdk/`, and `docs/mysdk/` (if present). Uninstalling in reference mode deletes only the link entry, leaving the original directory untouched.

Further reading: toolchain resolution priority is covered in [Toolchains](concepts/toolchains.md), components and stdx in [Components](concepts/components.md), runtime environment injection in [Runtime environment](runtime-environment.md), and the full command signatures in the [Command reference](command-reference.md).
