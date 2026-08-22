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

Error handling is centralized here too: a `*cjverr.ExitCodeError` returned by the implementation layer is unwrapped into a process exit code, while all other errors are printed to stderr uniformly and 1 is returned (in JSON mode the envelope has already been written to stdout by `cli.Execute`, keeping stderr clean). The UTF-8 switch for the Windows console and the pause prompt for double-click runs are also handled at this `main` level, because they are process-level concerns that should not leak into business logic.

## Responsibilities of the `internal/` packages

Each directory under `internal/` is a package, divided by subsystem. They are listed below from upper to lower layers, following roughly the direction of dependencies within a single command.

### `cli`: command definitions

`internal/cli` is the cobra command tree. `root.go` defines the root command `cjv` and the `Execute` entry point: it registers the global `--json` flag, hands the version number to cobra, attaches the subcommands, and then calls `rootCmd.Execute()`. Each subcommand has its own file, `install.go`, `uninstall.go`, `toolchain.go`, `run.go`, `exec.go`, `which.go`, `show.go`, `check.go`, `update.go`, `component.go` and so on, with file names that largely match the command names.

`cli` does not implement business logic itself; what it does is parse arguments, call the lower-level packages, and hand the result to the rendering layer. A few subpackages take on the cross-cutting concerns:

- `cli/output` renders command results. Each command defines a struct that implements the `Result` interface (a single `Text()` method), and `output` decides, based on the global `--json` flag, whether to call `Text()` to produce human-readable text or to marshal the struct directly to JSON. The JSON envelope for errors is also assembled here; it recognizes the `Coded` interface from `cjverr` to fill in the machine-readable error code.
- `cli/settings` is the group of configuration subcommands behind `cjv settings` (`set`, `default`, `override`, and so on).
- `cli/selfmgmt` is the group of self-management subcommands behind `cjv self` (update, uninstall), along with the privilege-escalation safety checks.

### `lifecycle`: installation orchestration

`internal/lifecycle` orchestrates download, extraction, verification, components, PATH configuration, and proxy links as one installation flow. It receives adapters such as `IsJSON`, `ComponentInstall`, `CreateProxyLinks`, and `ValidateInstallation` through `Options`, with dependencies directed from `cli` toward `lifecycle`. The same flow serves `cli install` and proxy auto-install.

Files inside the package are split by responsibility: `install.go` owns orchestration, `source.go` turns channel requests into `ResolvedToolchain` values, `component_install.go` owns component batches and rollback, and `resolved_install.go` owns materialization, validation, and transactional replacement. Distribution-source selection stays local to `source.go`.

### `resolve`: active toolchain resolution

`internal/resolve` answers the question of which toolchain to use right now. `Active` combines the command-line `+toolchain` override, the `CJV_TOOLCHAIN` environment variable, the directory-level and global overrides, and the default setting to determine the name and directory of the active toolchain, and returns them as an `ActiveToolchain` together with its target platform and components. If the toolchain is not installed during resolution, it can trigger an automatic install through the `AutoInstallFunc` test seam; in production this seam is wired to `lifecycle` by default, so `resolve` does not need a reverse dependency on `cli`.

### `toolchain` and `component`: models of what is installed

`internal/toolchain` manages the installed SDKs: it lists the installed toolchains (`ListInstalled`), resolves the active toolchain directory, and cleans up leftover staging and backup directories. It defines the directory-suffix conventions for staging (`.staging`), backups (`.old`), and transactions (`.fstx-`), as well as the parsing of toolchain names and version comparison.

`internal/component` manages the add-on components of a toolchain: `stdx`, `docs`, `stdx-docs`. Each component is a separately downloaded archive, and its extracted files are recorded through a per-component manifest, so it can be uninstalled independently. `component` also defines where each component installs to (`InstallLocation`: some land inside the toolchain directory tree, while others are placed as pure data under `<CJV_HOME>/docs/<tc>/`) and which environment variables a component needs to inject.

### `dist`: download and unpacking

`internal/dist` owns distribution sources and network artifacts. `source.go` is the unified entry point: LTS, STS, nightly, and every component share one manifest. The default source reads `manifest_url`, while an explicit `dist_server` reads `<root>/versions.json`. Relative URLs resolve against the manifest directory and absolute URLs are honored as written. `manifest.go` parses and validates all three channels; `download.go` handles progress, retries, and SHA-256; `install.go` unpacks archives; `nightly.go` reads nightly SHA-256 sidecars; and `platform.go` centralizes host platform keys.

### `target`: platform identity

`internal/target` is the single source of truth for platforms and target tuples. It parses the target tuple (the host part plus an optional cross-compilation environment suffix) and produces structured views such as the manifest index key and stdx platform token, sparing every caller from slicing strings on its own. `catalog.go` lists every `(GOOS, GOARCH)` combination for which cjv ships a host binary, and is the source for both the release artifacts and the download entries on the landing page; it carries a `go:generate` directive and is generated by running `scripts/gen-platform-surfaces.go`.

