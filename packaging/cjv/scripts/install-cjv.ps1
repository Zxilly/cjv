param(
    [Parameter(Mandatory = $true)]
    [string]$Mode
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$moduleDir = Split-Path -Parent $scriptDir
$releaseEnvPath = Join-Path $moduleDir "release.env"

function Get-Sha256Hex {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $stream = [System.IO.File]::OpenRead($Path)
    try {
        $sha256 = [System.Security.Cryptography.SHA256]::Create()
        try {
            return ([System.BitConverter]::ToString($sha256.ComputeHash($stream)) -replace "-", "").ToLowerInvariant()
        }
        finally {
            $sha256.Dispose()
        }
    }
    finally {
        $stream.Dispose()
    }
}

if (-not (Test-Path $releaseEnvPath)) {
    throw "release.env is missing: $releaseEnvPath"
}

$releaseEnv = @{}
foreach ($line in Get-Content $releaseEnvPath) {
    if ([string]::IsNullOrWhiteSpace($line) -or $line.TrimStart().StartsWith("#")) {
        continue
    }

    $parts = $line -split "=", 2
    if ($parts.Count -ne 2) {
        throw "invalid release.env line: $line"
    }
    $releaseEnv[$parts[0].Trim()] = $parts[1].Trim()
}

$version = $releaseEnv["CJV_VERSION"]
$tag = $releaseEnv["CJV_TAG"]
$repository = $releaseEnv["CJV_REPOSITORY"]
$baseUrl = $env:CJV_RELEASE_BASE_URL
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
    $baseUrl = $releaseEnv["CJV_RELEASE_BASE_URL"]
}
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
    $baseUrl = "https://github.com/$repository/releases/download"
}

if ([string]::IsNullOrWhiteSpace($version) -or [string]::IsNullOrWhiteSpace($tag) -or [string]::IsNullOrWhiteSpace($repository)) {
    throw "release.env must define CJV_VERSION, CJV_TAG and CJV_REPOSITORY"
}

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
switch ($arch) {
    "X64" { $asset = "cjv_windows_amd64.zip" }
    "Arm64" { throw "platform windows/arm64 is not supported" }
    default { throw "platform windows/$arch is not supported" }
}

switch ($Mode) {
    "build" {
        $destination = Join-Path $moduleDir "target\release\bin\main.exe"
    }
    "install" {
        if ([string]::IsNullOrWhiteSpace($env:USERPROFILE)) {
            throw "USERPROFILE is not set; cannot resolve install destination"
        }
        $destination = Join-Path $env:USERPROFILE ".cjpm\bin\cjv.exe"
    }
    default {
        throw "unsupported mode: $Mode"
    }
}

$assetUrl = "$baseUrl/$tag/$asset"
$checksumsUrl = "$baseUrl/$tag/checksums.txt"
$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("cjv-package-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmpDir | Out-Null

try {
    $assetPath = Join-Path $tmpDir $asset
    $checksumsPath = Join-Path $tmpDir "checksums.txt"
    $extractDir = Join-Path $tmpDir "extract"

    Write-Host "Downloading $assetUrl"
    Invoke-WebRequest -Uri $assetUrl -OutFile $assetPath
    Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath

    $checksumPattern = "^[0-9a-fA-F]{64}\s+\*?$([regex]::Escape($asset))$"
    $checksumLine = Get-Content $checksumsPath |
        Where-Object { $_ -match $checksumPattern } |
        Select-Object -First 1
    if (-not $checksumLine) {
        throw "checksum entry for $asset is missing from checksums.txt"
    }

    $expected = ($checksumLine -split "\s+", 2)[0].ToLowerInvariant()
    $actual = Get-Sha256Hex -Path $assetPath
    if ($expected -ne $actual) {
        throw "checksum mismatch for $assetPath (expected $expected, actual $actual)"
    }

    Expand-Archive -Path $assetPath -DestinationPath $extractDir -Force
    $downloadedBinary = Join-Path $extractDir "cjv.exe"
    if (-not (Test-Path $downloadedBinary)) {
        throw "expected extracted binary at $downloadedBinary"
    }

    $destinationDir = Split-Path -Parent $destination
    New-Item -ItemType Directory -Path $destinationDir -Force | Out-Null

    $stagedBinary = Join-Path $tmpDir "cjv.exe"
    Copy-Item -Path $downloadedBinary -Destination $stagedBinary -Force
    Move-Item -Path $stagedBinary -Destination $destination -Force

    Write-Host "Installed cjv $version to $destination"
}
finally {
    if (Test-Path $tmpDir) {
        Remove-Item -LiteralPath $tmpDir -Recurse -Force
    }
}
