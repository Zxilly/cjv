# Installing a Toolchain from a URL or Archive

The `<path>` in `cjv toolchain link <name> <path>` can take three forms: a local directory, a local archive file (`.zip` / `.tar.gz`), or an `http(s)://` URL. When given a directory, cjv only creates a link to it; when given an archive — whether a local file or a URL — cjv extracts it and materializes it into a real toolchain owned by cjv.

## Two behaviors: reference and materialize

`cjv toolchain link` first checks whether `<path>` matches `^https?://`: if so, it downloads and then materializes; otherwise it treats the path as local — a regular file is materialized as an archive, a directory is only referenced via a link.

|Aspect|Reference mode (local directory)|Materialize mode (local archive / URL)|
|------|--------------------------------|--------------------------------------|
|`<path>` form|Local directory, e.g. `/path/to/sdk`|Local archive `sdk.zip`, or `https://...`|
|`toolchains/<name>` contents|Symlink / junction pointing to your directory|The real directory materialized after extraction|
|Data ownership|cjv does not own it, only references it|Owned by cjv|
|Bundled stdx|Not applicable|Optional, auto-installed (see below)|
|Uninstall behavior|Only the link is deleted; the original directory is kept|Deletes the entire directory (including stdx)|
|Whether the default toolchain is changed|No|No|

In materialize mode a local archive and a URL go through the same extraction, materialization, and stdx-install logic; there is exactly one difference: a URL is first downloaded to `<CJV_HOME>/downloads/` for staging and cleaned up on success, whereas a local archive is read in place and cjv never moves or deletes your file. This chapter covers materialize mode; for referencing a local directory see [Toolchains](concepts/toolchains.md) and [Components](concepts/components.md).

```bash
# Materialize: download from a URL, extract, and materialize a real directory owned by cjv
cjv toolchain link mysdk https://example.com/cangjie-linux-x64-1.0.0.zip

# Materialize: extract from a local archive into a real directory owned by cjv (the source file is kept)
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip

# Reference (for contrast): just create a link to the local directory
cjv toolchain link mysdk /path/to/local/sdk
```

## The name must be a custom name

`<name>` must be a custom name. It cannot collide with the reserved channel names `lts`, `sts`, `nightly`, and it cannot contain path separators, a `+` prefix, or be an illegal name such as empty, `.`, `..`. This check applies to all three forms and runs before any download or extraction happens:

```bash
# Error: lts is a reserved channel name; no download is triggered
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

When `--sha256` is not provided, for a URL cjv relies on TLS transport security, and for a local archive it trusts the file you supplied. In both cases it checks that the file is a valid archive (zip or gzip) before extraction, rejecting content that is obviously corrupt or not an archive. The inner SDK and stdx archives are not verified separately; if the outer archive is intact, the inner contents are trusted. For end-to-end integrity, provide `--sha256`.

## Expected archive format

Materialize mode expects the archive to be a build artifact produced by the `cangjie-build` CI, and also accepts a bare SDK archive directly. The detection logic is identical whether the archive comes from a local file or a URL.

### CI build artifact (nested layout)

The build artifact `cangjie-<target>-<version>` downloaded from GitHub Actions is an outer ZIP that contains exactly two files:

```text
<outer .zip>
├── cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>            (inner SDK, required)
└── cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip> (inner stdx, optional)
```

- The outer wrapper is always a `.zip`. This is the packaging behavior of GitHub `actions/upload-artifact` at download time, not something cangjie-build produces itself.
- The inner archive is a `.tar.gz` on Linux and a `.zip` on Windows. The same extraction logic handles both.
- The inner SDK extracts to a single top-level directory `cangjie/`, containing `bin/`, `lib/`, `tools/`, `runtime/`, and so on.
- The inner stdx extracts to a single top-level directory `<platform>_cjnative/` (such as `linux_x86_64_cjnative`, `windows_x86_64_cjnative`), containing `dynamic/` and `static/`.

When scanning the top level of the outer zip, cjv matches by prefix: an inner archive whose filename starts with `cangjie-sdk-` is the SDK (required), one starting with `cangjie-stdx-` is stdx (optional); any other extra files (such as README, checksums) are silently ignored.

### Direct-archive fallback

If the archive is itself a bare SDK archive, that is, it extracts to exactly one top-level directory rather than two `cangjie-*` inner archives, cjv treats that directory as the SDK root and materializes it directly, without extracting a second time. In this case there is no bundled stdx.

If the archive contains neither a `cangjie-sdk-*` inner archive nor a single top-level directory (for example, a pile of loose files), cjv reports the error "no cangjie-sdk-\* archive found inside the archive" and makes no further attempt.

## Automatic installation of bundled stdx

When the archive contains `cangjie-stdx-*` and `--no-stdx` is not given, cjv installs that stdx as the toolchain's stdx component automatically, materializing it to `dynamic/` and `static/` under `<CJV_HOME>/stdx/<name>/` and writing the component manifest. Later, during proxying, `CANGJIE_STDX_PATH_DYNAMIC` and `CANGJIE_STDX_PATH_STATIC` are injected automatically, with no manual configuration.

```bash
# archive contains stdx -> SDK and stdx are installed together
cjv toolchain link mysdk ./cangjie-linux-x64-1.0.0.zip

# verify stdx is in place
cjv component list --toolchain mysdk
```

A locally linked stdx (`cjv component link stdx`) creates only a symlink, whereas a materialized stdx is real data owned by cjv and is deleted along with the toolchain on uninstall. For more details on the component mechanism, see [Components](concepts/components.md).

## Only the current system is supported

A materialize install supports only an SDK that matches the current operating system; cross-OS installs are not supported. If the SDK targets a different system than the current one (for example, installing a Windows or macOS SDK on Linux), cjv errors out before anything is materialized. This applies to both nested artifacts and bare archives.

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

## Behavior notes

- Does not change the default toolchain: like referencing, a materialize install does not set the new toolchain as the default. Use `cjv default mysdk` to set it explicitly when needed.
- Download staging (URL only): in URL mode the outer zip is downloaded to `<CJV_HOME>/downloads/` for staging and cleaned up on success. A local archive is not staged — it is extracted in place and never moved or deleted. Any failure partway through rolls back entirely, leaving no half-built `toolchains/<name>`.
- Staging kept on download failure: in URL mode, if `--sha256` does not match or the download is interrupted, the outer zip is kept for retry; the staged file is cleaned up only on a successful install.
- No symlink permission on Windows: proxy links fall back to junctions automatically, the symlinks inside stdx are materialized as copies, and the install does not abort.

Further reading: toolchain resolution priority is covered in [Toolchains](concepts/toolchains.md), components and stdx in [Components](concepts/components.md), runtime environment injection in [Runtime environment](runtime-environment.md), and the full command signatures in the [Command reference](command-reference.md).
