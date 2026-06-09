$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir "..\..")).Path
$InstallPs1 = Join-Path $RepoRoot "web\public\install.ps1"

$TargetPowerShell = if ($env:CJV_INSTALL_TEST_POWERSHELL) { $env:CJV_INSTALL_TEST_POWERSHELL } else { "powershell" }
$DefaultToolchain = if ($env:CJV_INSTALL_TEST_DEFAULT_TOOLCHAIN) { $env:CJV_INSTALL_TEST_DEFAULT_TOOLCHAIN } else { "none" }
$TempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("cjv-install-script-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $TempRoot -Force | Out-Null

function Remove-TestTemp {
    if (Test-Path $TempRoot) {
        Remove-Item -Path $TempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Fail {
    param([string]$Message)
    [Console]::Error.WriteLine("install-ps1-test: error: " + $Message)
    Remove-TestTemp
    exit 1
}

function Say {
    param([string]$Message)
    Write-Host ("install-ps1-test: " + $Message)
}

function Require-File {
    param([string]$Path)
    if (-not (Test-Path $Path -PathType Leaf)) {
        Fail "expected file missing: $Path"
    }
}

function Require-Directory {
    param([string]$Path)
    if (-not (Test-Path $Path -PathType Container)) {
        Fail "expected directory missing: $Path"
    }
}

function Require-AnyToolchain {
    param([string]$Path)
    Require-Directory $Path
    $children = @(Get-ChildItem -Path $Path | Where-Object { $_.PSIsContainer })
    if ($children.Count -eq 0) {
        Fail "expected at least one installed toolchain under $Path"
    }
}

function Assert-BaseInstall {
    param([string]$CjvHome)

    Require-File (Join-Path $CjvHome "bin\cjv.exe")
    Require-File (Join-Path $CjvHome "bin\cjc.exe")
    Require-File (Join-Path $CjvHome "bin\cjpm.exe")
    Require-File (Join-Path $CjvHome "env.ps1")
    Require-File (Join-Path $CjvHome "env.bat")
}

function Get-EnvironmentSnapshot {
    param([string[]]$Names)

    $snapshot = @{}
    foreach ($name in $Names) {
        $snapshot[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
    }
    return $snapshot
}

function Restore-EnvironmentSnapshot {
    param(
        [hashtable]$Snapshot,
        [string[]]$Names
    )

    foreach ($name in $Names) {
        [Environment]::SetEnvironmentVariable($name, $Snapshot[$name], "Process")
    }
}

function Invoke-WithInstallerEnvironment {
    param(
        [string]$CjvHome,
        [scriptblock]$Body
    )

    $names = @(
        "CJV_UPDATE_ROOT",
        "CJV_GITHUB_ROOT",
        "CJV_GITCODE_ROOT",
        "CJV_MIRROR",
        "CJV_FALLBACK_SETTINGS",
        "CJV_NO_PATH_SETUP",
        "CJV_HOME",
        "HOME",
        "USERPROFILE",
        "HOMEDRIVE",
        "HOMEPATH",
        "CJV_LANG",
        "CJV_TOOLCHAIN"
    )
    $snapshot = Get-EnvironmentSnapshot -Names $names

    try {
        foreach ($name in @("CJV_UPDATE_ROOT", "CJV_GITHUB_ROOT", "CJV_GITCODE_ROOT", "CJV_MIRROR", "CJV_FALLBACK_SETTINGS", "CJV_NO_PATH_SETUP")) {
            [Environment]::SetEnvironmentVariable($name, $null, "Process")
        }
        [Environment]::SetEnvironmentVariable("CJV_HOME", $CjvHome, "Process")
        [Environment]::SetEnvironmentVariable("HOME", $CjvHome, "Process")
        [Environment]::SetEnvironmentVariable("USERPROFILE", $CjvHome, "Process")
        [Environment]::SetEnvironmentVariable("HOMEDRIVE", (Split-Path -Qualifier $CjvHome), "Process")
        [Environment]::SetEnvironmentVariable("HOMEPATH", ($CjvHome.Substring((Split-Path -Qualifier $CjvHome).Length)), "Process")
        [Environment]::SetEnvironmentVariable("CJV_LANG", "en", "Process")
        [Environment]::SetEnvironmentVariable("CJV_TOOLCHAIN", "", "Process")

        & $Body
    } finally {
        Restore-EnvironmentSnapshot -Snapshot $snapshot -Names $names
    }
}

function Invoke-Installer {
    param(
        [string]$CjvHome,
        [string]$Mode
    )

    $shell = Get-Command $TargetPowerShell -ErrorAction Stop
    $shellPath = if ($shell.Path) { $shell.Path } else { $shell.Source }

    $installArgs = @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        $InstallPs1,
        "-Yes",
        "-NoModifyPath"
    )

    switch ($Mode) {
        "none" {
            $installArgs += "-DefaultToolchain"
            $installArgs += "none"
        }
        "lts" {}
        default {
            Fail "unsupported CJV_INSTALL_TEST_DEFAULT_TOOLCHAIN mode: $Mode"
        }
    }

    Invoke-WithInstallerEnvironment -CjvHome $CjvHome -Body {
        Say "running install.ps1 through $TargetPowerShell ($Mode)"
        & $shellPath @installArgs
        if ($LASTEXITCODE -ne 0) {
            Fail "install.ps1 exited with code $LASTEXITCODE"
        }
    }
}

function Run-Case {
    param([string]$Mode)

    $cjvHome = Join-Path $TempRoot $Mode
    New-Item -ItemType Directory -Path $cjvHome -Force | Out-Null

    Invoke-Installer -CjvHome $cjvHome -Mode $Mode
    Assert-BaseInstall -CjvHome $cjvHome

    if ($Mode -eq "lts") {
        Require-AnyToolchain (Join-Path $cjvHome "toolchains")
    }
}

try {
    if ($DefaultToolchain -eq "all") {
        $modes = @("none", "lts")
    } else {
        $modes = @($DefaultToolchain)
    }

    foreach ($mode in $modes) {
        Run-Case -Mode $mode
    }

    Say "ok"
} finally {
    Remove-TestTemp
}
