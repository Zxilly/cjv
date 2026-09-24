# Code Architecture

This chapter covers how cjv's code is organized: which directories the repository has, which packages the Go CLI's `internal/` is split into, what each package is responsible for, and how a command flows through these packages from process startup to completion. The module path is `github.com/Zxilly/cjv`, and the Go version follows the `go` directive in `go.mod` (currently 1.26.0).

## Repository layout

The main makeup of the repository root was already covered in the [Introduction](introduction.md), so here is just one addition: what is tracked under version control is `cmd/`, `internal/`, `web/`, `docs/`, `tests/`, `scripts/`, plus a few configuration files in the root (`go.mod`, `.goreleaser.yml`, `.golangci.yml`, and so on). The rest of this chapter is about the Go CLI.

```text
cmd/cjv/        binary entry point (the main package)
internal/       all implementation, split into packages by subsystem
scripts/        build-time helper scripts (code generation, CI)
tests/          cross-package integration and smoke tests
web/            landing page (see the "Landing page" chapter)
docs/           the two mdBooks (see the "Documentation site" chapter)
```

Under `scripts/` are two helper scripts that take no part in compiling the CLI: `gen-platform-surfaces.go` is the `go:generate` target for `internal/target`, generating code from the platform manifest; `extract-init-binaries.sh` is used by the release process. Under `tests/`, `integration/` holds the end-to-end integration tests, `smoke/` verifies real downloads, and `install-scripts/` tests the install scripts. Unit tests, following Go convention, live in the same directory as the code under test, with `_test.go` right next to the source file, so the `*_test.go` files you see in each `internal/` package are that package's own unit tests.

## Entry point: `cmd/cjv/main.go`

`cmd/cjv/main.go` is the only `main` package, and it is very thin. It does a few process-level things and then hands control over to `internal/`.

The two variables `version` and `updateURL` are injected by the linker at build time (see [Building from source](building.md) for details); when not injected, `version` is `"dev"`.

`main` calls `run`, which first initializes logging (`logging.Init`), records the version number, then takes the invoked program name from `os.Args[0]` (`proxy.ExtractToolName`) and branches three ways based on it:

- If the program name is a known SDK tool (`cjc`, `cjpm`, and so on, decided by `proxy.IsProxyTool`), it takes the proxy path `proxy.Run`, passing the arguments through to the real tool.
- If the program name starts with `cjv-init` / `cjv-setup`, it is treated as an installer, rewriting `os.Args` to `cjv init` before continuing.
- Otherwise it is an ordinary `cjv ...` invocation, handed to `cli.Execute(version, updateURL)`.

For ordinary CLI invocations, `cli.Execute` renders each error once: text goes to stderr, while JSON envelopes go to stdout. `main` only unwraps `*cjverr.ExitCodeError` into a process exit code or returns 1 for other errors. The proxy path bypasses the CLI, so `main` still handles its errors. The Windows console UTF-8 switch and the pause prompt for double-click runs also remain in `main` as process-level concerns.

## Responsibilities of the `internal/` packages

Each directory under `internal/` is a package, divided by subsystem. They are listed below from upper to lower layers, following roughly the direction of dependencies within a single command.

### `cli`: command definitions

`internal/cli` is the cobra command tree. Each `Execute` creates a fresh `application` that owns the invocation's commands, flag values, version and update URL, and `output.Renderer`. `root.go` registers the root-level `--json` flag and attaches subcommands without reusing a previous invocation's commands or output mode. Each subcommand has its own file, `install.go`, `uninstall.go`, `toolchain.go`, `run.go`, `exec.go`, `which.go`, `show.go`, `check.go`, `update.go`, `component.go` and so on, with file names that largely match the command names.

`cli` does not implement business logic itself; what it does is parse arguments, call the lower-level packages, and hand the result to the rendering layer. A few subpackages take on the cross-cutting concerns:

