# Automation and JSON interfaces

This chapter explains how to use cjv in CI pipelines, automated scripts, and offline build environments. It covers project version pinning, isolated CI directories, offline caches, diagnostics, update rollback, and the JSON output contract.

## Pin a project version

In a team or CI repository, run `cjv pin` to write a concrete SDK release name into [cangjie-sdk.toml](toolchain-file.md) in the current directory:

```bash
cjv pin lts
cjv pin --check
```

- `cjv pin lts` writes the resolved release name for `lts` and keeps existing `components`, `targets`, and custom keys. It reformats the file and removes TOML comments.
- If you omit the toolchain name, `cjv pin` pins the active selection. If a tracked channel is already installed locally, `cjv pin` uses that local release first without downloading an SDK.
- `cjv pin --check` checks that `cangjie-sdk.toml` pins a concrete version without writing to disk or using the network.

Commit `cangjie-sdk.toml` to version control. Because `cangjie-sdk.toml` fixes the exact version and the distribution manifest supplies SHA-256 checksums, you do not need a separate lockfile.

## CI isolation and offline caches

### Isolate CI settings and data

In CI jobs, set an dedicated data directory and user settings file so builds do not read or change the runner's home directory:

```bash
export CJV_HOME="$PWD/.cjv-data"
export CJV_CONFIG_FILE="$PWD/ci-settings.toml"
cjv install lts
cjv --offline --no-auto-install exec cjc --version
```

The `CJV_HOME` path must be an absolute path. The `--config` flag and `CJV_CONFIG_FILE` replace the user settings file only; system fallback settings, `cangjie-sdk.toml`, and other [environment variables](environment-variables.md) still apply. Set `CJV_FALLBACK_SETTINGS` if you also need to isolate the system fallback file.

### Prepare and inspect an offline cache

To run `cjv install` with `--offline`, you must populate `<CJV_HOME>/cache/` ahead of time from the same distribution source. When you install a toolchain online, cjv saves the manifest and all SDK and component archives with valid SHA-256 hashes in `cache/`. Archives without a SHA-256 checksum are not saved in `cache/`.

Copy the populated `cache/` directory and `cangjie-sdk.toml` into the offline runner, and keep the same distribution source configuration. Run `cjv cache list --json` to inspect the cached items.

If `cache/` contains damaged metadata, orphaned archive files, or incomplete temporary files, `cjv cache list --json` reports them in the optional `problem` field while still listing valid entries. Run `cjv cache clean` to delete these cjv-managed cache files, or run `cjv cache prune --max-age <duration>` to delete entries by age. If an online command succeeds but cannot write to the optional cache, cjv prints a warning and completes the command.

### Control network and installation behavior

- `--no-auto-install` stops proxy commands and `cjv exec` from installing missing items implicitly. Explicit `cjv install` commands still work.
- `--offline` stops cjv itself from making network requests. It does not block network access for child programs started by `run`, `exec`, or proxy binaries.

## Diagnostics, dry runs, and repair

Use these commands in scripts to check installation health, preview updates, or repair an SDK:

```bash
cjv doctor --json
cjv update --dry-run --json
cjv install lts --reinstall
```

- `cjv doctor` checks your settings, active selection, proxy binaries, installation metadata, and key SDK files without changing disk state. It does not compute SHA-256 hashes for every file in an installed SDK. If it finds an issue, it suggests a repair command and exits with a nonzero code.
- `cjv update --dry-run` calculates compatible target versions and update blockers without downloading archives, changing installations, or updating `cache/`.
- `cjv install <name> --reinstall` reinstalls the same SDK release and keeps your installed components and cross targets. By contrast, `--force` lets an install or update skip and remove unavailable optional components or targets, and does not reinstall an unchanged SDK.

## Update history and rollback

### How update snapshots work

When `cjv update` changes the release or digest of a tracked channel's host SDK or cross targets, cjv saves a snapshot of the previous toolchain group under `<CJV_HOME>/history/`. A toolchain group snapshot includes the host SDK, tracked cross targets, offline documentation, and `stdx`.

By default, cjv keeps the 2 most recent snapshots for each tracked channel. First-time installations, `--reinstall` repairs of the same release, and standalone `target add` or `component add` commands do not create snapshots. Pinned versions such as `lts-1.0.5` do not use channel history.

