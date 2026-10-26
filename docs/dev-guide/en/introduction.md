# cjv developer guide

This guide explains how to build cjv from source, how its internal packages work together, and how to test and release changes. If you want to install Cangjie SDKs and use cjv in your projects, read the [cjv user guide](https://cjv.zxilly.com/book/user-guide/en/).

## Repository layout

| Directory | Contents |
| --- | --- |
| `cmd/cjv/` | Main entry point for `cjv`, `cjv-init`, and SDK proxy binaries |
| `internal/` | CLI command groups, installation lifecycle, distribution, transactions, and runtime setup |
| `tests/` | End-to-end CLI integration tests, installer script tests, and live download smoke tests |
| `web/` | React landing page, platform detector, and public installer scripts (`install.sh`, `install.ps1`) |
| `docs/` | Bilingual mdBook sources for the user guide and developer guide, plus the shared theme |
| `deployment/` | Edge functions and site verification tools for Cloudflare and EdgeOne |
| `scripts/` | Platform catalog generators, CI scope filter, documentation builder, and HarmonyOS test runners |

## How to use this guide

1. Start with [Build from source](building.md) to compile the `cjv` binary, build the `mirror` variant, or cross-compile for OpenHarmony.
2. Read [Architecture](architecture.md) to find the package that owns the behavior you want to change and understand its locking and transaction guarantees.
3. Run the relevant commands in [Tests and checks](testing.md) to check your change across Go packages, installer scripts, the frontend, and documentation.
4. Follow [Contribute to cjv](contributing.md) to format commits and open a pull request.
5. For website and release maintenance, read [Landing page](web.md), [Documentation site](documentation.md), and [Releases](releasing.md).
