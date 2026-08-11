<#
.SYNOPSIS
Remove AEM's Windows executable and user environment integration.

.PARAMETER RemoveData
Also deletes AEM_HOME and all AEM-managed runtimes, SDKs, caches, and state.
#>
[CmdletBinding()]
param(
    [string]$AemHome = $(if ($env:AEM_HOME) { $env:AEM_HOME } else { Join-Path $HOME '.aem' }),
    [switch]$RemoveData
)

$ErrorActionPreference = 'Stop'
$managedEnvironment = [ordered]@{
    AEM_HOME         = $AemHome
    JAVA_HOME        = Join-Path $AemHome 'current\java'
    ANDROID_HOME     = Join-Path $AemHome 'current\android'
    ANDROID_SDK_ROOT = Join-Path $AemHome 'current\android'
}
$pathEntries = @(
    (Join-Path $AemHome 'bin'),
    (Join-Path $AemHome 'current\node'),
    (Join-Path $AemHome 'current\java\bin'),
    (Join-Path $AemHome 'current\android\platform-tools'),
    (Join-Path $AemHome 'current\android\cmdline-tools\latest\bin')
)

$statePath = Join-Path $AemHome 'installer-environment.json'
if (Test-Path -LiteralPath $statePath) {
    $state = Get-Content -Raw -LiteralPath $statePath | ConvertFrom-Json
    foreach ($name in $managedEnvironment.Keys) {
        $previousValue = $state.PreviousEnvironment.$name
        [Environment]::SetEnvironmentVariable($name, $previousValue, 'User')
        if ($null -eq $previousValue) {
            Remove-Item -Path "Env:$name" -ErrorAction SilentlyContinue
        } else {
            Set-Item -Path "Env:$name" -Value $previousValue
        }
    }
    Remove-Item -Force $statePath
} else {
    foreach ($entry in $managedEnvironment.GetEnumerator()) {
        if ([Environment]::GetEnvironmentVariable($entry.Key, 'User') -eq $entry.Value) {
            [Environment]::SetEnvironmentVariable($entry.Key, $null, 'User')
        }
        Remove-Item -Path "Env:$($entry.Key)" -ErrorAction SilentlyContinue
    }
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$remaining = @($userPath -split ';' | Where-Object {
    $candidate = $_.TrimEnd('\\')
    -not ($pathEntries | Where-Object { $_.TrimEnd('\\') -ieq $candidate })
})
[Environment]::SetEnvironmentVariable('Path', ($remaining -join ';'), 'User')

Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $AemHome 'bin\aem.exe')
Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $AemHome 'bin')

if ($RemoveData) {
    $resolvedHome = [IO.Path]::GetFullPath($AemHome)
    if ($resolvedHome -eq [IO.Path]::GetPathRoot($resolvedHome)) {
        throw 'Refusing to remove an unsafe AEM_HOME path.'
    }
    Remove-Item -Recurse -Force -LiteralPath $resolvedHome
    Write-Host "Removed AEM and all managed data from $resolvedHome."
} else {
    Write-Host "Removed the AEM executable. Managed data in $AemHome was retained."
    Write-Host 'Use -RemoveData to delete managed runtimes and SDKs as well.'
}
