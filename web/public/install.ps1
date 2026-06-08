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
$script:CjvInstallTmpDir = $null

if ($env:CJV_MIRROR -eq "1") {
    $Mirror = $true
}

if ($env:CJV_GITHUB_ROOT) {
    $CjvGithubRoot = $env:CJV_GITHUB_ROOT
} else {
    $CjvGithubRoot = "https://github.com/Zxilly/cjv/releases/latest/download"
}

if ($env:CJV_GITCODE_ROOT) {
    $CjvGitcodeRoot = $env:CJV_GITCODE_ROOT
} else {
    $CjvGitcodeRoot = "https://gitcode.com/Zxilly/cjv/releases/latest/download"
}

if ($env:CJV_UPDATE_ROOT) {
    $CjvUpdateRoot = $env:CJV_UPDATE_ROOT
} elseif ($Mirror) {
    $CjvUpdateRoot = $CjvGitcodeRoot
} else {
    $CjvUpdateRoot = $CjvGithubRoot
}

if ($Mirror) {
    $BinaryName = "cjv-mirror"
} else {
    $BinaryName = "cjv"
}

function Cleanup-CjvInstall {
    if ($script:CjvInstallTmpDir -and (Test-Path $script:CjvInstallTmpDir)) {
        Remove-Item -Path $script:CjvInstallTmpDir -Recurse -Force -ErrorAction SilentlyContinue
        $script:CjvInstallTmpDir = $null
    }
}

function Fail {
    param([string]$Message)
    [Console]::Error.WriteLine("cjv-install: error: " + $Message)
    Cleanup-CjvInstall
    exit 1
}

trap {
    [Console]::Error.WriteLine($_)
    Cleanup-CjvInstall
    exit 1
}

function Get-Architecture {
    if ($env:PROCESSOR_ARCHITEW6432) {
        $arch = $env:PROCESSOR_ARCHITEW6432
    } else {
        $arch = $env:PROCESSOR_ARCHITECTURE
    }
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { Fail "Windows ARM64 is not currently supported" }
        default { Fail "Unsupported architecture: $arch" }
    }
}

function Enable-Tls12 {
    $protocols = [Enum]::GetNames([Net.SecurityProtocolType])
    if ($protocols -contains "Tls12") {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    }
}

function Has-Command {
    param([string]$Name)
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    return $cmd -ne $null
}

function Download-File {
    param(
        [string]$Uri,
        [string]$OutFile
    )

    Enable-Tls12

    if (Has-Command "Invoke-WebRequest") {
        $savedProgress = $global:ProgressPreference
        $global:ProgressPreference = "SilentlyContinue"
        Invoke-WebRequest -Uri $Uri -OutFile $OutFile -UseBasicParsing
        $global:ProgressPreference = $savedProgress
        return
    }

    $client = New-Object Net.WebClient
    $client.DownloadFile($Uri, $OutFile)
    $client.Dispose()
}

function Expand-ZipArchive {
    param(
        [string]$ZipPath,
        [string]$DestinationPath,
        [string]$ExpectedFile
    )

    if (Has-Command "Expand-Archive") {
        Expand-Archive -Path $ZipPath -DestinationPath $DestinationPath -Force
        return
    }

    $shell = New-Object -ComObject Shell.Application
    $zip = $shell.NameSpace($ZipPath)
    if ($zip -eq $null) {
        Fail "failed to open downloaded archive: $ZipPath"
    }

    $dest = $shell.NameSpace($DestinationPath)
    if ($dest -eq $null) {
        Fail "failed to open extraction directory: $DestinationPath"
    }

    $dest.CopyHere($zip.Items(), 20)

    $expectedPath = Join-Path $DestinationPath $ExpectedFile
    $deadline = [DateTime]::UtcNow.AddSeconds(60)
    while ((-not (Test-Path $expectedPath)) -and ([DateTime]::UtcNow -lt $deadline)) {
        Start-Sleep -Seconds 1
    }

    if (-not (Test-Path $expectedPath)) {
        Fail "timed out extracting $ExpectedFile from downloaded archive"
    }
}

function Install-Cjv {
    $arch = Get-Architecture
    $url = "$CjvUpdateRoot/${BinaryName}_windows_$arch.zip"
    $tmpName = "cjv-install-" + [System.Guid]::NewGuid().ToString("N").Substring(0, 8)
    $script:CjvInstallTmpDir = Join-Path ([System.IO.Path]::GetTempPath()) $tmpName
    New-Item -ItemType Directory -Path $script:CjvInstallTmpDir -Force | Out-Null

    $zipPath = Join-Path $script:CjvInstallTmpDir "cjv.zip"

    Write-Host "cjv-install: downloading cjv from $url"
    Download-File -Uri $url -OutFile $zipPath

    Write-Host "cjv-install: extracting"
    Expand-ZipArchive -ZipPath $zipPath -DestinationPath $script:CjvInstallTmpDir -ExpectedFile "$BinaryName.exe"

    $cjvExe = Join-Path $script:CjvInstallTmpDir "$BinaryName.exe"
    if (-not (Test-Path $cjvExe)) {
        Fail "$BinaryName.exe not found in downloaded archive"
    }

    Write-Host "cjv-install: running cjv init"
    $initArgs = @("init")
    if ($Yes) { $initArgs += "-y" }
    if ($NoModifyPath) { $initArgs += "--no-modify-path" }
    if ($DefaultToolchain -ne "lts") {
        $initArgs += "--default-toolchain"
        $initArgs += $DefaultToolchain
    }

    & $cjvExe $initArgs
    $exitCode = $LASTEXITCODE
    Cleanup-CjvInstall
    if ($exitCode -ne 0) {
        exit $exitCode
    }
}

Install-Cjv
