# Quick start

This chapter shows how to check your setup, select a default SDK, pin a project version, and update toolchains. If you have not installed cjv yet, start with [Install cjv](installation/index.md).

## Do a check of your installation

After you complete the setup wizard, open a new terminal and check the active toolchain:

```bash
cjv show
cjc --version
```

If your shell cannot find `cjc`, make sure that `<CJV_HOME>/bin` is on your `PATH`. The default path is `~/.cjv/bin`. Read [Install cjv](installation/index.md#configure-path-and-check-the-setup) for `PATH` configuration details.

## Set a default toolchain

If you installed cjv without an SDK, find an available channel and set it as the default. Run these commands to use the LTS channel:

```bash
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

The `default` command installs the toolchain if it is not on your machine. If LTS has no release for your platform, check `sts` or `nightly` and select that channel instead. You can also run `cjv init` again to select an available channel through the wizard.

cjv uses the default toolchain in directories that have no project file or directory override. Run `cjv show active` to see the active toolchain and its selection source. Read [Toolchains and versions](concepts/toolchains.md) for channel and version syntax.

## Configure a version for a project

### Create a project toolchain file

To share the same SDK requirement across a team, create `cangjie-sdk.toml` in the project root directory:

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

Replace `lts-1.0.5` with a version that your distribution source provides. You can also run `cjv pin lts` to write the full release name automatically.

When you run `cjpm build` in this directory, cjv uses the toolchain from `cangjie-sdk.toml`. By default, cjv installs missing SDKs and components automatically. Running a proxy command does not upgrade an SDK that is already installed.

### Set a local directory override

If you want to change the toolchain for a directory on your machine only, set a directory override:

```bash
cjv override set sts
cjv show active
cjv override unset
```

cjv saves directory overrides in your user settings file instead of the project repository. Read [Project toolchains](toolchain-file.md) for the full selection order between flags, environment variables, overrides, and project files.

## Run a command with another version

To run one command with an explicit toolchain without changing your default or project settings, use `cjv run`:

```bash
cjv run --install sts cjc --version
```

The `run` command selects the named toolchain for that child process only. The `--install` flag installs the version if it is missing. To run your own compiled Cangjie binary with the SDK runtime library paths, use `cjv exec ./my_binary`. Read [Run commands and set the environment](runtime-environment.md) for details on `run`, `exec`, and `envsetup`.

## Check installed toolchains

Use these commands to check the active SDK, the tool paths, and all local installations:

```bash
cjv show active
cjv which cjc
cjv toolchain list
```

- `cjv show active` prints the selected toolchain and the rule that selected it.
- `cjv which cjc` prints the resolved file path of the compiler binary inside the active SDK.
- `cjv toolchain list` lists all installed host SDKs and their attached cross targets.

## Update and uninstall toolchains

### Update SDKs and cjv

Run these commands to check for updates and install them:

```bash
cjv check
cjv update
```

The `check` command lists available updates for installed SDKs and for cjv without installing them. The `update` command upgrades tracked channels such as `lts`, `sts`, and `nightly`. It keeps pinned versions and custom toolchains unchanged.

The `auto_self_update` setting controls whether a full `cjv update` also checks or updates cjv itself. Read [Configuration](configuration.md) to change this behavior, or update cjv manually at any time:

```bash
cjv self update
```

### Remove a toolchain

To delete an installed toolchain that you no longer need, run `cjv uninstall`:

```bash
cjv uninstall sts
```

Uninstalling a toolchain also removes its attached components, cross targets, and offline documentation. Read [Command reference](command-reference.md) for all command options, or continue to [Run commands and set the environment](runtime-environment.md).