- `cli/output.Renderer` holds the JSON mode for one invocation. Commands define structs implementing `Result` (a single `Text()` method), and the renderer emits text or JSON. It also builds JSON error envelopes, using `cjverr.Coded` to supply machine-readable error codes.
- `cli/settings` constructs the `set`, `default`, and `override` commands afresh on every registration. Each command closure owns its directory paths and cleanup flags.
- `cli/selfmgmt` constructs `cjv self` with the invocation's renderer and keeps uninstall confirmation local to the command. Explicit and automatic self-updates share `UpdateManaged`, which prepares the managed binary, updates it, and refreshes proxy links and env scripts. Callers decide how to render the result and whether failures are fatal.

### `lifecycle`: installation and content lifecycle

`internal/lifecycle` orchestrates download, extraction, verification, components, PATH configuration, and proxy links as one installation flow. It receives adapters such as `Report`, `ComponentInstall`, `CreateProxyLinks`, and `ValidateInstallation` through `Options`, with dependencies directed from `cli` toward `lifecycle`. `Report` carries progress and leaves operations silent when unset; the CLI owns output formatting. The same flow serves `cli install` and proxy auto-install. Reinstalling an existing SDK without force succeeds in both text and JSON modes, and the CLI also renders the JSON result of `component add`.

Files inside the package are split by responsibility: `install.go` owns orchestration, `source.go` turns channel requests into `ResolvedToolchain` values, `component_install.go` orchestrates component batches with rollback delegated to `component.ApplyChanges`, and `resolved_install.go` owns materialization, validation, and transactional replacement. Distribution-source selection stays local to `source.go`. Toolchain replacement regressions call these production installation entry points to verify restoration after finalization fails and propagation of rollback errors.

`UpgradeToolchain` and `RemoveToolchain` coordinate the SDK, external stdx/docs content, the default toolchain, and directory override references. Upgrades fetch downloaded components for the replacement version and retain the original sources of linked components. An existing replacement keeps its own component choices, with only missing components added. A failed operation retracts a newly created, unreferenced replacement; blocked recovery retains content still needed by references and reports the error. A forced reinstall under the same name preserves existing component manifests and external content. Bundled stdx in a URL install is still handled after SDK installation, so the SDK can succeed while stdx fails.

### `resolve`: active toolchain resolution

`internal/resolve` answers the question of which toolchain to use right now. `Active` combines the command-line `+toolchain` override, the `CJV_TOOLCHAIN` environment variable, the directory-level and global overrides, and the default setting to determine the name and directory of the active toolchain, and returns them as an `ActiveToolchain` together with its target platform and components. If the toolchain is not installed during resolution, it can trigger an automatic install through the `AutoInstallFunc` test seam; in production this seam is wired to `lifecycle` by default, so `resolve` does not need a reverse dependency on `cli`.

### `toolchain` and `component`: models of what is installed

`internal/toolchain` manages the installed SDKs: it lists the installed toolchains (`ListInstalled`), resolves the active toolchain directory, and parses toolchain names and compares versions. `RecoverHome` is the single recovery entry point for CJV_HOME: it first lets `fstx` resume unfinished transactions under `toolchains/`, then removes abandoned staging trees and restores legacy backups whose original is missing. A blocked recovery is returned unchanged as an `fstx.RecoveryError` and no residue is touched; backups needed for recovery are not deleted as ordinary leftovers. Install, upgrade and removal call it before changing files; proxy resolution and `update` call it at startup and log a blocked recovery as a warning.

`internal/component` manages the add-on components of a toolchain: `stdx`, `docs`, `stdx-docs`. Each component is a separately downloaded archive, and its extracted files are recorded through a per-component manifest, so it can be uninstalled independently. `component` also defines where each component installs to (`InstallLocation`: some land inside the toolchain directory tree, while others are placed as pure data under `<CJV_HOME>/docs/<tc>/`) and which environment variables a component needs to inject.

`ApplyChanges` owns backups, failure recovery, and cleanup for a component change or a batch of changes. Archive installation and local linking share the replacement flow. Backups include component files and manifests; a failed restore retains the backup and reports its location for recovery, so callers do not manage snapshot lifetimes themselves.

