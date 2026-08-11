<#
.SYNOPSIS
Build and install AEM from the current local source checkout.

.DESCRIPTION
Compiles the repository containing this script, installs aem.exe in AEM_HOME\bin,
and configures the current user's AEM environment. It does not download an AEM
release or clone from source control.
#>
[CmdletBinding()]
param(
    [string]$AemHome = $(if ($env:AEM_HOME) { $env:AEM_HOME } else { Join-Path $HOME '.aem' })
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go is required to build AEM from source.'
}
if (-not (Test-Path -LiteralPath (Join-Path $repoRoot 'go.mod') -PathType Leaf)) {
    throw "Could not find go.mod in the AEM source tree: $repoRoot"
}

$binDir = Join-Path $AemHome 'bin'
$targetBinary = Join-Path $binDir 'aem.exe'
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$temporaryBinary = Join-Path ([System.IO.Path]::GetTempPath()) ('aem-build-' + [guid]::NewGuid().ToString() + '.exe')

try {
    Write-Host "Building AEM from $repoRoot..."
    Push-Location $repoRoot
    try {
        & go build -trimpath -o $temporaryBinary .
        if ($LASTEXITCODE -ne 0) {
            throw "Go build failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        Pop-Location
    }
    Copy-Item -LiteralPath $temporaryBinary -Destination $targetBinary -Force
}
finally {
    Remove-Item -LiteralPath $temporaryBinary -Force -ErrorAction SilentlyContinue
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

Write-Host "Built and installed AEM to $targetBinary."
Write-Host 'Open a new terminal to use the persistent environment variables.'
