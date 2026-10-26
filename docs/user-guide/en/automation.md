# Automation and JSON interfaces

`cjv` is designed to work reliably in CI pipelines, build scripts, and editor integrations. This chapter covers pinning project versions, isolating CI configuration and offline caches, running non-destructive diagnostics and update dry runs, rolling back channel updates, and consuming `cjv`'s `--json` output.

## Pin a project version

In a project repository, running `cjv pin lts` writes the resolved concrete release name (such as `lts-1.0.5`) to [`cangjie-sdk.toml`](toolchain-file.md) in the current directory while preserving any existing `components`, `targets`, and custom keys (comments and formatting are rewritten). If you omit the toolchain name (`cjv pin`), `cjv` pins whichever toolchain is currently active. When pinning a tracked channel that is already installed on your machine, `cjv pin` prefers your locally installed release first and never downloads an SDK.

Commit `cangjie-sdk.toml` to version control so every contributor and CI job selects the exact same SDK version:

```bash
cjv pin lts
cjv pin --check
```

In CI pipelines, `cjv pin --check` verifies that the current directory's `cangjie-sdk.toml` pins a concrete release without modifying the file or making network requests. Because `cangjie-sdk.toml` fixes the exact version and the distribution manifest supplies SHA-256 checksums for every artifact, no separate lockfile is needed.

## Configuration and network access

### Isolating CI directories and settings

In CI jobs, you often want to isolate both the data directory and the user settings file inside the workspace so builds do not read or mutate the runner's home directory:

```bash
export CJV_HOME="$PWD/.cjv-data"
export CJV_CONFIG_FILE="$PWD/ci-settings.toml"
cjv install lts
cjv --offline --no-auto-install exec cjc --version
```

`CJV_HOME` must be an absolute path. Note that `--config` and `CJV_CONFIG_FILE` replace only the user settings file; system fallback settings, `cangjie-sdk.toml` project selection, and other [environment variables](environment-variables.md) still apply. If your runner also needs to isolate or override the system fallback file, set `CJV_FALLBACK_SETTINGS` separately.

### Offline caches and network flags

To run `cjv` with `--offline` (or `CJV_OFFLINE=1`), the required manifests, SDK archives, and component archives from the configured distribution source must be present in `<CJV_HOME>/cache/` beforehand. Whenever an online installation succeeds, `cjv` retains the manifest and all SHA-256-verified archives in `cache/` (archives without checksums are not retained in the reusable cache). You can inspect the cached items with `cjv cache list --json`, and supply the populated `cache/` directory alongside `cangjie-sdk.toml` to your offline build environment under the same distribution source configuration.

If `cache/` contains damaged metadata records, orphaned archive files, or incomplete temporary files, `cjv cache list --json` reports them via the optional `problem` field while still listing healthy entries. Running `cjv cache clean` removes all `cjv`-owned cache files (including damaged ones), whereas `cjv cache prune --max-age <duration>` filters entries by age. Note that if an online query succeeds but saving its optional cache entry fails, `cjv` prints a warning to `stderr` and still returns the successful query result.

- **`--no-auto-install`** prevents proxy commands and `cjv exec` from implicitly installing missing SDKs or components; explicit `cjv install` commands remain available.
- **`--offline`** prevents `cjv` itself from making network requests, but does not restrict network access by child programs spawned via `run`, `exec`, or proxy binaries.

## Inspect and repair

You can use these commands in automation or troubleshooting scripts to inspect environment health, preview updates, or repair an installation:

```bash
cjv doctor --json
cjv update --dry-run --json
cjv install lts --reinstall
```

- **`cjv doctor`** performs a read-only check of your settings, active toolchain selection, proxy entry points in `<CJV_HOME>/bin`, installation records, and the presence of key SDK files, suggesting concrete repair commands when it detects an issue. It does not compute SHA-256 digests for every individual file inside an installed SDK, and it exits with a nonzero code (`1`) whenever a check fails.
- **`cjv update --dry-run`** resolves compatible target releases and any update blockers without downloading archives, modifying installations, or refreshing `cache/`.
- **`cjv install <name> --reinstall`** prepares and installs the named SDK release again while preserving your selected components and cross targets. By contrast, `--force` continues to mean skipping and removing unavailable optional components or targets—it does not reinstall an unchanged SDK.

## Update history and rollback

Whenever updating a tracked channel (`lts`, `sts`, `nightly`) changes the release or SHA-256 digest of its host SDK or an installed cross target, `cjv` retains a complete snapshot of the previous toolchain group—including the host SDK, tracked cross targets, offline documentation, and `stdx`—under `<CJV_HOME>/history/`.

By default, each tracked channel retains its **2** most recent snapshots. Initial installations, `--reinstall` repairs of the same release, and standalone `target add` or `component add` operations do not create update snapshots. Fixed-version installations (such as `lts-1.0.5`) remain independent of tracking-channel history.

