# Custom SDKs: directories, archives, and URLs

Use `cjv toolchain link` when you need to register an unpacked SDK directory, install a local `.zip` or `.tar.gz` archive, or download a custom SDK archive from an HTTP or HTTPS URL. This chapter compares directory links with archive installations and describes the supported archive layouts.

## Register a custom SDK

Run `cjv toolchain link` with a custom name and a local path or URL:

```bash
cjv toolchain link local-sdk /path/to/sdk
cjv toolchain link archive-sdk ./cangjie-sdk.zip
cjv toolchain link remote-sdk https://example.com/cangjie-sdk.tar.gz
```

Each command registers an independent toolchain under the name that you give. The custom name must obey the naming rules in [Toolchains and versions](concepts/toolchains.md#channel-and-version-names) and cannot be an official name such as `lts`, `sts`, or `nightly`.

Linking a toolchain does not change your global default selection. Run `cjv default <name>` if you want to make the custom SDK your default, or reference `<name>` in [cangjie-sdk.toml](toolchain-file.md).

## Compare directory links and archive installs

Directory links and archive installations differ in file ownership, component support, and removal behavior:

| Source type | How cjv installs it | Component support | What `cjv uninstall` removes |
| --- | --- | --- | --- |
| Unpacked directory | Creates a reference to your directory; checks for `bin/cjc` (`cjc.exe` on Windows) | Maintained by the directory owner; `cjv component` cannot edit it | Removes the cjv link and keeps your source directory |
| Local archive or URL | Extracts files into a managed directory under `<CJV_HOME>/toolchains/` | Uses bundled `stdx` or accepts `cjv component link stdx` | Deletes the managed SDK copy and its components; keeps your local archive file |

Read [Components and offline documentation](concepts/components.md#manage-components-on-custom-sdks) for examples of linking `stdx` to an archive-installed custom SDK.

## Archive installation options

These flags apply only when the source is a local archive file or an HTTP/HTTPS URL:

| Option | Effect |
| --- | --- |
| `--sha256 <hex>` | Check the archive against the expected 64-character hexadecimal SHA-256 hash |
| `--force` | Replace an existing custom toolchain that has the same name |
| `--no-stdx` | Skip installing a bundled `stdx` archive from an outer bundle |

Do not pass `--sha256`, `--force`, or `--no-stdx` when linking an unpacked directory; cjv returns an error. If you omit `--sha256` for an archive, cjv skips the expected hash check but still checks the archive structure and SDK layout.

## Supported archive layouts

### Bare SDK archive

A bare `.zip` or `.tar.gz` archive contains a single complete `cangjie/` SDK directory with `bin/`, `lib/`, `tools/`, and `runtime/`.

### Outer CI bundle archive

cjv also accepts the outer `.zip` bundle produced by `cangjie-build` CI jobs. That outer archive contains an inner SDK archive and an optional inner `stdx` archive:

```text
cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>
cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip>  # optional
```

Inside the inner `stdx` archive, cjv expects `dynamic/` and `static/` under a platform directory. By default, cjv installs the bundled `stdx` automatically unless you pass `--no-stdx`. If the `stdx` extraction fails, cjv keeps the installed SDK and reports a warning.

### Host platform requirement

An archive installation must match your current host operating system. If you need to compile code for another target operating system, install a [cross-compilation](cross-compilation.md) target SDK that runs on your host instead of installing a regular SDK built for another host.

## Inspect and remove an archive SDK

Use the standard component and toolchain commands to inspect or remove a custom archive installation:

```bash
cjv component list --toolchain archive-sdk
cjv toolchain uninstall archive-sdk
```

For air-gapped enterprise deployments that use custom archives, read [Enterprise and offline deployment](enterprise/index.md#fully-offline-clients).
