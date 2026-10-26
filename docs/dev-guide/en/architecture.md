# Architecture

`cjv` separates command parsing, installation orchestration, and low-level filesystem or network operations into distinct packages under `internal/`. Proxy execution (`cjc`, `cjpm`) resolves the active SDK from the same local state without constructing the Cobra CLI command tree. This chapter explains the package boundaries, command state isolation, transactional installation flow, locking model, and runtime resolution.

## Entry point and package responsibilities

`cmd/cjv/main.go` routes execution according to `os.Args[0]`:

- **SDK tool name** (`cjc`, `cjpm`, and others): calls `proxy.Run` directly.
- **`cjv-init` or `cjv-setup` prefix**: starts the interactive installation wizard.
- **`cjv` (or any other name)**: calls `cli.Execute`.

| Package | Responsibility |
| --- | --- |
| `cli` | Cobra command tree, flag parsing, text and JSON rendering, and error formatting |
| `lifecycle` | High-level orchestration for installing, updating, linking, rolling back, and removing SDKs, targets, and components |
| `toolchain` | Installation identities, version selectors, host-target associations, and local state recovery |
| `resolve` | Active toolchain preparation and automatic installation of missing project requirements |
| `component` | Component directory roots, file manifests, local `stdx` links, and batch updates |
| `dist` | Lazy manifest loading, resumable downloads, SHA-256 verification, archive extraction, and staging lifetimes |
| `target` | Platform tuple conversions, SDK and component platform mappings, and the release platform catalog |
| `env`, `sdktools` | Environment variable merging, shell script generation (`envsetup`), and SDK tool binary lookup |
| `proxy`, `process` | Proxy command forwarding, child process execution, exit code propagation, and signal handling |
| `config` | Settings file inheritance, `cangjie-sdk.toml` lookup, and `<CJV_HOME>` directory layout |
| `reachable`, `selfupdate` | Managed `cjv` binary placement, proxy links in `<CJV_HOME>/bin`, `PATH` setup, and self-updates |
| `fstx`, `fsops` | Durable filesystem transactions, atomic renames, directory tree merges, and Windows retry loops |
| `ohos` | Preparing and self-signing native ELF binaries for OpenHarmony hosts |

Supporting packages provide cross-cutting utilities:

- `progress` defines progress events and terminal/JSON sinks.
- `i18n`, `cjverr`, and `logging` provide localized text, typed error codes, and structured logs.
- `retry` implements the backoff loop shared by filesystem and HTTP retry policies.
- `testutil`, `config/configtest`, and `cli/clitest` provide mock distribution servers, progress recorders, isolated settings, and CLI test harnesses; read [Tests and checks](testing.md).

## Command state and output rendering

Each call to `cli.Execute` constructs a fresh `clictx.Context` (output renderer, global `+toolchain` selector, and root command) and a fresh command tree. Subcommand groups live in dedicated subpackages (`showcmd`, `componentcmd`, `runcmd`, `initcmd`, `toolchaincmd`, `settingscmd`, `selfcmd`). Each subpackage stores its flags in a local struct and registers its subcommands on the root command.

Subcommands pass request structs to business packages (`lifecycle`, `resolve`, `toolchain`, `config`). Business packages emit progress events through `progress.Sink` and return structured result values instead of writing directly to `stdout`:

- **Text mode**: renders live progress through `progress.Text` and prints the result's `Text()` output.
- **JSON mode**: discards progress events and writes a single JSON document or error object to `stdout`.
- **Proxy mode**: reserves `stdout` for the child SDK tool and writes automatic installation progress to `stderr`.

The CLI layer renders each returned error once, and `main` translates the error into a process exit code. When you add or change CLI commands, keep all state inside `clictx.Context` or command structs; never store flags, writers, or JSON mode in package-level global variables.

## Transactional installation and publication

### Read-only vs. mutating distribution sessions

`lifecycle.OpenDistribution` loads merged settings, platform definitions, and remote manifests without scanning or recovering local installations. Remote queries such as `list-remote` and `check` therefore succeed even if local state needs recovery.

Mutating commands open an installation distribution session that recovers any interrupted transactions and migrates legacy directory layouts before reading local installation snapshots.

### Six-step managed SDK installation

A managed channel or version installation executes in six steps:

1. `dist.Preparation` takes the installation lock (`downloads/.lock`) and creates a private staging directory.
2. Under `HomeLock`, cjv reads the target installation identity, existing component selections, and host-target dependency snapshots.
3. cjv releases `HomeLock` while it resolves manifest versions, downloads archives, and extracts the host SDK, tracked cross targets, and components into staging. It keeps the installation lock throughout this phase.
4. cjv takes `HomeLock` again and checks that local snapshots and dependencies did not change while `HomeLock` was free.
5. `publication.go` commits the SDK, `stdx`, and documentation roots together in a single durable transaction (`fstx.NewToolchainGroupTransaction`).
6. cjv updates the proxy entry points in `<CJV_HOME>/bin`, marks the preparation complete, and deletes owned staging files and temporary downloads.

Policy flags adjust the resolution step without weakening transaction safety:

- `--force` lets the operation skip and remove missing optional components or targets, and does not reinstall an unchanged SDK.
- `--no-update` keeps the installed host release and adds requested targets or components only.
- `nightly` resolution checks the needed component and target set against published history, and needs `--allow-downgrade` before selecting an older release.

### Component batches, deduplication, and archive links

Managed component commands (`component add` and `component remove`) also stage files outside `HomeLock`, revalidate under `HomeLock`, and publish the component contents and file manifest in one `fstx` transaction while leaving the SDK root untouched.

