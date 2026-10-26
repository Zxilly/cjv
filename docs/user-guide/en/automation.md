# Automation and JSON interfaces

## Pin a project version

`cjv pin lts` writes the resolved release name to `cangjie-sdk.toml` in the current directory, preserving existing components, targets, and other key values. Comments and formatting are rewritten. With no name, it uses the active selection. An installed tracking channel uses its locally installed release first. Commit this file so project members select the same SDK version.

```bash
cjv pin lts
cjv pin --check
```

`--check` verifies that the current directory's project file pins a concrete release without writing it. The project file fixes the version and the distribution manifest supplies artifact checksums; no separate lockfile is needed.

## Configuration and network access

Select a dedicated settings file and data directory for CI:

```bash
export CJV_HOME="$PWD/.cjv-data"
export CJV_CONFIG_FILE="$PWD/ci-settings.toml"
cjv install lts
cjv --offline --no-auto-install exec cjc --version
```

`CJV_HOME` must be absolute. `--config` and `CJV_CONFIG_FILE` replace only the user settings file. System fallback settings, project selection, and other environment variables still apply. Set `CJV_FALLBACK_SETTINGS` separately when you need to control the fallback file.

Offline installation requires metadata, SDK archives, and component archives from the configured source to have been downloaded beforehand. Successful installation retains verified archives; inspect them with `cjv cache list --json`. Supply the populated `cache/` and project toolchain file to the offline environment and keep the same distribution source configuration. Archives without checksums are not retained in the reusable cache.

Cache listings use the optional `problem` field to identify damaged records, orphaned archives, and incomplete temporary files while retaining other entries. `cjv cache clean` removes these cjv-owned files; `cache prune` still filters by age. A successful online query remains usable if saving its optional cache fails, with a warning explaining the failure.

`--no-auto-install` prevents implicit installation; explicit `install` remains available. `--offline` controls cjv's requests and does not restrict networking by child processes.

## Inspect and repair

```bash
cjv doctor --json
cjv update --dry-run --json
cjv install lts --reinstall
```

`doctor` reads configuration, the selected SDK, command entry points, and installation records, and checks that key files exist before suggesting repairs. It does not verify a digest for every SDK file. It exits nonzero when it finds a problem. `update --dry-run` resolves compatible versions and blockers without downloading archives, installing, or refreshing the cache.

`--reinstall` prepares the named SDK again while retaining selected components and targets. `--force` continues to permit removal of unavailable optional components or targets; it does not request reinstallation.

## Update history and rollback

When a tracking channel's host or an existing target changes release or digest, cjv retains the previous complete toolchain group: the host SDK, tracked targets, documentation, and stdx. Each group retains its two most recent snapshots by default. Initial installation, repair of the same release, and adding targets or components alone do not create snapshots.

```bash
cjv toolchain history lts
cjv toolchain rollback lts
cjv toolchain history prune lts --keep 1
```

`rollback` selects the newest snapshot when its ID is omitted. Use `cjv toolchain rollback lts <snapshot>` to select one explicitly. Rollback verifies local content before restoring the entire group and retains the current group so the rollback can itself be undone. Fixed-version installations remain independent of tracking-channel history.

Snapshots contain independent copies, including the content of linked components, and require additional disk space. Cache cleanup does not remove snapshots. `history prune --keep 0` clears history while preserving current installations. `--keep` applies only to that cleanup; updates still retain two snapshots automatically. History lives under `<CJV_HOME>/history/`; do not edit its files directly.

History listings retain readable entries in `snapshots` and report damaged entries and their causes in the optional `invalid_snapshots` array. A nonzero `--keep` retains that many readable entries and leaves damaged entries for inspection; `--keep 0` also clears damaged entries. When the newest entry is damaged, rollback without an ID requires an explicit valid snapshot instead of silently choosing an older release. Every selected snapshot must still pass full content verification.

## JSON compatibility

Commands supporting JSON write one JSON document to stdout and logs to stderr. `run`, `exec`, and `init` do not support `--json`; child process output and exit codes are passed through.

- Success fields are specific to each command. Clients should ignore unknown fields and enum values they do not use.
- Objects with `schema_version` retain existing field meanings and types within that version. Optional additions do not change the version; incompatible changes must increment it.
- Existing commands without a version field keep their output structure. Do not assume they share fields with other commands.
- Required collections are arrays, with `[]` for an empty collection; optional fields may be absent. Existing `null` and omission behavior is retained for compatibility and must be handled per command.
- Use error `code` and the exit code for automation, not the localized `message`.

An empty cache returns:

```json
{"schema_version":1,"path":"/home/user/.cjv/cache","entries":[]}
```

This JSON Schema describes the stable fields of `cache list --json` and permits future additions:

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

Execution failures return `{"error":{"code":"…","message":"…","details":{}}}` on stdout and a nonzero process exit code. New error codes may be added, so clients must handle unknown codes. The fields of `details` depend on the error code. Ordinary successful commands exit with zero. When `doctor` finds diagnostic issues, it still returns its report with `checks`, sets `ok: false`, and exits with code 1 without an additional error object.

When automatic installation fails, the top-level code remains `TOOLCHAIN_NOT_INSTALLED` or `COMPONENT_NOT_INSTALLED`. Its `details.cause` preserves the underlying failure's `code`, `message`, and `details`. For example, `OFFLINE_CACHE_MISS` indicates missing cached data, `CHECKSUM_MISMATCH` indicates failed verification, `HTTP_ERROR` and `NETWORK_ERROR` distinguish HTTP responses from transport failures, and `CANCELED` or `DEADLINE_EXCEEDED` indicates cancellation or a timeout. A missing installation with automatic installation disabled has no `cause`, because no installation was attempted.