```bash
cjv toolchain history lts
cjv toolchain rollback lts
cjv toolchain history prune lts --keep 1
```

- **`cjv toolchain rollback <channel> [snapshot]`** restores the newest snapshot when `[snapshot]` is omitted, or restores a specific snapshot ID when given. Before replacing the installation, `rollback` verifies the local snapshot's contents and saves the current toolchain group as a snapshot so that the rollback itself can be undone.
- **`cjv toolchain history prune [channel] [--keep N]`** removes older snapshots. Because each snapshot stores independent file copies (including the target contents of linked components), snapshots consume additional disk space and are not removed by `cjv cache clean`. Passing `--keep 0` clears all snapshots while preserving your active installations. Note that `--keep` applies only to that `prune` invocation; future updates still retain 2 snapshots automatically. Never edit files inside `<CJV_HOME>/history/` by hand.

### Handling damaged snapshots

`cjv toolchain history` lists valid, readable snapshots in `snapshots` and reports any damaged records—along with their failure causes—in the optional `invalid_snapshots` array.

When you run `history prune` with a nonzero `--keep` value, `cjv` retains that many readable snapshots and leaves damaged entries in place so you can inspect them; running `history prune --keep 0` removes both readable and damaged entries. If the newest snapshot is damaged, running `cjv toolchain rollback <channel>` without an explicit snapshot ID refuses to silently guess an older release and requires you to specify a valid `<snapshot>` ID explicitly. Every selected snapshot must pass full content verification before restoration.

## JSON compatibility

Commands that support `--json` write a single structured JSON document to `stdout` and send all logs or warnings to `stderr`. The `run`, `exec`, and `init` commands do not support `--json` because their `stdout` and exit codes belong directly to the child process or interactive wizard.

When building automation around `--json`, follow these compatibility rules:

- **Per-command success payloads**: Success fields are specific to each command. Clients should ignore unrecognized object keys and enum values they do not use.
- **Versioned schemas (`schema_version`)**: Payloads that include a `schema_version` field preserve existing field meanings and types within that version. Adding new optional fields does not increment `schema_version`; incompatible changes increment it.
- **Unversioned payloads**: Existing commands without a `schema_version` field keep their current output structure; do not assume they share fields with other commands.
- **Collections and optional fields**: Required collections are always JSON arrays (`[]` when empty), while optional fields may be omitted or `null` according to each command's existing behavior.
- **Machine-readable error handling**: Always branch on the process exit code and `error.code`, never on the localized human-readable `error.message`.

For example, an empty cache returns:

```json
{"schema_version":1,"path":"/home/user/.cjv/cache","entries":[]}
```

The following JSON Schema describes the stable fields of `cjv cache list --json` while allowing future optional fields:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["schema_version", "path", "entries"],
  "properties": {
    "schema_version": {"const": 1},
    "path": {"type": "string"},
    "entries": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["key", "kind", "url", "size", "updated"],
        "properties": {
          "key": {"type": "string"},
          "kind": {"type": "string", "examples": ["metadata", "artifact"]},
          "url": {"type": "string"},
          "sha256": {"type": "string"},
          "size": {"type": "integer", "minimum": 0},
          "updated": {"type": "string", "format": "date-time"}
        }
      }
    }
  }
}
```

### Error objects and automatic installation causes

When a command fails, `cjv` writes `{"error":{"code":"...","message":"...","details":{}}}` to `stdout` and exits with a nonzero status code. New error codes may be added over time, so clients should handle unknown `code` values gracefully; the fields inside `details` depend on the specific `code`.

`cjv doctor --json` is an intentional exception: when it detects diagnostic problems, it still outputs its normal report object containing `checks` with `"ok": false` and exits with code `1`, rather than wrapping the response in an `error` object.

When automatic installation fails while resolving a toolchain or component, the top-level error code remains `TOOLCHAIN_NOT_INSTALLED` or `COMPONENT_NOT_INSTALLED`, and `details.cause` preserves the underlying failure's `code`, `message`, and `details`:

- **`OFFLINE_CACHE_MISS`** — A required manifest or archive was not found in `<CJV_HOME>/cache/` while running with `--offline`.
- **`CHECKSUM_MISMATCH`** — A downloaded or cached archive failed SHA-256 verification.
- **`HTTP_ERROR`** and **`NETWORK_ERROR`** — Distinguish unsuccessful HTTP status responses from lower-level connection or transport failures.
- **`CANCELED`** and **`DEADLINE_EXCEEDED`** — Indicate that the operation was canceled by a signal or exceeded the download timeout.

When automatic installation is disabled (`--no-auto-install`, `CJV_AUTO_INSTALL=0`, or `auto_install = false`), a missing installation has no `details.cause` because no installation was attempted.
