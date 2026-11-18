# Run synchronous QA with a disposable workspace, keeping output/logs explicitly
# written outside it. Example: .\scripts\with-build-workspace.ps1 { go test ./internal/... }
param([Parameter(Mandatory=$true, Position=0)][scriptblock]$Command)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'env.ps1')
$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\')
$work = Join-Path $tempRoot ('whaleisle-build-' + [guid]::NewGuid().ToString('N'))
$saved = @{}
# A cold-cache command may point these variables into $work. Restore them too,
# so the next command cannot recreate a just-reclaimed private cache directory.
foreach ($name in @('TEMP', 'TMP', 'GOTMPDIR', 'GOPATH', 'GOCACHE', 'GOMODCACHE', 'npm_config_cache', 'PSModuleAnalysisCachePath')) { $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
New-Item -ItemType Directory -Path $work | Out-Null
try {
    # An isolated desktop may change LOCALAPPDATA inside Command. Resolve Go's
    # defaults while still in the caller's environment, or every such run gets
    # a new compile cache inside its disposable application profile.
    if ((-not $env:GOCACHE -or -not $env:GOMODCACHE) -and (Get-Command go -ErrorAction SilentlyContinue)) {
        $goCacheJSON = & go env -json GOCACHE GOMODCACHE
        if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve the caller Go cache paths.' }
        $goCaches = $goCacheJSON | ConvertFrom-Json
        foreach ($name in @('GOCACHE', 'GOMODCACHE')) {
            if (-not [Environment]::GetEnvironmentVariable($name, 'Process')) {
                [Environment]::SetEnvironmentVariable($name, [string]$goCaches.$name, 'Process')
            }
        }
    }
    foreach ($name in @('TEMP', 'TMP', 'GOTMPDIR')) { [Environment]::SetEnvironmentVariable($name, $work, 'Process') }
    $global:LASTEXITCODE = 0
    & $Command $work
    if ($LASTEXITCODE -ne 0) { throw "Build/QA command failed (exit $LASTEXITCODE)" }
} finally {
    foreach ($name in $saved.Keys) {
        if ($null -eq $saved[$name]) { Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue }
        else { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
    }
    $target = [IO.Path]::GetFullPath($work)
    if ((Split-Path $target -Parent) -ne $tempRoot -or (Split-Path $target -Leaf) -notmatch '^whaleisle-build-[a-f0-9]{32}$') {
        throw "Unexpected owned workspace: $target"
    }
    $active = Get-CimInstance Win32_Process | Where-Object { $_.ProcessId -ne $PID -and (($_.ExecutablePath + ' ' + $_.CommandLine).IndexOf($target, [StringComparison]::OrdinalIgnoreCase) -ge 0) }
    if ($active) { Write-Warning "Owned workspace still referenced by a child process; retained: $target" }
    else {
        # Reuse the same no-follow removal implementation with an owned parent.
        # The disposable directory is never an input/source or user profile.
        & node (Join-Path $PSScriptRoot 'build-storage.mjs') workspace $target $tempRoot
        if ($LASTEXITCODE -ne 0) { throw "Owned workspace cleanup failed: $target" }
    }
}
