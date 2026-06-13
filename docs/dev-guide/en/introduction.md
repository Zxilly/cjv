# Introduction

cjv is a toolchain manager for the [Cangjie](https://cangjie-lang.cn/) programming language SDK, written in Go. It manages multiple Cangjie SDK installations, handles version switching, and provides transparent proxy execution for SDK tools such as `cjc` and `cjpm`. User-facing features are described in the cjv user guide: <https://cjv.zxilly.dev/book/user-guide/zh-CN/>.

This development guide is aimed at cjv contributors and developers. It covers how to build from source, how the code is organized, how to write and run tests, how the documentation site and landing page work, and what the CI and release processes look like. If you only want to use cjv to install a toolchain, the user guide is all you need; this guide is about how cjv itself is built.

## Repository Layout

The repository root is divided into four parts.

The Go CLI is the core. The command entry point is in `cmd/cjv/main.go`, while the actual logic lives under `internal/`, split by subsystem: `cli` holds the cobra command definitions, `resolve` performs toolchain version resolution, `toolchain` and `component` manage installed SDKs and components, `proxy` implements the transparent proxy, `env` handles the runtime environment, and `config`, `dist`, `selfupdate`, and others each have their own role. The module path is `github.com/Zxilly/cjv`.

`web/` is the landing page, a Vite + React 19 + TypeScript project hosted at <https://cjv.zxilly.dev>. It provides the one-line `install.sh` / `install.ps1` install scripts and platform-specific download entry points. Its copy is internationalized with Lingui.

`docs/` contains two mdBooks: `docs/user-guide/` is the user guide, and `docs/dev-guide/` is this development guide you are reading. Each keeps two independent sources: `src/` is Simplified Chinese (the source language) and `en/` is English.

`tests/` holds cross-package integration and smoke tests: `tests/integration/` contains end-to-end integration tests, `tests/smoke/` verifies real downloads, and `tests/install-scripts/` tests the install scripts. Unit tests, by convention, live in the same directory as the code under test, with `_test.go` files right next to the source files.

## Prerequisites

Building the Go CLI requires Go, at the version specified by the `go` directive in `go.mod` (currently 1.26.0). Once Go is installed, `go build ./...` compiles the entire project; see [Building from Source](building.md) for details.

Developing the landing page requires Node and pnpm. The package manager is pinned in the `packageManager` field of `web/package.json` (pnpm 10.x). Run `pnpm install` to install dependencies and `pnpm dev` to start the dev server, both from `web/`; see [Landing Page](web.md).

Building the documentation locally requires only [mdBook](https://github.com/rust-lang/mdBook). The version used in CI is recorded in `.github/workflows/pages.yml`; see [Documentation site](documentation.md).

## Next Steps

[Building from Source](building.md) covers how to compile and run the CLI, [Code Architecture](architecture.md) explains how the subsystems under `internal/` work together, and [Testing](testing.md) and [Linting and Formatting](linting.md) cover how to keep changes correct and consistent with the project style. [Documentation Site](documentation.md), [Landing Page](web.md), [Continuous Integration](ci.md), and [Release Process](releasing.md) each cover one of the four subsystems. When you are ready to submit a change, see the [Contributing Guide](contributing.md).
