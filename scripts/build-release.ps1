# Shared native production build. The diagnostic variant keeps symbols only;
# it uses the same source, tags, resources and module inputs as the release.
param(
    [Parameter(Mandatory=$true)][string]$Package,
    [Parameter(Mandatory=$true)][string]$Output,
    [string]$Version = '0.0.0-dev',
    [string]$GoExecutable = 'go',
    [switch]$KeepSymbols
)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$contract = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'release-build.json') -Raw | ConvertFrom-Json
$outputPath = [IO.Path]::GetFullPath($Output)
if ($Version -notmatch '^[0-9A-Za-z.+-]+$') { throw 'Invalid build version.' }
$packagePath = [IO.Path]::GetFullPath((Join-Path $projectRoot $Package))
$cmdRoot = [IO.Path]::GetFullPath((Join-Path $projectRoot 'cmd')) + [IO.Path]::DirectorySeparatorChar
if (-not $packagePath.StartsWith($cmdRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Release package must be beneath cmd/.' }
$targetOS = (& $GoExecutable env GOOS).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve Go target OS.' }
$targetArch = (& $GoExecutable env GOARCH).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve Go target architecture.' }
$parent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\', '/')
$work = Join-Path $parent ('whaleisle-release-build-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $work | Out-Null
$resourceTarget = $null
try {
    $flags = @()
    if (-not $KeepSymbols) { $flags += @($contract.ldflags) }
    if ((Split-Path $packagePath -Leaf) -eq 'whaleisle') { $flags += "-X main.buildVersion=$Version" }
    if ($targetOS -eq 'windows' -and $contract.windowsGUI) { $flags += '-H windowsgui' }
    $buildArgs = @('build', '-mod=readonly', '-tags', ($contract.tags -join ','))
    if ($contract.trimpath) { $buildArgs += '-trimpath' }
    if ($flags.Count) { $buildArgs += @('-ldflags', ($flags -join ' ')) }
    if ($targetOS -eq 'windows') {
        $resource = Join-Path $work 'windows.syso'
        $savedOS, $savedArch = $env:GOOS, $env:GOARCH
        try {
            $env:GOOS = (& $GoExecutable env GOHOSTOS).Trim()
            $env:GOARCH = (& $GoExecutable env GOHOSTARCH).Trim()
            & $GoExecutable tool mygo winres -arch $targetArch -version $Version -out $resource $projectRoot
        } finally {
            if ($null -eq $savedOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $savedOS }
            if ($null -eq $savedArch) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $savedArch }
        }
        if ($LASTEXITCODE -ne 0) { throw 'Windows resource generation failed.' }
        # Go's archive packer opens .syso paths directly, bypassing -overlay.
        # Publish only this build's resource, refusing existing/concurrent input.
        $candidate = Join-Path $packagePath "mygo_windows_$targetArch.syso"
        $owned = [IO.File]::Open($candidate, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::Read)
        $resourceTarget = $candidate
        try {
            $source = [IO.File]::OpenRead($resource)
            try { $source.CopyTo($owned) } finally { $source.Dispose() }
        } finally { $owned.Dispose() }
    }
    $buildArgs += @('-o', $outputPath, $packagePath)
    Push-Location $projectRoot
    try { & $GoExecutable @buildArgs } finally { Pop-Location }
    if ($LASTEXITCODE -ne 0) { throw "go build failed (exit $LASTEXITCODE)" }
    if (-not $KeepSymbols -and $targetOS -eq 'windows' -and $targetArch -eq 'amd64' -and (Split-Path $packagePath -Leaf) -eq 'whaleisle') {
        $size = (Get-Item -LiteralPath $outputPath).Length
        Write-Host ('Production desktop: {0} bytes ({1:N3} MiB)' -f $size, ($size / 1MB))
    }
} finally {
    if ($resourceTarget) { Remove-Item -LiteralPath $resourceTarget -Force }
    if ([IO.Path]::GetFullPath((Split-Path $work -Parent)) -ne $parent -or (Split-Path $work -Leaf) -notmatch '^whaleisle-release-build-[a-f0-9]{32}$') { throw 'Unexpected release build workspace.' }
    # Only ordinary files written above exist here. Never clean package sources.
    foreach ($name in @('windows.syso')) {
        $file = Join-Path $work $name
        if (Test-Path -LiteralPath $file) { Remove-Item -LiteralPath $file -Force }
    }
    Remove-Item -LiteralPath $work -Force
}
