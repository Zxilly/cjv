$ErrorActionPreference = "Stop"
$installer = Join-Path (Split-Path -Parent (Split-Path -Parent $PSScriptRoot)) "web/public/install.ps1"
$names = @("CJV_VERSION", "CJV_MIRROR", "CJV_UPDATE_ROOT", "CJV_GITHUB_ROOT", "CJV_GITCODE_ROOT", "PROCESSOR_ARCHITECTURE", "PROCESSOR_ARCHITEW6432")
$saved = @{}
$capture = @{ URL = $null }
foreach ($name in $names) {
    $saved[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
}

# Capture the first archive request and stop before any download or execution.
function Invoke-WebRequest {
    param([string]$Uri, [string]$OutFile, [switch]$UseBasicParsing)
    $capture.URL = $Uri
    throw "CJV_INSTALL_REQUEST_CAPTURED"
}

function Check-Request {
    param(
        [string]$Name,
        [hashtable]$Parameters,
        [hashtable]$Environment,
        [string]$ExpectedURL,
        [switch]$Invalid
    )
    foreach ($variable in $names) {
        [Environment]::SetEnvironmentVariable($variable, $null, "Process")
    }
    [Environment]::SetEnvironmentVariable("PROCESSOR_ARCHITECTURE", "AMD64", "Process")
    foreach ($variable in $Environment.Keys) {
        [Environment]::SetEnvironmentVariable($variable, $Environment[$variable], "Process")
    }
    $capture.URL = $null
    $failure = $null
    $savedProgress = $global:ProgressPreference
    $savedError = [Console]::Error
    $errorOutput = New-Object System.IO.StringWriter
    try {
        $global:ProgressPreference = "Continue"
        [Console]::SetError($errorOutput)
        & $installer @Parameters *> $null
    } catch {
        $failure = $_.Exception.Message
    } finally {
        $progressAfterFailure = $global:ProgressPreference
        $global:ProgressPreference = $savedProgress
        [Console]::SetError($savedError)
        $errorOutput.Dispose()
    }
    if ($progressAfterFailure -ne "Continue") {
        throw "$Name changed ProgressPreference after a failed request"
    }
    if ($Invalid) {
        if ($capture.URL -or $failure -notlike "invalid release tag:*") {
            throw "$Name accepted an invalid tag or made a download request: $failure"
        }
    } elseif ($capture.URL -ne $ExpectedURL -or $failure -notlike "*CJV_INSTALL_REQUEST_CAPTURED*") {
        throw "$Name expected '$ExpectedURL', got '$($capture.URL)' ($failure)"
    }
}

try {
    $github = "https://github.com/Zxilly/cjv/releases"
    $gitcode = "https://gitcode.com/Zxilly/cjv/releases"
    Check-Request "latest" @{} @{} "$github/latest/download/cjv_windows_amd64.zip"
    Check-Request "explicit version" @{ Version = "v0.4.1" } @{} "$github/download/v0.4.1/cjv_windows_amd64.zip"
    Check-Request "environment version" @{} @{ CJV_VERSION = "v0.4.1" } "$github/download/v0.4.1/cjv_windows_amd64.zip"
    Check-Request "explicit wins" @{ Version = "v0.4.1" } @{ CJV_VERSION = "v0.3.0" } "$github/download/v0.4.1/cjv_windows_amd64.zip"
    Check-Request "prerelease mirror" @{ Version = "v0.0.0-hmos.1"; Mirror = $true } @{} "$gitcode/download/v0.0.0-hmos.1/cjv-mirror_windows_amd64.zip"
    Check-Request "mirror environment" @{} @{ CJV_VERSION = "v0.4.1"; CJV_MIRROR = "1" } "$gitcode/download/v0.4.1/cjv-mirror_windows_amd64.zip"
    Check-Request "custom official root" @{ Version = "v0.4.1" } @{ CJV_GITHUB_ROOT = "https://example.invalid/approved" } "https://example.invalid/approved/cjv_windows_amd64.zip"
    Check-Request "custom mirror root" @{ Version = "v0.4.1"; Mirror = $true } @{ CJV_GITCODE_ROOT = "https://example.invalid/mirror" } "https://example.invalid/mirror/cjv-mirror_windows_amd64.zip"
    Check-Request "update root wins" @{ Version = "v0.4.1"; Mirror = $true } @{ CJV_UPDATE_ROOT = "https://example.invalid/final"; CJV_GITCODE_ROOT = "https://example.invalid/mirror" } "https://example.invalid/final/cjv-mirror_windows_amd64.zip"
    foreach ($version in @("", "../v0.4.1", "v0.4.1?x=1", "--help", "v0.4.1 extra")) {
        Check-Request "invalid explicit version '$version'" @{ Version = $version } @{} "" -Invalid
    }
    Check-Request "invalid environment version" @{} @{ CJV_VERSION = "../v0.4.1" } "" -Invalid
    Write-Host "install.ps1 version parameters: 15 cases passed"
} finally {
    foreach ($name in $names) {
        [Environment]::SetEnvironmentVariable($name, $saved[$name], "Process")
    }
}
