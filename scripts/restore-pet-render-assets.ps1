# Restores the pinned THA4 pet-render assets that must ship beside the shell.
#
# The binary model, ORT runtime and the two MIT license files are deliberately
# never committed (.gitignore: "Verified THA4 runtime is staged locally").
# internal/desktop/pet/assets/render-manifest.json is the single source of
# truth: this script takes each pinned URL, size and SHA256 from it, so the
# bytes restored here are exactly the bytes the executable accepts at run time
# (internal/desktop/pet/neural.go) and scripts/pet-render-assets.mjs stages.
#
#   pwsh -File scripts/restore-pet-render-assets.ps1                 # restore
#   pwsh -File scripts/restore-pet-render-assets.ps1 -Verify         # check only
#   pwsh -File scripts/restore-pet-render-assets.ps1 -Force          # re-download
#
# Without a restored directory the packaged pet silently degrades to the
# bundled rig (storelocale "missing or changed THA4 assets"), which is the
# regression this script exists to remove.
[CmdletBinding()]
param(
    [switch]$Verify,
    [switch]$Force,
    [string]$Destination
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$root = Split-Path -Parent $PSScriptRoot
if (-not $Destination) { $Destination = Join-Path $root 'runtime\pet-render-assets' }
$Destination = [System.IO.Path]::GetFullPath($Destination)
$manifestPath = Join-Path $root 'internal\desktop\pet\assets\render-manifest.json'
$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json

function Test-Pinned([string]$path, [object]$file) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { return $false }
    if ((Get-Item -LiteralPath $path).Length -ne $file.size) { return $false }
    return (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() -eq $file.sha256
}

$verified = 0
$failures = New-Object System.Collections.Generic.List[string]
foreach ($file in $manifest.files) {
    $target = Join-Path $Destination ($file.path -replace '/', [System.IO.Path]::DirectorySeparatorChar)
    if ((Test-Pinned $target $file) -and -not $Force) {
        $verified++
        continue
    }
    if ($Verify) {
        $failures.Add("$($file.path): missing or changed")
        continue
    }
    $parent = Split-Path -Parent $target
    New-Item -ItemType Directory -Force -Path $parent | Out-Null
    $partial = "$target.partial"
    Remove-Item -LiteralPath $partial -ErrorAction SilentlyContinue
    try {
        Invoke-WebRequest -Uri $file.url -OutFile $partial -TimeoutSec 600 -UseBasicParsing
        if (-not (Test-Pinned $partial $file)) {
            $length = if (Test-Path -LiteralPath $partial) { (Get-Item -LiteralPath $partial).Length } else { 0 }
            throw "downloaded $($file.path) is $length bytes and does not match the pinned $($file.size) bytes / SHA256"
        }
        Move-Item -LiteralPath $partial -Destination $target -Force
        $verified++
        Write-Host "restored $($file.path)"
    } catch {
        Remove-Item -LiteralPath $partial -ErrorAction SilentlyContinue
        $failures.Add("$($file.path): $($_.Exception.Message)")
    }
}

if ($failures.Count) {
    Write-Host "THA4 pet-render assets are incomplete in $Destination"
    foreach ($failure in $failures) { Write-Host "  $failure" }
    exit 1
}
Write-Host "Verified $verified/$($manifest.files.Count) pinned THA4 files in $Destination (commit $($manifest.commit))"
exit 0