### `dist`: download and unpacking

`internal/dist` owns distribution sources and network artifacts. `source.go` is the unified entry point: LTS/STS lazily cache `versions.json`, while nightly lazily caches sibling `nightly.json`. Under `dist_server`, both live at the distribution root. Relative URLs resolve against the manifest directory and absolute URLs are honored as written. `manifest.go` parses and validates channel data; `download.go` handles progress, retries, and SHA-256; `install.go` unpacks archives; `nightly.go` reads nightly SHA-256 sidecars; and `platform.go` centralizes host platform keys.

### `target`: platform identity

`internal/target` is the single source of truth for platforms and target tuples. It parses the target tuple (the host part plus an optional cross-compilation environment suffix) and produces structured views such as the manifest index key and stdx platform token, sparing every caller from slicing strings on its own. `catalog.go` lists every `(GOOS, GOARCH)` combination for which cjv ships a host binary, and is the source for both the release artifacts and the download entries on the landing page; it carries a `go:generate` directive and is generated by running `scripts/gen-platform-surfaces.go`.

### `env`: runtime environment

`internal/env` assembles the environment needed to run the Cangjie tools. `Runtime` owns the active toolchain and private SDK configuration, which callers no longer inspect or mutate. `ProxyEnv` and `ToolchainEnv` take an explicit base environment and share the rules for merging PATH, library paths, `SDKROOT`, and component variables, without reading a different environment from the process during the merge. `Contributions` returns SDK additions without inherited values; `ShellScript` derives shell changes from the same merged result. Platform variable names, path ordering, and casing rules stay inside this module, while shell detection and formatting remain in `shelldetect.go`, `shell_*.go`, and `shellformat.go`.

### `proxy`: transparent proxy

`internal/proxy` implements the transparent proxy: when the binary is invoked under a tool name such as `cjc` or `cjpm`, `Run` resolves the active toolchain (through `env.ResolveRuntime`), locates the real tool binary inside the toolchain directory (`toolPathMap` in `tools.go` maps tool names to relative paths), assembles the proxy environment, and passes arguments through. Unix replaces the current process with `syscall.Exec`; Windows starts and waits for a child through `process.Run`. A recursion counter (`CJV_RECURSION_COUNT`) prevents the proxy from calling itself indefinitely. `link.go` creates the proxy links at install time (`CreateAllProxyLinks`).

### `process`: child process execution

`internal/process.Run` accepts a configured `exec.Cmd` and owns starting, waiting, and termination-signal handling. Nonzero child exits become `cjverr.ExitCodeError`. `cjv run`, `cjv exec`, and the Windows proxy share it; callers still configure command lookup, arguments, environment, and standard streams. Unix SIGTERM forwarding and timeout escalation, along with keeping the parent alive while the child handles Ctrl+C on each platform, stay in this package.

### `config`: configuration and paths

`internal/config` is the configuration layer. It defines all `CJV_*` variables, including `CJV_DIST_SERVER`, resolves `CJV_HOME`, reads user and system fallback settings, reads the toolchain file, and manages directory overrides. `layout.go` is the one place that spells the CJV_HOME layout: the `toolchains/`, `stdx/`, `docs/`, `downloads/` and `bin/` subdirectories and each toolchain's directory within them (`ToolchainDirFor`, `StdxDirFor`, `DocsDirFor`), plus the naming rules for install residue: `StagingDir(dest)` names the staging tree, and `IsScratchName` tells whether a directory name is a staging tree, a legacy backup or an `fstx` transaction directory. Other packages derive paths from here instead of building suffixes themselves. `manifest_url` supplies the release manifest and locates its nightly sibling, `dist_server` selects an enterprise root containing both files, and the `mirror` build tag selects the default address.

