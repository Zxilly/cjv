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

$releaseEnv = @{}
if (Test-Path $releaseEnvPath) {
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
}

function Resolve-EnvValue {
    param(
        [string]$Name,
        [string]$Default
    )
    $value = [Environment]::GetEnvironmentVariable($Name)
    if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }
    if ($releaseEnv.ContainsKey($Name) -and -not [string]::IsNullOrWhiteSpace($releaseEnv[$Name])) {
        return $releaseEnv[$Name]
    }
    return $Default
}

$repository  = Resolve-EnvValue -Name "CJV_REPOSITORY"        -Default "Zxilly/cjv"
$baseUrl     = Resolve-EnvValue -Name "CJV_RELEASE_BASE_URL"  -Default "https://github.com/$repository/releases/download"
$apiBaseUrl  = Resolve-EnvValue -Name "CJV_API_BASE_URL"      -Default "https://api.github.com"

$tag = Resolve-EnvValue -Name "CJV_VERSION" -Default ""
if ($tag) {
    if (-not $tag.StartsWith("v")) { $tag = "v$tag" }
}
else {
    $latestUrl = "$apiBaseUrl/repos/$repository/releases/latest"
    $headers = @{ "Accept" = "application/vnd.github+json"; "User-Agent" = "cjv-installer" }
    $response = Invoke-RestMethod -Uri $latestUrl -Headers $headers
    $tag = $response.tag_name
    if ([string]::IsNullOrWhiteSpace($tag)) {
        throw "failed to resolve latest cjv release tag from $apiBaseUrl"
    }
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

    Write-Host "Installed cjv $tag to $destination"
}
finally {
    if (Test-Path $tmpDir) {
        Remove-Item -LiteralPath $tmpDir -Recurse -Force
    }
}
