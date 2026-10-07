# Custom SDKs: directories, archives, and URLs

`cjv toolchain link` accepts an existing SDK directory, a local `.zip` / `.tar.gz` archive, or an HTTP(S) URL:

```bash
cjv toolchain link local-sdk /path/to/sdk
cjv toolchain link archive-sdk ./cangjie-sdk.zip
cjv toolchain link remote-sdk https://example.com/cangjie-sdk.tar.gz
```

These examples use separate names and can be run independently. They do not change the default toolchain; use `cjv default <name>` when needed.

## Directories and archives

| Source | Installation | Uninstallation |
| --- | --- | --- |
| Extracted directory | Create a reference; requires `bin/cjc` (`cjc.exe` on Windows) | Remove the link and preserve the source SDK |
| Local archive or URL | Extract into a real directory managed by cjv | Delete the installed copy and components; preserve the local source archive |

The source SDK owner maintains a linked directory; cjv component commands cannot edit it. Archive installations can use bundled stdx or attach local stdx through `component link`.

The name must be a [custom toolchain name](concepts/toolchains.md), not an official name such as `lts`, `sts`, or `nightly`.

## Archive options

| Option | Effect |
| --- | --- |
| `--sha256 <hex>` | Verify content against the expected 64-digit hexadecimal SHA-256 |
| `--force` | Replace an existing toolchain with the same name |
| `--no-stdx` | Skip bundled stdx |

These options apply only to archives and URLs. Using them with a directory is an error. Without SHA-256, cjv does not check an expected content hash, but still validates the archive and SDK layout.

## Supported archives

A bare SDK archive contains one complete SDK directory. The outer ZIP produced by `cangjie-build` CI is also supported, with these inner archives:

```text
cangjie-sdk-<sdk_name>-<version>.<tar.gz|zip>
cangjie-stdx-<sdk_name>-<version>.<stdxver>.<tar.gz|zip>  # optional
```

The SDK contains `cangjie/` with its `bin/`, `lib/`, `tools/`, `runtime/`, and other SDK contents. The stdx archive contains `dynamic/` and `static/` inside a platform directory. Bundled stdx is installed unless `--no-stdx` is set. If stdx installation fails, the successfully installed SDK remains.

Archive installation requires an SDK for the current operating system. Cross-compilation uses a [target SDK](cross-compilation.md) that runs on the host; installing another operating system's regular SDK does not provide it.

```bash
cjv component list --toolchain archive-sdk
cjv toolchain uninstall archive-sdk
```
