# Custom SDKs: directories, archives, and URLs

`cjv toolchain link` references an unpacked SDK directory; `cjv toolchain import` creates a managed installation from a local archive or HTTP(S) URL:

```bash
cjv toolchain link local-sdk /path/to/sdk
cjv toolchain import archive-sdk ./cangjie-sdk.zip
cjv toolchain import remote-sdk https://example.com/cangjie-sdk.tar.gz
```

Each of these commands registers an independent custom toolchain under the name you choose. Linking a custom toolchain does not change your global default selection; run `cjv default <name>` if you want to make it the default, or set `channel = "<name>"` in a project's [`cangjie-sdk.toml`](toolchain-file.md).

The name you choose must follow the [custom toolchain naming rules](concepts/toolchains.md#channels-and-names) and cannot be an official channel or version name such as `lts`, `sts`, or `nightly`.

## Directory links vs. archive installations

Depending on whether you pass an unpacked directory or an archive/URL, `cjv` manages the files differently:

| Source | How `cjv` installs it | How `cjv` uninstalls it |
| --- | --- | --- |
| Unpacked directory | Creates a lightweight reference to the directory; requires `bin/cjc` (`cjc.exe` on Windows) | Removes the link in `cjv` and leaves your source SDK directory untouched |
| Local archive or URL | Extracts the archive into a real directory under `<CJV_HOME>/toolchains/` managed by `cjv` | Deletes the installed SDK copy and its attached components, while preserving your local source archive file |

This distinction also affects [component management](concepts/components.md#managing-components-on-custom-sdks):

- **Linked directories** are maintained directly by the owner of that directory, so `cjv component` commands cannot add, link, or remove components on them.
- **Archive and URL installations** live in a directory owned by `cjv`, so they can automatically install a bundled `stdx` archive or have a local `stdx` attached via `cjv component link stdx`.

## Archive installation options

`cjv toolchain import` accepts these archive installation options:

| Option | Effect |
| --- | --- |
| `--sha256 <hex>` | Verify the archive against an expected 64-character hexadecimal SHA-256 digest before extracting |
| `--replace` | Replace an existing custom toolchain with the same name |
| `--no-stdx` | Skip installing a bundled `stdx` archive when extracting an outer CI bundle |

These three options apply only to archives and URLs; passing them when linking an unpacked directory produces an error. If you omit `--sha256` for an archive or URL, `cjv` skips checking an expected digest but still validates the archive format and SDK directory structure.

The old archive/URL forms of `toolchain link` still work with a migration note; their `--force` replacement flag corresponds to `import --replace`.

## Supported archive layouts

`cjv toolchain import` supports two archive layouts out of the box:

1. **A bare SDK archive** (`.tar.gz` or `.zip`) containing a single complete `cangjie/` SDK directory with its `bin/`, `lib/`, `tools/`, `runtime/`, and other standard subdirectories.
2. **An outer CI bundle ZIP** (such as the artifact produced by `cangjie-build` CI workflows) that wraps an inner SDK archive and an optional inner `stdx` archive:

```text
cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>
cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip>  # optional
```

Inside the inner `stdx` archive, `cjv` looks for `dynamic/` and `static/` inside a platform directory. When a bundled `stdx` archive is present, `cjv` installs it automatically unless you pass `--no-stdx`. If extracting the bundled `stdx` fails, `cjv` keeps the successfully installed SDK and reports a warning.

Note that archive installations require an SDK built for your current host operating system. If you want to cross-compile for another operating system, attach a [cross-compilation target SDK](cross-compilation.md) that runs on your host rather than installing another operating system's host SDK archive.

Once installed, you can inspect or remove an archive-based custom toolchain with the standard commands:

```bash
cjv component list --toolchain archive-sdk
cjv uninstall archive-sdk
```
