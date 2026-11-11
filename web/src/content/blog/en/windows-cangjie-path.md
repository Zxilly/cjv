If PowerShell says that `cjc` is not recognized as a command after installation, first locate the missing link: the terminal's PATH, the cjv shim, or the selected SDK. Reinstalling everything is rarely a useful first diagnostic step.

This guide covers Windows PowerShell environments managed by cjv. For a manually extracted SDK, also follow its installation instructions and check which installation supplies each command.

## Start with a new terminal

cjv initialization attempts to update your user PATH. An existing PowerShell or editor process may still hold the previous environment. Open a new PowerShell window. If you use an integrated terminal, you may also need to restart its editor.

```powershell
Get-Command cjv,cjc,cjpm -All -ErrorAction SilentlyContinue |
  Select-Object Name,Source
```

If the commands now resolve, run `cjv show active` and `cjc --version`. The first explains how cjv selected the SDK; the second checks that the compiler starts.

## Check that the cjv shims exist

The default shim directory is `%USERPROFILE%\.cjv\bin`. If you set `CJV_HOME`, inspect its `bin` directory instead.

```powershell
$cjvRoot = if ($env:CJV_HOME) { $env:CJV_HOME } else {
  Join-Path $env:USERPROFILE '.cjv'
}
$cjvBin = Join-Path $cjvRoot 'bin'
Test-Path (Join-Path $cjvBin 'cjv.exe')
Test-Path (Join-Path $cjvBin 'cjc.exe')
$env:Path -split ';'
```

When the files exist but the directory is absent from PATH, temporarily add it to the current session to confirm the diagnosis:

```powershell
$env:Path = "$cjvBin;$env:Path"
cjv --version
```

This assignment affects only the current terminal. Run `cjv init` and follow its setup prompts to configure subsequent terminals, then open a new terminal. If you chose `--no-modify-path` or set `CJV_NO_PATH_SETUP=1`, you are responsible for managing PATH.

If `cjv.exe` is missing too, use the script or binary instructions in the [installation guide](https://cjv.zxilly.com/book/user-guide/en/installation/). Adding a nonexistent directory to PATH will not restore the executable.

## Separate command lookup from SDK selection

If `cjv --version` works and `cjc --version` produces a cjv toolchain error, command lookup is already working. Inspect the SDK selection and installation:

```powershell
cjv show active
cjv toolchain list
cjv doctor
```

If no default is selected, check which versions are available for the host and choose the required channel. This example assumes an LTS SDK is available:

```powershell
cjv toolchain list-remote --channel lts
cjv default lts
cjc --version
```

`cjv default lts` installs the toolchain when necessary. Use an available channel if the distribution does not provide LTS for your host. If your project already has `cangjie-sdk.toml`, inspect that declaration first: changing the global default does not override the project file. When automatic installation is disabled, explicitly install the required version.

## Check whether an old SDK takes precedence

A manually installed SDK earlier in PATH can bypass cjv. Compare the command found by PowerShell with the compiler selected by cjv:

```powershell
Get-Command cjc -All | Select-Object Name,Source
cjv which cjc
cjv show active
```

With cjv, `Get-Command` should resolve the shim first; `cjv which cjc` points to the actual compiler inside the selected SDK. These paths are expected to differ. If an old SDK appears first, adjust PATH order and check whether a shell profile adds that directory again.

Finish by running `cjc --version` and `cjpm build` from the real project directory. A compiler that starts but cannot build the project needs investigation of the build error. For an executable that cannot find runtime libraries, see [running commands and setting the environment](https://cjv.zxilly.com/book/user-guide/en/runtime-environment.html).