`SettingsFile.Load` returns a copy of the effective settings while retaining the provenance of user-defined fields. `Update(SettingsUpdate)` persists only explicit choices, leaving unspecified fields inherited from system or built-in defaults. An explicit choice can pin a value equal to an inherited one; `false` and empty strings are also explicit values. `Save` can restore a loaded snapshot's values and field presence. Validation and cache preparation happen before publication, so a successful write cannot be reported as failed because a subsequent read failed. Environment overrides remain transient.

### `selfupdate`: self-update

`internal/selfupdate` discovers, verifies, and installs cjv updates. Whether it uses GitHub or GitCode is selected at compile time by the `mirror` build tag (`update_default.go` / `update_mirror.go`). `Update` returns a status (`skipped`, `dev`, `up-to-date`, or `updated`) and version information for `cli/selfmgmt` to render, without printing its own results to stdout. It also establishes the current binary as the managed executable and replaces running binaries during updates, with platform-specific code in `replace_windows.go` / `replace_other.go`.

### Supporting packages

The remaining few are supporting packages shared across the layers:

- `i18n` internationalization. Messages live in `locales/en.toml` and `locales/zh-CN.toml` and are embedded into the binary, and `i18n.T` looks up a string by message ID. All user-facing text goes through it, error messages included.
- `cjverr` error types. It defines structured errors carrying a stable machine code (`ErrorCode`); the `Error()` method produces the human-readable message through `i18n`, and the `Coded` interface lets `output` emit the error code in JSON mode. `ExitCodeError` carries the process exit code.
- `fstx` filesystem transactions. On-disk journals record managed paths, backups, and transaction state, with size and path limits enforced when reading them; transaction directory names, staging trees and the `toolchains/`, `stdx/`, `docs/` entries a toolchain transaction owns all come from the `config` layout. `toolchain.RecoverHome` calls `Recover` first during startup cleanup and installation or removal retries. Committed and prepared-for-publication states retain ready content before cleaning backups. If recovery is blocked, the journal and backups remain and their location is reported for a later retry, without relying on in-process undo closures.
- `utils` miscellaneous utilities: atomic writes, file operations, Windows junctions, retries, console UTF-8, opening a browser, version-number parsing, and so on, most of them split into per-platform files.
- `logging` configures the global `slog` logger via the `CJV_LOG` environment variable (defaulting to `warn`).
- `testutil` test helpers: a mock download server and a Windows registry guard. It carries source files outside of `_test.go` so that the tests of other packages can import them.

## The flow of a single command

Tying the above together, here is roughly how `cjv install <toolchain>` runs.

The process starts in `run` in `cmd/cjv/main.go`: `logging.Init` sets up logging, the program name is `cjv` rather than some tool name, so it takes the `cli.Execute` path. cobra routes the `install` subcommand to `runInstall` in `internal/cli/install.go`. `runInstall` collects the `--target`, `--component`, `--force` and other flags, assembles a `lifecycle.Options` (wiring in the implementations of `output`, `component`, `proxy`, `selfupdate`), and calls into `internal/lifecycle`.

`lifecycle` first asks `dist.Source` to resolve the requested channel, version, platform, and component artifacts from the manifest. The shared download and extraction path then materializes a staging directory, after which `component`, `proxy`, and `fstx` complete component installation, proxy links, and transactional replacement. Every channel shares this installation path. The CLI selects progress reporting for its output mode and renders the command result through the invocation's renderer. `cli.Execute` renders errors, and `main` translates them into exit codes.

The proxy path is the other main line. When you run `cjc build`, what is actually invoked is the cjv link named `cjc`, and `main` recognizes the tool name and takes the `proxy.Run` path: `proxy`, through `env.ResolveRuntime`, has `resolve` determine the active toolchain, finds the real `cjc`, and assembles its environment before replacing the current process or running a child, depending on the platform. This path bypasses cobra while preserving the tool's standard streams and exit semantics.

To dig into a particular area, start with `internal/cli/root.go` for commands, `internal/lifecycle/install.go` for orchestration, `internal/dist/source.go` for distribution selection, `internal/lifecycle/resolved_install.go` for materialization, and `internal/proxy/proxy.go` for proxying. See [Testing](testing.md) for test organization.
