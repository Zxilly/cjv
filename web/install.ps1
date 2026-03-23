#Requires -Version 5.1
<#
.SYNOPSIS
    Installs cjv - Cangjie Version Manager
.DESCRIPTION
    Downloads and installs cjv, then runs 'cjv init' to complete setup.
.PARAMETER Mirror
    Use mirror source for toolchain downloads
.PARAMETER Yes
    Skip confirmation prompt
.PARAMETER DefaultToolchain
    Default toolchain to install (default: lts, use 'none' to skip)
.PARAMETER NoModifyPath
    Do not modify PATH
.EXAMPLE
    irm https://cjv.zxilly.dev/install.ps1 | iex
.EXAMPLE
    & ([scriptblock]::Create((irm https://cjv.zxilly.dev/install.ps1))) -Mirror -Yes
#>
param(
    [switch]$Mirror,
    [switch]$Yes,
    [string]$DefaultToolchain = "lts",
    [switch]$NoModifyPath
)

$ErrorActionPreference = "Stop"

# Support CJV_MIRROR env var
if ($env:CJV_MIRROR -eq "1") {
    $Mirror = $true
}

$CjvUpdateRoot = if ($env:CJV_UPDATE_ROOT) { $env:CJV_UPDATE_ROOT } else { "https://github.com/Zxilly/cjv/releases/latest/download" }

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { Write-Error "Windows arm64 is not currently supported"; exit 1 }
        default { Write-Error "Unsupported architecture: $arch"; exit 1 }
    }
}

function Install-Cjv {
    $arch = Get-Architecture
    $url = "$CjvUpdateRoot/cjv_windows_$arch.zip"
    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "cjv-install-$([System.Guid]::NewGuid().ToString('N').Substring(0, 8))"

    try {
        New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
        $zipPath = Join-Path $tmpDir "cjv.zip"

        Write-Host "cjv-install: downloading cjv from $url"
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing

        Write-Host "cjv-install: extracting"
        Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

        $cjvExe = Join-Path $tmpDir "cjv.exe"
        if (-not (Test-Path $cjvExe)) {
            Write-Error "cjv.exe not found in downloaded archive"
            exit 1
        }

        Write-Host "cjv-install: running cjv init"
        $initArgs = @("init")
        if ($Yes)          { $initArgs += "-y" }
        if ($Mirror)       { $initArgs += "--mirror" }
        if ($NoModifyPath) { $initArgs += "--no-modify-path" }
        if ($DefaultToolchain -ne "lts") { $initArgs += "--default-toolchain"; $initArgs += $DefaultToolchain }

        & $cjvExe @initArgs
        if ($LASTEXITCODE -ne 0) {
            exit $LASTEXITCODE
        }
    }
    finally {
        if (Test-Path $tmpDir) {
            Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Install-Cjv
