# cjv developer guide

This guide covers cjv builds, architecture, testing, website maintenance, and releases. See the [user guide](https://cjv.zxilly.com/book/user-guide/en/) for installing and using Cangjie SDKs.

## Repository layout

| Directory | Contents |
| --- | --- |
| `cmd/cjv/` | Go entry point for `cjv`, `cjv-init`, and SDK proxy binaries |
| `internal/` | CLI command groups, installation lifecycle, distribution, transactions, and runtime environment |
| `tests/` | End-to-end CLI integration, installer script, and live-download smoke tests |
| `web/` | React landing page, browser platform detection, and public installer scripts (`install.sh`, `install.ps1`) |
| `docs/` | User and developer guides (with Chinese in `src/` and English in `en/`) plus the shared theme |
| `deployment/` | Edge functions and site verification tools for Cloudflare Worker and Tencent Cloud EdgeOne |
| `scripts/` | Platform catalog generators, CI scope filter, documentation builder, and HarmonyOS test helpers |

## Quick workflow for contributors

1. Start by [building `cjv` from source](building.md) and trying out your local build in an isolated `CJV_HOME` directory.
2. Read the [Architecture guide](architecture.md) to find the package that owns the behavior you want to change and understand its locking, transaction, and state-isolation invariants.
3. Run the relevant commands in [Testing and checks](testing.md) to verify your changes across Go packages, installer scripts, the frontend, and the bilingual mdBook manuals.
4. Follow the [Contributing guide](contributing.md) for atomic commits, commit message conventions, and pull request expectations.
5. For maintaining the public website, documentation build, or release pipelines, see [Landing page](web.md), [Documentation site](documentation.md), and [Releases](releasing.md).
