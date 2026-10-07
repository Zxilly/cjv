# Architecture

The CLI parses commands, `lifecycle` coordinates installation and updates, and lower-level modules own distribution, file ownership, and transactions. Proxy execution resolves SDKs from the same installation state without using the CLI command tree.

## Entry point and modules

`cmd/cjv/main.go` dispatches by executable name. SDK tool names enter `proxy.Run`; `cjv-init` / `cjv-setup` prefixes enter the installer; other invocations enter `cli.Execute`.

| Module | Responsibility |
| --- | --- |
| `cli` | Cobra command tree, argument validation, results, and errors |
| `lifecycle` | SDK, target, component, custom installation, and removal orchestration |
| `toolchain` | Installation identities, versions, host-target associations, and local recovery |
| `resolve` | Active toolchain preparation and automatic installation of missing requirements |
| `component` | Component roots, file manifests, links, and batch edits |
| `dist` | Lazy manifest loading, downloads, verification, extraction, and preparation lifetime |
| `target` | Platform tuples, SDK/component mappings, and release platform catalog |
| `env`, `sdktools` | Environment merging, shell output, and SDK tool layout |
| `proxy`, `process` | Tool forwarding, child processes, exit codes, and signals |
| `config` | Setting origins, project selection, and data layout |
| `reachable`, `selfupdate` | Managed binary, proxies, PATH, and cjv updates |
| `fstx`, `fsops` | Durable transactions, atomic file operations, and platform retries |

`progress` defines events and output adapters. `i18n`, `cjverr`, and `logging` provide messages, structured errors, and logs. `testutil` supplies distribution fixtures, progress recording, and platform test helpers.

## Command state and output

Each `cli.Execute` creates a separate application, command tree, flags, and `output.Renderer`. Commands pass requests to business modules, which report events through `progress.Sink` instead of printing final results.

Text mode uses `progress.Text` and result `Text()` methods. JSON mode discards progress and emits structured results or errors. The CLI renders each error once; `main` sets the exit code. Proxy stdout belongs to the SDK tool, so automatic installation reports progress on stderr.

Keep invocation state isolated when changing the CLI. Do not move flags, writers, or JSON mode into global variables.

## Installation and publication

`lifecycle.OpenDistribution` reads settings, platforms, and manifests without recovering or scanning local installations. Remote queries therefore do not require successful local recovery. Mutations use an installation distribution session, recovering interrupted transactions and migrating legacy layouts before taking snapshots.

A managed SDK installation proceeds as follows:

1. `dist.Preparation` takes the installation lock and creates private staging space.
2. Under the home lock, read installation identity, component selection, and host dependency snapshots.
3. Release the home lock while resolving releases, downloading, and preparing the host, tracked targets, and components. Keep the installation lock.
4. Reacquire the home lock and check snapshots and dependencies for changes.
5. `publication.go` publishes SDK, stdx, and documentation roots together through `fstx.NewToolchainGroupTransaction`.
6. Finish command entry point setup, mark preparation complete, and clean owned downloads and staging files.

`--force` allows missing optional entries; it does not reinstall unchanged SDKs. `--no-update` preserves the installed release. Nightly resolution considers the required component and target set, with downgrades requiring explicit permission.

Managed component batches also prepare and revalidate before publishing contents and manifests in one transaction. The SDK remains in place. Pinned versions and tracked channels are independent. Hard-link deduplication applies only to regular SDK files with matching provenance and content; metadata and components are separate.

URL and local archive installations use `resolved_install.go`. Bundled stdx is processed after the SDK commits. A stdx failure preserves that SDK and reports partial success, unlike the all-member publication used for managed channels.

## Recovery and file ownership

`fstx` journals commit state, owned paths, and backups. Recovery after interruption uses the same commit or rollback decision. Blocked recovery retains journals and backups and returns `RecoveryError`. Installation, removal, and active toolchain preparation require recovery to succeed. Backups needed for recovery must not be treated as disposable leftovers.

Before snapshots or mutations, `component` checks owned roots, the complete file manifest, and parent directory links. Shared files are retained. Tracked leaf links may be removed without deleting their sources, and archive merges cannot traverse untracked links. External SDK directory links cannot be component edit roots; rollback also validates destination roots.

Managed component batches have durable publication journals. Direct component linking and archive edits use backups and rollback for ordinary errors, retaining backups if restoration fails. These operations do not all have the same cross-process recovery guarantees.

Derive data paths from `config/layout.go`. Use `fsops` for tree copies, merges, and platform retries. Archive and tree operations reject paths or links that escape owned directories.

## Active toolchains and cross targets

`toolchain.SelectActive` handles explicit selection, environment variables, per-directory overrides and project files, and defaults. `PrepareActive` recovers and locates an installation. It calls an installation callback only for a missing official SDK; invalid selections, corrupt records, and filesystem errors are returned directly.

`resolve.Active` supplies project requirements according to `auto_install`. `run --install` enables installation explicitly; status queries do not install. `toolchain.ReadHostTargets` combines host identity, actual release, and platform. Target commands, `component --target`, `resolve.ActiveTarget`, and proxy target preparation all use it to avoid selecting another host's or release's cross SDK.

`env.Runtime` merges SDK paths and component variables into an explicitly supplied base environment, then creates a child environment or shell script. `sdktools` owns tool paths. Unix proxies replace the current process; Windows proxies, `run`, and `exec` use `process.Run` for child execution.

## Distribution and downloads

`dist.Source` lazily reads and caches `versions.json` for LTS/STS and a separate `nightly.json`. Components are resolved from the same channel data. cjv consumes static manifests; upstream discovery and generation belong to the separate `cangjie-version-manifest` repository.

`Preparation` owns the installation lock, staging space, and downloaded archives through publication and cleanup. Failure retains reusable downloads. Success cleans only owned files, excluding user-provided archives and recovery journals.

Archives and nightly SHA-256 sidecars share cancellable retries. HTTP 408, 429, and server errors may retry; permanent HTTP errors and malformed content do not. Sidecar 404 means unpublished. Downloads with SHA-256 can resume across commands and verify the complete file. Downloads without a hash resume only within one operation.

## Settings and self-management

`config.SettingsFile` retains field presence. `Update` saves explicitly selected fields; `Save` can restore both values and presence from a snapshot. Environment overrides remain temporary. Validation precedes publication, and the cache is updated after a successful write.

`reachable.Ensure` manages the cjv binary, proxy entry points, and environment scripts, configuring PATH according to caller policy. `selfupdate.Check` queries metadata only; `Update` verifies and replaces the binary, then returns a status for the CLI to render. Default and mirror builds implement GitHub and GitCode discovery respectively.