### `env`: runtime environment

`internal/env` assembles the environment needed to run the Cangjie tools. `Runtime` wraps the active toolchain together with the SDK environment derived from it, and exposes several narrow views: the environment for proxied subprocesses, the environment for executing the toolchain directly, and the environment to write into a shell. It handles `LD_LIBRARY_PATH` / `PATH` assembly (split by platform across `ldpath_unix.go` / `ldpath_windows.go`), `SDKROOT`, shell detection, and the script formats for each shell (`shelldetect.go`, `shell_*.go`, `shellformat.go`), and is the foundation shared by `cjv env` and proxy execution.

### `proxy`: transparent proxy

`internal/proxy` implements the transparent proxy: when the binary is invoked under a tool name such as `cjc` or `cjpm`, `Run` resolves the active toolchain (through `env.ResolveRuntime`), locates the real tool binary inside the toolchain directory (`toolPathMap` in `tools.go` maps tool names to relative paths), assembles the proxy environment, and then `exec`s that binary, passing the arguments straight through. It carries a recursion counter (`CJV_RECURSION_COUNT`) to prevent the proxy from calling itself indefinitely. `link.go` is responsible for creating these proxy links at install time (`CreateAllProxyLinks`).

### `config`: configuration and paths

`internal/config` is the configuration layer. It defines all `CJV_*` variables, including `CJV_DIST_SERVER`, resolves `CJV_HOME`, reads user and system fallback settings, reads the toolchain file, and manages directory overrides. `manifest_url` supplies the default distribution manifest, `dist_server` selects an enterprise source represented by `<root>/versions.json`, and the `mirror` build tag selects the default manifest address.

### `selfupdate`: self-update

`internal/selfupdate` implements `cjv self update`. Whether it goes through GitHub or GitCode is chosen at compile time by the `mirror` build tag (`update_default.go` / `update_mirror.go`). It also manages establishing the current binary as the managed executable, as well as replacing the running binary during an update (split by platform across `replace_windows.go` / `replace_other.go`).

### Supporting packages

The remaining few are supporting packages shared across the layers:

- `i18n` internationalization. Messages live in `locales/en.toml` and `locales/zh-CN.toml` and are embedded into the binary, and `i18n.T` looks up a string by message ID. All user-facing text goes through it, error messages included.
- `cjverr` error types. It defines structured errors carrying a stable machine code (`ErrorCode`); the `Error()` method produces the human-readable message through `i18n`, and the `Coded` interface lets `output` emit the error code in JSON mode. `ExitCodeError` carries the process exit code.
- `fstx` filesystem transactions. It wraps a set of file additions, deletions, and modifications into a rollbackable transaction, which operations such as toolchain replacement rely on to ensure nothing half-finished is left behind on failure.
- `utils` miscellaneous utilities: atomic writes, file operations, Windows junctions, retries, console UTF-8, opening a browser, version-number parsing, and so on, most of them split into per-platform files.
- `logging` configures the global `slog` logger via the `CJV_LOG` environment variable (defaulting to `warn`).
- `testutil` test helpers: a mock download server and a Windows registry guard. It carries source files outside of `_test.go` so that the tests of other packages can import them.

## The flow of a single command

Tying the above together, here is roughly how `cjv install <toolchain>` runs.

The process starts in `run` in `cmd/cjv/main.go`: `logging.Init` sets up logging, the program name is `cjv` rather than some tool name, so it takes the `cli.Execute` path. cobra routes the `install` subcommand to `runInstall` in `internal/cli/install.go`. `runInstall` collects the `--target`, `--component`, `--force` and other flags, assembles a `lifecycle.Options` (wiring in the implementations of `output`, `component`, `proxy`, `selfupdate`), and calls into `internal/lifecycle`.

`lifecycle` first asks `dist.Source` to resolve the requested channel, version, platform, and component artifacts from the manifest. The shared download and extraction path then materializes a staging directory, after which `component`, `proxy`, and `fstx` complete component installation, proxy links, and transactional replacement. Every channel shares this installation path. Progress goes through `output`, and errors are translated into an exit code at `main`.

The proxy path is the other main line. When you run `cjc build`, what is actually invoked is the cjv link named `cjc`, and `main` recognizes the tool name and takes the `proxy.Run` path: `proxy`, through `env.ResolveRuntime`, has `resolve` determine the active toolchain, finds the real `cjc` inside the toolchain directory, assembles the run environment, and then `exec`s into it. This line does not touch `cli` and does not render any of cjv's own output; it purely passes the tool straight through.

To dig into a particular area, start with `internal/cli/root.go` for commands, `internal/lifecycle/install.go` for orchestration, `internal/dist/source.go` for distribution selection, `internal/lifecycle/resolved_install.go` for materialization, and `internal/proxy/proxy.go` for proxying. See [Testing](testing.md) for test organization.
