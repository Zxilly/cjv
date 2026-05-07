#Requires -Version 5.1
<#
.SYNOPSIS
    Installs cjv - Cangjie Version Manager
.DESCRIPTION
    Downloads and installs cjv, then runs 'cjv init' to complete setup.
.PARAMETER Mirror
    Download the cjv-mirror archive from GitCode (for environments without
    reliable GitHub access).
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

if ($env:CJV_MIRROR -eq "1") {
    $Mirror = $true
}

$CjvGithubRoot  = if ($env:CJV_GITHUB_ROOT)  { $env:CJV_GITHUB_ROOT }  else { "https://github.com/Zxilly/cjv/releases/latest/download" }
$CjvGitcodeRoot = if ($env:CJV_GITCODE_ROOT) { $env:CJV_GITCODE_ROOT } else { "https://gitcode.com/Zxilly/cjv/releases/latest/download" }

if ($env:CJV_UPDATE_ROOT) {
    $CjvUpdateRoot = $env:CJV_UPDATE_ROOT
} elseif ($Mirror) {
    $CjvUpdateRoot = $CjvGitcodeRoot
} else {
    $CjvUpdateRoot = $CjvGithubRoot
}

$BinaryName = if ($Mirror) { "cjv-mirror" } else { "cjv" }

function Get-Architecture {
    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { Write-Error "Windows ARM64 is not currently supported"; exit 1 }
        default { Write-Error "Unsupported architecture: $arch"; exit 1 }
    }
}

function Install-Cjv {
    $arch = Get-Architecture
    $url = "$CjvUpdateRoot/${BinaryName}_windows_$arch.zip"
    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "cjv-install-$([System.Guid]::NewGuid().ToString('N').Substring(0, 8))"

    try {
        New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
        $zipPath = Join-Path $tmpDir "cjv.zip"

        Write-Host "cjv-install: downloading cjv from $url"
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

        # PS 5.1's Write-Progress slows Invoke-WebRequest / Expand-Archive
        # ~100x for binary payloads.
        $savedProgress = $global:ProgressPreference
        try {
            $global:ProgressPreference = 'SilentlyContinue'
            Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing

            Write-Host "cjv-install: extracting"
            Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force
        }
        finally {
            $global:ProgressPreference = $savedProgress
        }

        $cjvExe = Join-Path $tmpDir "$BinaryName.exe"
        if (-not (Test-Path $cjvExe)) {
            Write-Error "$BinaryName.exe not found in downloaded archive"
            exit 1
        }

        Write-Host "cjv-install: running cjv init"
        $initArgs = @("init")
        if ($Yes)          { $initArgs += "-y" }
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
