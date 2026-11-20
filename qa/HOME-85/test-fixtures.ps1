#Requires -Version 5.1
<#
Window-free proof that the live fixtures are what the scenarios need.

The scenarios themselves need Windows and a quiet machine. The fixtures they
depend on do not: a seed config, a vendor tree that is present but is not this
build's runtime, an empty vendor tree, and a wall of occupied ports can all be
built and checked here. A scenario that fails because its fixture was wrong
would spend the operator's window on our own mistake.

    pwsh -NoProfile -File qa/HOME-85/test-fixtures.ps1

Exit 0 only when every fixture matches what the launch code expects.
#>
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'run-all.ps1')

$root = Join-Path ([IO.Path]::GetTempPath()) ('mygo-fixture-check-' + [guid]::NewGuid().ToString('n').Substring(0, 8))
New-Item -ItemType Directory -Path $root -Force | Out-Null
$failures = 0

function Assert-Fixture {
    param([Parameter(Mandatory)][bool]$Condition, [Parameter(Mandatory)][string]$Name, [string]$Detail = '')
    if ($Condition) {
        Write-Host "fixture $Name : PASS$Detail"
    } else {
        Write-Host "fixture $Name : FAIL$Detail"
        $script:failures++
    }
}

try {
    # The seed must be a readable config with the keys the app loads.
    $seed = New-SeedConfig -Name 'fixture-check' -Config @{ locale = 'zh'; autoStartDesktop = $true; workspace = $root }
    $parsed = Get-Content -LiteralPath $seed -Raw | ConvertFrom-Json
    Assert-Fixture -Name 'seed config' -Condition ($parsed.locale -eq 'zh' -and $parsed.autoStartDesktop -eq $true -and $parsed.workspace -eq $root) -Detail " $seed"

    # The mismatch tree must look like a staged harness to the resolver and carry
    # an incomplete session-event registry, which is the refusal the copy calls a
    # version mismatch.
    $mismatch = New-MismatchVendor
    $harness = Join-Path $mismatch 'deepseek-harness'
    $binJS = Join-Path $harness 'apps\cli\lib\bin.js'
    $web = Join-Path $harness 'apps\web\dist\index.html'
    $reader = Join-Path $harness 'packages\core\session\lib\index.js'
    Assert-Fixture -Name 'mismatch vendor tree' -Condition ((Test-Path -LiteralPath $binJS) -and (Test-Path -LiteralPath $web) -and (Test-Path -LiteralPath $reader)) -Detail " $harness"
    $readerText = Get-Content -LiteralPath $reader -Raw
    Assert-Fixture -Name 'mismatch registry is incomplete' -Condition ($readerText -match 'KNOWN_SESSION_EVENT_TYPES\s*=\s*new\s+Set\(\s*\[' -and $readerText -notmatch 'job/settled') -Detail " $(Split-Path -Leaf $reader)"
    $noManifest = -not (Test-Path -LiteralPath (Join-Path $mismatch 'current-host-build.json'))
    $noVisual = -not (Test-Path -LiteralPath (Join-Path $harness 'packages\client\ui-visual-replies\package.json'))
    $noStaged = -not (Test-Path -LiteralPath (Join-Path $mismatch 'session-runtime.json'))
    Assert-Fixture -Name 'mismatch vendor skips the staged paths' -Condition ($noManifest -and $noVisual -and $noStaged)

    # The empty vendor must not look like a harness at all.
    $empty = New-EmptyVendor
    Assert-Fixture -Name 'empty vendor has no harness' -Condition (-not (Test-Path -LiteralPath (Join-Path $empty 'deepseek-harness')))

    # The wall must actually hold every port the launcher scans from 3080.
    $wall = Start-PortWall -First 3080 -Count 50
    try {
        $held = 0
        foreach ($listener in $wall) { if ($listener.Server.IsBound) { $held++ } }
        Assert-Fixture -Name 'port wall' -Condition ($held -eq 50) -Detail " $held/50 bound"
        $blocked = $false
        try {
            $probe = New-Object System.Net.Sockets.TcpListener ([System.Net.IPAddress]::Loopback), 3080
            $probe.Server.ExclusiveAddressUse = $true
            $probe.Start()
            $probe.Stop()
        } catch {
            $blocked = $true
        }
        Assert-Fixture -Name 'port wall is exclusive' -Condition $blocked
    } finally {
        Stop-PortWall -Listeners $wall
    }
    $free = $false
    try {
        $probe = New-Object System.Net.Sockets.TcpListener ([System.Net.IPAddress]::Loopback), 3080
        $probe.Start()
        $probe.Stop()
        $free = $true
    } catch { }
    Assert-Fixture -Name 'wall releases its ports' -Condition $free
} finally {
    Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue
}

if ($failures -gt 0) {
    Write-Host "RESULT: FAIL ($failures fixture(s))"
    exit 1
}
Write-Host 'RESULT: PASS'
exit 0