> **CAUTION:** Each snapshot stores an independent copy of the files, including the target contents of linked components, which uses additional disk space. Do not edit files inside `<CJV_HOME>/history/` directly.

### Roll back or prune history

Use these commands to view snapshots, roll back an update, or free disk space:

```bash
cjv toolchain history lts
cjv toolchain rollback lts
cjv toolchain history prune lts --keep 1
```

- `cjv toolchain rollback lts` restores the newest snapshot for `lts`. Pass an explicit `<snapshot>` ID to select an older snapshot. Before restoring, cjv checks the integrity of the snapshot on disk and saves the current installation as a snapshot so you can undo the rollback.
- `cjv toolchain history prune lts --keep 1` keeps the 1 newest valid snapshot for `lts` and deletes older ones. Pass `--keep 0` to delete all snapshots while keeping your active installation. The `--keep` flag applies to that `prune` run only; future updates still keep 2 snapshots automatically. Note that `cjv cache clean` does not delete update history.

### Damaged snapshot handling

`cjv toolchain history` lists valid snapshots in `snapshots` and lists damaged records with their failure reasons in the optional `invalid_snapshots` array.

When you run `history prune` with a nonzero `--keep`, cjv keeps that many valid snapshots and leaves damaged snapshots in place so you can check them. Running `history prune --keep 0` deletes both valid and damaged snapshots. If the newest snapshot has damage, `cjv toolchain rollback <channel>` without a snapshot ID refuses to guess an older version and asks you to pass a valid `<snapshot>` ID explicitly.

## JSON output contract

### Output streams and schema stability

Commands that support `--json` write a single JSON document to `stdout` and write logs or warnings to `stderr`. The `run`, `exec`, and `init` commands do not support `--json` because they pass child process streams through directly.

Obey these rules when you parse `--json` output in scripts:

- **Per-command fields**: Each command defines its own success payload. Your parser must ignore unknown object keys and unknown enum strings.
- **Versioned schemas**: Payloads that include `schema_version` keep existing field names, types, and meanings within that version number. cjv may add optional fields without changing `schema_version`, and increments `schema_version` only for incompatible changes.
- **Unversioned payloads**: Older commands without `schema_version` keep their existing JSON structure. Do not assume that different commands share the same top-level fields.
- **Collections and optional fields**: Mandatory lists are always JSON arrays (`[]` when empty). Commands may omit optional fields or set them to `null` according to each command's existing behavior.
- **Error matching**: Always branch on the process exit code and the machine-readable `error.code` string, never on the localized `error.message` text.

### Example: `cache list --json` schema

An empty cache returns this JSON document:

```json
{"schema_version":1,"path":"/home/user/.cjv/cache","entries":[]}
```

The JSON Schema below defines the stable fields of `cjv cache list --json`:

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

### Error payloads and diagnostic exit codes

When a command fails, cjv writes `{"error":{"code":"...","message":"...","details":{}}}` to `stdout` and exits with a nonzero status code. Future releases may add new error codes, and the keys inside `details` depend on `error.code`.

The `cjv doctor --json` command is an exception: when it finds environment problems, it returns its normal report object with `checks` and `"ok": false`, and exits with status code `1` instead of wrapping the output in an `error` object.

When automatic installation fails during a command, the top-level `error.code` is `TOOLCHAIN_NOT_INSTALLED` or `COMPONENT_NOT_INSTALLED`, and `details.cause` contains the underlying `code`, `message`, and `details`:

| `details.cause.code` | Meaning |
| --- | --- |
| `OFFLINE_CACHE_MISS` | Target manifest or archive is not in `<CJV_HOME>/cache/` during `--offline` mode |
| `CHECKSUM_MISMATCH` | Downloaded or cached archive failed SHA-256 verification |
| `HTTP_ERROR` | Server returned an unsuccessful HTTP status code |
| `NETWORK_ERROR` | Connection or transport failure occurred before a complete HTTP response |
| `CANCELED` | A signal or caller context canceled the operation |
| `DEADLINE_EXCEEDED` | Download or request exceeded the configured timeout |

If you disable automatic installation, cjv omits `details.cause` because it did not try to install the missing item. Read [Command reference](command-reference.md) for all CLI commands and flags.
