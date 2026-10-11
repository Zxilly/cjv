# Quick start

## Check your installation

If you completed the installation wizard, open a new terminal and check the selected toolchain:

```bash
cjv show
cjc --version
```

If the shell cannot find `cjc`, check that `<CJV_HOME>/bin` is on `PATH`; the default is `~/.cjv/bin`. See [installation](installation/index.md) for PATH setup.

## Choose a default toolchain

If you installed cjv without an SDK, query an available channel and select it. For LTS:

```bash
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

`default` installs the toolchain if it is missing. If LTS has no SDK for your platform, query STS or nightly and use that channel instead. You can also run `cjv init` to let the wizard select an available channel.

The default applies to directories without a project selection or override. To inspect its source, run `cjv show active`.

## Choose a project version

Create `cangjie-sdk.toml` in the project root:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

Replace the example version with one your project needs and your distribution provides. Running `cjpm build` in this directory uses the declared toolchain. Automatic installation is enabled by default and supplies missing SDKs and components. Running a command does not automatically upgrade an existing toolchain.

For a selection that applies only on your machine, use a directory override:

```bash
cjv override set sts
cjv show active
cjv override unset
```

Overrides are stored in user settings; project files can be committed to version control. See [project toolchains](toolchain-file.md) for precedence.

## Run a command with another version

```bash
cjv run --install sts cjc --version
```

`run` selects a toolchain explicitly; `--install` allows a missing version to be installed. Use `cjv exec ./my_binary` to run your own compiled program with its runtime library paths.

## Inspect installed toolchains

```bash
cjv show active
cjv which cjc
cjv toolchain list
```

These commands report the active toolchain and its source, the compiler path, and the installed toolchains, respectively.

## Update and uninstall

```bash
cjv check
cjv update
```

`check` reports available SDK and cjv updates without installing them. `update` updates tracked channels and leaves pinned versions and custom toolchains unchanged. The [auto_self_update](configuration.md) setting controls whether a full update checks or updates cjv itself.

To update cjv manually:

```bash
cjv self update
```

To remove a toolchain you no longer need:

```bash
cjv uninstall sts
```

Uninstalling a toolchain also removes its components and documentation. For more options, see the [command reference](command-reference.md).
