<#
.SYNOPSIS
Download and install a released AEM binary for the current Windows user.

.DESCRIPTION
Downloads the latest AEM release (or a selected version), installs aem.exe in
AEM_HOME\\bin, and sets persistent user environment
variables for AEM, Node, Java, and the Android SDK. Restart terminals after
running it so they receive the updated environment.
#>
[CmdletBinding()]
param(
    [string]$AemHome = $(if ($env:AEM_HOME) { $env:AEM_HOME } else { Join-Path $HOME '.aem' }),
    [string]$Version = 'latest'
)

$ErrorActionPreference = 'Stop'
$githubRepository = 'firmanjml/aem-go'

if ($Version -ne 'latest' -and $Version -notmatch '^[0-9A-Za-z._-]+$') {
    throw "Invalid version: $Version"
}

$binDir = Join-Path $AemHome 'bin'
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$targetBinary = Join-Path $binDir 'aem.exe'
$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ('aem-install-' + [guid]::NewGuid().ToString())

try {
    if ($Version -eq 'latest') {
        $releaseApi = "https://api.github.com/repos/$githubRepository/releases/latest"
    }
    else {
        $releaseApi = "https://api.github.com/repos/$githubRepository/releases/tags/v$Version"
    }
    $release = Invoke-RestMethod -Uri $releaseApi -Headers @{ Accept = 'application/vnd.github+json' }
    $architecture = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
        'X64' { 'amd64' }
        'Arm64' { 'arm64' }
        default { throw "Unsupported CPU architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
    }
    $archiveVersion = $release.tag_name -replace '^v', ''
    $archiveName = "aem_${archiveVersion}_windows_${architecture}.zip"
    $archive = $release.assets | Where-Object { $_.name -eq $archiveName } | Select-Object -First 1
    $checksums = $release.assets | Where-Object { $_.name -eq 'checksums.txt' } | Select-Object -First 1
    if (-not $archive -or -not $checksums) {
        throw "Release $($release.tag_name) does not contain $archiveName and checksums.txt."
    }

    New-Item -ItemType Directory -Force -Path $temporaryDirectory | Out-Null
    $archivePath = Join-Path $temporaryDirectory $archiveName
    $checksumsPath = Join-Path $temporaryDirectory 'checksums.txt'
    Write-Host "Downloading AEM $($release.tag_name) for windows/$architecture..."
    Invoke-WebRequest -Uri $archive.browser_download_url -OutFile $archivePath
    Invoke-WebRequest -Uri $checksums.browser_download_url -OutFile $checksumsPath
    $checksumLine = Get-Content -LiteralPath $checksumsPath |
        Where-Object { $_ -match "^\s*[0-9a-fA-F]{64}\s+\*?$([regex]::Escape($archiveName))\s*$" } |
        Select-Object -First 1
    if (-not $checksumLine) { throw "No checksum found for $archiveName." }
    $expectedChecksum = [regex]::Match($checksumLine, '^[0-9a-fA-F]{64}').Value
    $actualChecksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash
    if ($actualChecksum -ine $expectedChecksum) { throw "Checksum verification failed for $archiveName." }

    $unpackDirectory = Join-Path $temporaryDirectory 'unpacked'
    Expand-Archive -LiteralPath $archivePath -DestinationPath $unpackDirectory -Force
    $releasedBinary = Join-Path $unpackDirectory 'aem.exe'
    if (-not (Test-Path -LiteralPath $releasedBinary -PathType Leaf)) {
        throw 'Release archive did not contain aem.exe.'
    }
    Copy-Item -LiteralPath $releasedBinary -Destination $targetBinary -Force
}
finally {
    Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
}

$managedEnvironment = [ordered]@{
    AEM_HOME         = $AemHome
    JAVA_HOME        = Join-Path $AemHome 'current\java'
    ANDROID_HOME     = Join-Path $AemHome 'current\android'
    ANDROID_SDK_ROOT = Join-Path $AemHome 'current\android'
}

$statePath = Join-Path $AemHome 'installer-environment.json'
if (-not (Test-Path -LiteralPath $statePath)) {
    $previous = [ordered]@{}
    foreach ($name in $managedEnvironment.Keys) {
        $previous[$name] = [Environment]::GetEnvironmentVariable($name, 'User')
    }
    [ordered]@{ Version = 1; PreviousEnvironment = $previous } |
        ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath $statePath
}

foreach ($entry in $managedEnvironment.GetEnumerator()) {
    [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'User')
    Set-Item -Path "Env:$($entry.Key)" -Value $entry.Value
}

$pathEntries = @(
    $binDir,
    (Join-Path $AemHome 'current\node'),
    (Join-Path $AemHome 'current\java\bin'),
    (Join-Path $AemHome 'current\android\platform-tools'),
    (Join-Path $AemHome 'current\android\cmdline-tools\latest\bin')
)
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$currentEntries = @($userPath -split ';' | Where-Object { $_ })
foreach ($entry in $pathEntries) {
    if (-not ($currentEntries | Where-Object { $_.TrimEnd('\\') -ieq $entry.TrimEnd('\\') })) {
        $currentEntries += $entry
    }
}
$newPath = $currentEntries -join ';'
[Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
$env:Path = $newPath + ';' + $env:Path

Write-Host "Installed AEM to $targetBinary."
Write-Host 'Open a new terminal to use the persistent environment variables.'
