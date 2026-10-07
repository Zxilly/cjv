# cjv developer guide

This guide covers building cjv, its code structure, tests, and releases. For SDK installation and daily use, see the [user guide](https://cjv.zxilly.dev/book/user-guide/en/).

| Directory | Contents |
| --- | --- |
| `cmd/cjv/` | Go CLI entry point |
| `internal/` | Commands, installation lifecycle, distribution, and runtime environment |
| `tests/` | CLI integration, installer, and real-download tests |
| `web/` | React landing page and installer scripts |
| `docs/` | User and developer guides, with Chinese and English sources |
| `scripts/` | Platform generation and release helpers |

Start by [building](building.md) the CLI, then use the [architecture guide](architecture.md) to find the module that owns your change. [Testing and checks](testing.md) lists validation commands; [contributing](contributing.md) covers submission conventions.
