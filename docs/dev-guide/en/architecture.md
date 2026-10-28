# Architecture

`cli` parses commands and renders output, `lifecycle` coordinates toolchain and component installations, and lower-level packages own distribution manifests, file ownership, and durable filesystem transactions. Proxy execution (`cjc`, `cjpm`, etc.) resolves the active SDK from the same local installation state without constructing the Cobra CLI command tree.

## Entry point and modules

`cmd/cjv/main.go` dispatches execution based on the invoked executable name (`os.Args[0]`):

- **SDK tool names** (`cjc`, `cjpm`, and others) enter `proxy.Run` directly.
- **`cjv-init` or `cjv-setup` prefixes** launch the interactive installation wizard.
- **All other invocations** enter `cli.Execute`.

| Module | Responsibility |
| --- | --- |
| `cli` | Cobra command tree, argument validation, result rendering, and error formatting |
| `lifecycle` | Orchestration for SDK, cross-target, component, custom installation, rollback, and removal |
| `toolchain` | Installation identities, version selectors, host-target associations, and local recovery |
| `resolve` | Active toolchain preparation and automatic installation of missing project requirements |
| `component` | Component directory roots, file manifests, local `stdx` links, and batch edits |
| `dist` | Lazy manifest loading, resumable downloads, SHA-256 verification, extraction, and staging lifetime |
| `target` | Platform tuples, SDK/component platform mappings, and the release platform catalog |
| `env`, `sdktools` | Environment variable merging, shell script output (`envsetup`), and SDK tool layout |
| `proxy`, `process` | Tool proxy forwarding, child process execution, exit codes, and signal propagation |
| `config` | Settings file origins and inheritance, `cangjie-sdk.toml` lookup, and `<CJV_HOME>` data layout |
| `reachable`, `selfupdate` | Managed `cjv` binary, proxy links in `<CJV_HOME>/bin`, `PATH` setup, and self-updates |
| `fstx`, `fsops` | Durable filesystem transactions, atomic file operations, tree merges, and platform retries |
| `ohos` | Preparing and self-signing native ELF payloads for OpenHarmony hosts |

Supporting packages provide shared infrastructure across the codebase:

- `progress` defines progress event types and output adapters.
- `i18n`, `cjverr`, and `logging` provide localized messages, structured error codes, and debug/warning logs.
- `retry` implements the exponential backoff loop shared by filesystem and HTTP download retry policies.
- `testutil` supplies mock distribution servers, progress recorders, and platform test helpers; `config/configtest` isolates settings files and `cli/clitest` runs command groups in unit tests.

## Command state and output

Each call to `cli.Execute` creates an isolated `clictx.Context` (holding the output renderer, global `+toolchain` selector, and root command) along with a fresh command tree. Command groups live in their own subpackages (`showcmd`, `componentcmd`, `runcmd`, `initcmd`, `toolchaincmd`, `settingscmd`, `selfcmd`); each group stores its flags in its own struct and registers itself on the root command. Subcommands pass request structs to business packages, which report progress events through `progress.Sink` and return structured results rather than printing directly to `stdout`.

- **Text mode** renders progress via `progress.Text` and prints each result's `Text()` method.
- **JSON mode** discards progress events and emits a single structured JSON document or error object to `stdout`.
- **Proxy mode** reserves `stdout` exclusively for the invoked SDK tool, writing automatic installation progress to `stderr`.

The CLI layer renders each error once, and `main` maps the result to a process exit code. When adding or modifying CLI commands, keep invocation state strictly isolated—never move flags, `io.Writer` handles, or JSON mode into package-level global variables.

## Installation and publication

`lifecycle.OpenDistribution` reads settings, platform definitions, and remote manifests without recovering or scanning local installations. As a result, remote-only queries (`toolchain list-remote`, `check`) do not depend on local transaction recovery succeeding. Mutating operations open an installation distribution session, which recovers any interrupted transactions and migrates legacy directory layouts before taking local snapshots.

A managed SDK installation proceeds through six distinct phases:

1. `dist.Preparation` acquires the installation lock (`<CJV_HOME>/.install.lock`) and creates a private staging directory for the operation.
2. Under `HomeLock`, `cjv` reads the target installation identity, current component selections, and host-target dependency snapshots.
3. `cjv` releases `HomeLock` while resolving manifest versions, downloading archives, and extracting the host SDK, tracked cross targets, and components into staging (retaining the installation lock throughout).
4. `cjv` reacquires `HomeLock` and verifies that local snapshots and dependencies did not change while the lock was released.
5. `publication.go` publishes the directories and metadata changed by each member in one durable transaction through `fstx.NewToolchainGroupTransaction`.
6. `cjv` finishes command entry point setup in `<CJV_HOME>/bin`, marks preparation complete, and cleans up owned downloads and staging files.

Installation flags adjust resolution without bypassing transactional publication: `--force` permits skipping and removing unavailable optional components or targets without reinstalling unchanged SDKs; `--no-update` preserves the installed host release; and `nightly` resolution searches published history for a release compatible with the required component and target set (requiring `--allow-downgrade` before selecting an older build).

When the SDK release and provenance are unchanged and only components need updating, the group revalidates under the lock and copies only affected component directories and component metadata, leaving the SDK payload in place. Component manifests and installation records still commit in the same transaction as changes to other host or cross-target members of the group. SDK replacements and explicit `--reinstall` prepare and publish the complete SDK. All publication paths are staged in `.staging` trees beside their SDK, documentation, or `stdx` roots so interrupted operations can use the same recovery and cleanup paths.

