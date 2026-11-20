# Windows local entry point; source and package output stay under the caller's control.
[CmdletBinding(DefaultParameterSetName='Verify')]
param(
    [string]$Worktree = (Get-Location).Path,
    [Parameter(ParameterSetName='Verify')][switch]$Verify,
    [Parameter(Mandatory, ParameterSetName='Package')][string]$OutputRoot,
    [Parameter(Mandatory, ParameterSetName='Package')][ValidateRange(1, 475)][double]$PeakGiB,
    [Parameter(ParameterSetName='Package')][string]$GoExecutable = 'go'
)

$ErrorActionPreference = 'Stop'
function Invoke-Checked([string]$Exe, [string[]]$Arguments) {
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Exe failed (exit $LASTEXITCODE)." }
}

$repo = (Get-Item -LiteralPath $Worktree).FullName
$top = [string](Invoke-Checked 'git' @('-C', $repo, 'rev-parse', '--show-toplevel'))
if ([IO.Path]::GetFullPath($top) -ine $repo) { throw 'Use the root of a MyGo worktree.' }
$scripts = Join-Path $repo 'scripts'
$packager = Join-Path $scripts 'package.ps1'
if (-not (Test-Path -LiteralPath $packager -PathType Leaf)) { throw 'This worktree has no MyGo package.ps1.' }

# Repository-local .git/config is shared by Git linked worktrees, not by unrelated projects.
$paths = @(& git -C $repo config --local --get-all mygo.sharedRuntimeRoot)
if ($LASTEXITCODE -ne 0 -or $paths.Count -ne 1 -or -not [IO.Path]::IsPathFullyQualified($paths[0])) { throw 'Set one absolute mygo.sharedRuntimeRoot in the MyGo repository-local Git config.' }
$runtime = (Get-Item -LiteralPath $paths[0]).FullName
$vendor = Join-Path $runtime 'vendor'
$node = Join-Path $runtime 'node\node.exe'
$manifest = Get-Content -Raw -LiteralPath (Join-Path $vendor 'host-session-runtime.json') | ConvertFrom-Json
if (-not (Test-Path -LiteralPath $node -PathType Leaf)) { throw "Shared runtime has no bundled Node: $node" }

if ($PSCmdlet.ParameterSetName -eq 'Package') {
    if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -eq '1') { throw 'Use the original package.ps1 for shell-only builds.' }
    $out = [IO.Path]::GetFullPath($OutputRoot)
    $root = [IO.Path]::GetPathRoot($out)
    if ($root -ine 'C:\') { throw 'This local entry point budgets the C: volume only. For another volume, budget that build and use package.ps1 directly.' }
    if ($out -ieq $runtime -or $out.StartsWith($runtime + '\', [StringComparison]::OrdinalIgnoreCase)) { throw 'Package output cannot be inside the shared runtime.' }
    for ($dir=$out; $dir; $dir=(Split-Path -Parent $dir)) {
        if (Test-Path -LiteralPath $dir) {
            $item = Get-Item -Force -LiteralPath $dir
            if (-not $item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw "Output path is redirected or not a directory: $dir" }
        }
    }
    if ((Test-Path -LiteralPath $out) -and (Get-ChildItem -LiteralPath $out -Force | Select-Object -First 1)) { throw 'Choose a new or empty package output directory.' }
    $free = (Get-PSDrive C).Free / 1GB
    if ($free - $PeakGiB -lt 50) {
        throw ('C: has {0:N2} GiB; need {1:N2} GiB free to keep 50 GiB after the estimated peak. No build started.' -f $free, (50 + $PeakGiB))
    }
    $cmd = Get-Command $packager
    if (-not $cmd.Parameters.ContainsKey('CleanStage') -or -not $cmd.Parameters.ContainsKey('RuntimeRoot')) { throw 'This checkout predates the shared-runtime packaging contract.' }
    # The packager rechecks the source, builds in its fresh stage, and only cleans it after finalizing the installer and metadata.
    Push-Location -LiteralPath $repo
    try { & $packager -RuntimeRoot $runtime -OutputRoot $out -CleanStage -GoExecutable $GoExecutable }
    finally { Pop-Location }
    return
}

Invoke-Checked $node @((Join-Path $scripts 'session-runtime.mjs'), 'verify', $vendor) | Out-Null
$composition = Invoke-Checked $node @((Join-Path $scripts 'session-runtime-composition.mjs'), 'verify', $runtime)
[pscustomobject]@{
    Worktree = $repo
    RuntimeRoot = $runtime
    Host = $manifest.cliVersion
    HostPin = $manifest.harnessPin.sha
    Composition = $composition -join "`n"
}