Hard-link deduplication (`link_mode = "hardlink"`) shares regular SDK files only when two installations have the same release, platform, and SHA-256 digest. Installation metadata and components are never shared across toolchains.

Custom archive and URL installations (`resolved_install.go`) commit the SDK first and then process any bundled `stdx` archive. If the bundled `stdx` step fails, cjv keeps the committed SDK and reports partial success, unlike managed channel updates which publish or roll back the entire group together.

## Crash recovery, locking, and file ownership

### Journaled transactions (`fstx`)

`fstx` writes the transaction state, owned target paths, and backup paths to a journal file on disk before moving files into place. If a process crashes or stops unexpectedly, the next mutating or active-toolchain command replays the journal and applies the same commit or rollback decision.

If a filesystem error blocks recovery, `fstx` keeps the journal and backup directories intact and returns `RecoveryError`. Never delete recovery backup directories as ordinary temporary files.

### Concurrent read/write locking (`HomeLock` and `WithStableReads`)

`HomeLock` takes the global writer lock first, and then takes the active-lookup gate. During long operations that do not mutate installed SDK directories (such as copying, checking, or pruning update history snapshots), the writer calls `WithStableReads` to release the active-lookup gate while holding the global writer lock.

This design lets concurrent reader processes (`cjc`, `cjv show active`, `cjv which`) resolve and run an installed SDK without waiting for history snapshot operations:

- A reader holds the active-lookup gate during directory resolution and checks the writer lock without blocking.
- If a live writer holds the writer lock during a stable-read phase, the reader uses the existing installed roots directly.
- If no live writer holds the writer lock (for example, after a writer crash), the reader runs normal transaction recovery before resolving the SDK.
- Before publishing changes to installed roots, the writer takes the active-lookup gate again and atomically clears the stable-read marker.
- Readers must never wait for the writer lock while holding the active-lookup gate, preventing lock-order deadlocks. Stale markers left by a crashed process never bypass recovery, and `WithStableReads` stays disabled until the initial legacy-layout migration finishes.

### Component ownership and path safety

Before taking a snapshot or changing files, `component` checks owned directory roots, the complete file manifest, and parent directory symlinks:

- Component removal keeps files that belong to the base SDK.
- Component removal can delete tracked leaf symlinks without deleting their target directories, and archive extraction refuses to traverse untracked symlinks.
- Component commands cannot edit external SDK directories linked with `cjv toolchain link`.
- Managed component batches use durable `fstx` journals across processes, whereas direct `component link` and custom archive edits use in-process backups and rollback (keeping backups on disk if restoration fails).

Always derive `<CJV_HOME>` subdirectories through `config/layout.go`, and use `fsops` for directory copies, merges, and Windows file-lock retries. All archive and tree operations reject paths or links that escape their owned root directory.

## Active toolchain resolution and runtime execution

1. `toolchain.SelectActive` evaluates explicit CLI selectors (`+name`, `--toolchain`), `CJV_TOOLCHAIN`, per-directory overrides and `cangjie-sdk.toml` files, and `default_toolchain`.
2. `toolchain.PrepareActive` runs pending recovery and finds the installation on disk. It calls the auto-install callback only when a valid official toolchain is missing; invalid names, corrupt metadata, and I/O errors return immediately.
3. `resolve.Active` checks and installs missing project components and cross targets when `auto_install` is `true` (or when you pass `run --install`). Read-only status commands (`show`, `which`) never trigger installation.
4. `toolchain.ReadHostTargets` binds each cross target to its host toolchain identity, exact release, and host platform. `target` commands, `component --target`, `resolve.ActiveTarget`, and proxy target preparation all call `ReadHostTargets` so they never mix cross SDKs from another host or release.
5. `env.Runtime` merges SDK binary paths, library paths, and `stdx` variables into an explicit base environment to produce a child process environment or an `envsetup` shell script. `resolve.ActiveRuntime` pairs active toolchain resolution with `env.Runtime`, and `sdktools` resolves tool binary paths. On Unix, proxy commands replace the current process via `execve`; on Windows (and for `cjv run` and `cjv exec`), `process.Run` spawns the child process and forwards signals and exit codes.

## Distribution manifests, downloads, and settings

- **Manifests (`dist.Source`)**: lazily loads and caches `versions.json` (for `lts` and `sts`) and `nightly.json` (for `nightly`). `cjv` consumes static manifests; upstream scraping and manifest generation live in the separate `cangjie-version-manifest` repository.
- **Downloads (`dist.Preparation`)**: manages the installation lock, staging directory, and archive downloads until publication and cleanup finish. Failed downloads keep partial files for resumption; successful operations delete only their own staging files and keep user-supplied local archives and recovery journals.
- **Retry rules**: archive downloads and nightly `.sha256` sidecar requests share a cancellable backoff loop. The client retries HTTP `408`, `429`, and `5xx` responses, whereas permanent `4xx` errors and invalid content fail immediately (a `404` on a nightly `.sha256` sidecar means the checksum is not published yet). Downloads with a known SHA-256 resume across separate CLI invocations; downloads without a hash resume only within a single command run.
- **Settings (`config.SettingsFile`)**: tracks which keys were explicitly set in each file so `Update` writes only the changed key and keeps inheritance for unset keys. `Save` can restore a complete snapshot of values and key presence.
- **Self-management (`reachable`, `selfupdate`)**: `reachable.Ensure` installs the managed `cjv` binary, proxy links, and shell environment scripts, and updates `PATH` when allowed. `selfupdate.Check` queries release metadata only, while `selfupdate.Update` checks and replaces the executable and returns a status struct for the CLI to display.
