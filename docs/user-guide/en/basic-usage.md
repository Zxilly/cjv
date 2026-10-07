# Quick start

Install the LTS toolchain and make it the default:

```bash
cjv install lts
cjv default lts
cjc --version
```

If the shell cannot find `cjc`, open a new terminal to load the installed `PATH` setting. If it is still missing, check that `<CJV_HOME>/bin` is on `PATH`; the default is `~/.cjv/bin`.

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

## Run once and inspect

```bash
cjv run --install sts cjc --version
cjv show
cjv which cjc
cjv toolchain list
```

`run` selects a toolchain explicitly; `--install` allows a missing version to be installed. Use `cjv exec ./my_binary` to run your own compiled program with its runtime library paths.

## Update and uninstall

```bash
cjv check
cjv update
cjv self update
cjv uninstall sts
```

`check` looks for SDK and cjv updates. `update` updates tracked channels and leaves pinned versions and custom toolchains unchanged. The [auto_self_update](configuration.md) setting controls automatic updates to cjv itself. `uninstall` also removes the selected toolchain's components and documentation.