Managed component installation batches (`component add`) likewise stage outside `HomeLock`, revalidate under `HomeLock`, and publish contents and file manifests in a single transaction while leaving the SDK root in place. `component remove` removes each requested component under `HomeLock`, continues after individual failures, and reports successful removals with the combined errors; the batch is not atomic. Tracked channels and pinned versions remain independent, and hard-link deduplication (`link_mode = "hardlink"`) applies only to regular SDK files with matching provenance and SHA-256 content (metadata and components are never shared).

Custom URL and local archive installations (`resolved_install.go`) commit the SDK first and then process any bundled `stdx` archive. A failure while extracting bundled `stdx` preserves the committed SDK and reports partial success, unlike the all-or-nothing group publication used for managed channels.

## Recovery and file ownership

`fstx` journals commit state, owned target paths, and backup locations on disk before moving files. If a process crashes or is interrupted, the next recovery pass replays the journal and applies the same commit or rollback decision. If a filesystem error blocks recovery, `fstx` retains the journal and backups and returns `RecoveryError`. Installation, removal, and active toolchain preparation all require recovery to succeed first—backups needed for recovery must never be treated as disposable leftovers.

### Concurrent read/write locking (`HomeLock` and `WithStableReads`)

`HomeLock` always acquires the global writer lock _before_ acquiring the active-lookup gate. During long-running phases that leave installed SDK roots unchanged—such as copying, verifying, or pruning update history snapshots—the writer calls `WithStableReads` to temporarily release the active-lookup gate while retaining the global writer lock.

This allows concurrent readers (`cjc`, `cjv show active`, `cjv which`) to resolve an existing SDK without blocking behind history snapshot I/O:

- A reader holds the active-lookup gate across directory lookup and probes the writer lock without waiting.
- If a live writer holds the writer lock during a stable-read phase, the reader resolves the installed SDK directly; if no live writer owns the writer lock (for instance, after a writer crash), ordinary transaction recovery runs instead.
- Before mutating installed roots, the writer reacquires the active-lookup gate and atomically clears the read-phase marker.
- Readers must **never** wait for the writer lock while holding the active-lookup gate, avoiding lock-order inversion. A stale marker left after a crash cannot bypass recovery, and this fast path remains disabled until the first legacy-layout migration completes.

### Component manifests and path safety

Before creating snapshots or modifying files, `component` validates owned roots, the complete file manifest, and parent directory symlinks. Files shared with the base SDK are retained during removal; tracked leaf links can be removed without deleting their target sources; and archive merges refuse to traverse untracked symlinks. External SDK directory links cannot serve as component edit roots, and rollback validates destination roots as well.

Managed component installation batches use durable cross-process `fstx` journals. `component remove` directly removes files for each requested component. `component link` and custom archive component edits use backups and in-process rollback for ordinary errors, retaining backups if restoration fails. Removal and linking do not provide the same durable transaction guarantees as managed component installation.

Always derive `<CJV_HOME>` paths from `config/layout.go`, and use `fsops` for directory tree copies, merges, and Windows retry loops. All archive and tree operations reject paths or links that escape owned directories.

## Active toolchains and cross targets

`toolchain.SelectActive` evaluates explicit CLI selections (`+name`, `--toolchain`), `CJV_TOOLCHAIN`, per-directory overrides and `cangjie-sdk.toml` files, and the global default. `PrepareActive` runs pending recovery and locates the installation on disk, invoking its auto-install callback only when a valid official SDK is missing (invalid selections, corrupt records, and I/O errors are returned directly).

`resolve.Active` supplies missing project requirements according to `auto_install` (`run --install` enables installation explicitly, while read-only status queries never install). `toolchain.ReadHostTargets` pairs host identity, actual release, and platform; `target` commands, `component --target`, `resolve.ActiveTarget`, and proxy target preparation all use it so they never associate a cross SDK belonging to another host or release.

`env.Runtime` merges SDK paths and component variables into an explicitly supplied base environment to build a child process environment or `envsetup` shell script (`env` itself does not select toolchains; `resolve.ActiveRuntime` pairs active selection with `env.Runtime`, and `sdktools` owns tool paths). On Unix, proxies replace the current process via `execve`; on Windows—and for `cjv run` and `cjv exec`—`process.Run` manages child execution, exit codes, and signals.

## Distribution and downloads

`dist.Source` lazily loads and caches `versions.json` for LTS/STS and `nightly.json` for nightly, resolving components from the same channel data. `cjv` consumes static manifests; upstream scraping and manifest generation live in the separate `cangjie-version-manifest` repository.

`Preparation` owns the installation lock, staging directory, and downloaded archives through publication and cleanup. Failures retain reusable partial downloads; successes clean only files owned by that operation, leaving user-supplied local archives and recovery journals untouched.

Archive downloads and nightly `.sha256` sidecar requests share a cancellable retry loop. HTTP `408`, `429`, and `5xx` responses may retry, whereas permanent `4xx` errors and malformed content fail immediately (a `404` on a nightly sidecar indicates that the checksum has not been published yet). Downloads with an expected SHA-256 can resume across commands and verify the complete file; downloads without a hash resume only within a single operation.

## Settings and self-management

`config.SettingsFile` tracks which fields were explicitly set in each file: `Update` writes only explicitly modified fields so unset fields continue to inherit from system fallback settings, while `Save` can restore both values and field presence from a snapshot. Environment overrides remain temporary, validation precedes publication, and the in-memory cache is updated after a successful write.

`reachable.Ensure` manages the installed `cjv` binary, proxy entry points, and environment scripts, configuring `PATH` according to caller policy. `selfupdate.Check` queries release metadata only, whereas `selfupdate.Update` verifies and replaces the executable and returns a status value for the CLI to render. Default and `mirror` builds implement GitHub and GitCode release discovery respectively.
