# Quick start

After [installing cjv](installation/index.md), check the selected toolchain, set a default version, configure a project version, and manage updates.

## Checking your installation

After completing the setup wizard, open a new terminal so your updated `PATH` is loaded, and check the active toolchain:

```bash
cjv show
cjc --version
```

`cjv show` displays your default host platform, all installed toolchains, and which toolchain is currently active. If your shell reports that `cjc` cannot be found, double-check that `<CJV_HOME>/bin` (by default `~/.cjv/bin`) is on your `PATH`—see [PATH configuration and verification](installation/index.md#path-configuration-and-verification) for details.

## Choosing a default toolchain

If you installed `cjv` with `--default-toolchain none` (or skipped SDK installation during setup), you can query the releases available on a channel and set a global default toolchain. Let's take the `lts` channel as an example:

```bash
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

If the toolchain you pass to `cjv default` is not installed yet, `cjv` downloads and installs it automatically before saving it as the default. If `lts` does not publish an SDK for your platform yet, run `cjv toolchain list-remote` for `sts` or `nightly` and select that channel instead—or run `cjv init` again to let the interactive wizard pick an available channel for you.

The global default toolchain is used whenever you are in a directory that does not have a project toolchain file or a local directory override. To see which toolchain is active in your current directory and why it was chosen, run:

```bash
cjv show active
```

See [Toolchains and versions](concepts/toolchains.md) for a full explanation of channels and version naming.

## Choosing a version for a project

### Using `cangjie-sdk.toml`

In a team repository, you usually want every contributor and CI runner to build with the exact same Cangjie release and extension libraries. You can declare these requirements in a `cangjie-sdk.toml` file at the root of your project:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

Replace `lts-1.0.5` with a version that your project needs and your distribution source provides (or run `cjv pin lts` to generate the file with the resolved version automatically).

Whenever you run `cjpm build` or `cjc` inside that project directory, `cjv` automatically selects the toolchain declared in `cangjie-sdk.toml`. By default, `auto_install` is enabled, so `cjv` will download and install any missing SDK, component, or cross-compilation target on the fly. Running a proxy command will never unexpectedly upgrade an SDK that is already installed.

### Setting a local directory override

If you want to test a project with a different toolchain on your own machine without modifying `cangjie-sdk.toml`, you can set a directory override:

```bash
cjv override set sts
cjv show active
cjv override unset
```

Directory overrides are stored in your personal `~/.cjv/settings.toml` file rather than inside the project directory, while `cangjie-sdk.toml` is meant to be committed to version control. See [Project toolchains](toolchain-file.md) for how overrides and project files interact.

## Running a command with another version

Sometimes you just want to run a single command with a specific toolchain without changing your default or directory settings. Use `cjv run`:

```bash
cjv run --install sts cjc --version
```

`cjv run` invokes the command using the specified toolchain (`sts` here), and `--install` tells `cjv` to install that toolchain first if it is missing.

If you have compiled your own Cangjie binary and need to run it with the dynamic libraries from the SDK or `stdx`, use `cjv exec`:

```bash
cjv exec ./my_binary
```

See [Running commands and setting the environment](runtime-environment.md) for more on `run`, `exec`, and `envsetup`.

## Inspecting installed toolchains

Here are the most common commands for inspecting your local setup:

```bash
cjv show active
cjv which cjc
cjv toolchain list
```

- `cjv show active` prints the currently selected toolchain and the rule that selected it (such as a `cangjie-sdk.toml` file, a directory override, or the global default).
- `cjv which cjc` prints the full filesystem path to the `cjc` executable inside the active SDK.
- `cjv toolchain list` (or `cjv show installed`) lists all installed host toolchains and any cross-compilation targets attached to them.

## Keeping toolchains and `cjv` up to date

### Updating tracked channels

When new versions of the Cangjie SDK are released, you can check for available updates or install them across all of your tracked channels (`lts`, `sts`, `nightly`):

```bash
cjv check
cjv update
```

`cjv check` queries the remote manifest and reports available updates for your installed SDKs and for `cjv` itself without installing anything. `cjv update` downloads and installs updates for all tracked channels, while leaving pinned versions (like `lts-1.0.5`) and custom toolchains untouched.

Whether a full `cjv update` also checks for or installs updates to `cjv` itself is controlled by the [`auto_self_update`](configuration.md#automatic-updates) setting (`check` by default). You can also update `cjv` manually at any time by running:

```bash
cjv self update
```

### Uninstalling a toolchain

To remove an installed toolchain that you no longer need and reclaim disk space:

```bash
cjv uninstall sts
```

Uninstalling a host toolchain also removes its attached components, cross-compilation targets, and offline documentation. For a complete list of commands and flags, see the [Command reference](command-reference.md).
